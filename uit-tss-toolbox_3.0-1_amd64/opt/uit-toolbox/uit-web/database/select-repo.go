package database

import (
	"context"
	"database/sql"
	"errors"
	"time"
)

type Repo struct {
	DB *sql.DB
}

func NewRepo(db *sql.DB) *Repo { return &Repo{DB: db} }

func (repo *Repo) GetAllTags(ctx context.Context) ([]int, error) {
	sqlCode := `SELECT tagnumber FROM (SELECT tagnumber, time, ROW_NUMBER() OVER (PARTITION BY tagnumber ORDER BY time DESC) AS
		row_nums FROM locations WHERE tagnumber IS NOT NULL) t1 WHERE t1.row_nums = 1 ORDER BY t1.time DESC;`

	rows, err := repo.DB.QueryContext(ctx, sqlCode)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var allTags []AllTags
	for rows.Next() {
		var tag AllTags
		if err := rows.Scan(&tag.Tagnumber); err != nil {
			return nil, err
		}
		allTags = append(allTags, tag)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	allTagsSlice := make([]int, len(allTags))
	for i := range allTags {
		allTagsSlice[i] = allTags[i].Tagnumber
	}

	return allTagsSlice, nil
}

func (repo *Repo) GetDepartments(ctx context.Context) (map[string]string, error) {
	rows, err := repo.DB.QueryContext(ctx, "SELECT department_name, department_name_formatted FROM static_department_info ORDER BY department_name_formatted;")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var departmentMap = make(map[string]string)
	for rows.Next() {
		var department, departmentFormatted string
		if err := rows.Scan(&department, &departmentFormatted); err != nil {
			return nil, err
		}
		departmentMap[department] = departmentFormatted
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return departmentMap, nil
}

type Domains struct {
	DomainName          string `json:"domain_name"`
	DomainNameFormatted string `json:"domain_name_formatted"`
	DomainSortOrder     int64  `json:"domain_sort_order"`
}

func (repo *Repo) GetDomains(ctx context.Context) ([]Domains, error) {
	rows, err := repo.DB.QueryContext(ctx, "SELECT domain_name, domain_name_formatted FROM static_ad_domains ORDER BY domain_sort_order DESC;")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var domains []Domains
	for rows.Next() {
		var domain Domains
		if err := rows.Scan(&domain.DomainName, &domain.DomainNameFormatted); err != nil {
			return nil, err
		}
		domains = append(domains, domain)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return domains, nil
}

func (repo *Repo) GetStatuses(ctx context.Context) (map[string]string, error) {
	rows, err := repo.DB.QueryContext(ctx, "SELECT status, status_formatted FROM static_client_statuses ORDER BY sort_order;")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var statusMap = make(map[string]string)
	for rows.Next() {
		var status, statusFormatted string
		if err := rows.Scan(&status, &statusFormatted); err != nil {
			return nil, err
		}
		statusMap[status] = statusFormatted
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return statusMap, nil
}

func (repo *Repo) GetLocations(ctx context.Context) (map[string]string, error) {
	sqlCode := `SELECT location, locationFormatting(location) AS location_formatted FROM locations WHERE time IN (SELECT MAX(time) FROM locations GROUP BY tagnumber) AND location IS NOT NULL GROUP BY location ORDER BY location ASC;`
	rows, err := repo.DB.QueryContext(ctx, sqlCode)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var locationMap = make(map[string]string)

	for rows.Next() {
		var location, locationFormatted string
		if err := rows.Scan(&location, &locationFormatted); err != nil {
			return nil, err
		}
		locationMap[location] = locationFormatted
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return locationMap, nil
}

func (repo *Repo) GetManufacturersAndModels(ctx context.Context) ([]ManufacturersAndModels, error) {
	sqlCode := `SELECT system_model, 
		(CASE WHEN LENGTH(system_model) > 17 THEN CONCAT(LEFT(system_model, 8), '...', RIGHT(system_model, 9)) ELSE system_model END) AS system_model_formatted,
		system_manufacturer, 
		(CASE WHEN LENGTH(system_manufacturer) > 10 THEN CONCAT(LEFT(system_manufacturer, 10), '...') ELSE system_manufacturer END) AS system_manufacturer_formatted
		FROM system_data 
		WHERE system_manufacturer IS NOT NULL 
			AND system_model IS NOT NULL
		GROUP BY system_manufacturer, system_model 
		ORDER BY system_manufacturer ASC, system_model ASC;`

	var manufacturersAndModels []ManufacturersAndModels
	rows, err := repo.DB.QueryContext(ctx, sqlCode)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var mam ManufacturersAndModels
		if err := rows.Scan(&mam.SystemModel, &mam.SystemModelFormatted, &mam.SystemManufacturer, &mam.SystemManufacturerFormatted); err != nil {
			return nil, err
		}
		manufacturersAndModels = append(manufacturersAndModels, mam)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return manufacturersAndModels, nil
}

func (repo *Repo) ClientLookupByTag(ctx context.Context, tag int64) (*ClientLookup, error) {
	var clientLookup ClientLookup
	row := repo.DB.QueryRowContext(ctx, "SELECT tagnumber, system_serial FROM locations WHERE tagnumber = $1 ORDER BY time DESC LIMIT 1;", tag)
	if err := row.Scan(&clientLookup.Tagnumber, &clientLookup.SystemSerial); err != nil {
		return nil, err
	}
	return &clientLookup, nil
}

func (repo *Repo) ClientLookupBySerial(ctx context.Context, serial string) (*ClientLookup, error) {
	var clientLookup ClientLookup
	row := repo.DB.QueryRowContext(ctx, "SELECT tagnumber, system_serial FROM locations WHERE system_serial = $1 ORDER BY time DESC LIMIT 1;", serial)
	if err := row.Scan(&clientLookup.Tagnumber, &clientLookup.SystemSerial); err != nil {
		return nil, err
	}
	return &clientLookup, nil
}

func (repo *Repo) GetHardwareIdentifiers(ctx context.Context, tag int64) (*HardwareData, error) {
	sqlQuery := `SELECT locations.tagnumber, locations.system_serial, jobstats.etheraddress, system_data.wifi_mac,
	system_data.system_model, system_data.system_uuid, system_data.system_sku, system_data.chassis_type, 
	system_data.motherboard_manufacturer, system_data.motherboard_serial, system_data.system_manufacturer
	FROM locations
	LEFT JOIN jobstats ON locations.tagnumber = jobstats.tagnumber AND jobstats.time IN (SELECT MAX(time) FROM jobstats GROUP BY tagnumber)
	LEFT JOIN system_data ON locations.tagnumber = system_data.tagnumber
	WHERE locations.time IN (SELECT MAX(time) FROM locations GROUP BY tagnumber)
	AND locations.tagnumber = $1;`

	var hardwareData HardwareData
	row := repo.DB.QueryRowContext(ctx, sqlQuery, tag)
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
		return nil, err
	}
	return &hardwareData, nil
}

func (repo *Repo) GetBiosData(ctx context.Context, tag int64) (*BiosData, error) {
	sqlQuery := `SELECT client_health.tagnumber, client_health.bios_version, client_health.bios_updated, 
	client_health.tpm_version 
	FROM client_health WHERE client_health.tagnumber = $1;`

	var biosData BiosData
	row := repo.DB.QueryRowContext(ctx, sqlQuery, tag)
	if err := row.Scan(
		&biosData.Tagnumber,
		&biosData.BiosVersion,
		&biosData.BiosUpdated,
		&biosData.BiosDate,
		&biosData.TpmVersion,
	); err != nil {
		return nil, err
	}
	return &biosData, nil
}

func (repo *Repo) GetOsData(ctx context.Context, tag int64) (*OsData, error) {
	sqlQuery := `SELECT locations.tagnumber, client_health.os_installed, client_health.os_name,
	client_health.last_imaged_time AT TIME ZONE 'America/Chicago', client_health.tpm_version, jobstats.boot_time
	FROM locations
	LEFT JOIN client_health ON locations.tagnumber = client_health.tagnumber
	LEFT JOIN jobstats ON locations.tagnumber = jobstats.tagnumber AND jobstats.time IN (SELECT MAX(time) FROM jobstats GROUP BY tagnumber)
	WHERE locations.time IN (SELECT MAX(time) FROM locations GROUP BY tagnumber)
	AND locations.tagnumber = $1;`

	var osData OsData
	row := repo.DB.QueryRowContext(ctx, sqlQuery, tag)
	if err := row.Scan(
		&osData.Tagnumber,
		&osData.OsInstalled,
		&osData.OsName,
		&osData.OsInstalledTime,
		&osData.TPMversion,
		&osData.BootTime,
	); err != nil {
		return nil, err
	}
	return &osData, nil
}

func (repo *Repo) GetActiveJobs(ctx context.Context, tag int64) (*ActiveJobs, error) {
	sqlQuery := `SELECT remote.tagnumber, remote.job_queued, remote.job_active, t1.queue_position
	FROM remote
	LEFT JOIN (SELECT tagnumber, ROW_NUMBER() OVER (PARTITION BY tagnumber ORDER BY present DESC) AS queue_position FROM remote) AS t1 
		ON remote.tagnumber = t1.tagnumber
	WHERE remote.tagnumber = $1;`

	var activeJobs ActiveJobs
	row := repo.DB.QueryRowContext(ctx, sqlQuery, tag)
	if err := row.Scan(
		&activeJobs.Tagnumber,
		&activeJobs.QueuedJob,
		&activeJobs.JobActive,
		&activeJobs.QueuePosition,
	); err != nil {
		return nil, err
	}
	return &activeJobs, nil
}

func (repo *Repo) GetAvailableJobs(ctx context.Context, tag int64) (*AvailableJobs, error) {
	sqlQuery := `SELECT 
	remote.tagnumber,
	(CASE 
		WHEN (remote.job_queued IS NULL) THEN TRUE
		ELSE FALSE
	END) AS job_available
	FROM remote
	WHERE remote.tagnumber = $1`

	var availableJobs AvailableJobs
	row := repo.DB.QueryRowContext(ctx, sqlQuery, tag)
	if err := row.Scan(
		&availableJobs.Tagnumber,
		&availableJobs.JobAvailable,
	); err != nil {
		return nil, err
	}
	return &availableJobs, nil
}

func (repo *Repo) GetJobQueueOverview(ctx context.Context) (*JobQueueOverview, error) {
	sqlQuery := `SELECT t1.total_queued_jobs, t2.total_active_jobs, t3.total_active_blocking_jobs
	FROM 
	(SELECT COUNT(*) AS total_queued_jobs FROM remote WHERE job_queued IS NOT NULL AND (NOW() - present < INTERVAL '30 SECOND')) AS t1,
	(SELECT COUNT(*) AS total_active_jobs FROM remote WHERE job_active IS NOT NULL AND job_active = TRUE AND (NOW() - present < INTERVAL '30 SECOND')) AS t2,
	(SELECT COUNT(*) AS total_active_blocking_jobs FROM remote WHERE job_active IS NOT NULL AND job_active = TRUE AND job_queued IS NOT NULL AND job_queued IN ('hpEraseAndClone', 'hpCloneOnly', 'generic-erase+clone', 'generic-clone')) AS t3;`

	var jobQueueOverview JobQueueOverview
	row := repo.DB.QueryRowContext(ctx, sqlQuery)
	if err := row.Scan(
		&jobQueueOverview.TotalQueuedJobs,
		&jobQueueOverview.TotalActiveJobs,
		&jobQueueOverview.TotalActiveBlockingJobs,
	); err != nil {
		return nil, err
	}
	return &jobQueueOverview, nil
}

func (repo *Repo) GetNotes(ctx context.Context, noteType string) (*NotesTable, error) {
	sqlQuery := `SELECT time AT TIME ZONE 'America/Chicago', note_type, note FROM notes WHERE note_type = $1 ORDER BY time DESC LIMIT 1;`

	var notesTable NotesTable
	row := repo.DB.QueryRowContext(ctx, sqlQuery, noteType)
	if err := row.Scan(
		&notesTable.Time,
		&notesTable.NoteType,
		&notesTable.Note,
	); err != nil {
		return nil, err
	}
	return &notesTable, nil
}

func (repo *Repo) GetDashboardInventorySummary(ctx context.Context) ([]DashboardInventorySummary, error) {
	sqlQuery := `WITH latest_locations AS (
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
		SELECT system_data.tagnumber, system_data.system_model
		FROM system_data
		WHERE system_data.system_model IS NOT NULL
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
		return nil, err
	}
	defer rows.Close()

	var dashboardInventorySummary []DashboardInventorySummary
	for rows.Next() {
		var summary DashboardInventorySummary
		if err := rows.Scan(
			&summary.SystemModel,
			&summary.SystemModelCount,
			&summary.TotalCheckedOut,
			&summary.AvailableForCheckout,
		); err != nil {
			return nil, err
		}
		dashboardInventorySummary = append(dashboardInventorySummary, summary)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return dashboardInventorySummary, nil
}

func (repo *Repo) GetLocationFormData(ctx context.Context, tag int64) (*InventoryFormAutofill, error) {
	sqlQuery := `SELECT locations.time, locations.tagnumber, locations.system_serial, locations.location, locations.building, locations.room, system_data.system_manufacturer, system_data.system_model,
	locations.department_name, locations.property_custodian, locations.ad_domain, locations.is_broken, locations.client_status, locations.disk_removed, locations.note, locations.acquired_date
	FROM locations
	LEFT JOIN system_data ON locations.tagnumber = system_data.tagnumber
	WHERE locations.time IN (SELECT MAX(time) FROM locations GROUP BY tagnumber)
	AND locations.tagnumber = $1
	ORDER BY locations.time DESC
	LIMIT 1;`
	row := repo.DB.QueryRowContext(ctx, sqlQuery, tag)

	inventoryUpdateForm := &InventoryFormAutofill{}
	if err := row.Scan(
		&inventoryUpdateForm.Time,
		&inventoryUpdateForm.Tagnumber,
		&inventoryUpdateForm.SystemSerial,
		&inventoryUpdateForm.Location,
		&inventoryUpdateForm.Building,
		&inventoryUpdateForm.Room,
		&inventoryUpdateForm.SystemManufacturer,
		&inventoryUpdateForm.SystemModel,
		&inventoryUpdateForm.Department,
		&inventoryUpdateForm.PropertyCustodian,
		&inventoryUpdateForm.Domain,
		&inventoryUpdateForm.Broken,
		&inventoryUpdateForm.Status,
		&inventoryUpdateForm.DiskRemoved,
		&inventoryUpdateForm.Note,
		&inventoryUpdateForm.AcquiredDate,
	); err != nil {
		return nil, err
	}

	return inventoryUpdateForm, nil
}

func (repo *Repo) GetClientImagePaths(ctx context.Context, tag int64) ([]string, error) {
	sqlQuery := `SELECT filepath FROM client_images WHERE tagnumber = $1 ORDER BY time DESC;`
	rows, err := repo.DB.QueryContext(ctx, sqlQuery, tag)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var filepaths []string
	for rows.Next() {
		var filepath string
		if err := rows.Scan(&filepath); err != nil {
			return nil, err
		}
		filepaths = append(filepaths, filepath)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return filepaths, nil
}

func (repo *Repo) GetClientImageUUIDs(ctx context.Context, tag int64) ([]string, error) {
	sqlQuery := `SELECT uuid FROM client_images WHERE tagnumber = $1 ORDER BY time DESC;`
	rows, err := repo.DB.QueryContext(ctx, sqlQuery, tag)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var uuids []string
	for rows.Next() {
		var uuid string
		if err := rows.Scan(&uuid); err != nil {
			return nil, err
		}
		uuids = append(uuids, uuid)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return uuids, nil
}

func (repo *Repo) GetClientImageByUUID(ctx context.Context, uuid string) (*ClientImagesTable, error) {
	sqlQuery := `SELECT uuid, time, tagnumber, filename, filepath, thumbnail_filepath, filesize, mime_type, exif_timestamp, resolution_x, resolution_y, note, hidden, primary_image
	FROM client_images WHERE uuid = $1;`
	row := repo.DB.QueryRowContext(ctx, sqlQuery, uuid)
	clientImage := &ClientImagesTable{}
	if err := row.Scan(
		&clientImage.UUID,
		&clientImage.Time,
		&clientImage.Tagnumber,
		&clientImage.Filename,
		&clientImage.FilePath,
		&clientImage.ThumbnailFilePath,
		&clientImage.Filesize,
		&clientImage.MimeType,
		&clientImage.ExifTimestamp,
		&clientImage.ResolutionX,
		&clientImage.ResolutionY,
		&clientImage.Note,
		&clientImage.Hidden,
		&clientImage.PrimaryImage,
	); err != nil {
		return nil, err
	}
	return clientImage, nil
}

func (repo *Repo) GetClientImageManifestByUUID(ctx context.Context, uuid string) (*time.Time, *int64, *string, *string, *bool, *bool, *string, error) {
	sqlQuery := `SELECT time, tagnumber, filepath, thumbnail_filepath, hidden, primary_image, note FROM client_images WHERE uuid = $1;`

	var (
		time              sql.NullTime
		tagnumber         sql.NullInt64
		filepath          sql.NullString
		thumbnailFilepath sql.NullString
		hidden            sql.NullBool
		primaryImage      sql.NullBool
		note              sql.NullString
	)
	row := repo.DB.QueryRowContext(ctx, sqlQuery, uuid)
	if err := row.Scan(
		&time,
		&tagnumber,
		&filepath,
		&thumbnailFilepath,
		&hidden,
		&primaryImage,
		&note,
	); err != nil {
		return nil, nil, nil, nil, nil, nil, nil, err
	}
	return ptrTime(time), ptrInt64(tagnumber), ptrString(filepath), ptrString(thumbnailFilepath), ptrBool(hidden), ptrBool(primaryImage), ptrString(note), nil
}

func (repo *Repo) GetInventoryTableData(ctx context.Context, filterOptions *InventoryFilterOptions) ([]*InventoryTableData, error) {
	sqlCode := `SELECT locations.tagnumber, locations.system_serial, locations.location, 
		locationFormatting(locations.location) AS location_formatted,
		system_data.system_manufacturer, system_data.system_model, locations.department_name, static_department_info.department_name_formatted,
		locations.ad_domain, static_ad_domains.domain_name_formatted, client_health.os_installed, client_health.os_name, static_client_statuses.status_formatted,
		locations.is_broken, locations.note, locations.time AS last_updated
		FROM locations
		LEFT JOIN system_data ON locations.tagnumber = system_data.tagnumber
		LEFT JOIN client_health ON locations.tagnumber = client_health.tagnumber
		LEFT JOIN static_department_info ON locations.department_name = static_department_info.department_name
		LEFT JOIN static_ad_domains ON locations.ad_domain = static_ad_domains.domain_name
		LEFT JOIN static_client_statuses ON locations.client_status = static_client_statuses.status
		WHERE locations.time IN (SELECT MAX(time) FROM locations GROUP BY tagnumber)
		AND ($1::bigint IS NULL OR locations.tagnumber = $1)
		AND ($2::text IS NULL OR locations.system_serial = $2)
		AND ($3::text IS NULL OR locations.location = $3)
		AND ($4::text IS NULL OR system_data.system_manufacturer = $4)
		AND ($5::text IS NULL OR system_data.system_model = $5)
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

	rows, err := repo.DB.QueryContext(ctx, sqlCode,
		toNullInt64(filterOptions.Tagnumber),
		toNullString(filterOptions.SystemSerial),
		toNullString(filterOptions.Location),
		toNullString(filterOptions.SystemManufacturer),
		toNullString(filterOptions.SystemModel),
		toNullString(filterOptions.Department),
		toNullString(filterOptions.Domain),
		toNullString(filterOptions.Status),
		toNullBool(filterOptions.Broken),
		toNullBool(filterOptions.HasImages),
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []*InventoryTableData
	for rows.Next() {
		row := &InventoryTableData{}
		if err = rows.Err(); err != nil {
			return nil, errors.New("Query error: " + err.Error())
		}
		if err = ctx.Err(); err != nil {
			return nil, errors.New("Context error: " + err.Error())
		}
		err = rows.Scan(
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
		)
		if err != nil {
			return nil, err
		}
		results = append(results, row)
	}
	return results, nil
}

type ClientBatteryHealth struct {
	Time                *time.Time `json:"time"`
	Tagnumber           *int64     `json:"tagnumber"`
	JobstatsBattery     *string    `json:"jobstatsHealthPcnt"`
	ClientHealthBattery *string    `json:"clientHealthPcnt"`
	BatteryChargeCycles *int64     `json:"chargeCycles"`
}

func (repo *Repo) GetClientBatteryHealth(ctx context.Context, tagnumber int64) (*ClientBatteryHealth, error) {
	sqlQuery := `SELECT jobstats.time, jobstats.tagnumber, jobstats.battery_health, client_health.battery_health, 
	jobstats.battery_charge_cycles
	FROM jobstats 
	LEFT JOIN client_health ON jobstats.tagnumber = client_health.tagnumber
	WHERE jobstats.tagnumber = $1 AND jobstats.time IN (SELECT MAX(time) FROM jobstats WHERE tagnumber = $1);`

	var batteryHealth ClientBatteryHealth
	row := repo.DB.QueryRowContext(ctx, sqlQuery, tagnumber)
	if err := row.Scan(
		&batteryHealth.Time,
		&batteryHealth.Tagnumber,
		&batteryHealth.JobstatsBattery,
		&batteryHealth.ClientHealthBattery,
		&batteryHealth.BatteryChargeCycles,
	); err != nil {
		return nil, err
	}
	return &batteryHealth, nil
}
