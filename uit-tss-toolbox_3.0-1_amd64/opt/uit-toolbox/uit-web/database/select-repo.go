package database

import (
	"context"
	"database/sql"
)

type Repo struct {
	DB *sql.DB
}

func NewRepo(db *sql.DB) *Repo { return &Repo{DB: db} }

func GetDepartments(ctx context.Context, db *sql.DB) ([]string, error) {
	rows, err := db.QueryContext(ctx, "SELECT department, department_readable FROM static_departments ORDER BY department_readable;")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var departments []string
	for rows.Next() {
		var department DepartmentList
		if err := rows.Scan(&department.Department, &department.DepartmentReadable); err != nil {
			return nil, err
		}
		departments = append(departments, department.Department, department.DepartmentReadable)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return departments, nil
}

func (repo *Repo) ClientLookupByTag(ctx context.Context, tag int) (*ClientLookup, error) {
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

func (repo *Repo) GetHardwareIdentifiers(ctx context.Context, tag int) (*HardwareData, error) {
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

func (repo *Repo) GetBiosData(ctx context.Context, tag int) (*BiosData, error) {
	sqlQuery := `SELECT client_health.tagnumber, client_health.bios_version, client_health.bios_updated, 
	client_health.bios_date, client_health.tpm_version 
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

func (repo *Repo) GetOsData(ctx context.Context, tag int) (*OsData, error) {
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

func (repo *Repo) GetActiveJobs(ctx context.Context, tag int) (*ActiveJobs, error) {
	sqlQuery := `SELECT remote.tagnumber, remote.job_queued, remote.job_active, t1.queue_position
	FROM remote
	LEFT JOIN (SELECT tagnumber, ROW_NUMBER() OVER (PARTITION BY tagnumber ORDER BY time DESC) AS queue_position FROM job_queue) AS t1 
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

func (repo *Repo) GetAvailableJobs(ctx context.Context, tag int) (*AvailableJobs, error) {
	sqlQuery := `SELECT 
	remote.tagnumber,
	(CASE 
		WHEN (remote.job_queued IS NULL) THEN TRUE
		ELSE FALSE
	END) AS job_available,
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
		SELECT DISTINCT ON (locations.tagnumber) locations.tagnumber, locations.department
		FROM locations
		ORDER BY locations.tagnumber, locations.time DESC
	),
	latest_checkouts AS (
		SELECT DISTINCT ON (checkouts.tagnumber) checkouts.tagnumber, checkouts.checkout_date, checkouts.return_date
		FROM checkouts
		ORDER BY checkouts.tagnumber, checkouts.time DESC
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
			(latest_locations.department IS NOT NULL AND latest_locations.department NOT IN ('property', 'pre-property')) AS loc_ok
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
