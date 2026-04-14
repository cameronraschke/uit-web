package database

// All SELECT queries should check for:
// 1. Basic input constraints/validation (type conversion should be done prior to)
// 2. Check context errors on row iteration
// 3. Get database connection from app state
// 4. Check for sql.ErrNoRows and return nil if error exists
// 5. Return any other errors

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"
	"uit-toolbox/config"
	"uit-toolbox/types"
)

type Select interface {
	GetManufacturersAndModels(ctx context.Context) ([]types.AllManufacturersAndModelsRow, error)
	CheckTwoFactorCode(ctx context.Context, twoFactorCode *string) (string, error)
	CheckAuthCredentials(ctx context.Context, username *string, password *string) (bool, *string, error)
	GetActiveJobs(ctx context.Context, tag *int64) (*types.ActiveJobs, error)
	GetJobQueueOverview(ctx context.Context) (*types.JobQueueOverview, error)
	GetNotes(ctx context.Context, noteType *string) (*types.GeneralNoteRow, error)
	GetLocationFormData(ctx context.Context, tag *int64, serial *string) (*types.InventoryFormPrefill, error)
	GetFileHashesFromTag(ctx context.Context, tag *int64) ([][]uint8, error)
	GetClientImageManifestByTag(ctx context.Context, tagnumber *int64) ([]types.ImageManifestView, error)
	GetClientBatteryReport(ctx context.Context) ([]types.ClientReportRow, error)
	GetAllJobs(ctx context.Context) ([]types.AllJobsRow, error)
	GetAllLocations(ctx context.Context) ([]types.AllLocationsRow, error)
	GetAllDeviceTypes(ctx context.Context) ([]types.DeviceType, error)
	GetClientHardwareOverview(ctx context.Context, tag int64) ([]types.ClientHardwareView, error)
	GetJobQueuePosition(ctx context.Context, tag int64) (int64, error)
	GetJobName(ctx context.Context, tag int64) (string, error)
	GetFormattedJobName(ctx context.Context, jobName string) (string, error)
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

func SelectAllIDs(ctx context.Context) ([]types.ClientLookupRow, error) {
	dbConn, err := config.GetDatabaseConn()
	if err != nil {
		return nil, fmt.Errorf("%w: %w", types.DatabaseConnError, err)
	}
	const sqlQuery = `
		SELECT 
			ids.tagnumber,
			ids.system_serial,
			locations.time AS "last_inventory_entry"
		FROM 
			ids
		INNER JOIN locations ON ids.uuid = locations.client_uuid
		GROUP BY ids.tagnumber, ids.system_serial, locations.time
		ORDER BY 
			locations.time DESC NULLS LAST, 
			ids.tagnumber ASC NULLS LAST, 
			ids.system_serial DESC NULLS LAST
	;`

	rows, err := dbConn.QueryContext(ctx, sqlQuery)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var globalLookupRow []types.ClientLookupRow
	for rows.Next() {
		if ctx.Err() != nil {
			return nil, fmt.Errorf("%w: %w", types.DatabaseRowIterationError, ctx.Err())
		}
		var tag types.ClientLookupRow
		if err := rows.Scan(
			&tag.Tagnumber,
			&tag.SystemSerial,
			&tag.LastInventoryEntry,
		); err != nil {
			return nil, fmt.Errorf("%w: %w", types.DatabaseRowScanError, err)
		}
		globalLookupRow = append(globalLookupRow, tag)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("%w: %w", types.DatabaseRowIterationError, err)
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
	if tag == nil || *tag <= 0 {
		tagErr = fmt.Errorf("tagnumber is nil or invalid")
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
	if ctx.Err() != nil {
		return nil, fmt.Errorf("context error: %w", ctx.Err())
	}

	dbConn, err := config.GetDatabaseConn()
	if err != nil {
		return nil, fmt.Errorf("%w: %w", types.DatabaseConnError, err)
	}

	sqlQuery := fmt.Sprintf(`
		SELECT 
			tagnumber, 
			system_serial 
		FROM 
			ids 
		%s
		ORDER BY 
			time DESC NULLS LAST 
		LIMIT 1
	;`, whereClause)

	var clientLookup types.ClientLookupRow
	row := dbConn.QueryRowContext(ctx, sqlQuery, whereArgs...)
	if err := row.Scan(
		&clientLookup.Tagnumber,
		&clientLookup.SystemSerial,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("query error: %w", err)
	}
	return &clientLookup, nil
}

func GetAllDepartments(ctx context.Context) ([]types.AllDepartmentsRow, error) {
	dbConn, err := config.GetDatabaseConn()
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

	rows, err := dbConn.QueryContext(ctx, sqlQuery)
	if err != nil {
		return nil, fmt.Errorf("error executing query: %w", err)
	}
	defer rows.Close()

	var departments []types.AllDepartmentsRow
	for rows.Next() {
		if ctx.Err() != nil {
			return nil, fmt.Errorf("%w: %w", types.DatabaseRowIterationError, ctx.Err())
		}
		var dept types.AllDepartmentsRow
		if err := rows.Scan(
			&dept.DepartmentName,
			&dept.DepartmentNameFormatted,
			&dept.DepartmentSortOrder,
			&dept.OrganizationName,
			&dept.OrganizationNameFormatted,
			&dept.OrganizationSortOrder,
			&dept.ClientCount,
		); err != nil {
			return nil, fmt.Errorf("%w: %w", types.DatabaseRowScanError, err)
		}
		departments = append(departments, dept)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("%w: %w", types.DatabaseRowIterationError, err)
	}
	if len(departments) == 0 {
		return nil, nil
	}

	return departments, nil
}

func GetAllDomains(ctx context.Context) ([]types.AllDomainsRow, error) {
	dbConn, err := config.GetDatabaseConn()
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

	rows, err := dbConn.QueryContext(ctx, sqlQuery)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", types.DatabaseQueryError, err)
	}
	defer rows.Close()

	var domains []types.AllDomainsRow
	for rows.Next() {
		if ctx.Err() != nil {
			return nil, fmt.Errorf("%w: %w", types.DatabaseRowIterationError, ctx.Err())
		}
		var domain types.AllDomainsRow
		if err := rows.Scan(
			&domain.DomainName,
			&domain.DomainNameFormatted,
			&domain.DomainSortOrder,
			&domain.ClientCount,
		); err != nil {
			return nil, fmt.Errorf("%w: %w", types.DatabaseRowScanError, err)
		}
		domains = append(domains, domain)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("%w: %w", types.DatabaseRowIterationError, err)
	}
	if len(domains) == 0 {
		return nil, nil
	}

	return domains, nil
}

func (repo *SelectRepo) GetManufacturersAndModels(ctx context.Context) ([]types.AllManufacturersAndModelsRow, error) {
	const sqlQuery = `SELECT system_manufacturer, system_model, COUNT(*) AS "system_model_count"
		FROM hardware_data 
		WHERE system_manufacturer IS NOT NULL 
			AND system_model IS NOT NULL
		GROUP BY system_manufacturer, system_model 
		ORDER BY system_manufacturer ASC, system_model ASC;`

	var manufacturersAndModels []types.AllManufacturersAndModelsRow
	rows, err := repo.DB.QueryContext(ctx, sqlQuery)
	if err != nil {
		return nil, fmt.Errorf("cannot execute query: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		if ctx.Err() != nil {
			return nil, fmt.Errorf("context error: %w", ctx.Err())
		}
		var row types.AllManufacturersAndModelsRow
		if err := rows.Scan(
			&row.SystemManufacturer,
			&row.SystemModel,
			&row.SystemModelCount); err != nil {
			return nil, fmt.Errorf("error during row scan: %w", err)
		}
		manufacturersAndModels = append(manufacturersAndModels, row)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error during row iteration: %w", err)
	}
	if len(manufacturersAndModels) == 0 {
		return nil, nil
	}

	manufacturerCountMap := make(map[string]int64, len(manufacturersAndModels))
	for _, row := range manufacturersAndModels {
		if row.SystemManufacturer == nil || row.SystemModelCount == nil {
			continue
		}
		manufacturerCountMap[*row.SystemManufacturer] += *row.SystemModelCount
	}

	for i := range manufacturersAndModels {
		if manufacturersAndModels[i].SystemManufacturer == nil {
			manufacturersAndModels[i].SystemManufacturerCount = nil
			continue
		}
		count := manufacturerCountMap[*manufacturersAndModels[i].SystemManufacturer]
		manufacturersAndModels[i].SystemManufacturerCount = &count
	}

	return manufacturersAndModels, nil
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
		return "", fmt.Errorf("error during row scan: %w", err)
	}

	return dbCode, nil
}

func (repo *SelectRepo) CheckAuthCredentials(ctx context.Context, username *string, password *string) (bool, *string, error) {
	if username == nil || password == nil || strings.TrimSpace(*username) == "" || strings.TrimSpace(*password) == "" {
		return false, nil, fmt.Errorf("username or password is empty")
	}

	if ctx.Err() != nil {
		return false, nil, fmt.Errorf("context error: %w", ctx.Err())
	}

	const sqlQuery = `SELECT password FROM logins WHERE username = $1 LIMIT 1;`

	var dbBcryptHash sql.NullString
	row := repo.DB.QueryRowContext(ctx, sqlQuery, ptrToNullString(username))
	if err := row.Scan(&dbBcryptHash); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil, fmt.Errorf("no results from db")
		}
		return false, nil, fmt.Errorf("error during row scan: %w", err)
	}

	if !dbBcryptHash.Valid || strings.TrimSpace(dbBcryptHash.String) == "" {
		return false, nil, nil
	}

	return true, &dbBcryptHash.String, nil
}

