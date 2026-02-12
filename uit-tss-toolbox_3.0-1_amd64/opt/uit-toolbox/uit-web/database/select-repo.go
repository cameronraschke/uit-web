package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"uit-toolbox/config"
)

type Select interface {
	AllTags(ctx context.Context) ([]int64, error)
	GetDepartments(ctx context.Context) ([]Department, error)
	GetDomains(ctx context.Context) ([]Domain, error)
	GetManufacturersAndModels(ctx context.Context) ([]ManufacturersAndModels, error)
	ClientLookupByTag(ctx context.Context, tag *int64) (*ClientLookup, error)
	ClientLookupBySerial(ctx context.Context, serial *string) (*ClientLookup, error)
	GetHardwareIdentifiers(ctx context.Context, tag *int64) (*HardwareData, error)
	GetBiosData(ctx context.Context, tag *int64) (*BiosData, error)
	GetOsData(ctx context.Context, tag *int64) (*OsData, error)
	GetActiveJobs(ctx context.Context, tag *int64) (*ActiveJobs, error)
	GetAvailableJobs(ctx context.Context, tag *int64) (*AvailableJobs, error)
	GetJobQueueOverview(ctx context.Context) (*JobQueueOverview, error)
	GetNotes(ctx context.Context, noteType *string) (*NotesTable, error)
	GetDashboardInventorySummary(ctx context.Context) ([]DashboardInventorySummary, error)
	GetLocationFormData(ctx context.Context, tag *int64, serial *string) (*InventoryUpdateForm, error)
	GetClientImageFilePathFromUUID(ctx context.Context, uuid *string) (*ImageManifest, error)
	GetClientImageManifestByTag(ctx context.Context, tagnumber *int64) ([]ImageManifest, error)
	GetInventoryTableData(ctx context.Context, filterOptions *InventoryAdvSearchOptions) ([]InventoryTableData, error)
	GetClientBatteryHealth(ctx context.Context, tagnumber *int64) (*ClientBatteryHealth, error)
	GetJobQueueTable(ctx context.Context) ([]JobQueueTableRow, error)
	GetBatteryStandardDeviation(ctx context.Context) ([]ClientReport, error)
	GetAllJobs(ctx context.Context) ([]AllJobs, error)
	GetAllLocations(ctx context.Context) ([]AllLocations, error)
	GetAllStatuses(ctx context.Context) ([]ClientStatus, error)
	GetAllDeviceTypes(ctx context.Context) ([]DeviceType, error)
}

type SelectRepo struct {
	DB *sql.DB
}

func NewSelectRepo() (Select, error) {
	db := config.GetDatabaseConn()
	if db == nil {
		return nil, fmt.Errorf("db connection is nil in NewSelectRepo")
	}
	return &SelectRepo{DB: db}, nil
}

var _ Select = (*SelectRepo)(nil)

func (repo *SelectRepo) AllTags(ctx context.Context) ([]int64, error) {
	const sqlQuery = `SELECT tagnumber FROM (SELECT tagnumber, time, ROW_NUMBER() OVER (PARTITION BY tagnumber ORDER BY time DESC) AS
		row_nums FROM locations WHERE tagnumber IS NOT NULL) t1 WHERE t1.row_nums = 1 ORDER BY t1.time DESC;`

	rows, err := repo.DB.QueryContext(ctx, sqlQuery)
	if err != nil {
		return nil, fmt.Errorf("error executing query: %w", err)
	}
	defer rows.Close()

	var allTags []AllTags
	for rows.Next() {
		if ctx.Err() != nil {
			return nil, fmt.Errorf("context error: %w", ctx.Err())
		}
		var tag AllTags
		if err := rows.Scan(&tag.Tagnumber); err != nil {
			return nil, fmt.Errorf("error scanning row: %w", err)
		}
		allTags = append(allTags, tag)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error during row iteration: %w", err)
	}
	if len(allTags) == 0 {
		return nil, nil
	}

	allTagsSlice := make([]int64, len(allTags))
	for i := range allTags {
		if allTags[i].Tagnumber != nil {
			allTagsSlice[i] = *allTags[i].Tagnumber
		}
	}

	return allTagsSlice, nil
}

func (repo *SelectRepo) GetDepartments(ctx context.Context) ([]Department, error) {
	const sqlQuery = `SELECT 
			static_department_info.department_name, 
			static_department_info.department_name_formatted, 
			static_department_info.department_sort_order,
			COALESCE(static_organizations.organization_name, ''),
			COALESCE(static_organizations.organization_name_formatted, ''),
			COALESCE(static_organizations.organization_sort_order, 101)
		FROM static_department_info 
		LEFT JOIN static_organizations ON static_organizations.organization_name = static_department_info.organization_name
		ORDER BY static_organizations.organization_sort_order, static_department_info.department_sort_order;`

	rows, err := repo.DB.QueryContext(ctx, sqlQuery)
	if err != nil {
		return nil, fmt.Errorf("error executing query: %w", err)
	}
	defer rows.Close()

	var departments []Department
	for rows.Next() {
		if ctx.Err() != nil {
			return nil, fmt.Errorf("context error: %w", ctx.Err())
		}
		var dept Department
		if err := rows.Scan(
			&dept.DepartmentName,
			&dept.DepartmentNameFormatted,
			&dept.DepartmentSortOrder,
			&dept.OrganizationName,
			&dept.OrganizationNameFormatted,
			&dept.OrganizationSortOrder,
		); err != nil {
			return nil, fmt.Errorf("error during row scan: %w", err)
		}
		departments = append(departments, dept)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error during row iteration: %w", err)
	}
	if len(departments) == 0 {
		return nil, nil
	}

	return departments, nil
}

