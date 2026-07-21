package database

// All SELECT queries should check for:
// 1. Basic input constraints/validation (type conversion should be done prior to)
// 2. Check context errors on row iteration
// 3. Get database connection from app state
// 4. Check for sql.ErrNoRows and return nil if error exists
// 5. Return any other errors

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/csv"
	"errors"
	"fmt"
	"strings"
	"time"
	"uit-toolbox/config"
	"uit-toolbox/types"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type onlineClientData struct {
	ClientUUID uuid.UUID
	LastHeard  *time.Time
}

type Select interface {
	CheckTwoFactorCode(ctx context.Context, twoFactorCode *string) (string, error)
	CheckAuthCredentials(ctx context.Context, username *string, password *string) (bool, *string, error)
}

type SelectRepo struct {
	DB *sql.DB
}

const approxClientCount = 600

func NewSelectRepo() (Select, error) {
	db, err := config.GetDatabaseConn()
	if err != nil {
		return nil, fmt.Errorf("error getting database connection in NewSelectRepo: %w", err)
	}
	return &SelectRepo{DB: db}, nil
}

var _ Select = (*SelectRepo)(nil)

func GetClientUUIDByTag(ctx context.Context, pgxPool *pgxpool.Pool, tagnumber int64) (clientUUID uuid.UUID, err error) {
	if err := types.IsTagnumberInt64Valid(&tagnumber); err != nil {
		return uuid.Nil, fmt.Errorf("tagnumber is nil or invalid: %w", err)
	}
	if pgxPool == nil {
		pgxPool, err = config.GetPGXPool()
		if err != nil {
			return uuid.Nil, fmt.Errorf("%w: %w", types.DatabaseConnError, err)
		}
	}

	const sqlCode = `
		SELECT uuid
		FROM ids
		WHERE tagnumber = $1
	;`

	row := pgxPool.QueryRow(ctx, sqlCode,
		toNullInt64(tagnumber),
	)
	if err := row.Scan(
		&clientUUID,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return uuid.Nil, fmt.Errorf("%w: no client found for tagnumber '%d'", types.DatabaseQueryError, tagnumber)
		}
		return uuid.Nil, fmt.Errorf("%w: %w", types.DatabaseQueryError, err)
	}

	return clientUUID, nil
}

func GetClientUUIDBySerial(ctx context.Context, pgxPool *pgxpool.Pool, systemSerial string) (clientUUID uuid.UUID, err error) {
	if strings.TrimSpace(systemSerial) == "" {
		return uuid.Nil, fmt.Errorf("%w: systemSerial is empty", types.InvalidStructureError)
	}
	if pgxPool == nil {
		pgxPool, err = config.GetPGXPool()
		if err != nil {
			return uuid.Nil, fmt.Errorf("%w: %w", types.DatabaseConnError, err)
		}
	}

	const sqlCode = `
		SELECT uuid
		FROM ids
		WHERE system_serial = $1
	;`

	row := pgxPool.QueryRow(ctx, sqlCode,
		toNullString(systemSerial),
	)
	if err := row.Scan(
		&clientUUID,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return uuid.Nil, fmt.Errorf("%w: no client found for system serial '%s'", types.DatabaseQueryError, systemSerial)
		}
		return uuid.Nil, fmt.Errorf("%w: %w", types.DatabaseQueryError, err)
	}

	return clientUUID, nil
}

func SelectAllIDs(ctx context.Context) ([]types.ClientLookupRow, error) {
	pgxPool, err := config.GetPGXPool()
	if err != nil {
		return nil, fmt.Errorf("%w: %w", types.DatabaseConnError, err)
	}
	const sqlQuery = `
		SELECT 
			ids.tagnumber,
			ids.system_serial,
			ids.uuid,
			locations.time AS "last_inventory_entry"
		FROM 
			ids
		INNER JOIN locations ON ids.uuid = locations.client_uuid
		GROUP BY 
			ids.uuid,
			ids.tagnumber, 
			ids.system_serial, 
			locations.time
		ORDER BY 
			locations.time DESC NULLS LAST, 
			ids.tagnumber ASC NULLS LAST, 
			ids.system_serial DESC NULLS LAST
	;`

	rows, err := pgxPool.Query(ctx, sqlQuery)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", types.DatabaseQueryError, err)
	}
	defer rows.Close()

	var globalLookupRow []types.ClientLookupRow
	for rows.Next() {
		if ctx.Err() != nil {
			return nil, fmt.Errorf("%w: %w", types.DatabaseRowIterationError, ctx.Err())
		}
		var tag types.ClientLookupRow
		scanErr := rows.Scan(
			&tag.Tagnumber,
			&tag.SystemSerial,
			&tag.ClientUUID,
			&tag.LastInventoryEntry,
		)
		if scanErr != nil {
			return nil, fmt.Errorf("%w: %w", types.DatabaseRowScanError, scanErr)
		}
		globalLookupRow = append(globalLookupRow, tag)
	}
	if rows.Err() != nil {
		return nil, fmt.Errorf("%w: %w", types.DatabaseRowIterationError, rows.Err())
	}
	if len(globalLookupRow) == 0 {
		return nil, nil
	}

	return globalLookupRow, nil
}

func ClientIDLookup(ctx context.Context, tag *int64, serial *string) (*types.ClientLookupRow, error) {
	var tagErr error
	var serialErr error
	whereClause := "WHERE ids.uuid IS NOT NULL "
	whereArgs := make([]any, 0, 2)
	if err := types.IsTagnumberInt64Valid(tag); err != nil {
		tagErr = fmt.Errorf("tagnumber is nil or invalid: %w", err)
	} else {
		whereArgs = append(whereArgs, *tag)
		whereClause += fmt.Sprintf("AND ids.tagnumber = $%d ", len(whereArgs))
	}

	if serial == nil || strings.TrimSpace(*serial) == "" {
		serialErr = fmt.Errorf("system serial is nil or empty")
	} else {
		trimmedSerial := strings.TrimSpace(*serial)
		whereArgs = append(whereArgs, trimmedSerial)
		whereClause += fmt.Sprintf("AND ids.system_serial = $%d ", len(whereArgs))
	}

	if tagErr != nil && serialErr != nil {
		return nil, fmt.Errorf("%s", "both tagnumber and system serial are invalid/empty")
	}

	pgxPool, err := config.GetPGXPool()
	if err != nil {
		return nil, fmt.Errorf("%w: %w", types.DatabaseConnError, err)
	}

	sqlQuery := fmt.Sprintf(`
		SELECT 
			tagnumber, 
			system_serial,
			uuid
		FROM 
			ids 
		%s
		ORDER BY 
			time DESC NULLS LAST 
		LIMIT 1
	;`, whereClause)

	var clientLookup types.ClientLookupRow
	row := pgxPool.QueryRow(ctx, sqlQuery, whereArgs...)
	rowScanErr := row.Scan(
		&clientLookup.Tagnumber,
		&clientLookup.SystemSerial,
		&clientLookup.ClientUUID,
	)
	if rowScanErr != nil {
		if errors.Is(rowScanErr, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("%w: %v", types.DatabaseRowScanError, rowScanErr)
	}
	return &clientLookup, nil
}

func GetAllDepartments(ctx context.Context) ([]types.AllDepartmentsRow, error) {
	pgxPool, err := config.GetPGXPool()
	if err != nil {
		return nil, fmt.Errorf("%w: %w", types.DatabaseConnError, err)
	}

	const sqlQuery = `
		SELECT 
			static_department_info.department_name, 
			static_department_info.department_name_formatted, 
			static_department_info.department_sort_order,
			COALESCE(static_organizations.organization_name, '') AS organization_name,
			COALESCE(static_organizations.organization_name_formatted, '') AS organization_name_formatted,
			COALESCE(static_organizations.organization_sort_order, 101) AS organization_sort_order,
			COUNT(*) AS client_count
		FROM 
			static_department_info
		LEFT JOIN locations ON static_department_info.department_name = locations.department_name
		LEFT JOIN static_organizations ON static_department_info.organization_name = static_organizations.organization_name
		GROUP BY
			static_department_info.department_name,
			static_department_info.department_name_formatted,
			static_department_info.department_sort_order,
			static_organizations.organization_name,
			static_organizations.organization_name_formatted,
			static_organizations.organization_sort_order
		ORDER BY 
			static_organizations.organization_sort_order, 
			static_department_info.department_sort_order
	;`

	rows, err := pgxPool.Query(ctx, sqlQuery)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", types.DatabaseQueryError, err)
	}
	defer rows.Close()

	allDepartments := make([]types.AllDepartmentsRow, 0, 10)
	for rows.Next() {
		if ctx.Err() != nil {
			return nil, fmt.Errorf("%w: %w", types.DatabaseRowIterationError, ctx.Err())
		}
		deptRow := new(types.AllDepartmentsRow)
		rowScanErr := rows.Scan(
			&deptRow.DepartmentName,
			&deptRow.DepartmentNameFormatted,
			&deptRow.DepartmentSortOrder,
			&deptRow.OrganizationName,
			&deptRow.OrganizationNameFormatted,
			&deptRow.OrganizationSortOrder,
			&deptRow.ClientCount,
		)
		if rowScanErr != nil {
			return nil, fmt.Errorf("%w: %v", types.DatabaseRowScanError, rowScanErr)
		}
		allDepartments = append(allDepartments, *deptRow)
	}
	if rows.Err() != nil {
		return nil, fmt.Errorf("%w: %w", types.DatabaseRowIterationError, rows.Err())
	}
	if len(allDepartments) == 0 {
		return nil, nil
	}

	return allDepartments, nil
}

func GetAllDomains(ctx context.Context) ([]types.AllDomainsRow, error) {
	pgxPool, err := config.GetPGXPool()
	if err != nil {
		return nil, fmt.Errorf("%w: %w", types.DatabaseConnError, err)
	}

	const sqlQuery = `
		SELECT 
			domain_name, 
			domain_name_formatted,
			domain_sort_order,
			COUNT(*) AS "client_count"
		FROM static_ad_domains
		LEFT JOIN locations ON static_ad_domains.domain_name = locations.ad_domain
		GROUP BY
			static_ad_domains.domain_name,
			static_ad_domains.domain_name_formatted,
			static_ad_domains.domain_sort_order
		ORDER BY 
			domain_sort_order NULLS LAST
	;`

	rows, err := pgxPool.Query(ctx, sqlQuery)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", types.DatabaseQueryError, err)
	}
	defer rows.Close()

	var allDomains []types.AllDomainsRow
	for rows.Next() {
		if ctx.Err() != nil {
			return nil, fmt.Errorf("%w: %w", types.DatabaseRowIterationError, ctx.Err())
		}
		var domainRow types.AllDomainsRow
		rowScanErr := rows.Scan(
			&domainRow.DomainName,
			&domainRow.DomainNameFormatted,
			&domainRow.DomainSortOrder,
			&domainRow.ClientCount,
		)
		if rowScanErr != nil {
			return nil, fmt.Errorf("%w: %v", types.DatabaseRowScanError, rowScanErr)
		}
		allDomains = append(allDomains, domainRow)
	}
	if rows.Err() != nil {
		return nil, fmt.Errorf("%w: %w", types.DatabaseRowIterationError, rows.Err())
	}
	if len(allDomains) == 0 {
		return nil, nil
	}

	return allDomains, nil
}

func SelectAllManufacturersAndModels(ctx context.Context) ([]types.AllManufacturersAndModelsRow, error) {
	pgxPool, err := config.GetPGXPool()
	if err != nil {
		return nil, fmt.Errorf("%w: %w", types.DatabaseConnError, err)
	}

	const sqlQuery = `
		SELECT 
			system_manufacturer, 
			system_model, 
			COUNT(*) AS "system_model_count"
		FROM hardware_data 
		WHERE 
			system_manufacturer IS NOT NULL 
			AND system_model IS NOT NULL
		GROUP BY 
			system_manufacturer, 
			system_model 
		ORDER BY 
			system_manufacturer ASC, 
			system_model ASC
	;`

	rows, err := pgxPool.Query(ctx, sqlQuery)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", types.DatabaseQueryError, err)
	}
	defer rows.Close()

	var allManufacturersAndModels []types.AllManufacturersAndModelsRow
	for rows.Next() {
		if ctx.Err() != nil {
			return nil, fmt.Errorf("%w: %w", types.DatabaseRowIterationError, ctx.Err())
		}
		var manufacturerModelRow types.AllManufacturersAndModelsRow
		rowScanErr := rows.Scan(
			&manufacturerModelRow.SystemManufacturer,
			&manufacturerModelRow.SystemModel,
			&manufacturerModelRow.SystemModelCount)
		if rowScanErr != nil {
			return nil, fmt.Errorf("%w: %v", types.DatabaseRowScanError, rowScanErr)
		}
		allManufacturersAndModels = append(allManufacturersAndModels, manufacturerModelRow)
	}
	if rows.Err() != nil {
		return nil, fmt.Errorf("%w: %w", types.DatabaseRowIterationError, rows.Err())
	}
	if len(allManufacturersAndModels) == 0 {
		return nil, nil
	}

	manufacturerCountMap := make(map[string]int64, len(allManufacturersAndModels))
	for _, row := range allManufacturersAndModels {
		if row.SystemManufacturer == nil || row.SystemModelCount == nil {
			continue
		}
		manufacturerCountMap[*row.SystemManufacturer] += *row.SystemModelCount
	}

	for i := range allManufacturersAndModels {
		if allManufacturersAndModels[i].SystemManufacturer == nil {
			allManufacturersAndModels[i].SystemManufacturerCount = nil
			continue
		}
		count := manufacturerCountMap[*allManufacturersAndModels[i].SystemManufacturer]
		allManufacturersAndModels[i].SystemManufacturerCount = &count
	}

	return allManufacturersAndModels, nil
}