func (repo *SelectRepo) GetActiveJobs(ctx context.Context, tag *int64) (*types.ActiveJobs, error) {
	if tag == nil {
		return nil, fmt.Errorf("tagnumber is nil")
	}

	if ctx.Err() != nil {
		return nil, fmt.Errorf("context error: %w", ctx.Err())
	}

	const sqlQuery = `
	WITH job_queue_position AS (
		SELECT 
			tagnumber, 
			ROW_NUMBER() OVER (ORDER BY job_queued_at ASC NULLS LAST) AS "position",
			job_name
		FROM 
			job_queue 
		WHERE 
			job_queued = TRUE OR job_name IS NOT NULL
	)
	SELECT * FROM (SELECT 
			job_queue.tagnumber, 
			job_queue.job_queued, 
			job_queue.job_name, 
			job_queue.job_active, 
			DENSE_RANK() OVER (ORDER BY
			(CASE
				WHEN job_queue.job_name IN ('hpEraseAndClone', 'hpCloneOnly', 'generic-erase+clone', 'generic-clone') THEN COALESCE(job_queue_position.position, 0)
				ELSE 0
			END)) - 1 AS "job_queue_position"
		FROM job_queue
		LEFT JOIN job_queue_position ON job_queue.tagnumber = job_queue_position.tagnumber) t1
	WHERE t1.tagnumber = $1;`

	var activeJobs types.ActiveJobs
	row := repo.DB.QueryRowContext(ctx, sqlQuery, ptrToNullInt64(tag))
	if err := row.Scan(
		&activeJobs.Tagnumber,
		&activeJobs.JobQueued,
		&activeJobs.JobName,
		&activeJobs.JobActive,
		&activeJobs.QueuePosition,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("error during row scan: %w", err)
	}
	return &activeJobs, nil
}

func SelectIsClientJobAvailable(ctx context.Context, tag *int64) (*bool, error) {
	if tag == nil || *tag <= 0 {
		return nil, fmt.Errorf("tagnumber is nil or invalid")
	}

	if ctx.Err() != nil {
		return nil, fmt.Errorf("context error: %w", ctx.Err())
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
		return nil, fmt.Errorf("error during row scan: %w", err)
	}
	return &jobAvailable, nil
}

func (repo *SelectRepo) GetJobQueueOverview(ctx context.Context) (*types.JobQueueOverview, error) {
	if ctx.Err() != nil {
		return nil, fmt.Errorf("context error: %w", ctx.Err())
	}

	const sqlQuery = `SELECT t1.total_queued_jobs, t2.total_active_jobs, t3.total_active_blocking_jobs
	FROM 
	(SELECT COUNT(*) AS total_queued_jobs FROM job_queue WHERE job_queued = TRUE AND job_name IS NOT NULL AND EXTRACT(EPOCH FROM (CURRENT_TIMESTAMP - last_heard)) < 30) AS t1,
	(SELECT COUNT(*) AS total_active_jobs FROM job_queue WHERE job_queued = TRUE AND job_name IS NOT NULL AND job_active = TRUE AND EXTRACT(EPOCH FROM (CURRENT_TIMESTAMP - last_heard)) < 30) AS t2,
	(SELECT COUNT(*) AS total_active_blocking_jobs FROM job_queue WHERE job_queued = TRUE AND job_active = FALSE AND job_name IN ('hpEraseAndClone', 'hpCloneOnly', 'generic-erase+clone', 'generic-clone')) AS t3;`

	var jobQueueOverview types.JobQueueOverview
	row := repo.DB.QueryRowContext(ctx, sqlQuery)
	if err := row.Scan(
		&jobQueueOverview.TotalQueuedJobs,
		&jobQueueOverview.TotalActiveJobs,
		&jobQueueOverview.TotalActiveBlockingJobs,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("error during row scan: %w", err)
	}
	return &jobQueueOverview, nil
}