func (repo *SelectRepo) GetDomains(ctx context.Context) ([]Domain, error) {
	const sqlQuery = `SELECT domain_name, domain_name_formatted 
		FROM static_ad_domains 
		ORDER BY domain_sort_order NULLS LAST;`
	rows, err := repo.DB.QueryContext(ctx, sqlQuery)
	if err != nil {
		return nil, fmt.Errorf("error executing query: %w", err)
	}
	defer rows.Close()

	var domains []Domain
	for rows.Next() {
		if ctx.Err() != nil {
			return nil, fmt.Errorf("context error: %w", ctx.Err())
		}
		var domain Domain
		if err := rows.Scan(&domain.DomainName, &domain.DomainNameFormatted); err != nil {
			return nil, fmt.Errorf("error during row scan: %w", err)
		}
		domains = append(domains, domain)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error during row iteration: %w", err)
	}
	if len(domains) == 0 {
		return nil, nil
	}

	return domains, nil
}

func (repo *SelectRepo) GetManufacturersAndModels(ctx context.Context) ([]ManufacturersAndModels, error) {
	const sqlQuery = `SELECT system_manufacturer, system_model, COUNT(*) AS "system_model_count"
		FROM hardware_data 
		WHERE system_manufacturer IS NOT NULL 
			AND system_model IS NOT NULL
		GROUP BY system_manufacturer, system_model 
		ORDER BY system_manufacturer ASC, system_model ASC;`

	var manufacturersAndModels []ManufacturersAndModels
	rows, err := repo.DB.QueryContext(ctx, sqlQuery)
	if err != nil {
		return nil, fmt.Errorf("cannot execute query: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		if ctx.Err() != nil {
			return nil, fmt.Errorf("context error: %w", ctx.Err())
		}
		var row ManufacturersAndModels
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

func (repo *SelectRepo) ClientLookupByTag(ctx context.Context, tag *int64) (*ClientLookup, error) {
	if tag == nil {
		return nil, fmt.Errorf("tagnumber is nil")
	}

	if ctx.Err() != nil {
		return nil, fmt.Errorf("context error: %w", ctx.Err())
	}

	const sqlQuery = `SELECT tagnumber, system_serial 
		FROM locations 
		WHERE tagnumber = $1 
		ORDER BY time DESC NULLS LAST LIMIT 1;`

	var clientLookup ClientLookup
	row := repo.DB.QueryRowContext(ctx, sqlQuery, ToNullInt64(tag))
	if err := row.Scan(&clientLookup.Tagnumber, &clientLookup.SystemSerial); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("query error: %w", err)
	}
	return &clientLookup, nil
}

func (repo *SelectRepo) ClientLookupBySerial(ctx context.Context, serial *string) (*ClientLookup, error) {
	if serial == nil || strings.TrimSpace(*serial) == "" {
		return nil, fmt.Errorf("serial is nil")
	}

	if ctx.Err() != nil {
		return nil, fmt.Errorf("context error: %w", ctx.Err())
	}

	const sqlQuery = `SELECT tagnumber, system_serial 
		FROM locations 
		WHERE system_serial = $1 
		ORDER BY time DESC NULLS LAST LIMIT 1;`

	row := repo.DB.QueryRowContext(ctx, sqlQuery, ToNullString(serial))
	var clientLookup ClientLookup
	if err := row.Scan(&clientLookup.Tagnumber, &clientLookup.SystemSerial); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("query error: %w", err)
	}
	return &clientLookup, nil
}

func (repo *SelectRepo) GetHardwareIdentifiers(ctx context.Context, tag *int64) (*HardwareData, error) {
	if tag == nil {
		return nil, fmt.Errorf("tagnumber is nil")
	}

	if ctx.Err() != nil {
		return nil, fmt.Errorf("context error: %w", ctx.Err())
	}

	const sqlQuery = `SELECT locations.tagnumber, locations.system_serial, jobstats.etheraddress, hardware_data.wifi_mac,
	hardware_data.system_model, hardware_data.system_uuid, hardware_data.system_sku, hardware_data.chassis_type, 
	hardware_data.motherboard_manufacturer, hardware_data.motherboard_serial, hardware_data.system_manufacturer
	FROM locations
	LEFT JOIN jobstats ON locations.tagnumber = jobstats.tagnumber AND jobstats.time IN (SELECT MAX(time) FROM jobstats GROUP BY tagnumber)
	LEFT JOIN hardware_data ON locations.tagnumber = hardware_data.tagnumber
	WHERE locations.time IN (SELECT MAX(time) FROM locations GROUP BY tagnumber)
	AND locations.tagnumber = $1;`

	var hardwareData HardwareData
	row := repo.DB.QueryRowContext(ctx, sqlQuery, ToNullInt64(tag))
	if err := row.Scan(
		&hardwareData.Tagnumber,
		&hardwareData.SystemSerial,
		&hardwareData.EthernetMAC,
		&hardwareData.WifiMac,
		&hardwareData.SystemModel,
		&hardwareData.SystemUUID,
		&hardwareData.SystemSKU,
		&hardwareData.ChassisType,
		&hardwareData.MotherboardManufacturer,
		&hardwareData.MotherboardSerial,
		&hardwareData.SystemManufacturer,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("error during row scan: %w", err)
	}
	return &hardwareData, nil
}