func (repo *SelectRepo) CheckTwoFactorCode(ctx context.Context, twoFactorCode *string) (string, error) {
	if twoFactorCode == nil || strings.TrimSpace(*twoFactorCode) == "" {
		return "", fmt.Errorf("twoFactorCode is empty")
	}

	if ctx.Err() != nil {
		return "", fmt.Errorf("context error: %w", ctx.Err())
	}

	const sqlQuery = `SELECT two_factor_code FROM logins WHERE two_factor_code = $1 LIMIT 1;`

	var dbCode string
	row := repo.DB.QueryRowContext(ctx, sqlQuery, ptrToNullString(twoFactorCode))
	if err := row.Scan(&dbCode); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", nil
		}
		return "", fmt.Errorf("%w: %w", types.DatabaseRowScanError, err)
	}

	return dbCode, nil
}

func (repo *SelectRepo) CheckAuthCredentials(ctx context.Context, username *string, password *string) (bool, *string, error) {
	if username == nil || password == nil || strings.TrimSpace(*username) == "" || strings.TrimSpace(*password) == "" {
		return false, nil, fmt.Errorf("username or password is empty/nil")
	}

	const sqlQuery = `SELECT password FROM logins WHERE username = $1 LIMIT 1;`

	var dbBcryptHash sql.NullString
	row := repo.DB.QueryRowContext(ctx, sqlQuery, ptrToNullString(username))
	if err := row.Scan(&dbBcryptHash); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil, fmt.Errorf("%w: %w", types.DatabaseRowScanError, err)
		}
		return false, nil, fmt.Errorf("%w: %w", types.DatabaseRowScanError, err)
	}

	if !dbBcryptHash.Valid || strings.TrimSpace(dbBcryptHash.String) == "" {
		return false, nil, nil
	}

	return true, &dbBcryptHash.String, nil
}

func SelectIsClientJobAvailable(ctx context.Context, tag *int64) (*bool, error) {
	if err := types.IsTagnumberInt64Valid(tag); err != nil {
		return nil, fmt.Errorf("%w: %w", types.DatabaseRowScanError, err)
	}

	dbConn, err := config.GetDatabaseConn()
	if err != nil {
		return nil, fmt.Errorf("%w: %w", types.DatabaseConnError, err)
	}

	const sqlQuery = `
	SELECT 
		(CASE 
			WHEN (job_queue.job_queued = FALSE AND job_queue.job_name IS NULL) THEN TRUE
			ELSE FALSE
		END) AS "job_available"
	FROM 
		job_queue
	WHERE 
		job_queue.client_uuid = (SELECT uuid FROM ids WHERE tagnumber = $1 ORDER BY time DESC LIMIT 1)`

	var jobAvailable bool
	row := dbConn.QueryRowContext(ctx, sqlQuery, ptrToNullInt64(tag))
	if err := row.Scan(&jobAvailable); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("%w: %w", types.DatabaseRowScanError, err)
	}
	return &jobAvailable, nil
}

func GetNotes(ctx context.Context, noteType *string) (*types.GeneralNoteResponse, error) {
	if noteType == nil || strings.TrimSpace(*noteType) == "" {
		return nil, fmt.Errorf("%w: %s", types.InvalidFieldError, "noteType is nil or empty")
	}

	pgxPool, err := config.GetPGXPool()
	if err != nil {
		return nil, fmt.Errorf("%w: %w", types.DatabaseConnError, err)
	}

	const sqlQuery = `
	SELECT 
		time, 
		note_type, 
		note 
	FROM notes 
	WHERE note_type = $1 
	ORDER BY time DESC NULLS LAST
	LIMIT 1
	;`

	generalNoteRow := new(types.GeneralNoteResponse)
	row := pgxPool.QueryRow(ctx, sqlQuery,
		ptrToNullString(noteType),
	)
	if err := row.Scan(
		&generalNoteRow.Time,
		&generalNoteRow.NoteType,
		&generalNoteRow.NoteContent,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("%w: %w", types.DatabaseRowScanError, err)
	}
	return generalNoteRow, nil
}

func GetLocationFormData(ctx context.Context, tag *int64, serial *string) (*types.InventoryFormPrefillRow, error) {
	tagErr := types.IsTagnumberInt64Valid(tag)
	serialErr := types.IsSystemSerialValid(serial)

	if tagErr != nil && serialErr != nil {
		return nil, fmt.Errorf("%w: both tag and serial are invalid", types.DatabaseRowScanError)
	}

	pgxPool, err := config.GetPGXPool()
	if err != nil {
		return nil, fmt.Errorf("%w: %w", types.DatabaseConnError, err)
	}

	const sqlQuery = `
	WITH files AS (
		SELECT client_uuid, COUNT(client_images.client_uuid) AS "file_count" FROM client_images WHERE hidden = FALSE AND client_uuid = (SELECT uuid FROM ids WHERE (tagnumber = $1 OR system_serial = $2))
		GROUP BY client_uuid
	),
	default_system_model AS (
		SELECT 
			hardware_data.client_uuid, hardware_data.device_type 
		FROM 
			hardware_data 
		WHERE 
			hardware_data.system_model = 
				(SELECT 
					hardware_data.system_model 
				FROM 
					hardware_data 
				WHERE 
					hardware_data.system_model IS NOT NULL
				ORDER BY 
					hardware_data.time DESC NULLS LAST
				LIMIT 1) 
			AND hardware_data.device_type IS NOT NULL 
			GROUP BY hardware_data.client_uuid, hardware_data.device_type
			ORDER BY MAX(hardware_data.time) DESC NULLS LAST 
			LIMIT 1
	),
	most_recent_checkout AS (
		SELECT client_uuid, checkout_date, return_date, customer_name, checkout_bool FROM checkout_log WHERE client_uuid = (SELECT uuid FROM ids WHERE (tagnumber = $1 OR system_serial = $2)) ORDER BY time DESC NULLS LAST LIMIT 1
	)
	SELECT 
		locations.time, 
		ids.tagnumber, 
		ids.system_serial, 
		locations.location, 
		locations.building, 
		locations.room, 
		hardware_data.system_manufacturer, 
		hardware_data.system_model,
		COALESCE(hardware_data.device_type, default_system_model.device_type) AS "device_type",
		locations.department_name, 
		locations.ad_domain, 
		locations.property_custodian, 
		locations.acquired_date,
		locations.retired_date,
		locations.is_broken, 
		locations.disk_removed, 
		client_health.last_hardware_check,
		locations.client_status, 
		most_recent_checkout.checkout_date,
		most_recent_checkout.return_date,
		most_recent_checkout.customer_name,
		(CASE WHEN most_recent_checkout.checkout_bool = TRUE AND DATE(CURRENT_TIMESTAMP) >= most_recent_checkout.checkout_date AND DATE(CURRENT_TIMESTAMP) <= COALESCE(most_recent_checkout.return_date, DATE '9999-12-31') THEN TRUE ELSE FALSE END) AS "checkout_bool",
		locations.note,
		COALESCE(files.file_count, 0) AS "file_count"
	FROM ids
	LEFT JOIN locations ON ids.uuid = locations.client_uuid
	LEFT JOIN files ON ids.uuid = files.client_uuid
	LEFT JOIN hardware_data ON ids.uuid = hardware_data.client_uuid
	LEFT JOIN client_health ON ids.uuid = client_health.client_uuid
	LEFT JOIN static_department_info ON locations.department_name = static_department_info.department_name
	LEFT JOIN client_images ON ids.uuid = client_images.client_uuid
	LEFT JOIN default_system_model ON ids.uuid = default_system_model.client_uuid
	LEFT JOIN most_recent_checkout ON ids.uuid = most_recent_checkout.client_uuid
	WHERE ids.uuid = (SELECT uuid FROM ids WHERE (tagnumber = $1 OR system_serial = $2) ORDER BY time DESC LIMIT 1)
	GROUP BY 
		locations.time,
		ids.tagnumber,
		ids.system_serial,
		locations.location,
		locations.building,
		locations.room,
		hardware_data.system_manufacturer,
		hardware_data.system_model,
		hardware_data.device_type,
		default_system_model.device_type,
		locations.department_name,
		locations.ad_domain,
		locations.property_custodian,
		locations.acquired_date,
		locations.retired_date,
		locations.is_broken,
		locations.disk_removed,
		client_health.last_hardware_check,
		locations.client_status,
		most_recent_checkout.checkout_date,
		most_recent_checkout.return_date,
		most_recent_checkout.customer_name,
		most_recent_checkout.checkout_date,
		most_recent_checkout.checkout_bool,
		locations.note,
		files.file_count,
		files.client_uuid,
		client_images.client_uuid
	ORDER BY locations.time DESC NULLS LAST
	LIMIT 1;`

	row := pgxPool.QueryRow(ctx, sqlQuery,
		ptrToNullInt64(tag),
		ptrToNullString(serial),
	)

	inventoryFormPrefillRow := new(types.InventoryFormPrefillRow)
	if err := row.Scan(
		&inventoryFormPrefillRow.Time,
		&inventoryFormPrefillRow.Tagnumber,
		&inventoryFormPrefillRow.SystemSerial,
		&inventoryFormPrefillRow.Location,
		&inventoryFormPrefillRow.Building,
		&inventoryFormPrefillRow.Room,
		&inventoryFormPrefillRow.SystemManufacturer,
		&inventoryFormPrefillRow.SystemModel,
		&inventoryFormPrefillRow.DeviceType,
		&inventoryFormPrefillRow.Department,
		&inventoryFormPrefillRow.ADDomain,
		&inventoryFormPrefillRow.PropertyCustodian,
		&inventoryFormPrefillRow.AcquiredDate,
		&inventoryFormPrefillRow.RetiredDate,
		&inventoryFormPrefillRow.IsBroken,
		&inventoryFormPrefillRow.DiskRemoved,
		&inventoryFormPrefillRow.LastHardwareCheck,
		&inventoryFormPrefillRow.ClientStatus,
		&inventoryFormPrefillRow.CheckoutDate,
		&inventoryFormPrefillRow.ReturnDate,
		&inventoryFormPrefillRow.CustomerName,
		&inventoryFormPrefillRow.CheckoutBool,
		&inventoryFormPrefillRow.Note,
		&inventoryFormPrefillRow.FileCount,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("%w: %w", types.DatabaseRowScanError, err)
	}
	return inventoryFormPrefillRow, nil
}

func GetClientImageManifestByTag(ctx context.Context, tagnumber *int64) ([]types.ImageManifestResponse, error) {
	if err := types.IsTagnumberInt64Valid(tagnumber); err != nil {
		return nil, fmt.Errorf("%w: %w", types.InvalidFieldError, err)
	}

	pgxPool, err := config.GetPGXPool()
	if err != nil {
		return nil, fmt.Errorf("%w: %w", types.DatabaseConnError, err)
	}

	const sqlQuery = `
		SELECT 
			time, 
			client_uuid, 
			tagnumber, 
			uuid, 
			filename, 
			thumbnail_filename, 
			mime_type, 
			hidden, 
			pinned, 
			note 
		FROM client_images 
		WHERE 
			client_uuid = (SELECT uuid FROM ids WHERE tagnumber = $1);`

	imageManifests := make([]types.ImageManifestResponse, 0, 10)
	rows, err := pgxPool.Query(ctx, sqlQuery, tagnumber)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", types.DatabaseQueryError, err)
	}
	defer rows.Close()

	for rows.Next() {
		if ctx.Err() != nil {
			return nil, fmt.Errorf("%w: %w", types.DatabaseRowIterationError, ctx.Err())
		}
		var imageManifest types.ImageManifestResponse
		if err := rows.Scan(
			&imageManifest.Time,
			&imageManifest.ClientUUID,
			&imageManifest.Tagnumber,
			&imageManifest.FileUUID,
			&imageManifest.FileName,
			&imageManifest.ThumbnailFileName,
			&imageManifest.MimeType,
			&imageManifest.Hidden,
			&imageManifest.Pinned,
			&imageManifest.Caption,
		); err != nil {
			return nil, fmt.Errorf("%w: %w", types.DatabaseRowScanError, err)
		}
		imageManifests = append(imageManifests, imageManifest)
	}
	if rows.Err() != nil {
		return nil, fmt.Errorf("%w: %w", types.DatabaseRowIterationError, rows.Err())
	}
	if len(imageManifests) == 0 {
		return nil, nil
	}
	return imageManifests, nil
}