func (repo *SelectRepo) GetNotes(ctx context.Context, noteType *string) (*types.GeneralNoteRow, error) {
	if noteType == nil || strings.TrimSpace(*noteType) == "" {
		return nil, fmt.Errorf("noteType is nil or empty")
	}

	if ctx.Err() != nil {
		return nil, fmt.Errorf("context error: %w", ctx.Err())
	}

	const sqlQuery = `SELECT time, note_type, note 
		FROM notes 
		WHERE note_type = $1 
		ORDER BY time DESC NULLS LAST LIMIT 1;`

	var generalNoteRow types.GeneralNoteRow
	row := repo.DB.QueryRowContext(ctx, sqlQuery, ptrToNullString(noteType))
	if err := row.Scan(
		&generalNoteRow.Time,
		&generalNoteRow.NoteType,
		&generalNoteRow.Note,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("error during row scan: %w", err)
	}
	return &generalNoteRow, nil
}

func (repo *SelectRepo) GetLocationFormData(ctx context.Context, tag *int64, serial *string) (*types.InventoryFormPrefill, error) {
	if tag == nil && (serial == nil || strings.TrimSpace(*serial) == "") {
		return nil, fmt.Errorf("either tag or serial must be provided")
	}

	if ctx.Err() != nil {
		return nil, fmt.Errorf("context error: %w", ctx.Err())
	}

	const sqlQuery = `WITH files AS
	(
		SELECT client_uuid, COUNT(client_images.client_uuid) AS file_count from client_images WHERE hidden = FALSE GROUP BY client_uuid
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
	)
	SELECT 
		locations.time, 
		locations.tagnumber, 
		locations.system_serial, 
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
		checkout_log.checkout_date,
		checkout_log.return_date,
		locations.note,
		COALESCE(files.file_count, 0) AS "file_count"
	FROM locations
	LEFT JOIN files ON locations.client_uuid = files.client_uuid
	LEFT JOIN hardware_data ON locations.client_uuid = hardware_data.client_uuid
	LEFT JOIN client_health ON locations.client_uuid = client_health.client_uuid
	LEFT JOIN checkout_log ON locations.client_uuid = checkout_log.client_uuid AND checkout_log.log_entry_time IN (SELECT MAX(log_entry_time) FROM checkout_log WHERE log_entry_time IS NOT NULL GROUP BY client_uuid)
	LEFT JOIN static_department_info ON locations.department_name = static_department_info.department_name
	LEFT JOIN client_images ON locations.client_uuid = client_images.client_uuid
	LEFT JOIN default_system_model ON locations.client_uuid = default_system_model.client_uuid
	WHERE locations.client_uuid = (SELECT uuid FROM ids WHERE (tagnumber = $1 OR system_serial = $2) ORDER BY time DESC LIMIT 1)
	GROUP BY 
		locations.time,
		locations.tagnumber,
		locations.system_serial,
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
		checkout_log.checkout_date,
		checkout_log.return_date,
		locations.note,
		COALESCE(files.file_count, 0),
		files.client_uuid,
		client_images.client_uuid
	ORDER BY locations.time DESC NULLS LAST
	LIMIT 1;`
	row := repo.DB.QueryRowContext(ctx, sqlQuery,
		ptrToNullInt64(tag),
		ptrToNullString(serial),
	)

	inventoryUpdate := new(types.InventoryFormPrefill)
	if err := row.Scan(
		&inventoryUpdate.Time,
		&inventoryUpdate.Tagnumber,
		&inventoryUpdate.SystemSerial,
		&inventoryUpdate.Location,
		&inventoryUpdate.Building,
		&inventoryUpdate.Room,
		&inventoryUpdate.SystemManufacturer,
		&inventoryUpdate.SystemModel,
		&inventoryUpdate.DeviceType,
		&inventoryUpdate.Department,
		&inventoryUpdate.ADDomain,
		&inventoryUpdate.PropertyCustodian,
		&inventoryUpdate.AcquiredDate,
		&inventoryUpdate.RetiredDate,
		&inventoryUpdate.IsBroken,
		&inventoryUpdate.DiskRemoved,
		&inventoryUpdate.LastHardwareCheck,
		&inventoryUpdate.ClientStatus,
		&inventoryUpdate.CheckoutDate,
		&inventoryUpdate.ReturnDate,
		&inventoryUpdate.Note,
		&inventoryUpdate.FileCount,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("error during row scan: %w", err)
	}
	return inventoryUpdate, nil
}

func GetClientImageFilePathFromUUID(ctx context.Context, uuid *string) (*types.ImageManifestView, error) {
	if uuid == nil || strings.TrimSpace(*uuid) == "" {
		return nil, fmt.Errorf("%w: %s", types.MissingFieldError, "uuid")
	}

	dbConn, err := config.GetDatabaseConn()
	if err != nil {
		return nil, fmt.Errorf("%w: %w", types.DatabaseConnError, err)
	}

	const sqlQuery = `
		SELECT 
			tagnumber, 
			filename, 
			filepath, 
			thumbnail_filepath, 
			hidden
		FROM 
			client_images 
		WHERE
			uuid = $1
	;`

	row := dbConn.QueryRowContext(ctx, sqlQuery,
		ptrToNullString(uuid),
	)
	var imageManifest types.ImageManifestView
	if err := row.Scan(
		&imageManifest.Tagnumber,
		&imageManifest.FileName,
		&imageManifest.FilePath,
		&imageManifest.ThumbnailFilePath,
		&imageManifest.Hidden,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("%w: %w", types.DatabaseRowScanError, err)
	}
	return &imageManifest, nil
}

func (repo *SelectRepo) GetFileHashesFromTag(ctx context.Context, tag *int64) ([][]uint8, error) {
	if tag == nil {
		return nil, fmt.Errorf("tag is nil")
	}

	const maxHashByteLength = 64 // Some older hashes (PHP version) are 64 instead of 32 bytes

	const sqlQuery = `SELECT sha256_hash FROM client_images WHERE tagnumber = $1;`

	rows, err := repo.DB.QueryContext(ctx, sqlQuery, tag)
	if err != nil {
		return nil, fmt.Errorf("error during query execution: %w", err)
	}
	defer rows.Close()

	hashes := make([][]uint8, 0, 10)
	for rows.Next() {
		if ctx.Err() != nil {
			return nil, fmt.Errorf("context error: %w", ctx.Err())
		}
		var hash = make([]uint8, maxHashByteLength)
		if err := rows.Scan(&hash); err != nil {
			return nil, fmt.Errorf("error during row scan: %w", err)
		}
		if len(hash) > maxHashByteLength {
			return nil, fmt.Errorf("unexpected hash length: got %d, want less than %d", len(hash), maxHashByteLength)
		}
		hashes = append(hashes, hash)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error during row iteration: %w", err)
	}
	if len(hashes) == 0 {
		return nil, nil
	}
	return hashes, nil
}