func (repo *SelectRepo) GetBiosData(ctx context.Context, tag *int64) (*BiosData, error) {
	if tag == nil {
		return nil, fmt.Errorf("tagnumber is nil")
	}

	if ctx.Err() != nil {
		return nil, fmt.Errorf("context error: %w", ctx.Err())
	}

	const sqlQuery = `SELECT client_health.tagnumber, client_health.bios_version, client_health.bios_updated, 
	client_health.tpm_version 
	FROM client_health WHERE client_health.tagnumber = $1;`

	var biosData BiosData
	row := repo.DB.QueryRowContext(ctx, sqlQuery, ToNullInt64(tag))
	if err := row.Scan(
		&biosData.Tagnumber,
		&biosData.BiosVersion,
		&biosData.BiosUpdated,
		&biosData.BiosDate,
		&biosData.TpmVersion,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("error during row scan: %w", err)
	}
	return &biosData, nil
}

func (repo *SelectRepo) GetOsData(ctx context.Context, tag *int64) (*OsData, error) {
	if tag == nil {
		return nil, fmt.Errorf("tagnumber is nil")
	}

	if ctx.Err() != nil {
		return nil, fmt.Errorf("context error: %w", ctx.Err())
	}

	const sqlQuery = `SELECT locations.tagnumber, client_health.os_installed, client_health.os_name,
	client_health.last_imaged_time, client_health.tpm_version, jobstats.boot_time
	FROM locations
	LEFT JOIN client_health ON locations.tagnumber = client_health.tagnumber
	LEFT JOIN jobstats ON locations.tagnumber = jobstats.tagnumber AND jobstats.time IN (SELECT MAX(time) FROM jobstats GROUP BY tagnumber)
	WHERE locations.time IN (SELECT MAX(time) FROM locations GROUP BY tagnumber)
	AND locations.tagnumber = $1;`

	var osData OsData
	row := repo.DB.QueryRowContext(ctx, sqlQuery, ToNullInt64(tag))
	if err := row.Scan(
		&osData.Tagnumber,
		&osData.OsInstalled,
		&osData.OsName,
		&osData.OsInstalledTime,
		&osData.TPMversion,
		&osData.BootTime,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("error during row scan: %w", err)
	}
	return &osData, nil
}