func GetClientImageManifestByFileUUID(ctx context.Context, fileUUID string) (*types.ImageManifestResponse, error) {
	if strings.TrimSpace(fileUUID) == "" {
		return nil, fmt.Errorf("file UUID is nil")
	}

	pgxPool, err := config.GetPGXPool()
	if err != nil {
		return nil, fmt.Errorf("%w: %w", types.DatabaseConnError, err)
	}

	const sqlQuery = `
		SELECT 
			time, 
			client_uuid, 
			tagnumber, 
			uuid, 
			filename, 
			thumbnail_filename, 
			mime_type, 
			hidden, 
			pinned, 
			note 
		FROM client_images 
		WHERE 
			uuid = $1;`

	var imageManifest types.ImageManifestResponse
	row := pgxPool.QueryRow(ctx, sqlQuery,
		fileUUID,
	)
	if err := row.Scan(
		&imageManifest.Time,
		&imageManifest.ClientUUID,
		&imageManifest.Tagnumber,
		&imageManifest.FileUUID,
		&imageManifest.FileName,
		&imageManifest.ThumbnailFileName,
		&imageManifest.MimeType,
		&imageManifest.Hidden,
		&imageManifest.Pinned,
		&imageManifest.Caption,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("%w: %w", types.DatabaseRowScanError, err)
	}
	return &imageManifest, nil
}

func GetInventoryTableData(ctx context.Context, filterOptions *types.InventoryAdvSearchOptions) ([]types.InventoryTableRow, error) {
	if filterOptions == nil {
		return nil, fmt.Errorf("%w: %s", types.InvalidStructureError, "filterOptions")
	}

	whereClause := make([]string, 0, 12)
	whereArgs := make([]any, 0, 12)
	i := 1

	// Make sure WHERE clause is never empty
	whereClause = append(whereClause, "locations.client_uuid IS NOT NULL")

	// Location filter
	if filterOptions.Location != nil && strings.TrimSpace(*filterOptions.Location.ParamValue) != "" {
		if filterOptions.Location.Not != nil && *filterOptions.Location.Not {
			whereClause = append(whereClause, fmt.Sprintf("NOT locations.location = $%d", i))
		} else {
			whereClause = append(whereClause, fmt.Sprintf("locations.location = $%d", i))
		}
		whereArgs = append(whereArgs, strings.TrimSpace(*filterOptions.Location.ParamValue))
		i++
	}
	// Building filter
	if filterOptions.BuildingAndRoom != nil && filterOptions.Building != nil && strings.TrimSpace(*filterOptions.Building) != "" {
		if filterOptions.BuildingAndRoom.Not != nil && *filterOptions.BuildingAndRoom.Not {
			whereClause = append(whereClause, fmt.Sprintf("NOT locations.building = $%d", i))
		} else {
			whereClause = append(whereClause, fmt.Sprintf("locations.building = $%d", i))
		}
		whereArgs = append(whereArgs, strings.TrimSpace(*filterOptions.Building))
		i++
	}
	// Room filter
	if filterOptions.BuildingAndRoom != nil && filterOptions.Room != nil && strings.TrimSpace(*filterOptions.Room) != "" {
		if filterOptions.BuildingAndRoom.Not != nil && *filterOptions.BuildingAndRoom.Not {
			whereClause = append(whereClause, fmt.Sprintf("NOT locations.room = $%d", i))
		} else {
			whereClause = append(whereClause, fmt.Sprintf("locations.room = $%d", i))
		}
		whereArgs = append(whereArgs, strings.TrimSpace(*filterOptions.Room))
		i++
	}
	// Manufacturer filter
	if filterOptions.SystemManufacturer != nil && strings.TrimSpace(*filterOptions.SystemManufacturer.ParamValue) != "" {
		if filterOptions.SystemManufacturer.Not != nil && *filterOptions.SystemManufacturer.Not {
			whereClause = append(whereClause, fmt.Sprintf("NOT hardware_data.system_manufacturer = $%d", i))
		} else {
			whereClause = append(whereClause, fmt.Sprintf("hardware_data.system_manufacturer = $%d", i))
		}
		whereArgs = append(whereArgs, strings.TrimSpace(*filterOptions.SystemManufacturer.ParamValue))
		i++
	}
	// Model filter
	if filterOptions.SystemModel != nil && strings.TrimSpace(*filterOptions.SystemModel.ParamValue) != "" {
		if filterOptions.SystemModel.Not != nil && *filterOptions.SystemModel.Not {
			whereClause = append(whereClause, fmt.Sprintf("NOT hardware_data.system_model = $%d", i))
		} else {
			whereClause = append(whereClause, fmt.Sprintf("hardware_data.system_model = $%d", i))
		}
		whereArgs = append(whereArgs, strings.TrimSpace(*filterOptions.SystemModel.ParamValue))
		i++
	}
	// Device Type filter
	if filterOptions.DeviceType != nil && strings.TrimSpace(*filterOptions.DeviceType.ParamValue) != "" {
		if filterOptions.DeviceType.Not != nil && *filterOptions.DeviceType.Not {
			whereClause = append(whereClause, fmt.Sprintf("NOT hardware_data.device_type = $%d", i))
		} else {
			whereClause = append(whereClause, fmt.Sprintf("hardware_data.device_type = $%d", i))
		}
		whereArgs = append(whereArgs, strings.TrimSpace(*filterOptions.DeviceType.ParamValue))
		i++
	}
	// Department filter
	if filterOptions.Department != nil && strings.TrimSpace(*filterOptions.Department.ParamValue) != "" {
		if filterOptions.Department.Not != nil && *filterOptions.Department.Not {
			whereClause = append(whereClause, fmt.Sprintf("NOT locations.department_name = $%d", i))
		} else {
			whereClause = append(whereClause, fmt.Sprintf("locations.department_name = $%d", i))
		}
		whereArgs = append(whereArgs, strings.TrimSpace(*filterOptions.Department.ParamValue))
		i++
	}
	// AD Domain filter
	if filterOptions.ADDomain != nil && strings.TrimSpace(*filterOptions.ADDomain.ParamValue) != "" {
		if filterOptions.ADDomain.Not != nil && *filterOptions.ADDomain.Not {
			whereClause = append(whereClause, fmt.Sprintf("NOT locations.ad_domain = $%d", i))
		} else {
			whereClause = append(whereClause, fmt.Sprintf("locations.ad_domain = $%d", i))
		}
		whereArgs = append(whereArgs, strings.TrimSpace(*filterOptions.ADDomain.ParamValue))
		i++
	}
	// Status filter
	if filterOptions.Status != nil && strings.TrimSpace(*filterOptions.Status.ParamValue) != "" {
		if filterOptions.Status.Not != nil && *filterOptions.Status.Not {
			whereClause = append(whereClause, fmt.Sprintf("NOT locations.client_status = $%d", i))
		} else {
			whereClause = append(whereClause, fmt.Sprintf("locations.client_status = $%d", i))
		}
		whereArgs = append(whereArgs, strings.TrimSpace(*filterOptions.Status.ParamValue))
		i++
	}
	// IsBroken filter
	if filterOptions.IsBroken != nil && filterOptions.IsBroken.ParamValue != nil {
		if filterOptions.IsBroken.Not != nil && *filterOptions.IsBroken.Not {
			whereClause = append(whereClause, fmt.Sprintf("NOT locations.is_broken = $%d", i))
		} else {
			whereClause = append(whereClause, fmt.Sprintf("locations.is_broken = $%d", i))
		}
		whereArgs = append(whereArgs, *filterOptions.IsBroken.ParamValue)
		i++
	}
	// Has Images filter
	if filterOptions.HasImages != nil && filterOptions.HasImages.ParamValue != nil {
		if *filterOptions.HasImages.ParamValue {
			whereClause = append(whereClause, "COALESCE(files.file_count, 0) > 0")
		} else {
			whereClause = append(whereClause, "COALESCE(files.file_count, 0) = 0")
		}
		// No whereArgs needed here, can cause issues with index
	}

	whereSQL := strings.Join(whereClause, "\n  AND ")

	sqlQuery := fmt.Sprintf(`
		WITH files AS (
			SELECT client_uuid, COUNT(*) AS file_count from client_images WHERE hidden = FALSE GROUP BY client_uuid
		),
		latest_os_versions AS (
			SELECT DISTINCT ON (os_info.os_name)
				os_info.os_name,
				(CASE WHEN os_info.windows_build_number IS NOT NULL AND os_info.windows_ubr IS NOT NULL THEN CONCAT(os_info.windows_build_number, '.', os_info.windows_ubr) ELSE NULL END) AS latest_os_version
			FROM os_info
			WHERE os_info.os_name IS NOT NULL
			ORDER BY os_info.os_name, os_info.windows_build_number DESC NULLS LAST, os_info.windows_ubr DESC NULLS LAST
		),
		latest_historical_firmware_data AS (
			SELECT * FROM (
				SELECT
					ROW_NUMBER() OVER (PARTITION BY historical_firmware_data.client_uuid ORDER BY historical_firmware_data.time DESC NULLS LAST) AS "row_num",
					historical_firmware_data.client_uuid, 
					historical_firmware_data.bios_version,
					historical_firmware_data.has_2023_ca
				FROM 
					historical_firmware_data
				WHERE 
					historical_firmware_data.client_uuid IS NOT NULL 
					AND historical_firmware_data.bios_version IS NOT NULL
				ORDER BY historical_firmware_data.client_uuid, historical_firmware_data.time DESC NULLS LAST
			) t1 WHERE t1.row_num = 1
		),
		os_installed_table AS (
			SELECT * FROM (
			SELECT
				ROW_NUMBER() OVER (PARTITION BY jobstats.client_uuid ORDER BY jobstats.time DESC NULLS LAST) AS "row_num",
				jobstats.client_uuid,
				static_image_names.image_version,
				(CASE WHEN jobstats.erase_completed IS TRUE AND jobstats.clone_completed IS DISTINCT FROM TRUE THEN FALSE ELSE TRUE END) AS "os_installed"
			FROM jobstats
			LEFT JOIN hardware_data ON jobstats.client_uuid = hardware_data.client_uuid
			LEFT JOIN static_image_names ON hardware_data.system_model = static_image_names.system_model
			WHERE (jobstats.erase_completed = TRUE OR jobstats.clone_completed = TRUE)
			GROUP BY jobstats.time, jobstats.client_uuid, static_image_names.image_version, jobstats.erase_completed, jobstats.clone_completed
			ORDER BY jobstats.client_uuid, jobstats.time DESC NULLS LAST) t1 WHERE t1.row_num = 1
		),
		most_recent_checkout AS (
			SELECT DISTINCT ON (client_uuid)
				client_uuid,
				checkout_date,
				return_date,
				customer_name
			FROM checkout_log
			ORDER BY client_uuid, time DESC NULLS LAST
		)
		SELECT
			ids.tagnumber, 
			ids.system_serial, 
			locations.location, 
			locationFormatting(locations.location) AS "location_formatted", 
			locations.building, 
			locations.room,
			hardware_data.system_manufacturer, 
			hardware_data.system_model, 
			hardware_data.device_type, 
			static_device_types.device_type_formatted, 
			locations.department_name, 
			static_department_info.department_name_formatted,
			locations.ad_domain, 
			os_info.is_intune_joined,
			static_ad_domains.domain_name_formatted, 
			os_installed_table.os_installed,
			(CASE WHEN locations.disk_removed = TRUE THEN NULL ELSE COALESCE(os_info.os_name, os_installed_table.image_version) END) AS "os_name", 
			(CASE WHEN locations.disk_removed = TRUE THEN NULL WHEN os_info.windows_build_number IS NOT NULL AND os_info.windows_ubr IS NOT NULL THEN CONCAT(os_info.windows_build_number, '.', os_info.windows_ubr) ELSE NULL END) AS "os_version",
			latest_os_versions.latest_os_version,
			os_info.admin_users,
			os_info.is_disk_encrypted,
			os_info.secure_boot_enabled,
			latest_historical_firmware_data.has_2023_ca,
			client_health.last_hardware_check,
			(CASE 
				WHEN latest_historical_firmware_data.bios_version = static_bios_stats.bios_version THEN TRUE
				ELSE FALSE
			END) AS "bios_updated",
			latest_historical_firmware_data.bios_version,
			locations.client_status,
			static_client_statuses.status_formatted,
			locations.is_broken, 
			locations.disk_removed,
			locations.retired_date,
			(CASE WHEN DATE(CURRENT_TIMESTAMP) < most_recent_checkout.return_date THEN TRUE ELSE FALSE END) AS "checkout_bool",
			locations.note, 
			locations.time AS last_updated, 
			(CASE WHEN files.file_count IS NOT NULL AND files.file_count > 0 THEN files.file_count ELSE 0 END) AS "file_count"
		FROM ids
			LEFT JOIN locations ON ids.uuid = locations.client_uuid
			LEFT JOIN hardware_data ON ids.uuid = hardware_data.client_uuid
			LEFT JOIN client_health ON ids.uuid = client_health.client_uuid
			LEFT JOIN static_department_info ON locations.department_name = static_department_info.department_name
			LEFT JOIN static_ad_domains ON locations.ad_domain = static_ad_domains.domain_name
			LEFT JOIN static_client_statuses ON locations.client_status = static_client_statuses.status_name
			LEFT JOIN static_device_types ON hardware_data.device_type = static_device_types.device_type
			LEFT JOIN files ON ids.uuid = files.client_uuid
			LEFT JOIN latest_historical_firmware_data ON ids.uuid = latest_historical_firmware_data.client_uuid
			LEFT JOIN static_bios_stats ON hardware_data.system_model = static_bios_stats.system_model
			LEFT JOIN os_info ON ids.uuid = os_info.client_uuid
			LEFT JOIN latest_os_versions ON os_info.os_name = latest_os_versions.os_name
			LEFT JOIN os_installed_table ON ids.uuid = os_installed_table.client_uuid
			LEFT JOIN most_recent_checkout ON ids.uuid = most_recent_checkout.client_uuid
		WHERE %s
		GROUP BY 
			locations.client_uuid,
			ids.tagnumber, 
			ids.system_serial,
			locations.location,
			locations.building,
			locations.room,
			hardware_data.system_manufacturer,
			hardware_data.system_model,
			hardware_data.device_type,
			static_device_types.device_type_formatted,
			locations.department_name,
			static_department_info.department_name_formatted,
			locations.ad_domain,
			os_info.is_intune_joined,
			static_ad_domains.domain_name_formatted,
			os_installed_table.os_installed,
			os_installed_table.image_version,
			os_info.os_name,
			os_info.os_version,
			os_info.windows_build_number,
			os_info.windows_ubr,
			latest_os_versions.latest_os_version,
			os_info.admin_users,
			os_info.is_disk_encrypted,
			os_info.secure_boot_enabled,
			latest_historical_firmware_data.has_2023_ca,
			client_health.last_hardware_check,
			static_bios_stats.bios_version,
			latest_historical_firmware_data.bios_version,
			locations.client_status,
			static_client_statuses.status_formatted,
			locations.is_broken,
			locations.disk_removed,
			locations.retired_date,
			most_recent_checkout.return_date,
			locations.note,
			locations.time,
			files.file_count
		ORDER BY locations.time DESC NULLS LAST
	;`, whereSQL)

	pgxPool, err := config.GetPGXPool()
	if err != nil {
		return nil, fmt.Errorf("%w: %w", types.DatabaseConnError, err)
	}

	results := make([]types.InventoryTableRow, 0, approxClientCount)
	rows, err := pgxPool.Query(ctx, sqlQuery, whereArgs...)
	if err != nil {
		return nil, fmt.Errorf("query error: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		if ctx.Err() != nil {
			return nil, fmt.Errorf("%w: %w", types.DatabaseRowIterationError, ctx.Err())
		}
		var row types.InventoryTableRow
		var adminUsers []string
		if err := rows.Scan(
			&row.Tagnumber,
			&row.SystemSerial,
			&row.Location,
			&row.LocationFormatted,
			&row.Building,
			&row.Room,
			&row.SystemManufacturer,
			&row.SystemModel,
			&row.DeviceType,
			&row.DeviceTypeFormatted,
			&row.Department,
			&row.DepartmentFormatted,
			&row.ADDomain,
			&row.IsIntuneJoined,
			&row.DomainFormatted,
			&row.OsInstalled,
			&row.OsName,
			&row.OsVersion,
			&row.LatestOsVersion,
			&adminUsers,
			&row.IsDiskEncrypted,
			&row.SecureBootEnabled,
			&row.Has2023SecureBootCA,
			&row.LastHardwareCheck,
			&row.BIOSUpdated,
			&row.BIOSVersion,
			&row.Status,
			&row.StatusFormatted,
			&row.IsBroken,
			&row.DiskRemoved,
			&row.RetiredDate,
			&row.IsCheckedOut,
			&row.Note,
			&row.LastUpdated,
			&row.FileCount,
		); err != nil {
			return nil, fmt.Errorf("%w: %w", types.DatabaseRowScanError, err)
		}

		if adminUsers != nil {
			row.AdminUsers = adminUsers
		}
		results = append(results, row)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("%w: %w", types.DatabaseRowIterationError, err)
	}
	if len(results) == 0 {
		return nil, nil
	}

	resultsWithErrors, err := ModifyClientConfigErrorResults(results)
	if err != nil {
		return nil, fmt.Errorf("error modifying client config error results: %w", err)
	}
	return resultsWithErrors, nil
}