func (repo *SelectRepo) GetClientImageManifestByTag(ctx context.Context, tagnumber *int64) ([]types.ImageManifestView, error) {
	if tagnumber == nil {
		return nil, fmt.Errorf("tagnumber is nil")
	}

	const sqlQuery = `SELECT time, tagnumber, uuid, filename, filepath, thumbnail_filepath, mime_type, hidden, pinned, note FROM client_images WHERE tagnumber = $1;`

	imageManifests := make([]types.ImageManifestView, 0, 10)
	rows, err := repo.DB.QueryContext(ctx, sqlQuery, tagnumber)
	if err != nil {
		return nil, fmt.Errorf("error during query execution: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		if ctx.Err() != nil {
			return nil, fmt.Errorf("context error: %w", ctx.Err())
		}
		var imageManifest types.ImageManifestView
		if err := rows.Scan(
			&imageManifest.Time,
			&imageManifest.Tagnumber,
			&imageManifest.UUID,
			&imageManifest.FileName,
			&imageManifest.FilePath,
			&imageManifest.ThumbnailFilePath,
			&imageManifest.MimeType,
			&imageManifest.Hidden,
			&imageManifest.Pinned,
			&imageManifest.Caption,
		); err != nil {
			return nil, fmt.Errorf("error during row scan: %w", err)
		}
		imageManifests = append(imageManifests, imageManifest)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error during row iteration: %w", err)
	}
	if len(imageManifests) == 0 {
		return nil, nil
	}
	return imageManifests, nil
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
		latest_historical_hardware_data AS (
			SELECT DISTINCT ON (historical_hardware_data.client_uuid) 
				historical_hardware_data.client_uuid, 
				historical_hardware_data.bios_version
			FROM historical_hardware_data
			ORDER BY historical_hardware_data.client_uuid, historical_hardware_data.time DESC NULLS LAST
		)
		SELECT
			locations.tagnumber, 
			locations.system_serial, 
			locations.location, 
			locationFormatting(locations.location) AS location_formatted, 
			locations.building, 
			locations.room,
			hardware_data.system_manufacturer, 
			hardware_data.system_model, 
			hardware_data.device_type, 
			static_device_types.device_type_formatted, 
			locations.department_name, 
			static_department_info.department_name_formatted,
			locations.ad_domain, 
			static_ad_domains.domain_name_formatted, 
			client_health.os_installed, 
			client_health.os_name, 
			client_health.last_hardware_check,
			(CASE 
				WHEN latest_historical_hardware_data.bios_version = static_bios_stats.bios_version THEN TRUE
				ELSE FALSE
			END) AS "bios_updated",
			latest_historical_hardware_data.bios_version,
			locations.client_status,
			static_client_statuses.status_formatted,
			locations.is_broken, 
			locations.disk_removed,
			locations.note, 
			locations.time AS last_updated, 
			files.file_count
		FROM locations
			LEFT JOIN hardware_data ON locations.client_uuid = hardware_data.client_uuid
			LEFT JOIN client_health ON locations.client_uuid = client_health.client_uuid
			LEFT JOIN static_department_info ON locations.department_name = static_department_info.department_name
			LEFT JOIN static_ad_domains ON locations.ad_domain = static_ad_domains.domain_name
			LEFT JOIN static_client_statuses ON locations.client_status = static_client_statuses.status_name
			LEFT JOIN static_device_types ON hardware_data.device_type = static_device_types.device_type
			LEFT JOIN files ON locations.client_uuid = files.client_uuid
			LEFT JOIN latest_historical_hardware_data ON locations.client_uuid = latest_historical_hardware_data.client_uuid
			LEFT JOIN static_bios_stats ON hardware_data.system_model = static_bios_stats.system_model
		WHERE %s
		GROUP BY 
			locations.tagnumber, 
			locations.system_serial,
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
			static_ad_domains.domain_name_formatted,
			client_health.os_installed,
			client_health.os_name,
			client_health.last_hardware_check,
			static_bios_stats.bios_version,
			latest_historical_hardware_data.bios_version,
			locations.client_status,
			static_client_statuses.status_formatted,
			locations.is_broken,
			locations.disk_removed,
			locations.note,
			locations.time,
			files.file_count
		ORDER BY locations.time DESC NULLS LAST
	;`, whereSQL)

	dbConn, err := config.GetDatabaseConn()
	if err != nil {
		return nil, fmt.Errorf("%w: %w", types.DatabaseConnError, err)
	}

	rows, err := dbConn.QueryContext(ctx, sqlQuery, whereArgs...)
	if err != nil {
		return nil, fmt.Errorf("query error: %w", err)
	}
	defer rows.Close()

	results := make([]types.InventoryTableRow, 0, approxClientCount)
	for rows.Next() {
		if err := ctx.Err(); err != nil {
			return nil, fmt.Errorf("context error: %w", err)
		}
		var row types.InventoryTableRow
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
			&row.DomainFormatted,
			&row.OsInstalled,
			&row.OsName,
			&row.LastHardwareCheck,
			&row.BIOSUpdated,
			&row.BIOSVersion,
			&row.Status,
			&row.StatusFormatted,
			&row.IsBroken,
			&row.DiskRemoved,
			&row.Note,
			&row.LastUpdated,
			&row.FileCount,
		); err != nil {
			return nil, fmt.Errorf("query error: %w", err)
		}
		results = append(results, row)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error during row iteration: %w", err)
	}
	if len(results) == 0 {
		return nil, nil
	}

	// Set client configuration errors
	for i := range results {
		// If client is missing required info in the DB
		if results[i].Tagnumber == nil ||
			results[i].SystemSerial == nil ||
			results[i].Location == nil ||
			results[i].Building == nil ||
			results[i].Room == nil ||
			results[i].SystemManufacturer == nil ||
			results[i].SystemModel == nil ||
			results[i].DeviceType == nil ||
			results[i].Department == nil ||
			results[i].Status == nil {
			missingRequiredInfo := types.MissingRequiredInfo.String()
			results[i].ClientErrors = append(results[i].ClientErrors, missingRequiredInfo)
		}
		// If client is broken
		if results[i].IsBroken != nil && *results[i].IsBroken {
			isBroken := types.IsBroken.String()
			results[i].ClientErrors = append(results[i].ClientErrors, isBroken)
		}
		// If client has status pre-property or retired status, it need to be erased
		if results[i].Status != nil && (*results[i].Status == "pre-property" || *results[i].Status == "retired") {
			if results[i].OsInstalled != nil && *results[i].OsInstalled {
				needsErasing := types.NeedsErasing.String()
				results[i].ClientErrors = append(results[i].ClientErrors, needsErasing)
			}
			// If client has status pre-property or retired status but disk is not removed
			if results[i].DiskRemoved != nil && !*results[i].DiskRemoved {
				diskNotRemoved := types.DiskNotRemoved.String()
				results[i].ClientErrors = append(results[i].ClientErrors, diskNotRemoved)
			}
		}
		// If disk is removed but OS is still marked as installed (need to update OS info in DB)
		if (results[i].DiskRemoved != nil && *results[i].DiskRemoved) && (results[i].OsInstalled == nil || (results[i].OsInstalled != nil && *results[i].OsInstalled)) {
			osOutdated := types.OSOutdated.String()
			results[i].ClientErrors = append(results[i].ClientErrors, osOutdated)
		}
		// If OS is installed but OS name is missing (need to update OS info)
		if results[i].OsInstalled != nil && *results[i].OsInstalled {
			if results[i].OsName == nil || strings.TrimSpace(*results[i].OsName) == "" {
				osMissing := types.OSOutdated.String()
				results[i].ClientErrors = append(results[i].ClientErrors, osMissing)
			}
		} else { // If OS is not installed
			osNotInstalled := types.OSNotInstalled.String()
			results[i].ClientErrors = append(results[i].ClientErrors, osNotInstalled)
			// if results[i].DiskRemoved != nil && !*results[i].DiskRemoved {
			// 	osNotInstalled := types.OSNotInstalled.String()
			// 	results[i].ClientErrors = append(results[i].ClientErrors, osNotInstalled)
			// }
		}
		// If BIOS out of date
		if results[i].BIOSVersion != nil && (results[i].BIOSUpdated != nil && !*results[i].BIOSUpdated) {
			biosOutdated := types.BIOSOutdated.String()
			if results[i].BIOSVersion != nil {
				results[i].ClientErrors = append(results[i].ClientErrors, biosOutdated+": "+*results[i].BIOSVersion)
			} else {
				results[i].ClientErrors = append(results[i].ClientErrors, biosOutdated)
			}
		}
		// If no hardware check in over 3 months
		if results[i].LastHardwareCheck == nil || (results[i].LastHardwareCheck != nil && time.Since(*results[i].LastHardwareCheck) > 90*24*time.Hour) {
			needsHardwareCheck := types.NeedsHardwareCheck.String()
			results[i].ClientErrors = append(results[i].ClientErrors, needsHardwareCheck)
		}
		// If AD domain not joined
		if results[i].ADDomain == nil || (results[i].ADDomain != nil && strings.TrimSpace(*results[i].ADDomain) == "") {
			domainNotJoined := types.DomainNotJoined.String()
			results[i].ClientErrors = append(results[i].ClientErrors, domainNotJoined)
		}
		if results[i].ADDomain != nil && *results[i].ADDomain == "none" {
			domainNotJoined := types.DomainNotJoined.String()
			results[i].ClientErrors = append(results[i].ClientErrors, domainNotJoined)
		}
		// If client is missing images of itself
		if results[i].FileCount == nil || (results[i].FileCount != nil && *results[i].FileCount <= 0) {
			missingImages := types.MissingImages.String()
			results[i].ClientErrors = append(results[i].ClientErrors, missingImages)
		}
	}

	return results, nil
}

func GetJobQueueTable(ctx context.Context) ([]types.JobQueueTableRowView, error) {
	dbConn, err := config.GetDatabaseConn()
	if err != nil {
		return nil, fmt.Errorf("%w: %w", types.DatabaseConnError, err)
	}

	const sqlQuery = `
	WITH avg_battery_health AS (
		SELECT system_model, AVG(avg_battery_health_pcnt) AS "avg_battery_health_pcnt" 
		FROM (
			SELECT hardware_data.system_model, (historical_hardware_data.battery_current_max_capacity::decimal / historical_hardware_data.battery_design_capacity::decimal * 100) AS "avg_battery_health_pcnt" 
			FROM 
				historical_hardware_data 
			LEFT JOIN 
				hardware_data ON historical_hardware_data.tagnumber = hardware_data.tagnumber
			WHERE 
				historical_hardware_data.battery_design_capacity IS NOT NULL 
				AND historical_hardware_data.battery_current_max_capacity IS NOT NULL
			GROUP BY 
				hardware_data.system_model,
				historical_hardware_data.tagnumber, 
				historical_hardware_data.battery_current_max_capacity, 
				historical_hardware_data.battery_design_capacity
		)
		GROUP BY system_model
	),
	current_battery_health AS (
		SELECT 
			DISTINCT ON (historical_hardware_data.tagnumber) historical_hardware_data.tagnumber,
			ROUND((historical_hardware_data.battery_current_max_capacity::decimal / historical_hardware_data.battery_design_capacity::decimal * 100), 2) AS "battery_health_pcnt"
		FROM 
			historical_hardware_data
		WHERE
			historical_hardware_data.battery_design_capacity IS NOT NULL 
			AND historical_hardware_data.battery_current_max_capacity IS NOT NULL
			GROUP BY historical_hardware_data.tagnumber, historical_hardware_data.battery_current_max_capacity, historical_hardware_data.battery_design_capacity
			ORDER BY historical_hardware_data.tagnumber DESC NULLS LAST
	),
	latest_historical_hardware_data AS (
		SELECT DISTINCT ON (historical_hardware_data.tagnumber) historical_hardware_data.time, historical_hardware_data.tagnumber, 
			historical_hardware_data.disk_type, historical_hardware_data.disk_size_kb AS "disk_capacity",
			historical_hardware_data.bios_version, historical_hardware_data.disk_model,
			historical_hardware_data.battery_design_capacity, historical_hardware_data.battery_current_max_capacity
		FROM historical_hardware_data
		ORDER BY historical_hardware_data.tagnumber, historical_hardware_data.time DESC NULLS LAST),
	latest_job AS (
		SELECT DISTINCT ON (jobstats.tagnumber) jobstats.time, jobstats.tagnumber,
			jobstats.erase_completed, jobstats.erase_mode, jobstats.erase_time, 
			jobstats.clone_completed, jobstats.clone_image, jobstats.clone_master, jobstats.clone_time, 
			jobstats.job_cancelled
		FROM jobstats
		WHERE jobstats.erase_completed = TRUE OR jobstats.clone_completed = TRUE
		ORDER BY jobstats.tagnumber, jobstats.time DESC NULLS LAST),
	newest_image AS (
		SELECT * FROM (
			SELECT 
				jobstats.time, 
				hardware_data.system_model, 
				ROW_NUMBER() OVER (PARTITION BY hardware_data.system_model ORDER BY jobstats.time DESC NULLS LAST) AS "row_num"
			FROM jobstats
			LEFT JOIN hardware_data ON jobstats.tagnumber = hardware_data.tagnumber
			WHERE
				jobstats.clone_master = TRUE 
				GROUP BY hardware_data.system_model, jobstats.time
				ORDER BY jobstats.time DESC NULLS LAST
		) t1 WHERE t1.row_num = 1
	),
	job_queue_position AS (
		SELECT 
			client_uuid, 
			ROW_NUMBER() OVER (ORDER BY job_queued_at ASC NULLS LAST) AS "position",
			job_name
		FROM 
			job_queue 
		WHERE 
			job_queued = TRUE OR job_name IS NOT NULL
	)
	SELECT
		locations.tagnumber,
		locations.system_serial,
		hardware_data.system_manufacturer,
		hardware_data.system_model,
		locationFormatting(locations.location) AS "location",
		static_department_info.department_name AS "department_name_formatted",
		static_client_statuses.status_formatted AS "client_status",
		locations.is_broken,
		locations.disk_removed,
		FALSE AS "temp_warning",
		(CASE WHEN static_client_statuses.status_name = 'checked_out' THEN TRUE ELSE FALSE END) AS "checkout_bool",
		TRUE AS "kernel_updated",
		job_queue.last_heard,
		job_queue.system_uptime,
		job_queue.client_app_uptime,
		(CASE WHEN EXTRACT(EPOCH FROM (CURRENT_TIMESTAMP - job_queue.last_heard)) < 30 THEN TRUE ELSE FALSE END) AS "online",
		job_queue.job_active,
		job_queue.job_queued,
		job_queue.job_queued_at,
		DENSE_RANK() OVER (ORDER BY
		(CASE
			WHEN job_queue.job_name IN ('hpEraseAndClone', 'hpCloneOnly', 'generic-erase+clone', 'generic-clone') THEN COALESCE(job_queue_position.position, 0)
			ELSE 0
		END)) - 1 AS "job_queue_position",
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
			WHEN latest_job.erase_completed = TRUE THEN latest_job.time
			WHEN latest_job.clone_completed = TRUE THEN latest_job.time
			ELSE NULL
		END) AS "last_job_time",
		(CASE 
			WHEN latest_job.clone_completed = TRUE THEN TRUE
			ELSE FALSE
		END) AS "os_installed",
		static_image_names.image_name_readable AS "os_name",
		(CASE
			WHEN latest_job.clone_completed = TRUE AND newest_image.time <= latest_job.time THEN TRUE
			ELSE FALSE
		END) AS "os_updated",
		(CASE 
			WHEN locations.ad_domain IS NOT NULL AND NOT locations.ad_domain = 'none' THEN TRUE
			ELSE FALSE
		END) AS "domain_joined",
		static_ad_domains.domain_name,
		static_ad_domains.domain_name_formatted AS "ad_domain_formatted",
		(CASE 
			WHEN latest_historical_hardware_data.bios_version = static_bios_stats.bios_version THEN TRUE
			ELSE FALSE
		END) AS "bios_updated",
		latest_historical_hardware_data.bios_version,
		job_queue.cpu_usage,
		job_queue.cpu_mhz,
		job_queue.cpu_temp,
		(CASE 
			WHEN job_queue.cpu_temp > 90 THEN TRUE
			ELSE FALSE
		END) AS "cpu_temp_warning",
		job_queue.memory_usage_kb,
		job_queue.memory_capacity_kb,
		'0' AS "disk_usage",
		job_queue.disk_temp,
		static_disk_stats.disk_type,
		latest_historical_hardware_data.disk_capacity AS "disk_size_kb",
		'80' AS "max_disk_temp",
		(CASE
			WHEN job_queue.disk_temp > 80 THEN TRUE
			ELSE FALSE
		END) AS "disk_temp_warning",
		'UP' AS "network_link_status",
		job_queue.network_speed AS "network_link_speed",
		'0' AS "network_usage",
		job_queue.battery_charge_pcnt,
		job_queue.battery_status,
		ROUND((battery_current_max_capacity::decimal / battery_design_capacity::decimal * 100), 2) AS "battery_health_pcnt",
		ROUND(current_battery_health.battery_health_pcnt - avg_battery_health.avg_battery_health_pcnt, 2) AS "battery_health_deviation",
		NULL AS "plugged_in",
		job_queue.watts_now AS "power_usage"
	FROM locations
	LEFT JOIN job_queue ON locations.tagnumber = job_queue.tagnumber
	LEFT JOIN hardware_data ON locations.tagnumber = hardware_data.tagnumber
	LEFT JOIN latest_historical_hardware_data ON locations.tagnumber = latest_historical_hardware_data.tagnumber
	LEFT JOIN avg_battery_health ON hardware_data.system_model = avg_battery_health.system_model
	LEFT JOIN current_battery_health ON locations.tagnumber = current_battery_health.tagnumber
	LEFT JOIN latest_job ON locations.tagnumber = latest_job.tagnumber
	LEFT JOIN static_image_names ON latest_job.clone_image = static_image_names.image_name
	LEFT JOIN static_job_names ON job_queue.job_name = static_job_names.job_name
	LEFT JOIN static_bios_stats ON hardware_data.system_model = static_bios_stats.system_model
	LEFT JOIN static_disk_stats ON latest_historical_hardware_data.disk_model = static_disk_stats.disk_model
	LEFT JOIN static_ad_domains ON locations.ad_domain = static_ad_domains.domain_name
	LEFT JOIN job_queue_position ON locations.client_uuid = job_queue_position.client_uuid
	LEFT JOIN newest_image ON hardware_data.system_model = newest_image.system_model
	LEFT JOIN static_client_statuses ON locations.client_status = static_client_statuses.status_name
	LEFT JOIN static_department_info ON locations.department_name = static_department_info.department_name
	ORDER BY
		job_queue.last_heard DESC NULLS LAST
	LIMIT 50
	;`

	rows, err := dbConn.QueryContext(ctx, sqlQuery)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", types.DatabaseQueryError, err)
	}
	defer rows.Close()

	jobQueueRows := make([]types.JobQueueTableRowView, 0, approxClientCount)
	for rows.Next() {
		if ctx.Err() != nil {
			return nil, fmt.Errorf("%w: %w", types.DatabaseRowIterationError, ctx.Err())
		}
		var row types.JobQueueTableRowView
		if err := rows.Scan(
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
			&row.LastHeard,
			&row.SystemUptime,
			&row.AppUptime,
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
			&row.OSUpdated,
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
		jobQueueRows = append(jobQueueRows, row)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("%w: %w", types.DatabaseRowIterationError, err)
	}
	if len(jobQueueRows) == 0 {
		return nil, nil
	}
	return jobQueueRows, nil
}

func (repo *SelectRepo) GetClientBatteryReport(ctx context.Context) ([]types.ClientReportRow, error) {
	const sqlQuery = `
	WITH avg_battery_health AS (
		SELECT system_model, AVG(avg_battery_health_pcnt) AS "avg_battery_health_pcnt" 
		FROM (
			SELECT hardware_data.system_model, (historical_hardware_data.battery_current_max_capacity::decimal / historical_hardware_data.battery_design_capacity::decimal * 100) AS "avg_battery_health_pcnt" 
			FROM 
				historical_hardware_data 
			LEFT JOIN 
				hardware_data ON historical_hardware_data.tagnumber = hardware_data.tagnumber
			WHERE 
				historical_hardware_data.battery_design_capacity IS NOT NULL 
				AND historical_hardware_data.battery_current_max_capacity IS NOT NULL
			GROUP BY 
				hardware_data.system_model,
				historical_hardware_data.tagnumber, 
				historical_hardware_data.battery_current_max_capacity, 
				historical_hardware_data.battery_design_capacity
		)
		GROUP BY system_model
	),
	current_battery_health AS (
		SELECT 
			tagnumber, ROUND((historical_hardware_data.battery_current_max_capacity::decimal / historical_hardware_data.battery_design_capacity::decimal * 100), 2) AS "battery_health_pcnt"
		FROM 
			historical_hardware_data
		WHERE 
			historical_hardware_data.time IN (SELECT MAX(time) FROM historical_hardware_data GROUP BY tagnumber)
			AND historical_hardware_data.battery_design_capacity IS NOT NULL 
			AND historical_hardware_data.battery_current_max_capacity IS NOT NULL
	)
	SELECT 
		historical_hardware_data.time AS "battery_health_timestamp", 
		historical_hardware_data.tagnumber, 
		current_battery_health.battery_health_pcnt AS "battery_health_pcnt", 
		ROUND(current_battery_health.battery_health_pcnt - avg_battery_health.avg_battery_health_pcnt, 2) AS "battery_health_deviation"
	FROM historical_hardware_data
	LEFT JOIN hardware_data ON historical_hardware_data.tagnumber = hardware_data.tagnumber
	LEFT JOIN avg_battery_health ON hardware_data.system_model = avg_battery_health.system_model
	LEFT JOIN current_battery_health ON historical_hardware_data.tagnumber = current_battery_health.tagnumber
	WHERE 
		historical_hardware_data.time IN (SELECT MAX(time) FROM historical_hardware_data GROUP BY tagnumber)
		AND historical_hardware_data.battery_design_capacity IS NOT NULL 
		AND historical_hardware_data.battery_current_max_capacity IS NOT NULL
	GROUP BY 
		hardware_data.system_model,
		historical_hardware_data.tagnumber, 
		historical_hardware_data.time, 
		avg_battery_health.avg_battery_health_pcnt,
		current_battery_health.battery_health_pcnt,
		historical_hardware_data.battery_current_max_capacity,
		historical_hardware_data.battery_design_capacity
	ORDER BY historical_hardware_data.time DESC NULLS LAST;
	`
	var clientReports []types.ClientReportRow
	rows, err := repo.DB.QueryContext(ctx, sqlQuery)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		if ctx.Err() != nil {
			return nil, fmt.Errorf("context error: %w", ctx.Err())
		}
		var clientReport types.ClientReportRow
		if err := rows.Scan(
			&clientReport.BatteryHealthTimestamp,
			&clientReport.Tagnumber,
			&clientReport.BatteryHealthPcnt,
			&clientReport.BatteryHealthDeviation,
		); err != nil {
			return nil, err
		}
		clientReports = append(clientReports, clientReport)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error during row iteration: %w", err)
	}
	if len(clientReports) == 0 {
		return nil, nil
	}
	return clientReports, nil
}

func (repo *SelectRepo) GetAllJobs(ctx context.Context) ([]types.AllJobsRow, error) {
	const sqlQuery = `SELECT static_job_names.job_name, 
		static_job_names.job_name_readable, 
		static_job_names.job_sort_order, 
		static_job_names.job_hidden
	FROM static_job_names
	ORDER BY static_job_names.job_sort_order ASC;`

	var allJobs []types.AllJobsRow
	rows, err := repo.DB.QueryContext(ctx, sqlQuery)
	if err != nil {
		return nil, fmt.Errorf("error during query execution: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		if ctx.Err() != nil {
			return nil, fmt.Errorf("context error: %w", ctx.Err())
		}
		var job types.AllJobsRow
		if err := rows.Scan(
			&job.JobName,
			&job.JobNameReadable,
			&job.JobSortOrder,
			&job.JobHidden,
		); err != nil {
			return nil, fmt.Errorf("error during row scan: %w", err)
		}
		allJobs = append(allJobs, job)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error during row iteration: %w", err)
	}
	if len(allJobs) == 0 {
		return nil, nil
	}
	return allJobs, nil
}

func (repo *SelectRepo) GetAllLocations(ctx context.Context) ([]types.AllLocationsRow, error) {
	const sqlQuery = `
		SELECT 
			location, 
			MAX(time) AS "timestamp",
			COUNT(*) as "location_count"
		FROM 
			locations
		GROUP BY 
			location
		ORDER BY 
			location ASC, timestamp DESC
	;`

	var allLocations []types.AllLocationsRow
	rows, err := repo.DB.QueryContext(ctx, sqlQuery)
	if err != nil {
		return nil, fmt.Errorf("error during query execution: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		if ctx.Err() != nil {
			return nil, fmt.Errorf("context error: %w", ctx.Err())
		}
		var location types.AllLocationsRow
		if err := rows.Scan(
			&location.Location,
			&location.Timestamp,
			&location.LocationCount,
		); err != nil {
			return nil, fmt.Errorf("error during row scan: %w", err)
		}
		allLocations = append(allLocations, location)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error during row iteration: %w", err)
	}
	if len(allLocations) == 0 {
		return nil, nil
	}
	return allLocations, nil
}

func GetAllStatuses(ctx context.Context) (map[string][]types.AllClientStatuses, error) {
	dbConn, err := config.GetDatabaseConn()
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

	rows, err := dbConn.QueryContext(ctx, sqlQuery)
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
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("%w: %w", types.DatabaseRowIterationError, err)
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

func (repo *SelectRepo) GetAllDeviceTypes(ctx context.Context) ([]types.DeviceType, error) {
	const sqlQuery = `SELECT static_device_types.device_type, 
			static_device_types.device_type_formatted, 
			static_device_types.device_meta_category, 
			COUNT(hardware_data.device_type) AS "device_type_count"
		FROM static_device_types 
		LEFT JOIN hardware_data ON static_device_types.device_type = hardware_data.device_type
		GROUP BY static_device_types.device_type, 
			static_device_types.device_type_formatted, 
			static_device_types.device_meta_category, 
			static_device_types.sort_order;`
	rows, err := repo.DB.QueryContext(ctx, sqlQuery)
	if err != nil {
		return nil, fmt.Errorf("error during query execution: %w", err)
	}
	defer rows.Close()

	var allDeviceTypes []types.DeviceType
	for rows.Next() {
		if ctx.Err() != nil {
			return nil, fmt.Errorf("context error: %w", ctx.Err())
		}
		var deviceType types.DeviceType
		if err := rows.Scan(
			&deviceType.DeviceType,
			&deviceType.DeviceTypeFormatted,
			&deviceType.DeviceMetaCategory,
			&deviceType.DeviceTypeCount,
		); err != nil {
			return nil, fmt.Errorf("error during row scan: %w", err)
		}
		allDeviceTypes = append(allDeviceTypes, deviceType)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error during row iteration: %w", err)
	}
	if len(allDeviceTypes) == 0 {
		return nil, nil
	}
	return allDeviceTypes, nil
}

func (repo *SelectRepo) GetClientHardwareOverview(ctx context.Context, tag int64) ([]types.ClientHardwareView, error) {
	if tag == 0 {
		return nil, fmt.Errorf("tagnumer cannot be nil")
	}
	const sqlQuery = `
	SELECT 
		locations.tagnumber, 
		locations.system_serial, 
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
		locations
	LEFT JOIN hardware_data ON locations.client_uuid = hardware_data.client_uuid
	LEFT JOIN historical_hardware_data ON locations.client_uuid = historical_hardware_data.client_uuid
	WHERE 
		locations.client_uuid = (SELECT client_uuid FROM locations WHERE tagnumber = $1 ORDER BY time DESC NULLS LAST LIMIT 1)
		AND historical_hardware_data.time IN (SELECT MAX(time) FROM historical_hardware_data GROUP BY tagnumber)
	ORDER BY 
		locations.time DESC NULLS LAST LIMIT 1
	;`

	var clientHardwareData types.ClientHardwareView
	row := repo.DB.QueryRowContext(ctx, sqlQuery, tag)
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

func (repo *SelectRepo) GetJobQueuePosition(ctx context.Context, tag int64) (int64, error) {
	if tag == 0 {
		return 0, fmt.Errorf("tagnumer cannot be nil")
	}
	const sqlQuery = `
	WITH job_queue_position AS (
		SELECT 
			tagnumber, 
			ROW_NUMBER() OVER (ORDER BY job_queued_at ASC NULLS LAST) AS "position",
			job_name
		FROM 
			job_queue 
		WHERE 
			job_queued = TRUE OR job_name IS NOT NULL
	)
	SELECT job_queue_position FROM (
		SELECT
			job_queue.tagnumber,
			DENSE_RANK() OVER (ORDER BY
				(CASE
					WHEN job_queue.job_name IN ('hpEraseAndClone', 'hpCloneOnly', 'generic-erase+clone', 'generic-clone') THEN COALESCE(job_queue_position.position, 0)
					ELSE 0
				END)) - 1 AS "job_queue_position"
			FROM 
				job_queue
			LEFT JOIN 
				job_queue_position ON job_queue.tagnumber = job_queue_position.tagnumber
	) t1
	WHERE t1.tagnumber = $1;
	;`

	var queuePosition sql.NullInt64
	row := repo.DB.QueryRowContext(ctx, sqlQuery, tag)
	if err := row.Scan(
		&queuePosition,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, nil
		} else {
			return 0, fmt.Errorf("error during row scan: %w", err)
		}
	}
	return queuePosition.Int64, nil
}

func (repo *SelectRepo) GetJobName(ctx context.Context, tag int64) (string, error) {
	if tag == 0 {
		return "", fmt.Errorf("tagnumber is empty")
	}

	const sqlCode = `
	SELECT 
		job_queue.job_name
	FROM
		job_queue
	WHERE
		tagnumber = $1
	;`

	var jobName sql.NullString
	row := repo.DB.QueryRowContext(ctx, sqlCode, tag)
	if err := row.Scan(
		&jobName,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", nil
		} else {
			return "", fmt.Errorf("error during row scan: %w", err)
		}
	}
	return jobName.String, nil
}

func (repo *SelectRepo) GetFormattedJobName(ctx context.Context, jobName string) (string, error) {
	if strings.TrimSpace(jobName) == "" {
		return "", fmt.Errorf("job name is empty")
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
	row := repo.DB.QueryRowContext(ctx, sqlCode, jobName)
	if err := row.Scan(
		&jobNameFormatted,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", nil
		} else {
			return "", fmt.Errorf("error during row scan: %w", err)
		}
	}
	return jobNameFormatted.String, nil
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

	var buildingAndRooms []types.AllBuildingsAndRooms
	for rows.Next() {
		if ctx.Err() != nil {
			return nil, fmt.Errorf("%w: %w", types.DatabaseRowIterationError, ctx.Err())
		}
		var buildingAndRoom types.AllBuildingsAndRooms
		if err := rows.Scan(
			&buildingAndRoom.BuildingName,
			&buildingAndRoom.RoomName,
			&buildingAndRoom.ClientCount,
		); err != nil {
			return nil, fmt.Errorf("%w: %w", types.DatabaseRowScanError, err)
		}
		buildingAndRooms = append(buildingAndRooms, buildingAndRoom)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("%w: %w", types.DatabaseRowIterationError, err)
	}
	if len(buildingAndRooms) == 0 {
		return nil, nil
	}
	return buildingAndRooms, nil
}