func (repo *SelectRepo) GetActiveJobs(ctx context.Context, tag *int64) (*ActiveJobs, error) {
	if tag == nil {
		return nil, fmt.Errorf("tagnumber is nil")
	}

	if ctx.Err() != nil {
		return nil, fmt.Errorf("context error: %w", ctx.Err())
	}

	const sqlQuery = `SELECT job_queue.tagnumber, job_queue.job_queued, job_queue.job_active, t1.queue_position
	FROM job_queue
	LEFT JOIN (SELECT tagnumber, ROW_NUMBER() OVER (PARTITION BY tagnumber ORDER BY present DESC) AS queue_position FROM job_queue) AS t1 
		ON job_queue.tagnumber = t1.tagnumber
	WHERE job_queue.tagnumber = $1;`

	var activeJobs ActiveJobs
	row := repo.DB.QueryRowContext(ctx, sqlQuery, ToNullInt64(tag))
	if err := row.Scan(
		&activeJobs.Tagnumber,
		&activeJobs.QueuedJob,
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

func (repo *SelectRepo) GetAvailableJobs(ctx context.Context, tag *int64) (*AvailableJobs, error) {
	if tag == nil {
		return nil, fmt.Errorf("tagnumber is nil")
	}

	if ctx.Err() != nil {
		return nil, fmt.Errorf("context error: %w", ctx.Err())
	}

	const sqlQuery = `SELECT 
	job_queue.tagnumber,
	(CASE 
		WHEN (job_queue.job_queued IS NULL) THEN TRUE
		ELSE FALSE
	END) AS job_available
	FROM job_queue
	WHERE job_queue.tagnumber = $1`

	var availableJobs AvailableJobs
	row := repo.DB.QueryRowContext(ctx, sqlQuery, ToNullInt64(tag))
	if err := row.Scan(
		&availableJobs.Tagnumber,
		&availableJobs.JobAvailable,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("error during row scan: %w", err)
	}
	return &availableJobs, nil
}

func (repo *SelectRepo) GetJobQueueOverview(ctx context.Context) (*JobQueueOverview, error) {
	if ctx.Err() != nil {
		return nil, fmt.Errorf("context error: %w", ctx.Err())
	}

	const sqlQuery = `SELECT t1.total_queued_jobs, t2.total_active_jobs, t3.total_active_blocking_jobs
	FROM 
	(SELECT COUNT(*) AS total_queued_jobs FROM job_queue WHERE job_queued IS NOT NULL AND (NOW() - present < INTERVAL '30 SECOND')) AS t1,
	(SELECT COUNT(*) AS total_active_jobs FROM job_queue WHERE job_active IS NOT NULL AND job_active = TRUE AND (NOW() - present < INTERVAL '30 SECOND')) AS t2,
	(SELECT COUNT(*) AS total_active_blocking_jobs FROM job_queue WHERE job_active IS NOT NULL AND job_active = TRUE AND job_queued IS NOT NULL AND job_queued IN ('hpEraseAndClone', 'hpCloneOnly', 'generic-erase+clone', 'generic-clone')) AS t3;`

	var jobQueueOverview JobQueueOverview
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

func (repo *SelectRepo) GetNotes(ctx context.Context, noteType *string) (*NotesTable, error) {
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

	var notesTable NotesTable
	row := repo.DB.QueryRowContext(ctx, sqlQuery, ToNullString(noteType))
	if err := row.Scan(
		&notesTable.Time,
		&notesTable.NoteType,
		&notesTable.Note,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("error during row scan: %w", err)
	}
	return &notesTable, nil
}

func (repo *SelectRepo) GetDashboardInventorySummary(ctx context.Context) ([]DashboardInventorySummary, error) {
	if ctx.Err() != nil {
		return nil, fmt.Errorf("context error: %w", ctx.Err())
	}

	const sqlQuery = `WITH latest_locations AS (
		SELECT DISTINCT ON (locations.tagnumber) locations.tagnumber, locations.department_name
		FROM locations
		ORDER BY locations.tagnumber, locations.time DESC
	),
	latest_checkouts AS (
		SELECT DISTINCT ON (checkout_log.tagnumber) checkout_log.tagnumber, checkout_log.checkout_date, checkout_log.return_date
		FROM checkout_log
		ORDER BY checkout_log.tagnumber, checkout_log.log_entry_time DESC
	),
	systems AS (
		SELECT hardware_data.tagnumber, hardware_data.system_model
		FROM hardware_data
		WHERE hardware_data.system_model IS NOT NULL
	),
	joined AS (
		SELECT systems.system_model,
			(latest_checkouts.checkout_date IS NOT NULL AND latest_checkouts.return_date IS NULL)
				OR (latest_checkouts.return_date IS NOT NULL AND latest_checkouts.return_date > NOW()) AS is_checked_out,
			(latest_locations.department_name IS NOT NULL AND latest_locations.department_name NOT IN ('property', 'pre-property')) AS loc_ok
		FROM systems
		LEFT JOIN latest_checkouts ON latest_checkouts.tagnumber = systems.tagnumber
		LEFT JOIN latest_locations ON latest_locations.tagnumber = systems.tagnumber
	)
	SELECT system_model,
		COUNT(*) AS system_model_count,
		COUNT(*) FILTER (WHERE is_checked_out) AS total_checked_out,
		COUNT(*) FILTER (WHERE NOT is_checked_out AND loc_ok) AS available_for_checkout
	FROM joined
	GROUP BY system_model
	ORDER BY system_model_count DESC;`

	rows, err := repo.DB.QueryContext(ctx, sqlQuery)
	if err != nil {
		return nil, fmt.Errorf("query error: %w", err)
	}
	defer rows.Close()

	var dashboardInventorySummary []DashboardInventorySummary
	for rows.Next() {
		if ctx.Err() != nil {
			return nil, fmt.Errorf("context error: %w", ctx.Err())
		}
		var summary DashboardInventorySummary
		if err := rows.Scan(
			&summary.SystemModel,
			&summary.SystemModelCount,
			&summary.TotalCheckedOut,
			&summary.AvailableForCheckout,
		); err != nil {
			return nil, fmt.Errorf("error during row scan: %w", err)
		}
		dashboardInventorySummary = append(dashboardInventorySummary, summary)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error during row iteration: %w", err)
	}
	if len(dashboardInventorySummary) == 0 {
		return nil, nil
	}

	return dashboardInventorySummary, nil
}

func (repo *SelectRepo) GetLocationFormData(ctx context.Context, tag *int64, serial *string) (*InventoryUpdateForm, error) {
	if tag == nil && (serial == nil || strings.TrimSpace(*serial) == "") {
		return nil, fmt.Errorf("either tag or serial must be provided")
	}

	if ctx.Err() != nil {
		return nil, fmt.Errorf("context error: %w", ctx.Err())
	}

	const sqlQuery = `SELECT 
		locations.time, 
		locations.tagnumber, 
		locations.system_serial, 
		locations.location, 
		locations.building, 
		locations.room, 
		hardware_data.system_manufacturer, 
		hardware_data.system_model,
		hardware_data.device_type,
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
		locations.note
	FROM locations
	LEFT JOIN hardware_data ON locations.tagnumber = hardware_data.tagnumber
	LEFT JOIN client_health ON locations.tagnumber = client_health.tagnumber
	LEFT JOIN checkout_log ON locations.tagnumber = checkout_log.tagnumber AND checkout_log.log_entry_time IN (SELECT MAX(log_entry_time) FROM checkout_log WHERE log_entry_time IS NOT NULL GROUP BY tagnumber)
	LEFT JOIN static_department_info ON locations.department_name = static_department_info.department_name
	WHERE (locations.tagnumber = $1 OR locations.system_serial = $2)
	ORDER BY locations.time DESC NULLS LAST
	LIMIT 1;`
	row := repo.DB.QueryRowContext(ctx, sqlQuery,
		ToNullInt64(tag),
		ToNullString(serial),
	)

	inventoryUpdateForm := &InventoryUpdateForm{}
	if err := row.Scan(
		&inventoryUpdateForm.Time,
		&inventoryUpdateForm.Tagnumber,
		&inventoryUpdateForm.SystemSerial,
		&inventoryUpdateForm.Location,
		&inventoryUpdateForm.Building,
		&inventoryUpdateForm.Room,
		&inventoryUpdateForm.SystemManufacturer,
		&inventoryUpdateForm.SystemModel,
		&inventoryUpdateForm.DeviceType,
		&inventoryUpdateForm.Department,
		&inventoryUpdateForm.Domain,
		&inventoryUpdateForm.PropertyCustodian,
		&inventoryUpdateForm.AcquiredDate,
		&inventoryUpdateForm.RetiredDate,
		&inventoryUpdateForm.Broken,
		&inventoryUpdateForm.DiskRemoved,
		&inventoryUpdateForm.LastHardwareCheck,
		&inventoryUpdateForm.ClientStatus,
		&inventoryUpdateForm.CheckoutDate,
		&inventoryUpdateForm.ReturnDate,
		&inventoryUpdateForm.Note,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("error during row scan: %w", err)
	}
	return inventoryUpdateForm, nil
}

func (repo *SelectRepo) GetClientImageFilePathFromUUID(ctx context.Context, uuid *string) (*ImageManifest, error) {
	if uuid == nil || strings.TrimSpace(*uuid) == "" {
		return nil, fmt.Errorf("uuid cannot be nil or empty")
	}

	if ctx.Err() != nil {
		return nil, fmt.Errorf("context error: %w", ctx.Err())
	}

	const sqlQuery = `SELECT tagnumber, filename, 
			filepath, thumbnail_filepath, hidden
		FROM client_images 
		WHERE uuid = $1;`
	row := repo.DB.QueryRowContext(ctx, sqlQuery, ToNullString(uuid))
	var imageManifest ImageManifest
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
		return nil, fmt.Errorf("error during row scan: %w", err)
	}
	return &imageManifest, nil
}

func (repo *SelectRepo) GetClientImageManifestByTag(ctx context.Context, tagnumber *int64) ([]ImageManifest, error) {
	if tagnumber == nil {
		return nil, fmt.Errorf("tagnumber is nil")
	}

	const sqlQuery = `SELECT time, tagnumber, uuid, filename, filepath, thumbnail_filepath, hidden, primary_image, note FROM client_images WHERE tagnumber = $1;`

	imageManifests := make([]ImageManifest, 0, 10)
	rows, err := repo.DB.QueryContext(ctx, sqlQuery, tagnumber)
	if err != nil {
		return nil, fmt.Errorf("error during query execution: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		if ctx.Err() != nil {
			return nil, fmt.Errorf("context error: %w", ctx.Err())
		}
		var imageManifest ImageManifest
		if err := rows.Scan(
			&imageManifest.Time,
			&imageManifest.Tagnumber,
			&imageManifest.UUID,
			&imageManifest.FileName,
			&imageManifest.FilePath,
			&imageManifest.ThumbnailFilePath,
			&imageManifest.Hidden,
			&imageManifest.PrimaryImage,
			&imageManifest.Note,
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

func (repo *SelectRepo) GetInventoryTableData(ctx context.Context, filterOptions *InventoryAdvSearchOptions) ([]InventoryTableData, error) {
	if filterOptions == nil {
		return nil, fmt.Errorf("filterOptions cannot be nil")
	}

	const sqlQuery = `SELECT locations.tagnumber, locations.system_serial, locations.location, 
		locationFormatting(locations.location) AS location_formatted,
		hardware_data.system_manufacturer, hardware_data.system_model, locations.department_name, static_department_info.department_name_formatted,
		locations.ad_domain, static_ad_domains.domain_name_formatted, client_health.os_installed, client_health.os_name, static_client_statuses.status_formatted,
		locations.is_broken, locations.note, locations.time AS last_updated
		FROM locations
		LEFT JOIN hardware_data ON locations.tagnumber = hardware_data.tagnumber
		LEFT JOIN client_health ON locations.tagnumber = client_health.tagnumber
		LEFT JOIN static_department_info ON locations.department_name = static_department_info.department_name
		LEFT JOIN static_ad_domains ON locations.ad_domain = static_ad_domains.domain_name
		LEFT JOIN static_client_statuses ON locations.client_status = static_client_statuses.status
		WHERE locations.time IN (SELECT MAX(time) FROM locations GROUP BY tagnumber)
		AND ($1::bigint IS NULL OR locations.tagnumber = $1)
		AND ($2::text IS NULL OR locations.system_serial = $2)
		AND ($3::text IS NULL OR locations.location = $3)
		AND ($4::text IS NULL OR hardware_data.system_manufacturer = $4)
		AND ($5::text IS NULL OR hardware_data.system_model = $5)
		AND ($6::text IS NULL OR locations.department_name = $6)
		AND ($7::text IS NULL OR locations.ad_domain = $7)
		AND ($8::text IS NULL OR locations.client_status = $8)
		AND ($9::boolean IS NULL OR locations.is_broken = $9)
		AND (
			$10::boolean IS NULL OR 
			(
				($10 = TRUE AND EXISTS (SELECT 1 FROM client_images WHERE client_images.tagnumber = locations.tagnumber))
				OR ($10 = FALSE AND NOT EXISTS (SELECT 1 FROM client_images WHERE client_images.tagnumber = locations.tagnumber)))
			)
		ORDER BY locations.time DESC;`

	rows, err := repo.DB.QueryContext(ctx, sqlQuery,
		ToNullInt64(filterOptions.Tagnumber),
		ToNullString(filterOptions.SystemSerial),
		ToNullString(filterOptions.Location),
		ToNullString(filterOptions.SystemManufacturer),
		ToNullString(filterOptions.SystemModel),
		ToNullString(filterOptions.Department),
		ToNullString(filterOptions.Domain),
		ToNullString(filterOptions.Status),
		ToNullBool(filterOptions.Broken),
		ToNullBool(filterOptions.HasImages),
	)
	if err != nil {
		return nil, fmt.Errorf("query error: %w", err)
	}
	defer rows.Close()

	var results []InventoryTableData
	for rows.Next() {
		if err := ctx.Err(); err != nil {
			return nil, fmt.Errorf("context error: %w", err)
		}
		row := InventoryTableData{}
		if err := rows.Scan(
			&row.Tagnumber,
			&row.SystemSerial,
			&row.Location,
			&row.LocationFormatted,
			&row.SystemManufacturer,
			&row.SystemModel,
			&row.Department,
			&row.DepartmentFormatted,
			&row.Domain,
			&row.DomainFormatted,
			&row.OsInstalled,
			&row.OsName,
			&row.Status,
			&row.Broken,
			&row.Note,
			&row.LastUpdated,
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
	return results, nil
}

func (repo *SelectRepo) GetClientBatteryHealth(ctx context.Context, tagnumber *int64) (*ClientBatteryHealth, error) {
	if tagnumber == nil {
		return nil, fmt.Errorf("tagnumber cannot be nil (GetClientBatteryHealth)")
	}

	if ctx.Err() != nil {
		return nil, fmt.Errorf("context error: %w", ctx.Err())
	}

	const sqlQuery = `SELECT jobstats.time, jobstats.tagnumber, jobstats.battery_health, client_health.battery_health, 
	jobstats.battery_charge_cycles
	FROM jobstats 
	LEFT JOIN client_health ON jobstats.tagnumber = client_health.tagnumber
	WHERE jobstats.tagnumber = $1 
	ORDER BY jobstats.time DESC NULLS LAST LIMIT 1;`

	var batteryHealth ClientBatteryHealth
	row := repo.DB.QueryRowContext(ctx, sqlQuery, ToNullInt64(tagnumber))
	if err := row.Scan(
		&batteryHealth.Time,
		&batteryHealth.Tagnumber,
		&batteryHealth.JobstatsBattery,
		&batteryHealth.ClientHealthBattery,
		&batteryHealth.BatteryChargeCycles,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("query error: %w", err)
	}

	return &batteryHealth, nil
}

func (repo *SelectRepo) GetJobQueueTable(ctx context.Context) ([]JobQueueTableRow, error) {
	const sqlQuery = `WITH latest_locations AS (
		SELECT DISTINCT ON (locations.tagnumber) locations.time, locations.tagnumber, locations.system_serial, locations.location,
			locationFormatting(locations.location) AS location_formatted, locations.department_name, locations.ad_domain,
			locations.client_status, locations.is_broken,
			locations.disk_removed
		FROM locations
		ORDER BY locations.tagnumber, locations.time DESC),
	latest_jobstats AS (
		SELECT DISTINCT ON (jobstats.tagnumber) jobstats.time, jobstats.tagnumber, 
			jobstats.disk_type, jobstats.disk_size AS "disk_capacity",
			jobstats.battery_health, jobstats.bios_version, jobstats.ram_capacity, jobstats.disk_model, jobstats.disk_temp
		FROM jobstats
		ORDER BY jobstats.tagnumber, jobstats.time DESC),
	latest_job AS (
		SELECT DISTINCT ON (jobstats.tagnumber) jobstats.time, jobstats.tagnumber,
			jobstats.erase_completed, jobstats.erase_mode, jobstats.erase_time, 
			jobstats.clone_completed, jobstats.clone_image, jobstats.clone_master, jobstats.clone_time, 
			jobstats.job_failed
		FROM jobstats
		WHERE jobstats.erase_completed = TRUE OR jobstats.clone_completed = TRUE
		ORDER BY jobstats.tagnumber, jobstats.time DESC NULLS LAST)
	SELECT
		latest_locations.tagnumber,
		latest_locations.system_serial,
		hardware_data.system_manufacturer,
		hardware_data.system_model,
		latest_locations.location_formatted AS "location",
		latest_locations.department_name,
		latest_locations.client_status,
		latest_locations.is_broken,
		latest_locations.disk_removed,
		FALSE AS "temp_warning",
		FALSE AS "battery_warning",
		(CASE WHEN latest_locations.client_status = 'checked_out' THEN TRUE ELSE FALSE END) AS checkout_bool,
		TRUE AS "kernel_updated",
		job_queue.present AS "last_heard",
		job_queue.uptime,
		(CASE WHEN (CURRENT_TIMESTAMP - job_queue.present) < INTERVAL '30 SECONDS' THEN TRUE ELSE FALSE END) AS "online",
		job_queue.job_active,
		(CASE WHEN job_queue.job_queued IS NOT NULL THEN TRUE ELSE FALSE END) AS "job_queued",
		job_queue.job_queued_position AS "queue_position",
		job_queue.job_queued AS "job_name",
		static_job_names.job_name_readable,
		(CASE
			WHEN job_queue.job_active = TRUE AND job_queue.job_queued IS NOT NULL THEN job_queue.clone_mode
			ELSE 'N/A'
		END) AS "job_clone_mode",
		(CASE
			WHEN job_queue.job_active = TRUE AND job_queue.job_queued IS NOT NULL THEN job_queue.erase_mode
			ELSE 'N/A'
		END) AS "job_erase_mode",
		(CASE 
			WHEN job_queue.job_active = TRUE THEN 'In Progress' || job_queue.status
			WHEN job_queue.job_queued IS NOT NULL AND job_queue.job_active = FALSE THEN 'Queued' || job_queue.status
			ELSE 'Idle'
		END) AS "job_status",
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
		NULL AS "os_updated",
		(CASE 
			WHEN latest_locations.ad_domain IS NOT NULL OR NOT latest_locations.ad_domain = 'none' THEN TRUE
			ELSE FALSE
		END) AS "domain_joined",
		static_ad_domains.domain_name,
		(CASE 
			WHEN latest_jobstats.bios_version = static_bios_stats.bios_version THEN TRUE
			ELSE FALSE
		END) AS "bios_updated",
		static_bios_stats.bios_version,
		'0' AS "cpu_usage",
		'0' AS "cpu_temp",
		FALSE AS "cpu_temp_warning",
		'0' AS ram_usage,
		latest_jobstats.ram_capacity,
		'0' AS "disk_usage",
		latest_jobstats.disk_temp,
		static_disk_stats.disk_type,
		latest_jobstats.disk_capacity AS "disk_size",
		'80' AS "max_disk_temp",
		FALSE AS "disk_temp_warning",
		'UP' AS "network_link_status",
		job_queue.network_speed AS "network_link_speed",
		'0' AS "network_usage",
		job_queue.battery_charge,
		job_queue.battery_status,
		latest_jobstats.battery_health,
		NULL AS "plugged_in",
		job_queue.watts_now AS "power_usage"
	FROM locations
	LEFT JOIN job_queue ON locations.tagnumber = job_queue.tagnumber
	LEFT JOIN hardware_data ON locations.tagnumber = hardware_data.tagnumber
	LEFT JOIN latest_locations ON locations.tagnumber = latest_locations.tagnumber
	LEFT JOIN latest_jobstats ON locations.tagnumber = latest_jobstats.tagnumber
	LEFT JOIN latest_job ON locations.tagnumber = latest_job.tagnumber
	LEFT JOIN static_image_names ON latest_job.clone_image = static_image_names.image_name
	LEFT JOIN static_job_names ON job_queue.job_queued = static_job_names.job_name
	LEFT JOIN static_bios_stats ON hardware_data.system_model = static_bios_stats.system_model
	LEFT JOIN static_disk_stats ON latest_jobstats.disk_model = static_disk_stats.disk_model
	LEFT JOIN static_ad_domains ON latest_locations.ad_domain = static_ad_domains.domain_name
	WHERE locations.time IN (SELECT MAX(time) FROM locations GROUP BY tagnumber)
	ORDER BY locations.tagnumber;`

	jobQueueRows := make([]JobQueueTableRow, 0, 560) // 560 is the approximate # of clients

	rows, err := repo.DB.QueryContext(ctx, sqlQuery)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		if ctx.Err() != nil {
			return nil, errors.New("context error: " + ctx.Err().Error())
		}
		var row JobQueueTableRow
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
			&row.BatteryHealthWarning,
			&row.CheckoutBool,
			&row.KernelUpdated,
			&row.LastHeard,
			&row.Uptime,
			&row.Online,
			&row.JobActive,
			&row.JobQueued,
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
			&row.BIOSUpdated,
			&row.BIOSVersion,
			&row.CPUUsage,
			&row.CPUTemp,
			&row.CPUTempWarning,
			&row.RAMUsage,
			&row.RAMCapacity,
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
			&row.BatteryHealth,
			&row.PluggedIn,
			&row.PowerUsage,
		); err != nil {
			return nil, err
		}
		jobQueueRows = append(jobQueueRows, row)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error during row iteration: %w", err)
	}
	if len(jobQueueRows) == 0 {
		return nil, nil
	}
	return jobQueueRows, nil
}

func (repo *SelectRepo) GetBatteryStandardDeviation(ctx context.Context) ([]ClientReport, error) {
	const sqlQuery = `SELECT jobstats.time AS "battery_health_timestamp", jobstats.tagnumber, jobstats.battery_health AS "battery_health_pcnt", 
		ROUND(jobstats.battery_health - AVG(jobstats.battery_health) OVER (), 2) AS "battery_health_stddev"
	FROM locations
	LEFT JOIN jobstats ON locations.tagnumber = jobstats.tagnumber AND locations.time IN (SELECT MAX(time) FROM locations GROUP BY tagnumber)
	WHERE locations.department_name NOT IN ('property')
		AND jobstats.battery_health IS NOT NULL
		AND jobstats.time IN (SELECT MAX(time) FROM jobstats GROUP BY tagnumber)
	GROUP BY jobstats.tagnumber, jobstats.time, jobstats.battery_health
	ORDER BY jobstats.time DESC;`
	var clientReports []ClientReport
	rows, err := repo.DB.QueryContext(ctx, sqlQuery)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		if ctx.Err() != nil {
			return nil, errors.New("context error: " + ctx.Err().Error())
		}
		var clientReport ClientReport
		if err := rows.Scan(
			&clientReport.BatteryHealthTimestamp,
			&clientReport.Tagnumber,
			&clientReport.BatteryHealthPcnt,
			&clientReport.BatteryHealthStdDev,
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

func (repo *SelectRepo) GetAllJobs(ctx context.Context) ([]AllJobs, error) {
	const sqlQuery = `SELECT static_job_names.job_name, 
		static_job_names.job_name_readable, 
		static_job_names.job_sort_order, 
		static_job_names.job_hidden
	FROM static_job_names
	ORDER BY static_job_names.job_sort_order ASC;`

	var allJobs []AllJobs
	rows, err := repo.DB.QueryContext(ctx, sqlQuery)
	if err != nil {
		return nil, fmt.Errorf("error during query execution: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		if ctx.Err() != nil {
			return nil, errors.New("context error: " + ctx.Err().Error())
		}
		var job AllJobs
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

func (repo *SelectRepo) GetAllLocations(ctx context.Context) ([]AllLocations, error) {
	const sqlQuery = `WITH latest_locations AS (
		SELECT DISTINCT ON (tagnumber) location, time, COUNT(location) OVER (PARTITION BY location) AS location_count
		FROM locations
		WHERE location IS NOT NULL 
			AND location != ''
			AND time IN (SELECT MAX(time) FROM locations GROUP BY tagnumber)
		ORDER BY tagnumber, time DESC
	)
	SELECT location, MAX(time) as time, MAX(location_count) as location_count
	FROM latest_locations
	GROUP BY location
	ORDER BY location ASC;`

	var allLocations []AllLocations
	rows, err := repo.DB.QueryContext(ctx, sqlQuery)
	if err != nil {
		return nil, fmt.Errorf("error during query execution: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		if ctx.Err() != nil {
			return nil, fmt.Errorf("context error: %w", ctx.Err())
		}
		var location AllLocations
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

func (repo *SelectRepo) GetAllStatuses(ctx context.Context) ([]ClientStatus, error) {
	const sqlQuery = `SELECT status, status_formatted, sort_order FROM static_client_statuses ORDER BY sort_order;`

	rows, err := repo.DB.QueryContext(ctx, sqlQuery)
	if err != nil {
		return nil, fmt.Errorf("error during query execution: %w", err)
	}
	defer rows.Close()

	var allStatuses []ClientStatus
	for rows.Next() {
		if ctx.Err() != nil {
			return nil, fmt.Errorf("context error: %w", ctx.Err())
		}
		var status ClientStatus
		if err := rows.Scan(
			&status.Status,
			&status.StatusFormatted,
			&status.SortOrder,
		); err != nil {
			return nil, fmt.Errorf("error during row scan: %w", err)
		}
		allStatuses = append(allStatuses, status)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error during row iteration: %w", err)
	}
	if len(allStatuses) == 0 {
		return nil, nil
	}
	return allStatuses, nil
}

func (repo *SelectRepo) GetAllDeviceTypes(ctx context.Context) ([]DeviceType, error) {
	const sqlQuery = `SELECT device_type, device_type_formatted, device_meta_category, sort_order FROM static_device_types ORDER BY sort_order;`
	rows, err := repo.DB.QueryContext(ctx, sqlQuery)
	if err != nil {
		return nil, fmt.Errorf("error during query execution: %w", err)
	}
	defer rows.Close()

	var allDeviceTypes []DeviceType
	for rows.Next() {
		if ctx.Err() != nil {
			return nil, fmt.Errorf("context error: %w", ctx.Err())
		}
		var deviceType DeviceType
		if err := rows.Scan(
			&deviceType.DeviceType,
			&deviceType.DeviceTypeFormatted,
			&deviceType.DeviceMetaCategory,
			&deviceType.SortOrder,
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