func ModifyClientConfigErrorResults(results []types.InventoryTableRow) ([]types.InventoryTableRow, error) {
	if len(results) == 0 {
		return results, nil
	}
	// Set client configuration errors
	for i := range results {
		// If client is missing required info in the DB
		tagErr := types.IsTagnumberInt64Valid(results[i].Tagnumber)
		serialErr := types.IsSystemSerialValid(results[i].SystemSerial)

		if tagErr != nil ||
			serialErr != nil ||
			results[i].Location == nil {
			missingRequiredGeneralInfo := types.MissingRequiredGeneralInfo.ToConfigErrorResponse()
			results[i].ClientErrors = append(results[i].ClientErrors, missingRequiredGeneralInfo)
		}

		if results[i].Building == nil ||
			results[i].Room == nil ||
			results[i].Department == nil ||
			results[i].Status == nil {
			missingOptionalInfo := types.MissingOptionalInfo.ToConfigErrorResponse()
			results[i].ClientErrors = append(results[i].ClientErrors, missingOptionalInfo)
		}

		if results[i].SystemManufacturer == nil ||
			results[i].SystemModel == nil ||
			results[i].DeviceType == nil {
			missingRequiredHardwareInfo := types.MissingRequiredHardwareInfo.ToConfigErrorResponse()
			results[i].ClientErrors = append(results[i].ClientErrors, missingRequiredHardwareInfo)
		}
		// // If client is missing images of itself
		// if results[i].FileCount == nil || (results[i].FileCount != nil && *results[i].FileCount <= 0) {
		// 	missingImages := types.MissingImages.String()
		// 	results[i].ClientErrors = append(results[i].ClientErrors, missingImages)
		// }

		// If client is retired, only pay attention to if disk removed or not
		if results[i].Status != nil && (*results[i].Status == "retired" || *results[i].Status == "pre-property") {
			if results[i].DiskRemoved != nil && !*results[i].DiskRemoved {
				diskNotRemoved := types.DiskNotRemoved.ToConfigErrorResponse()
				results[i].ClientErrors = append(results[i].ClientErrors, diskNotRemoved)
			}
			continue
		}

		// If no hardware check in over 3 months
		if results[i].LastHardwareCheck == nil || (results[i].LastHardwareCheck != nil && time.Since(*results[i].LastHardwareCheck) > 90*24*time.Hour) {
			needsHardwareCheck := types.NeedsHardwareCheck.ToConfigErrorResponse()
			results[i].ClientErrors = append(results[i].ClientErrors, needsHardwareCheck)
		}

		// If disk is removed but OS is still marked as installed (need to update OS info in DB)
		if (results[i].DiskRemoved != nil && *results[i].DiskRemoved) && (results[i].OsInstalled == nil || (results[i].OsInstalled != nil && *results[i].OsInstalled)) {
			osInvalidData := types.OSInvalidData.ToConfigErrorResponse()
			results[i].ClientErrors = append(results[i].ClientErrors, osInvalidData)
		}

		// If client has status pre-property or retired status, it need to be erased
		if results[i].Status != nil && (*results[i].Status == "pre-property" || *results[i].Status == "retired") {
			if results[i].OsInstalled != nil && *results[i].OsInstalled {
				needsErasing := types.NeedsErasing.ToConfigErrorResponse()
				results[i].ClientErrors = append(results[i].ClientErrors, needsErasing)
			}
			// If client has status pre-property or retired but disk is not removed
			if results[i].DiskRemoved != nil && !*results[i].DiskRemoved {
				diskNotRemoved := types.DiskNotRemoved.ToConfigErrorResponse()
				results[i].ClientErrors = append(results[i].ClientErrors, diskNotRemoved)
			}
			continue
		}

		// Check for Microsoft's 2023 Secure Boot CA
		if results[i].Has2023SecureBootCA == nil || (results[i].Has2023SecureBootCA != nil && !*results[i].Has2023SecureBootCA) {
			missing2023SecureBootCA := types.Missing2023SecureBootCA.ToConfigErrorResponse()
			results[i].ClientErrors = append(results[i].ClientErrors, missing2023SecureBootCA)
		}

		// If OS is installed
		if results[i].OsInstalled != nil && *results[i].OsInstalled {
			// if OS name is missing (need to update OS info)
			if results[i].OsName == nil || strings.TrimSpace(*results[i].OsName) == "" {
				osMissing := types.MissingRequiredSoftwareInfo.ToConfigErrorResponse()
				results[i].ClientErrors = append(results[i].ClientErrors, osMissing)
			}
			// If OS version is missing (need to update OS info)
			if results[i].OsVersion == nil || strings.TrimSpace(*results[i].OsVersion) == "" {
				osMissingInfo := types.MissingRequiredSoftwareInfo.ToConfigErrorResponse()
				results[i].ClientErrors = append(results[i].ClientErrors, osMissingInfo)
			}
			// If OS version is not the latest (need to update OS info and/or update OS)
			if results[i].LatestOsVersion != nil && results[i].OsVersion != nil {
				currentVersion := strings.TrimSpace(*results[i].OsVersion)
				latestVersion := strings.TrimSpace(*results[i].LatestOsVersion)
				if currentVersion != "" && latestVersion != "" && currentVersion != latestVersion {
					osOutdated := types.OSOutdated.ToConfigErrorResponse()
					results[i].ClientErrors = append(results[i].ClientErrors, osOutdated)
				}
			}
			// If OS is windows
			if results[i].OsName != nil && strings.Contains(strings.ToLower(*results[i].OsName), "windows") {
				// If secure boot is not enabled
				if results[i].SecureBootEnabled != nil && !*results[i].SecureBootEnabled {
					secureBootNotEnabled := types.SecureBootNotEnabled.ToConfigErrorResponse()
					results[i].ClientErrors = append(results[i].ClientErrors, secureBootNotEnabled)
				}
				// If OS is windows but Bitlocker is not enabled
				if results[i].IsDiskEncrypted != nil && !*results[i].IsDiskEncrypted {
					diskNotEncryptedErr := types.DiskNotEncrypted.ToConfigErrorResponse()
					results[i].ClientErrors = append(results[i].ClientErrors, diskNotEncryptedErr)
				}
				// If OS is windows and AD domain is nil/empty/default (WORKGROUP)
				if results[i].ADDomain == nil || (results[i].ADDomain != nil && (strings.TrimSpace(*results[i].ADDomain) == "" || *results[i].ADDomain == "none" || strings.ToLower(*results[i].ADDomain) == "workgroup")) {
					domainNotJoined := types.DomainNotJoined.ToConfigErrorResponse()
					results[i].ClientErrors = append(results[i].ClientErrors, domainNotJoined)
				} else { // If OS is windows and AD domain is valid
					// If OS is windows and AD domain is valid but not Intune joined
					if results[i].IsIntuneJoined != nil && !*results[i].IsIntuneJoined {
						intuneNotEnrolled := types.IntuneNotEnrolled.ToConfigErrorResponse()
						results[i].ClientErrors = append(results[i].ClientErrors, intuneNotEnrolled)
					}
					if len(results[i].AdminUsers) < 2 {
						adminUsersMissing := types.AdminUsersMissing.ToConfigErrorResponse()
						results[i].ClientErrors = append(results[i].ClientErrors, adminUsersMissing)
					}
				}
			}
		} else { // If OS is not installed
			osNotInstalled := types.OSNotInstalled.ToConfigErrorResponse()
			results[i].ClientErrors = append(results[i].ClientErrors, osNotInstalled)
			// if results[i].DiskRemoved != nil && !*results[i].DiskRemoved {
			// 	osNotInstalled := types.OSNotInstalled.ToConfigErrorResponse()
			// 	results[i].ClientErrors = append(results[i].ClientErrors, osNotInstalled)
			// }
		}
		// If BIOS out of date
		if results[i].BIOSVersion != nil && (results[i].BIOSUpdated != nil && !*results[i].BIOSUpdated) {
			biosOutdated := types.BIOSOutdated.ToConfigErrorResponse()
			if results[i].BIOSVersion != nil {
				biosOutdated.ErrorMessage = biosOutdated.ErrorMessage + ": " + *results[i].BIOSVersion
				results[i].ClientErrors = append(results[i].ClientErrors, biosOutdated)
			} else {
				results[i].ClientErrors = append(results[i].ClientErrors, biosOutdated)
			}
		}
	}
	return results, nil
}

func GetJobQueueTable(ctx context.Context) ([]types.JobQueueTableRowView, error) {
	pgxPool, err := config.GetPGXPool()
	if err != nil {
		return nil, fmt.Errorf("%w: %w", types.DatabaseConnError, err)
	}

	const sqlQuery = `
	WITH top_clients AS MATERIALIZED (
		SELECT ids.uuid
		FROM ids
		LEFT JOIN live_os_data ON ids.uuid = live_os_data.client_uuid
		ORDER BY
			(ids.uuid = ANY($1::uuid[])) DESC,
			live_os_data.last_heard DESC NULLS LAST
		LIMIT 50
	),
	avg_battery_health AS (
		SELECT 
			system_model, 
			AVG(avg_battery_health_pcnt) AS "avg_battery_health_pcnt" 
		FROM (
			SELECT 
				hardware_data.system_model, 
				(historical_battery_data.battery_current_max_capacity::decimal / historical_battery_data.battery_design_capacity::decimal * 100) AS "avg_battery_health_pcnt" 
			FROM 
				historical_battery_data 
			LEFT JOIN 
				hardware_data ON historical_battery_data.client_uuid = hardware_data.client_uuid
			WHERE 
				historical_battery_data.battery_design_capacity IS NOT NULL 
				AND historical_battery_data.battery_current_max_capacity IS NOT NULL 
		)
		GROUP BY system_model
	),
	job_queue_positions_cte AS (
		SELECT
			job_queue.client_uuid, 
			job_queue.job_name,
			ROW_NUMBER() OVER (
				ORDER BY job_queue.job_queued_at ASC NULLS LAST
			) AS "position_in_queue"
		FROM job_queue
		WHERE
			job_queue.job_name IN (
				'hpEraseAndClone',
				'hpCloneOnly',
				'generic-erase+clone',
				'generic-clone'
			)
			AND job_queue.job_queued_at IS NOT NULL
			AND (job_queue.job_queued = TRUE OR job_queue.job_name IS NOT NULL)
			AND job_queue.client_uuid = ANY($1::uuid[])
	)
	SELECT
		ids.uuid,
		ids.tagnumber,
		ids.system_serial,
		hardware_data.system_manufacturer,
		hardware_data.system_model,
		locationFormatting(locations.location) AS "location",
		static_department_info.department_name_formatted,
		static_client_statuses.status_formatted AS "client_status",
		locations.is_broken,
		locations.disk_removed,
		FALSE AS "temp_warning",
		(CASE WHEN static_client_statuses.status_name = 'checked_out' THEN TRUE ELSE FALSE END) AS "checkout_bool",
		TRUE AS "kernel_updated",
		(ids.uuid = ANY($1::uuid[])) AS "online",
		job_queue.job_active,
		job_queue.job_queued,
		job_queue.job_queued_at,
		job_queue_positions_cte.position_in_queue AS "job_queue_position",
		job_queue.job_name,
		static_job_names.job_name_readable,
		(CASE
			WHEN (job_queue.job_queued = TRUE OR job_queue.job_active = TRUE) AND job_queue.job_name IS NOT NULL THEN job_queue.clone_mode
			ELSE NULL
		END) AS "job_clone_mode",
		(CASE
			WHEN (job_queue.job_queued = TRUE OR job_queue.job_active = TRUE) AND job_queue.job_name IS NOT NULL THEN job_queue.erase_mode
			ELSE NULL
		END) AS "job_erase_mode",
		job_queue.job_status,
		(CASE
			WHEN latest_completed_job.erase_completed IS NOT NULL AND latest_completed_job.erase_completed = TRUE THEN latest_completed_job.time
			WHEN latest_completed_job.clone_completed IS NOT NULL AND latest_completed_job.clone_completed = TRUE THEN latest_completed_job.time
			ELSE NULL
		END) AS "last_job_time",
		(CASE 
			WHEN latest_completed_job.clone_completed IS NOT NULL AND latest_completed_job.clone_completed = TRUE THEN TRUE
			WHEN latest_completed_job.erase_completed IS NOT NULL AND latest_completed_job.erase_completed = TRUE THEN FALSE
			ELSE TRUE
		END) AS "os_installed",
		(CASE WHEN os_info.time >= latest_completed_job.time THEN os_info.os_name ELSE static_image_names.image_name_readable END) AS "os_name",
		(CASE
			WHEN 
				latest_completed_job.clone_completed IS NOT NULL 
				AND latest_completed_job.clone_completed = TRUE 
				AND static_image_names.last_updated <= latest_completed_job.time 
			THEN TRUE
			ELSE FALSE
		END) AS "latest_image_installed",
		(CASE 
			WHEN locations.ad_domain IS NOT NULL AND NOT locations.ad_domain = 'none' THEN TRUE
			ELSE FALSE
		END) AS "domain_joined",
		static_ad_domains.domain_name,
		static_ad_domains.domain_name_formatted AS "ad_domain_formatted",
		(CASE 
			WHEN latest_firmware_data.bios_version = static_bios_stats.bios_version THEN TRUE
			ELSE FALSE
		END) AS "bios_updated",
		latest_firmware_data.bios_version,
		live_os_data.cpu_usage,
		job_queue.cpu_mhz,
		job_queue.cpu_temp,
		(CASE 
			WHEN job_queue.cpu_temp > 90 THEN TRUE
			ELSE FALSE
		END) AS "cpu_temp_warning",
		live_os_data.memory_usage_kb,
		latest_historical_hardware_data.memory_capacity_kb,
		0::integer AS "disk_usage",
		job_queue.disk_temp,
		static_disk_stats.disk_type,
		latest_historical_disk_data.disk_size_kb,
		80::integer AS "max_disk_temp",
		(CASE
			WHEN job_queue.disk_temp > 80 THEN TRUE
			ELSE FALSE
		END) AS "disk_temp_warning",
		'UP' AS "network_link_status",
		live_os_data.link_speed AS "network_link_speed",
		0::integer AS "network_usage",
		job_queue.battery_charge_pcnt,
		job_queue.battery_status,
		current_battery_health.battery_health_pcnt AS "battery_health_pcnt",
		ROUND(current_battery_health.battery_health_pcnt - avg_battery_health.avg_battery_health_pcnt, 2) AS "battery_health_deviation",
		NULL AS "plugged_in",
		job_queue.watts_now AS "power_usage"
	FROM top_clients
	INNER JOIN ids ON ids.uuid = top_clients.uuid
	LEFT JOIN job_queue ON ids.uuid = job_queue.client_uuid
	LEFT JOIN hardware_data ON ids.uuid = hardware_data.client_uuid
	LEFT JOIN locations ON ids.uuid = locations.client_uuid
	LEFT JOIN LATERAL (
		SELECT 
			memory_serial, 
			memory_capacity_kb, 
			memory_speed_mhz
		FROM historical_hardware_data
		WHERE 
			client_uuid = ids.uuid
			AND memory_capacity_kb IS NOT NULL
		ORDER BY time DESC NULLS LAST
		LIMIT 1
	) latest_historical_hardware_data ON TRUE
	LEFT JOIN LATERAL (
		SELECT 
			disk_model, 
			disk_size_kb
		FROM historical_disk_data
		WHERE 
			client_uuid = ids.uuid
		ORDER BY time DESC NULLS LAST
		LIMIT 1
	) latest_historical_disk_data ON TRUE
	LEFT JOIN LATERAL (
		SELECT 
				bios_version
			FROM historical_firmware_data
			WHERE 
				client_uuid = ids.uuid
				AND bios_version IS NOT NULL
			ORDER BY time DESC NULLS LAST
			LIMIT 1
	) latest_firmware_data ON TRUE
	LEFT JOIN avg_battery_health ON hardware_data.system_model = avg_battery_health.system_model
	LEFT JOIN LATERAL (
		SELECT 
			ROUND((battery_current_max_capacity::decimal / battery_design_capacity::decimal * 100), 2) AS "battery_health_pcnt"
		FROM 
			historical_battery_data
		WHERE
			client_uuid = ids.uuid
			AND battery_design_capacity IS NOT NULL 
			AND battery_current_max_capacity IS NOT NULL
		ORDER BY time DESC NULLS LAST
		LIMIT 1
	) current_battery_health ON TRUE
	LEFT JOIN LATERAL (
		SELECT
			jobstats.time, 
			jobstats.erase_completed, 
			jobstats.erase_mode, 
			jobstats.erase_time, 
			jobstats.clone_completed, 
			jobstats.clone_image, 
			jobstats.clone_master, 
			jobstats.clone_time, 
			jobstats.job_cancelled
		FROM jobstats
		WHERE 
			ids.uuid = jobstats.client_uuid
			AND jobstats.job_cancelled IS DISTINCT FROM TRUE
			AND (jobstats.erase_completed = TRUE OR jobstats.clone_completed = TRUE)
		ORDER BY jobstats.time DESC NULLS LAST
		LIMIT 1
	) latest_completed_job ON TRUE
	LEFT JOIN os_info ON ids.uuid = os_info.client_uuid
	LEFT JOIN static_job_names ON job_queue.job_name = static_job_names.job_name
	LEFT JOIN live_os_data ON ids.uuid = live_os_data.client_uuid
	LEFT JOIN static_bios_stats ON hardware_data.system_model = static_bios_stats.system_model
	LEFT JOIN static_disk_stats ON latest_historical_disk_data.disk_model = static_disk_stats.disk_model
	LEFT JOIN static_ad_domains ON locations.ad_domain = static_ad_domains.domain_name
	LEFT JOIN static_image_names ON static_image_names.image_name = latest_completed_job.clone_image AND static_image_names.system_model = hardware_data.system_model
	LEFT JOIN job_queue_positions_cte ON job_queue_positions_cte.client_uuid = ids.uuid
	LEFT JOIN static_client_statuses ON static_client_statuses.status_name = locations.client_status
	LEFT JOIN static_department_info ON static_department_info.department_name = locations.department_name
	;`

	onlineClientsMap, err := config.GetAllOnlineClientsData()
	if err != nil {
		return nil, fmt.Errorf("%w: %w", types.ErrNoOnlineClients, err)
	}

	onlineClientsMapUUIDAsKey := make(map[uuid.UUID]types.JobQueueRealtimeData, len(onlineClientsMap))
	for _, realtimeData := range onlineClientsMap {
		onlineClientsMapUUIDAsKey[realtimeData.ClientUUID] = types.JobQueueRealtimeData{
			ClientUUID:   realtimeData.ClientUUID,
			LastHeard:    realtimeData.LastHeard,
			SystemUptime: realtimeData.SystemUptime,
			AppUptime:    realtimeData.AppUptime,
		}
	}

	onlineClientUUIDs := make([]uuid.UUID, 0, len(onlineClientsMap))
	for _, realtimeData := range onlineClientsMap {
		onlineClientUUIDs = append(onlineClientUUIDs, realtimeData.ClientUUID)
	}

	jobQueueRows := make([]types.JobQueueTableRowView, 0, approxClientCount)

	rows, err := pgxPool.Query(ctx, sqlQuery, onlineClientUUIDs)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", types.DatabaseQueryError, err)
	}
	defer rows.Close()

	for rows.Next() {
		if ctx.Err() != nil {
			return nil, fmt.Errorf("%w: %w", types.DatabaseRowIterationError, ctx.Err())
		}
		var row types.JobQueueTableRowView
		var clientUUID uuid.UUID
		if err := rows.Scan(
			&clientUUID,
			&row.Tagnumber,
			&row.SystemSerial,
			&row.SystemManufacturer,
			&row.SystemModel,
			&row.Location,
			&row.Department,
			&row.ClientStatus,
			&row.IsBroken,
			&row.DiskRemoved,
			&row.TempWarning,
			&row.CheckoutBool,
			&row.KernelUpdated,
			&row.Online,
			&row.JobActive,
			&row.JobQueued,
			&row.JobQueuedAt,
			&row.QueuePosition,
			&row.JobName,
			&row.JobNameReadable,
			&row.JobCloneMode,
			&row.JobEraseMode,
			&row.JobStatus,
			&row.LastJobTime,
			&row.OSInstalled,
			&row.OSName,
			&row.LatestImageInstalled,
			&row.DomainJoined,
			&row.DomainName,
			&row.DomainNameFormatted,
			&row.BIOSUpdated,
			&row.BIOSVersion,
			&row.CPUUsage,
			&row.CPUMHz,
			&row.CPUTemp,
			&row.CPUTempWarning,
			&row.MemoryUsageKB,
			&row.MemoryCapacityKB,
			&row.DiskUsage,
			&row.DiskTemp,
			&row.DiskType,
			&row.DiskSize,
			&row.MaxDiskTemp,
			&row.DiskTempWarning,
			&row.NetworkLinkStatus,
			&row.NetworkLinkSpeed,
			&row.NetworkUsage,
			&row.BatteryCharge,
			&row.BatteryStatus,
			&row.BatteryHealthPcnt,
			&row.BatteryHealthDeviation,
			&row.PluggedIn,
			&row.PowerUsage,
		); err != nil {
			return nil, fmt.Errorf("%w: %w", types.DatabaseRowScanError, err)
		}
		row.LastHeard = onlineClientsMapUUIDAsKey[clientUUID].LastHeard
		row.ClientUUID = &clientUUID
		row.SystemUptime = time.Duration(onlineClientsMapUUIDAsKey[clientUUID].SystemUptime.Seconds())
		row.AppUptime = time.Duration(onlineClientsMapUUIDAsKey[clientUUID].AppUptime.Seconds())
		jobQueueRows = append(jobQueueRows, row)
	}
	if rows.Err() != nil {
		return nil, fmt.Errorf("%w: %w", types.DatabaseRowIterationError, rows.Err())
	}
	if len(jobQueueRows) == 0 {
		return nil, nil
	}
	return jobQueueRows, nil
}

func SelectAllJobs(ctx context.Context) ([]types.AllJobsRow, error) {
	pgxPool, err := config.GetPGXPool()
	if err != nil {
		return nil, fmt.Errorf("%w: %w", types.DatabaseConnError, err)
	}

	const sqlQuery = `
		SELECT 
			job_name, 
			job_name_readable, 
			job_sort_order, 
			job_hidden
		FROM static_job_names
		ORDER BY 
			job_sort_order ASC
	;`

	var allJobs []types.AllJobsRow
	rows, err := pgxPool.Query(ctx, sqlQuery)
	if err != nil {
		return nil, fmt.Errorf("error during query execution: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		if ctx.Err() != nil {
			return nil, fmt.Errorf("%w: %w", types.DatabaseRowIterationError, ctx.Err())
		}
		var job types.AllJobsRow
		rowScanErr := rows.Scan(
			&job.JobName,
			&job.JobNameReadable,
			&job.JobSortOrder,
			&job.JobHidden,
		)
		if rowScanErr != nil {
			return nil, fmt.Errorf("%w: %w", types.DatabaseRowScanError, rowScanErr)
		}
		allJobs = append(allJobs, job)
	}
	if rows.Err() != nil {
		return nil, fmt.Errorf("%w: %w", types.DatabaseRowIterationError, rows.Err())
	}
	if len(allJobs) == 0 {
		return nil, nil
	}
	return allJobs, nil
}

func GetAllLocations(ctx context.Context) ([]types.AllLocationsRow, error) {
	const sqlQuery = `
		SELECT 
			location, 
			MAX(time) AS "timestamp",
			COUNT(*) as "location_count"
		FROM locations
		GROUP BY 
			location
		ORDER BY 
			location ASC, timestamp DESC
	;`

	pgxPool, err := config.GetPGXPool()
	if err != nil {
		return nil, fmt.Errorf("%w: %w", types.DatabaseConnError, err)
	}

	var allLocations []types.AllLocationsRow
	rows, err := pgxPool.Query(ctx, sqlQuery)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", types.DatabaseQueryError, err)
	}
	defer rows.Close()

	for rows.Next() {
		if ctx.Err() != nil {
			return nil, fmt.Errorf("%w: %w", types.DatabaseRowIterationError, ctx.Err())
		}
		location := new(types.AllLocationsRow)
		if err := rows.Scan(
			&location.Location,
			&location.Timestamp,
			&location.LocationCount,
		); err != nil {
			return nil, fmt.Errorf("%w: %w", types.DatabaseRowScanError, err)
		}
		allLocations = append(allLocations, *location)
	}
	if rows.Err() != nil {
		return nil, fmt.Errorf("%w: %w", types.DatabaseRowIterationError, rows.Err())
	}
	if len(allLocations) == 0 {
		return nil, nil
	}
	return allLocations, nil
}

func GetAllStatuses(ctx context.Context) (map[string][]types.AllClientStatuses, error) {
	pgxPool, err := config.GetPGXPool()
	if err != nil {
		return nil, fmt.Errorf("%w: %w", types.DatabaseConnError, err)
	}

	const sqlQuery = `
		SELECT 
			static_client_statuses.status_name, 
			static_client_statuses.status_formatted, 
			static_client_statuses.sort_order, 
			static_client_statuses.status_type,
			COUNT(locations.client_status) AS "status_count"
		FROM 
			static_client_statuses 
		LEFT JOIN locations ON 
			static_client_statuses.status_name = locations.client_status
		GROUP BY 
			static_client_statuses.status_name, 
			static_client_statuses.status_formatted,
			static_client_statuses.sort_order,
			static_client_statuses.status_type
		ORDER BY 
			static_client_statuses.sort_order ASC NULLS LAST
	;`

	rows, err := pgxPool.Query(ctx, sqlQuery)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", types.DatabaseQueryError, err)
	}
	defer rows.Close()

	var allStatuses []types.AllClientStatuses
	for rows.Next() {
		if ctx.Err() != nil {
			return nil, fmt.Errorf("%w: %w", types.DatabaseRowIterationError, ctx.Err())
		}
		var status types.AllClientStatuses
		if err := rows.Scan(
			&status.Status,
			&status.StatusFormatted,
			&status.SortOrder,
			&status.StatusType,
			&status.ClientCount,
		); err != nil {
			return nil, fmt.Errorf("%w: %w", types.DatabaseRowScanError, err)
		}
		allStatuses = append(allStatuses, status)
	}
	if rows.Err() != nil {
		return nil, fmt.Errorf("%w: %w", types.DatabaseRowIterationError, rows.Err())
	}
	if len(allStatuses) == 0 {
		return nil, nil
	}
	statusMap := make(map[string][]types.AllClientStatuses)
	for _, status := range allStatuses {
		if status.StatusType == nil {
			continue
		}
		statusMap[*status.StatusType] = append(statusMap[*status.StatusType], status)
	}
	return statusMap, nil
}

func GetAllDeviceTypes(ctx context.Context) ([]types.AllDeviceTypesRow, error) {
	const sqlQuery = `
	SELECT 
		static_device_types.device_type, 
		static_device_types.device_type_formatted, 
		static_device_types.device_meta_category, 
		COUNT(hardware_data.device_type) AS "device_type_count"
	FROM static_device_types 
	LEFT JOIN hardware_data ON static_device_types.device_type = hardware_data.device_type
	GROUP BY 
		static_device_types.device_type, 
		static_device_types.device_type_formatted, 
		static_device_types.device_meta_category, 
		static_device_types.sort_order
	;`

	pgxPool, err := config.GetPGXPool()
	if err != nil {
		return nil, fmt.Errorf("%w: %w", types.DatabaseConnError, err)
	}

	rows, err := pgxPool.Query(ctx, sqlQuery)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", types.DatabaseQueryError, err)
	}
	defer rows.Close()

	var allDeviceTypes []types.AllDeviceTypesRow
	for rows.Next() {
		if ctx.Err() != nil {
			return nil, fmt.Errorf("%w: %w", types.DatabaseRowIterationError, ctx.Err())
		}
		deviceType := new(types.AllDeviceTypesRow)
		if err := rows.Scan(
			&deviceType.DeviceType,
			&deviceType.DeviceTypeFormatted,
			&deviceType.DeviceMetaCategory,
			&deviceType.DeviceTypeCount,
		); err != nil {
			return nil, fmt.Errorf("%w: %w", types.DatabaseRowScanError, err)
		}
		allDeviceTypes = append(allDeviceTypes, *deviceType)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("%w: %w", types.DatabaseRowIterationError, err)
	}
	if len(allDeviceTypes) == 0 {
		return nil, nil
	}
	return allDeviceTypes, nil
}

func GetClientHardwareOverview(ctx context.Context, tag int64) ([]types.ClientHardwareView, error) {
	if err := types.IsTagnumberInt64Valid(&tag); err != nil {
		return nil, fmt.Errorf("%w: %w", types.InvalidFieldError, err)
	}
	const sqlQuery = `
	SELECT 
		ids.tagnumber, 
		ids.system_serial, 
		hardware_data.ethernet_mac, 
		hardware_data.wifi_mac,
		hardware_data.system_manufacturer,
		hardware_data.system_model,
		NULL AS "product_family",
		NULL AS "product_name",
		hardware_data.system_uuid,
		hardware_data.system_sku,
		hardware_data.chassis_type,
		hardware_data.motherboard_manufacturer,
		hardware_data.motherboard_serial,
		hardware_data.device_type,
		historical_hardware_data.memory_speed_mhz
	FROM 
		ids
	LEFT JOIN hardware_data ON ids.uuid = hardware_data.client_uuid
	LEFT JOIN historical_hardware_data ON ids.uuid = historical_hardware_data.client_uuid
	WHERE 
		ids.uuid = (SELECT uuid FROM ids WHERE tagnumber = $1 ORDER BY time DESC NULLS LAST LIMIT 1)
		AND historical_hardware_data.time IN (SELECT MAX(time) FROM historical_hardware_data WHERE client_uuid = ids.uuid AND historical_hardware_data.memory_speed_mhz IS NOT NULL GROUP BY client_uuid)
	ORDER BY 
		ids.time DESC NULLS LAST LIMIT 1
	;`

	pgxPool, err := config.GetPGXPool()
	if err != nil {
		return nil, fmt.Errorf("%w: %w", types.DatabaseConnError, err)
	}

	var clientHardwareData types.ClientHardwareView
	row := pgxPool.QueryRow(ctx, sqlQuery,
		tag,
	)
	if err := row.Scan(
		&clientHardwareData.Tagnumber,
		&clientHardwareData.SystemSerial,
		&clientHardwareData.EthernetMAC,
		&clientHardwareData.WiFiMAC,
		&clientHardwareData.SystemManufacturer,
		&clientHardwareData.SystemModel,
		&clientHardwareData.ProductFamily,
		&clientHardwareData.ProductName,
		&clientHardwareData.SystemUUID,
		&clientHardwareData.SystemSKU,
		&clientHardwareData.ChassisType,
		&clientHardwareData.MotherboardManufacturer,
		&clientHardwareData.MotherboardSerial,
		&clientHardwareData.DeviceType,
		&clientHardwareData.MemorySpeedMHz,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("error during row scan: %w", err)
	}
	return []types.ClientHardwareView{clientHardwareData}, nil
}

func SelectJobQueuePosition(ctx context.Context, tag int64) (int64, error) {
	const queuePositionMaxValue int64 = 1_000_000
	if err := types.IsTagnumberInt64Valid(&tag); err != nil {
		return queuePositionMaxValue, err
	}

	pgxPool, err := config.GetPGXPool()
	if err != nil {
		return queuePositionMaxValue, fmt.Errorf("%w: %w", types.DatabaseConnError, err)
	}

	clientUUID, err := GetClientUUIDByTag(ctx, pgxPool, tag)
	if err != nil {
		return queuePositionMaxValue, fmt.Errorf("%w: %w", types.DatabaseQueryError, err)
	}

	onlineClientData, err := config.GetAllOnlineClientsData()
	if err != nil {
		return queuePositionMaxValue, fmt.Errorf("%w: %w", types.ErrNoOnlineClients, err)
	}

	onlineClientUUIDs := make([]uuid.UUID, 0, len(onlineClientData))
	for _, realtimeData := range onlineClientData {
		onlineClientUUIDs = append(onlineClientUUIDs, realtimeData.ClientUUID)
	}

	if len(onlineClientData) == 0 || len(onlineClientUUIDs) == 0 {
		return queuePositionMaxValue, nil
	}

	const sqlQuery = `
		WITH job_queue_positions_cte AS (
			SELECT
				job_queue.client_uuid, 
				job_queue.job_name,
				ROW_NUMBER() OVER (
					ORDER BY job_queue.job_queued_at ASC NULLS LAST
				) AS "position_in_queue"
			FROM job_queue
			WHERE
				job_queue.job_name IN (
					'hpEraseAndClone',
					'hpCloneOnly',
					'generic-erase+clone',
					'generic-clone'
				)
				AND job_queue.job_queued_at IS NOT NULL
				AND (job_queue.job_queued = TRUE OR job_queue.job_name IS NOT NULL)
				AND job_queue.client_uuid = ANY($1::uuid[])
		)
		SELECT
			COALESCE((
				SELECT job_queue_positions_cte.position_in_queue
				FROM job_queue_positions_cte
				WHERE job_queue_positions_cte.client_uuid = $2
			), 0) AS "job_queue_position"
	;`

	var queuePosition sql.NullInt64
	row := pgxPool.QueryRow(ctx, sqlQuery,
		onlineClientUUIDs,
		clientUUID,
	)
	if err := row.Scan(
		&queuePosition,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return queuePositionMaxValue, nil
		}
		return queuePositionMaxValue, fmt.Errorf("error during row scan: %w", err)
	}
	if !queuePosition.Valid || queuePosition.Int64 < 0 || queuePosition.Int64 > queuePositionMaxValue {
		return queuePositionMaxValue, fmt.Errorf("invalid queue position value: %v", queuePosition.Int64)
	}
	return queuePosition.Int64, nil
}

func GetJobName(ctx context.Context, tag int64) (*string, error) {
	if err := types.IsTagnumberInt64Valid(&tag); err != nil {
		return nil, fmt.Errorf("%w: %w", types.InvalidFieldError, err)
	}

	const sqlCode = `
	SELECT 
		job_queue.job_name
	FROM
		job_queue
	WHERE
		tagnumber = $1
	;`

	pgxPool, err := config.GetPGXPool()
	if err != nil {
		return nil, fmt.Errorf("%w: %w", types.DatabaseConnError, err)
	}

	var jobName sql.NullString
	row := pgxPool.QueryRow(ctx, sqlCode, tag)
	if err := row.Scan(
		&jobName,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("%w: %w", types.DatabaseRowScanError, err)
	}
	return &jobName.String, nil
}

func GetFormattedJobName(ctx context.Context, jobName string) (*string, error) {
	if strings.TrimSpace(jobName) == "" {
		return nil, fmt.Errorf("job name is empty")
	}

	pgxPool, err := config.GetPGXPool()
	if err != nil {
		return nil, fmt.Errorf("%w: %w", types.DatabaseConnError, err)
	}

	const sqlCode = `
	SELECT 
		static_job_names.job_name_readable
	FROM
		static_job_names
	WHERE
		job_name = $1
	;`

	var jobNameFormatted sql.NullString
	row := pgxPool.QueryRow(ctx, sqlCode,
		jobName,
	)
	if err := row.Scan(
		&jobNameFormatted,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		} else {
			return nil, fmt.Errorf("%w: %w", types.DatabaseRowScanError, err)
		}
	}
	return &jobNameFormatted.String, nil
}

func GetAllBuildingsAndRooms(ctx context.Context) ([]types.AllBuildingsAndRooms, error) {
	dbConn, err := config.GetDatabaseConn()
	if err != nil {
		return nil, fmt.Errorf("%w: %w", types.DatabaseConnError, err)
	}
	const sqlQuery = `
		SELECT 
			building, 
			room, 
			COUNT(*) 
		FROM 
			locations 
		WHERE 
			building IS NOT NULL 
			AND room IS NOT NULL 
		GROUP BY 
			room, 
			building
		ORDER BY
			building ASC NULLS LAST,
			room ASC NULLS LAST
	;`

	rows, err := dbConn.QueryContext(ctx, sqlQuery)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", types.DatabaseQueryError, err)
	}
	defer rows.Close()

	var allBuildingsAndRooms []types.AllBuildingsAndRooms
	for rows.Next() {
		if ctx.Err() != nil {
			return nil, fmt.Errorf("%w: %w", types.DatabaseRowIterationError, ctx.Err())
		}
		var buildingRoomRow types.AllBuildingsAndRooms
		rowScanErr := rows.Scan(
			&buildingRoomRow.BuildingName,
			&buildingRoomRow.RoomName,
			&buildingRoomRow.ClientCount,
		)
		if rowScanErr != nil {
			return nil, fmt.Errorf("%w: %w", types.DatabaseRowScanError, rowScanErr)
		}
		allBuildingsAndRooms = append(allBuildingsAndRooms, buildingRoomRow)
	}
	if rows.Err() != nil {
		return nil, fmt.Errorf("%w: %w", types.DatabaseRowIterationError, rows.Err())
	}
	if len(allBuildingsAndRooms) == 0 {
		return nil, nil
	}
	return allBuildingsAndRooms, nil
}

func SelectCheckoutData(ctx context.Context, tag *int64) (*types.CheckoutLogResponse, error) {
	if err := types.IsTagnumberInt64Valid(tag); err != nil {
		return nil, err
	}
	dbConn, err := config.GetDatabaseConn()
	if err != nil {
		return nil, fmt.Errorf("%w: %w", types.DatabaseConnError, err)
	}
	const sqlQuery = `
	SELECT
		$1::BIGINT AS "tagnumber",
		customer_name,
		checkout_date,
		return_date
	FROM
		checkout_log
	WHERE
		client_uuid = (SELECT uuid FROM ids WHERE tagnumber = $1)
	ORDER BY
		time DESC NULLS LAST
	LIMIT 1
	;`

	var checkoutLogRow types.CheckoutLogResponse
	row := dbConn.QueryRowContext(ctx, sqlQuery,
		ptrToNullInt64(tag),
	)
	rowScanErr := row.Scan(
		&checkoutLogRow.Tagnumber,
		&checkoutLogRow.CustomerName,
		&checkoutLogRow.CheckoutDate,
		&checkoutLogRow.ReturnDate,
	)
	if rowScanErr != nil {
		if errors.Is(rowScanErr, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("%w: %w", types.DatabaseRowScanError, rowScanErr)
	}

	return &checkoutLogRow, nil
}

func SelectClientInfo(ctx context.Context, tag int64) (*types.ClientInfoResponse, error) {
	if err := types.IsTagnumberInt64Valid(&tag); err != nil {
		return nil, fmt.Errorf("%w: %w", types.InvalidFieldError, err)
	}

	pgxPool, err := config.GetPGXPool()
	if err != nil {
		return nil, fmt.Errorf("%w: %w", types.DatabaseConnError, err)
	}

	clientUUID, err := GetClientUUIDByTag(ctx, pgxPool, tag)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", types.DatabaseQueryError, err)
	}

	const sqlCode = `
	WITH client_files_cte AS (
		SELECT
			COUNT(*)::BIGINT AS file_count
		FROM client_images
		WHERE hidden = FALSE AND client_uuid = $1
	)
	SELECT 
		ids.tagnumber,
		ids.system_serial,
		ids.uuid,
		locations.time AS "locations_time",
		locations.location,
		locations.building,
		locations.room,
		static_department_info.department_name_formatted,
		static_ad_domains.domain_name_formatted AS "ou_name",
		locations.property_custodian,
		locations.acquired_date,
		locations.retired_date,
		static_client_statuses.status_formatted,
		locations.is_broken,
		locations.disk_removed,
		locations.note,
		jobstats.time AS "jobstats_time",
		jobstats.clone_completed,
		jobstats.clone_time,
		jobstats.clone_image,
		jobstats.erase_completed,
		jobstats.erase_time,
		jobstats.erase_mode,
		checkout_log.checkout_bool,
		checkout_log.checkout_date,
		checkout_log.return_date,
		checkout_log.customer_name,
		client_files_cte.file_count,
		os_info.time AS "os_info_time",
		(CASE WHEN jobstats.erase_completed IS TRUE AND jobstats.clone_completed IS DISTINCT FROM TRUE THEN FALSE ELSE TRUE END) AS "os_installed",
		COALESCE(os_info.os_name, image_version_cte.image_version) AS "os_name",
		(CASE WHEN locations.disk_removed = TRUE THEN NULL WHEN os_info.windows_build_number IS NOT NULL AND os_info.windows_ubr IS NOT NULL THEN CONCAT(os_info.windows_build_number, '.', os_info.windows_ubr) ELSE NULL END) AS "os_version",
		COALESCE(os_info.ad_computer_name, os_info.computer_name) AS "computer_name",
		os_info.admin_users,
		os_info.is_intune_joined,
		os_info.is_disk_encrypted,
		os_info.secure_boot_enabled,
		historical_firmware_data.bios_version,
		historical_firmware_data.bios_release_date,
		hardware_data.device_type,
		hardware_data.ethernet_mac,
		hardware_data.wifi_mac,
		hardware_data.tpm_version,
		hardware_data.system_manufacturer,
		hardware_data.system_model,
		hardware_data.system_sku,
		hardware_data.cpu_manufacturer,
		hardware_data.cpu_model,
		hardware_data.cpu_max_speed_mhz,
		hardware_data.cpu_core_count,
		hardware_data.cpu_thread_count,
		historical_hardware_data.time AS "historical_hardware_data_time",
		static_disk_stats.disk_type,
		( 100 - 
			historical_disk_data.disk_writes_kb::double precision
			/ NULLIF(static_disk_stats.max_kbw, 0)::double precision
			* 100
		) AS "disk_health_pcnt",
		(
			historical_battery_data.battery_current_max_capacity::double precision
			/ NULLIF(historical_battery_data.battery_design_capacity, 0)::double precision
			* 100
		) AS "battery_health_pcnt", 
 		historical_disk_data.disk_model,
		historical_disk_data.disk_size_kb,
		historical_disk_data.disk_serial,
		historical_disk_data.disk_writes_kb,
		historical_disk_data.disk_reads_kb,
		historical_disk_data.disk_power_on_hours,
		historical_disk_data.disk_error_count,
		historical_disk_data.disk_power_cycles,
		historical_disk_data.disk_firmware_version,
		historical_battery_data.battery_manufacturer,
		historical_battery_data.battery_model,
		historical_battery_data.battery_serial,
		historical_battery_data.battery_manufacture_date,
		historical_battery_data.battery_design_capacity,
		historical_battery_data.battery_current_max_capacity,
		historical_battery_data.battery_charge_cycles,
		historical_hardware_data.memory_serial,
		historical_hardware_data.memory_capacity_kb,
		historical_hardware_data.memory_speed_mhz,
		(CASE WHEN client_files_cte.file_count IS NOT NULL AND client_files_cte.file_count > 0 THEN client_files_cte.file_count ELSE 0 END) AS "file_count"
	FROM ids
		LEFT JOIN LATERAL (
			SELECT
				time,
				client_uuid,
				location,
				building,
				room,
				department_name,
				ad_domain,
				property_custodian,
				acquired_date,
				retired_date,
				client_status,
				is_broken,
				disk_removed,
				note
			FROM locations
			WHERE client_uuid = ids.uuid
			ORDER BY time DESC NULLS LAST
			LIMIT 1
		) locations ON TRUE
		LEFT JOIN static_department_info ON locations.department_name = static_department_info.department_name
		LEFT JOIN static_ad_domains ON locations.ad_domain = static_ad_domains.domain_name
		LEFT JOIN static_client_statuses ON locations.client_status = static_client_statuses.status_name
		LEFT JOIN LATERAL (
			SELECT
				client_uuid,
				device_type,
				ethernet_mac,
				wifi_mac,
				tpm_version,
				system_manufacturer,
				system_model,
				system_sku,
				cpu_manufacturer,
				cpu_model,
				cpu_max_speed_mhz,
				cpu_core_count,
				cpu_thread_count
			FROM hardware_data
			WHERE client_uuid = ids.uuid
			ORDER BY time DESC NULLS LAST
			LIMIT 1
		) hardware_data ON TRUE
		LEFT JOIN LATERAL (
			SELECT
				time,
				client_uuid,
				memory_serial,
				memory_capacity_kb,
				memory_speed_mhz
			FROM historical_hardware_data
			WHERE client_uuid = ids.uuid
			ORDER BY time DESC NULLS LAST
			LIMIT 1
		) historical_hardware_data ON TRUE
		LEFT JOIN LATERAL (
			SELECT
				time,
				client_uuid,
				disk_model,
				disk_size_kb,
				disk_serial,
				disk_writes_kb,
				disk_reads_kb,
				disk_power_on_hours,
				disk_error_count,
				disk_power_cycles,
				disk_firmware_version
			FROM historical_disk_data
			WHERE
				client_uuid = ids.uuid
				AND updated_from_windows = FALSE
				AND disk_writes_kb IS NOT NULL
				AND disk_model IS NOT NULL
			ORDER BY time DESC NULLS LAST
			LIMIT 1
		) historical_disk_data ON TRUE
		LEFT JOIN LATERAL (
			SELECT
				time,
				client_uuid,
				bios_version,
				bios_release_date
			FROM historical_firmware_data
			WHERE client_uuid = ids.uuid
			ORDER BY time DESC NULLS LAST
			LIMIT 1
		) historical_firmware_data ON TRUE
		LEFT JOIN LATERAL (
			SELECT
				time,
				client_uuid,
				battery_manufacturer,
				battery_model,
				battery_serial,
				battery_manufacture_date,
				battery_design_capacity,
				battery_current_max_capacity,
				battery_charge_cycles
			FROM historical_battery_data
			WHERE client_uuid = ids.uuid
			ORDER BY time DESC NULLS LAST
			LIMIT 1
		) historical_battery_data ON TRUE
		LEFT JOIN static_disk_stats ON historical_disk_data.disk_model = static_disk_stats.disk_model
		LEFT JOIN LATERAL (
			SELECT
				time,
				client_uuid,
				clone_completed,
				clone_time,
				clone_image,
				erase_completed,
				erase_time,
				erase_mode
			FROM jobstats
			WHERE
				client_uuid = ids.uuid
				AND (erase_completed = TRUE OR clone_completed = TRUE)
			ORDER BY time DESC NULLS LAST
			LIMIT 1
		) jobstats ON TRUE
		LEFT JOIN static_image_names ON jobstats.clone_image = static_image_names.image_name
		LEFT JOIN LATERAL (
			SELECT
				checkout_bool,
				checkout_date,
				return_date,
				customer_name
			FROM checkout_log
			WHERE client_uuid = ids.uuid
			ORDER BY time DESC NULLS LAST
			LIMIT 1
		) checkout_log ON TRUE
		LEFT JOIN LATERAL (
			SELECT
				time,
				client_uuid,
				os_name,
				windows_build_number,
				windows_ubr,
				os_version,
				ad_computer_name,
				computer_name,
				admin_users,
				is_intune_joined,
				is_disk_encrypted,
				secure_boot_enabled
			FROM os_info
			WHERE client_uuid = ids.uuid
			ORDER BY time DESC NULLS LAST
			LIMIT 1
		) os_info ON TRUE
		LEFT JOIN LATERAL (
			SELECT
				image_version
			FROM static_image_names
			WHERE system_model = hardware_data.system_model
			ORDER BY image_version DESC NULLS LAST
			LIMIT 1
		) image_version_cte ON TRUE
		LEFT JOIN client_files_cte ON TRUE
		LEFT JOIN LATERAL (
			SELECT 1 AS has_image
			FROM client_images
			WHERE client_uuid = ids.uuid
			LIMIT 1
		) image_presence ON TRUE
	WHERE 
		ids.uuid = $1
	;`

	rows, err := pgxPool.Query(ctx, sqlCode, clientUUID)
	if err != nil {
		return nil, fmt.Errorf("Error selecting client UUID: %w: %w", types.DatabaseQueryError, err)
	}
	defer rows.Close()

	var clientInfoResult types.ClientInfoResponse
	var adminUsers []string
	var memorySerialArr []string
	for rows.Next() {
		if ctx.Err() != nil {
			return nil, fmt.Errorf("%w: %w", types.DatabaseRowIterationError, ctx.Err())
		}
		rowScanErr := rows.Scan(
			&clientInfoResult.Tagnumber,
			&clientInfoResult.SystemSerial,
			&clientInfoResult.ClientUUID,
			&clientInfoResult.LocationEntryTime,
			&clientInfoResult.Location,
			&clientInfoResult.Building,
			&clientInfoResult.Room,
			&clientInfoResult.DepartmentName,
			&clientInfoResult.OUName,
			&clientInfoResult.PropertyCustodian,
			&clientInfoResult.AcquiredDate,
			&clientInfoResult.RetiredDate,
			&clientInfoResult.ClientStatus,
			&clientInfoResult.IsBroken,
			&clientInfoResult.DiskRemoved,
			&clientInfoResult.ClientNote,
			&clientInfoResult.JobStartTime,
			&clientInfoResult.CloneCompleted,
			&clientInfoResult.CloneJobDuration,
			&clientInfoResult.CloneImageName,
			&clientInfoResult.EraseCompleted,
			&clientInfoResult.EraseJobDuration,
			&clientInfoResult.EraseMode,
			&clientInfoResult.IsCheckedOut,
			&clientInfoResult.CheckoutDate,
			&clientInfoResult.ReturnDate,
			&clientInfoResult.CustomerName,
			&clientInfoResult.FileCount,
			&clientInfoResult.LastOSEntryTime,
			&clientInfoResult.OSInstalled,
			&clientInfoResult.OSName,
			&clientInfoResult.OSVersion,
			&clientInfoResult.ComputerName,
			&adminUsers,
			&clientInfoResult.IsIntuneJoined,
			&clientInfoResult.IsDiskEncrypted,
			&clientInfoResult.SecureBootEnabled,
			&clientInfoResult.BIOSVersion,
			&clientInfoResult.BIOSReleaseDate,
			&clientInfoResult.DeviceType,
			&clientInfoResult.EthernetMAC,
			&clientInfoResult.WiFiMAC,
			&clientInfoResult.TPMVersion,
			&clientInfoResult.SystemManufacturer,
			&clientInfoResult.SystemModel,
			&clientInfoResult.SystemSKU,
			&clientInfoResult.CPUManufacturer,
			&clientInfoResult.CPUModel,
			&clientInfoResult.CPUMaxSpeedMhz,
			&clientInfoResult.CPUCoreCount,
			&clientInfoResult.CPUThreadCount,
			&clientInfoResult.LastHardwareCheck,
			&clientInfoResult.DiskType,
			&clientInfoResult.DiskHealthPcnt,
			&clientInfoResult.BatteryHealthPcnt,
			&clientInfoResult.DiskModel,
			&clientInfoResult.DiskSizeKB,
			&clientInfoResult.DiskSerial,
			&clientInfoResult.DiskWritesKB,
			&clientInfoResult.DiskReadsKB,
			&clientInfoResult.DiskPowerOnHours,
			&clientInfoResult.DiskErrors,
			&clientInfoResult.DiskPowerCycles,
			&clientInfoResult.DiskFirmware,
			&clientInfoResult.BatteryManufacturer,
			&clientInfoResult.BatteryModel,
			&clientInfoResult.BatterySerial,
			&clientInfoResult.BatteryManufactureDate,
			&clientInfoResult.BatteryDesignCapacity,
			&clientInfoResult.BatteryCurrentMaxCapacity,
			&clientInfoResult.BatteryChargeCycles,
			&memorySerialArr,
			&clientInfoResult.MemoryCapacityKB,
			&clientInfoResult.MemorySpeedMHz,
			&clientInfoResult.FileCount,
		)
		if rowScanErr != nil {
			if errors.Is(rowScanErr, sql.ErrNoRows) {
				return nil, nil
			}
			return nil, fmt.Errorf("%w: %w", types.DatabaseRowScanError, rowScanErr)
		}
	}

	if len(adminUsers) > 0 {
		clientInfoResult.AdminUsers = adminUsers
	} else {
		clientInfoResult.AdminUsers = nil
	}
	if len(memorySerialArr) > 0 {
		clientInfoResult.MemorySerial = memorySerialArr
	} else {
		clientInfoResult.MemorySerial = nil
	}

	return &clientInfoResult, nil
}

func SelectDiskImageByModel(ctx context.Context, r *types.DiskImageNameRequest) (*types.DiskImageNameResponse, error) {
	if err := r.Validate(); err != nil {
		return nil, fmt.Errorf("%w: %w", types.InvalidFieldError, err)
	}

	const sqlCode = `
		SELECT
			static_image_names.system_model,
			static_image_names.image_name
		FROM
			static_image_names
		WHERE 
			static_image_names.image_name IS NOT NULL
			AND static_image_names.system_model = $1
	;`

	pgxPool, err := config.GetPGXPool()
	if err != nil {
		return nil, fmt.Errorf("%w: %w", types.DatabaseConnError, err)
	}

	row := pgxPool.QueryRow(ctx, sqlCode, r.SystemModel)
	diskImageName := new(types.DiskImageNameResponse)
	if err := row.Scan(
		&diskImageName.SystemModel,
		&diskImageName.ImageName,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("%w: %w", types.DatabaseRowScanError, err)
	}

	return diskImageName, nil
}

func ConvertClientInfoToCSV(ctx context.Context, tags []int64) (*bytes.Buffer, error) {
	if len(tags) == 0 {
		return nil, fmt.Errorf("no tags provided for CSV conversion")
	}

	var buf bytes.Buffer
	var dbQueryData []types.ClientInfoResponse
	buf.Grow(len(dbQueryData) * 200) // Grow by 200 bytes before another allocation
	for _, tag := range tags {
		if err := types.IsTagnumberInt64Valid(&tag); err != nil {
			return nil, fmt.Errorf("%w: %w", types.InvalidFieldError, err)
		}

		clientInfo, err := SelectClientInfo(ctx, tag)
		if err != nil {
			return nil, fmt.Errorf("%w: %w", types.DatabaseQueryError, err)
		}
		dbQueryData = append(dbQueryData, *clientInfo)
	}

	csvWriter := csv.NewWriter(&buf)

	var csvHeader = []string{
		"Tag",
		"Serial No.",
		"Ethernet MAC",
		"WiFi MAC",
		"Manufacturer",
		"Model",
		"SKU",
		"Device Type",
		"Location",
		"Building",
		"Room",
		"Department",
		"Property Owner",
		"Disk Removed",
		"OS Name",
		"OS Version",
		"Computer Name",
		"OU Name",
		"Is Intune Joined",
		"BIOS Version",
		"TPM Version",
		"Secure Boot",
		"CPU Model",
		"CPU Max Speed (MHz)",
		"CPU Core Count",
		"CPU Thread Count",
		"Memory Capacity (KB)",
		"Memory Speed (MHz)",
		"Memory Serial",
		"Disk Model",
		"Disk Serial",
		"Disk Capacity (KB)",
		"Disk Encrypted",
		"Status",
		"Broken",
		"Note",
		"Last Hardware Check",
	}

	if err := csvWriter.Write(csvHeader); err != nil {
		return nil, fmt.Errorf("Error writing CSV header in ConvertInventoryTableDataToCSV: %w", err)
	}
	csvWriter.Flush()
	if err := csvWriter.Error(); err != nil {
		return nil, fmt.Errorf("Error flushing CSV writer after writing header in ConvertInventoryTableDataToCSV: %w", err)
	}

	for _, row := range dbQueryData {
		record := []string{
			ptrIntToString(row.Tagnumber),
			ptrStringToString(row.SystemSerial),
			ptrStringToString(row.EthernetMAC),
			ptrStringToString(row.WiFiMAC),
			ptrStringToString(row.SystemManufacturer),
			ptrStringToString(row.SystemModel),
			ptrStringToString(row.SystemSKU),
			ptrStringToString(row.DeviceType),
			ptrStringToString(row.Location),
			ptrStringToString(row.Building),
			ptrStringToString(row.Room),
			ptrStringToString(row.DepartmentName),
			ptrStringToString(row.PropertyCustodian),
			ptrBoolToString(row.DiskRemoved),
			ptrStringToString(row.OSName),
			ptrStringToString(row.OSVersion),
			ptrStringToString(row.ComputerName),
			ptrStringToString(row.OUName),
			ptrBoolToString(row.IsIntuneJoined),
			ptrStringToString(row.BIOSVersion),
			ptrStringToString(row.TPMVersion),
			ptrBoolToString(row.SecureBootEnabled),
			ptrStringToString(row.CPUModel),
			ptrIntToString(row.CPUMaxSpeedMhz),
			ptrIntToString(row.CPUCoreCount),
			ptrIntToString(row.CPUThreadCount),
			ptrIntToString(row.MemoryCapacityKB),
			ptrIntToString(row.MemorySpeedMHz),
			ptrSliceToString(row.MemorySerial),
			ptrStringToString(row.DiskModel),
			ptrStringToString(row.DiskSerial),
			ptrIntToString(row.DiskSizeKB),
			ptrBoolToString(row.IsDiskEncrypted),
			ptrStringToString(row.ClientStatus),
			ptrBoolToString(row.IsBroken),
			ptrStringToString(row.ClientNote),
			ptrTimeToString(row.LastHardwareCheck),
		}
		if err := csvWriter.Write(record); err != nil {
			return nil, fmt.Errorf("Error writing CSV row in ConvertInventoryTableDataToCSV: %w", err)
		}
	}

	// Flush buffered data to the writer
	csvWriter.Flush()
	if err := csvWriter.Error(); err != nil {
		return nil, fmt.Errorf("Error flushing CSV writer in ConvertInventoryTableDataToCSV: %w", err)
	}

	return &buf, nil
}
