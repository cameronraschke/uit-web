package database

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"strconv"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
)

type AvailableJobs struct {
	Tagnumber   *int    `json:"tagnumber"`
	Job         *string `json:"job"`
	JobReadable *string `json:"job_readable"`
	JobActive   *bool   `json:"job_active"`
}

func GetAvailableJobs(ctx context.Context, db *sql.DB, tagnumber int) (string, error) {
	var sqlCode string
	var rows *sql.Rows
	var results []*AvailableJobs
	var resultsJson string
	var err error

	sqlCode = "SELECT remote.tagnumber, static_job_names.job, static_job_names.job_readable, remote.job_active FROM static_job_names LEFT JOIN remote ON (static_job_names.job = remote.job_queued OR (remote.job_active = TRUE OR remote.job_active = FALSE OR remote.job_queued IS NOT NULL OR remote.job_queued IS NULL)) WHERE remote.tagnumber = $1 AND static_job_names.job_html_bool = TRUE ORDER BY static_job_names.job_rank ASC;"

	rows, err = db.QueryContext(ctx, sqlCode, tagnumber)
	if err != nil {
		return "", errors.New("Timeout error: " + err.Error())
	}
	defer rows.Close()

	for rows.Next() {
		row := &AvailableJobs{}
		if err = rows.Err(); err != nil {
			return "", errors.New("Query error: " + err.Error())
		}
		if err = ctx.Err(); err != nil {
			return "", errors.New("Context error: " + err.Error())
		}
		err = rows.Scan(
			&row.Tagnumber,
			&row.Job,
			&row.JobReadable,
			&row.JobActive,
		)
		if err != nil && err != sql.ErrNoRows {
			return "", errors.New("Error scanning rows: " + err.Error())
		}
		results = append(results, row)
	}

	resultsJson, err = CreateJson(results)
	if err != nil {
		return "", errors.New("JSON error: " + err.Error())
	}
	return resultsJson, nil
}

type JobQueue struct {
	Tagnumber           *int    `json:"tagnumber"`
	PresentBool         *bool   `json:"present_bool"`
	KernelUpdated       *bool   `json:"kernel_updated"`
	BiosUpdated         *bool   `json:"bios_updated"`
	RemoteStatus        *string `json:"remote_status"`
	RemoteTimeFormatted *string `json:"remote_time_formatted"`
	JobQueued           *string `json:"job_queued"`
	JobQueuedFormatted  *string `json:"job_queued_formatted"`
	JobActive           *bool   `json:"job_active"`
}

func GetJobQueueByTagnumber(ctx context.Context, db *sql.DB, tagnumber int) (string, error) {
	var sqlCode string
	var rows *sql.Rows
	var results []*JobQueue
	var resultsJson string
	var err error

	sqlCode = `SELECT remote.tagnumber, remote.present_bool, remote.kernel_updated, client_health.bios_updated, 
  remote.status AS remote_status, TO_CHAR(remote.present, 'MM/DD/YY HH12:MI:SS AM') AS remote_time_formatted,
  (CASE WHEN remote.job_queued IS NOT NULL THEN remote.job_queued ELSE NULL END) AS job_queued,
  (CASE WHEN remote.job_queued IS NOT NULL THEN static_job_names.job_readable ELSE 'No Job' END) AS job_queued_formatted,
  remote.job_active
  FROM remote 
  LEFT JOIN client_health ON remote.tagnumber = client_health.tagnumber 
  LEFT JOIN static_job_names ON remote.job_queued = static_job_names.job 
  WHERE remote.tagnumber = $1`

	rows, err = db.QueryContext(ctx, sqlCode, tagnumber)
	if err != nil {
		return "", errors.New("Timeout error: " + err.Error())
	}
	defer rows.Close()

	for rows.Next() {
		row := &JobQueue{}
		if err = rows.Err(); err != nil {
			return "", errors.New("Query error: " + err.Error())
		}
		if err = ctx.Err(); err != nil {
			return "", errors.New("Context error: " + err.Error())
		}
		err = rows.Scan(
			&row.Tagnumber,
			&row.PresentBool,
			&row.KernelUpdated,
			&row.BiosUpdated,
			&row.RemoteStatus,
			&row.RemoteTimeFormatted,
			&row.JobQueued,
			&row.JobQueuedFormatted,
			&row.JobActive,
		)
		if err != nil && err != sql.ErrNoRows {
			return "", errors.New("Error scanning rows: " + err.Error())
		}
		results = append(results, row)
	}

	resultsJson, err = CreateJson(results)
	if err != nil {
		return "", errors.New("JSON error: " + err.Error())
	}
	return resultsJson, nil
}

type AllTags struct {
	Tagnumber *int32 `json:"tagnumber"`
}

func GetAllTags(ctx context.Context, db *sql.DB) (string, error) {
	var sqlCode string
	var rows *sql.Rows
	var results []*AllTags
	var resultsJson string
	var err error

	sqlCode = `SELECT t1.tagnumber FROM (SELECT time, tagnumber, ROW_NUMBER() OVER (PARTITION BY tagnumber ORDER BY time DESC) AS row_nums FROM locations) t1 WHERE t1.row_nums = 1 ORDER BY t1.time DESC`

	rows, err = db.QueryContext(ctx, sqlCode)
	if err != nil {
		return "", errors.New("Timeout error: " + err.Error())
	}
	defer rows.Close()

	for rows.Next() {
		row := &AllTags{}
		if err = rows.Err(); err != nil {
			return "", errors.New("Query error: " + err.Error())
		}
		if err = ctx.Err(); err != nil {
			return "", errors.New("Context error: " + err.Error())
		}
		err = rows.Scan(
			&row.Tagnumber,
		)
		if err != nil && err != sql.ErrNoRows {
			return "", errors.New("Error scanning rows: " + err.Error())
		}
		results = append(results, row)
	}

	resultsJson, err = CreateJson(results)
	if err != nil {
		return "", errors.New("JSON error: " + err.Error())
	}
	return resultsJson, nil

}

type RemoteOnlineTable struct {
	Tagnumber              *int32  `json:"tagnumber"`
	Screenshot             *string `json:"screenshot"`
	LastJobTimeFormatted   *string `json:"last_job_time_formatted"`
	LocationFormatted      *string `json:"location_formatted"`
	LocationsStatus        *bool   `json:"locations_status"`
	Status                 *string `json:"status"`
	OsInstalled            *bool   `json:"os_installed"`
	OsInstalledFormatted   *string `json:"os_installed_formatted"`
	BatteryChargeFormatted *string `json:"battery_charge_formatted"`
	Uptime                 *string `json:"uptime"`
	CpuTemp                *int32  `json:"cpu_temp"`
	CpuTempFormatted       *string `json:"cpu_temp_formatted"`
	DiskTemp               *int32  `json:"disk_temp"`
	DiskTempFormatted      *string `json:"disk_temp_formatted"`
	MaxDiskTemp            *int32  `json:"max_disk_temp"`
	WattsNow               *string `json:"watts_now"`
	Domain                 *string `json:"domain"`
	TimeFormatted          *string `json:"time_formatted"`
	JobQueued              *string `json:"job_queued"`
	QueuePosition          *int32  `json:"queue_position"`
	PresentBool            *bool   `json:"present_bool"`
	BiosUpdated            *bool   `json:"bios_updated"`
	BiosUpdatedFormatted   *string `json:"bios_updated_formatted"`
	KernelUpdated          *bool   `json:"kernel_updated"`
	JobActive              *bool   `json:"job_active"`
}

func GetRemoteOnlineTable(ctx context.Context, db *sql.DB) (string, error) {
	var sqlCode string
	var rows *sql.Rows
	var results []*RemoteOnlineTable
	var resultsJson string
	var err error

	sqlCode = `SELECT remote.tagnumber, live_images.screenshot, t1.domain, 
      TO_CHAR(remote.present, 'MM/DD/YY HH12:MI:SS AM') AS time_formatted, locationFormatting(t3.location) AS location_formatted, 
      TO_CHAR(remote.last_job_time, 'MM/DD/YY HH12:MI:SS AM') AS last_job_time_formatted, 
      remote.job_queued, remote.status, t2.queue_position, remote.present_bool, 
      client_health.os_name AS os_installed_formatted, client_health.os_installed, t1.locations_status, 
      client_health.bios_updated, (CASE WHEN client_health.bios_updated = TRUE THEN 'Yes' ELSE 'No' END) AS bios_updated_formatted, 
      remote.kernel_updated, CONCAT(remote.battery_charge, '%', ' - ', remote.battery_status) AS battery_charge_formatted, 
      AGE(NOW(), NOW() - (remote.uptime * INTERVAL '1 second')) AS uptime, 
      remote.cpu_temp, CONCAT(remote.cpu_temp, '°C') AS cpu_temp_formatted, 
      remote.disk_temp, CONCAT(remote.disk_temp, '°C') AS disk_temp_formatted, static_disk_stats.max_temp AS max_disk_temp, 
      CONCAT(remote.watts_now, ' watts') AS watts_now, remote.job_active
    FROM remote 
    LEFT JOIN (SELECT s1.time, s1.tagnumber, s1.domain, s1.status AS locations_status FROM (SELECT time, tagnumber, domain, status, ROW_NUMBER() OVER (PARTITION BY tagnumber ORDER BY time DESC) AS row_nums FROM locations) s1 WHERE s1.row_nums = 1) t1
      ON remote.tagnumber = t1.tagnumber
    LEFT JOIN client_health ON remote.tagnumber = client_health.tagnumber
    LEFT JOIN (SELECT tagnumber, location, row_nums FROM (SELECT tagnumber, location, ROW_NUMBER() OVER (PARTITION BY tagnumber ORDER BY time DESC) AS row_nums FROM locations) s3 WHERE s3.row_nums = 1) t3
      ON t3.tagnumber = remote.tagnumber
    LEFT JOIN (SELECT tagnumber, queue_position FROM (SELECT tagnumber, ROW_NUMBER() OVER (ORDER BY tagnumber ASC) AS queue_position FROM remote WHERE job_queued IS NOT NULL) s2) t2
      ON remote.tagnumber = t2.tagnumber
    LEFT JOIN (SELECT s4.tagnumber, s4.disk_model FROM (SELECT tagnumber, disk_model, ROW_NUMBER() OVER (PARTITION BY tagnumber ORDER BY time DESC) AS row_nums FROM jobstats WHERE tagnumber IS NOT NULL AND disk_model IS NOT NULL) s4 WHERE s4.row_nums = 1) t4
      ON remote.tagnumber = t4.tagnumber
    LEFT JOIN static_disk_stats ON static_disk_stats.disk_model = t4.disk_model
    LEFT JOIN live_images ON remote.tagnumber = live_images.tagnumber
    WHERE remote.present_bool = TRUE
    ORDER BY
      remote.status LIKE 'fail%' DESC, job_queued IS NOT NULL DESC, job_active = TRUE DESC, queue_position ASC,
      (CASE WHEN job_queued = 'data collection' THEN 20 WHEN job_queued = 'update' THEN 15 WHEN job_queued = 'nvmeVerify' THEN 14 WHEN job_queued =  'nvmeErase' THEN 12 WHEN job_queued =  'hpCloneOnly' THEN 11 WHEN job_queued = 'hpEraseAndClone' THEN 10 WHEN job_queued = 'findmy' THEN 8 WHEN job_queued = 'shutdown' THEN 7 WHEN job_queued = 'fail-test' THEN 5 END) DESC, 
      status = 'Waiting for job' ASC, client_health.os_installed = TRUE DESC, 
      remote.kernel_updated DESC, t1.locations_status = TRUE DESC, client_health.bios_updated = TRUE DESC, 
      remote.last_job_time DESC`

	rows, err = db.QueryContext(ctx, sqlCode)
	if err != nil {
		return "", errors.New("Timeout error: " + err.Error())
	}
	defer rows.Close()

	for rows.Next() {
		row := &RemoteOnlineTable{}
		if err = rows.Err(); err != nil {
			return "", errors.New("Query error: " + err.Error())
		}
		if err = ctx.Err(); err != nil {
			return "", errors.New("Context error: " + err.Error())
		}
		err = rows.Scan(
			&row.Tagnumber,
			&row.Screenshot,
			&row.Domain,
			&row.TimeFormatted,
			&row.LocationFormatted,
			&row.LastJobTimeFormatted,
			&row.JobQueued,
			&row.Status,
			&row.QueuePosition,
			&row.PresentBool,
			&row.OsInstalledFormatted,
			&row.OsInstalled,
			&row.LocationsStatus,
			&row.BiosUpdated,
			&row.BiosUpdatedFormatted,
			&row.KernelUpdated,
			&row.BatteryChargeFormatted,
			&row.Uptime,
			&row.CpuTemp,
			&row.CpuTempFormatted,
			&row.DiskTemp,
			&row.DiskTempFormatted,
			&row.MaxDiskTemp,
			&row.WattsNow,
			&row.JobActive,
		)
		if err != nil && err != sql.ErrNoRows {
			return "", errors.New("Error scanning rows: " + err.Error())
		}
		results = append(results, row)
	}

	resultsJson, err = CreateJson(results)
	if err != nil {
		return "", errors.New("JSON error: " + err.Error())
	}
	return resultsJson, nil
}

type RemoteOfflineTable struct {
	Tagnumber              *int32  `json:"tagnumber"`
	TimeFormatted          *string `json:"time_formatted"`
	Status                 *string `json:"status"`
	LocationsStatus        *string `json:"locations_status"`
	Location               *string `json:"location_formatted"`
	BatteryChargeFormatted *string `json:"battery_charge_formatted"`
	CpuTempFormatted       *string `json:"cpu_temp_formatted"`
	DiskTempFormatted      *string `json:"disk_temp_formatted"`
	WattsNowFormatted      *string `json:"watts_now_formatted"`
	OsInstalledFormatted   *string `json:"os_installed_formatted"`
	OsInstalled            *bool   `json:"os_installed"`
	DomainJoined           *bool   `json:"domain_joined"`
}

func GetRemoteOfflineTable(ctx context.Context, db *sql.DB) (string, error) {
	var sqlCode string
	var rows *sql.Rows
	var results []*RemoteOfflineTable
	var resultsJson string
	var err error

	sqlCode = `SELECT remote.tagnumber, TO_CHAR(remote.present, 'MM/DD/YY HH12:MI:SS AM') AS time_formatted, 
        remote.status, locations.status AS locations_status, locationFormatting(locations.location) AS location_formatted, CONCAT(remote.battery_charge, '%', ' - ', remote.battery_status) AS battery_charge_formatted, 
        CONCAT(remote.cpu_temp, '°C') AS cpu_temp_formatted, 
        CONCAT(remote.disk_temp, '°C') AS disk_temp_formatted, CONCAT(remote.watts_now, ' watts') AS watts_now_formatted,
        client_health.os_name AS os_installed_formatted, client_health.os_installed, 
        (CASE WHEN locations.domain IS NOT NULL THEN TRUE ELSE FALSE END) AS domain_joined
      FROM remote 
      LEFT JOIN client_health ON remote.tagnumber = client_health.tagnumber
      LEFT JOIN locations ON remote.tagnumber = locations.tagnumber AND locations.time IN (SELECT time FROM (SELECT time, tagnumber, ROW_NUMBER() OVER (PARTITION BY tagnumber ORDER BY time DESC) AS row_nums FROM locations) s1 WHERE s1.row_nums = 1)
      WHERE remote.present_bool IS FALSE 
        AND remote.present IS NOT NULL 
      ORDER BY remote.present DESC, remote.tagnumber DESC`

	rows, err = db.QueryContext(ctx, sqlCode)
	if err != nil {
		return "", errors.New("Timeout error: " + err.Error())
	}
	defer rows.Close()

	for rows.Next() {
		row := &RemoteOfflineTable{}
		if err = rows.Err(); err != nil {
			return "", errors.New("Query error: " + err.Error())
		}
		if err = ctx.Err(); err != nil {
			return "", errors.New("Context error: " + err.Error())
		}
		err = rows.Scan(
			&row.Tagnumber,
			&row.TimeFormatted,
			&row.Status,
			&row.LocationsStatus,
			&row.Location,
			&row.BatteryChargeFormatted,
			&row.CpuTempFormatted,
			&row.DiskTempFormatted,
			&row.WattsNowFormatted,
			&row.OsInstalledFormatted,
			&row.OsInstalled,
			&row.DomainJoined,
		)
		if err != nil && err != sql.ErrNoRows {
			return "", errors.New("Error scanning rows: " + err.Error())
		}
		results = append(results, row)
	}

	resultsJson, err = CreateJson(results)
	if err != nil {
		return "", errors.New("JSON error: " + err.Error())
	}
	return resultsJson, nil
}

type LiveImage struct {
	TimeFormatted *string `json:"time_formatted"`
	Screenshot    *string `json:"screenshot"`
}

func GetLiveImage(ctx context.Context, db *sql.DB, tagnumber int) (string, error) {
	var sqlCode string
	var rows *sql.Rows
	var results []*LiveImage
	var resultsJson string
	var err error

	sqlCode = `SELECT TO_CHAR(time, 'MM/DD/YY HH12:MI:SS AM') AS time_formatted, screenshot 
            FROM live_images 
            WHERE tagnumber = $1`

	rows, err = db.QueryContext(ctx, sqlCode, tagnumber)
	if err != nil {
		return "", errors.New("Timeout error: " + err.Error())
	}
	defer rows.Close()

	for rows.Next() {
		row := &LiveImage{}
		if err = rows.Err(); err != nil {
			return "", errors.New("Query error: " + err.Error())
		}
		if err = ctx.Err(); err != nil {
			return "", errors.New("Context error: " + err.Error())
		}
		err = rows.Scan(
			&row.TimeFormatted,
			&row.Screenshot,
		)
		if err != nil && err != sql.ErrNoRows {
			return "", errors.New("Error scanning rows: " + err.Error())
		}
		results = append(results, row)
	}

	resultsJson, err = CreateJson(results)
	if err != nil {
		return "", errors.New("JSON error: " + err.Error())
	}
	return resultsJson, nil

}

type RemotePresentHeader struct {
	TagnumberCount         *string `json:"tagnumber_count"`
	OsInstalledFormatted   *string `json:"os_installed_formatted"`
	BatteryChargeFormatted *string `json:"battery_charge_formatted"`
	CpuTempFormatted       *string `json:"cpu_temp_formatted"`
	DiskTempFormatted      *string `json:"disk_temp_formatted"`
	PowerUsageFormatted    *string `json:"power_usage_formatted"`
}

func GetRemotePresentHeader(ctx context.Context, db *sql.DB) (string, error) {
	var sqlCode string
	var rows *sql.Rows
	var results []*RemotePresentHeader
	var resultsJson string
	var err error

	sqlCode = `SELECT CONCAT('(', COUNT(remote.tagnumber), ')') AS tagnumber_count, 
        CONCAT('(', MIN(remote.battery_charge), '%', '/', MAX(remote.battery_charge), '%', '/', ROUND(AVG(remote.battery_charge), 2), '%', ')') AS battery_charge_formatted, 
        CONCAT('(', MIN(remote.cpu_temp), '°C', '/', MAX(remote.cpu_temp), '°C', '/', ROUND(AVG(remote.cpu_temp), 2), '°C', ')') AS cpu_temp_formatted, 
        CONCAT('(', MIN(remote.disk_temp), '°C',  '/', MAX(remote.disk_temp), '°C' , '/', ROUND(AVG(remote.disk_temp), 2), '°C' , ')') AS disk_temp_formatted, 
        CONCAT('(', SUM((CASE WHEN client_health.os_installed = TRUE THEN 1 ELSE 0 END)), ')') AS os_installed_formatted, 
        CONCAT('(', SUM(remote.watts_now), ' ', 'watts', ')') AS power_usage_formatted 
      FROM remote 
      LEFT JOIN client_health 
        ON remote.tagnumber = client_health.tagnumber 
      WHERE remote.present_bool = TRUE`

	rows, err = db.QueryContext(ctx, sqlCode)
	if err != nil {
		return "", errors.New("Timeout error: " + err.Error())
	}
	defer rows.Close()

	for rows.Next() {
		row := &RemotePresentHeader{}
		if err = rows.Err(); err != nil {
			return "", errors.New("Query error: " + err.Error())
		}
		if err = ctx.Err(); err != nil {
			return "", errors.New("Context error: " + err.Error())
		}

		err = rows.Scan(
			&row.TagnumberCount,
			&row.BatteryChargeFormatted,
			&row.CpuTempFormatted,
			&row.DiskTempFormatted,
			&row.OsInstalledFormatted,
			&row.PowerUsageFormatted,
		)
		if err != nil && err != sql.ErrNoRows {
			return "", errors.New("Error scanning rows: " + err.Error())
		}
		results = append(results, row)
	}

	resultsJson, err = CreateJson(results)
	if err != nil {
		return "", errors.New("JSON error: " + err.Error())
	}
	return resultsJson, nil
}

type TagnumberData struct {
	LocationTimeFormatted  *string    `json:"location_time_formatted"`
	PlaceholderBool        *bool      `json:"placeholder_bool"`
	JobstatsTime           *string    `json:"jobstatstime"`
	Tagnumber              *int       `json:"tagnumber"`
	SystemSerial           *string    `json:"system_serial"`
	Department             *string    `json:"department"`
	Location               *string    `json:"location"`
	RemoteStatus           *string    `json:"remote_status_formatted"`
	LocationsStatus        *bool      `json:"locations_status"`
	DepartmentReadable     *string    `json:"department_readable"`
	MostRecentNote         *string    `json:"most_recent_note"`
	Note                   *string    `json:"note"`
	NoteTime               *string    `json:"note_time_formatted"`
	DiskRemovedFormatted   *string    `json:"disk_removed_formatted"`
	DiskRemoved            *bool      `json:"disk_removed"`
	Etheraddress           *string    `json:"etheraddress_formatted"`
	WifiMac                *string    `json:"wifi_mac_formatted"`
	ChassicType            *string    `json:"chassis_type"`
	SystemModel            *string    `json:"system_model_formatted"`
	CpuModel               *string    `json:"cpu_model"`
	CpuMaxSpeed            *string    `json:"cpu_maxspeed_formatted"`
	MultithreadedFormatted *string    `json:"multithreaded_formatted"`
	RamCapacityFormatted   *string    `json:"ram_capacity_formatted"`
	DiskModel              *string    `json:"disk_model"`
	DiskSize               *string    `json:"disk_size"`
	DiskType               *string    `json:"disk_type"`
	DiskSerial             *string    `json:"disk_serial"`
	Identifier             *string    `json:"identifier"`
	RecoveryKey            *string    `json:"recovery_key"`
	BatteryHealth          *string    `json:"battery_health"`
	DiskHealth             *float32   `json:"disk_health"`
	AvgEraseTime           *int       `json:"avg_erase_time"`
	AvgCloneTime           *int       `json:"avg_clone_time"`
	AllJobs                *int       `json:"all_jobs"`
	NetworkSpeedFormatted  *string    `json:"network_speed_formatted"`
	BiosUpdated            *bool      `json:"bios_updated"`
	BiosUpdatedFormatted   *string    `json:"bios_updated_formatted"`
	DiskWrites             *float32   `json:"disk_writes"`
	DiskReads              *float32   `json:"disk_reads"`
	DiskPowerOnHours       *int       `json:"disk_power_on_hours"`
	DiskPowerCycles        *int       `json:"disk_power_cycles"`
	DiskErrors             *int       `json:"disk_errors"`
	Domain                 *string    `json:"domain"`
	DomainReadable         *string    `json:"domain_readable"`
	OsInstalledFormatted   *string    `json:"os_installed_formatted"`
	CustomerName           *string    `json:"customer_name"`
	CheckoutDate           *time.Time `json:"checkout_date"`
	CheckoutBool           *bool      `json:"checkout_bool"`
	TpmVersion             *int       `json:"tpm_version"`
}

func GetTagnumberData(ctx context.Context, db *sql.DB, tagnumber int) (string, error) {
	var sqlCode string
	var rows *sql.Rows
	var results []*TagnumberData
	var resultsJson string
	var err error

	sqlCode = `SELECT TO_CHAR(t10.time, 'MM/DD/YY HH12:MI:SS AM') AS location_time_formatted,
    (CASE WHEN t3.time = t10.time THEN 1 ELSE 0 END) AS placeholder_bool,
    jobstats.time AS jobstatsTime, locations.tagnumber, locations.system_serial, locations.department, 
    locationFormatting(locations.location) AS location, 
    (CASE WHEN locations.status = TRUE THEN 'Broken' ELSE 'Yes' END) AS remote_status_formatted, locations.status AS locations_status, t2.department_readable, t3.note AS most_recent_note,
    locations.note, TO_CHAR(t3.time, 'MM/DD/YY HH12:MI:SS AM') AS note_time_formatted, 
    (CASE WHEN locations.disk_removed = TRUE THEN 'Yes' ELSE 'No' END) AS disk_removed_formatted, locations.disk_removed,
    (CASE 
      WHEN jobstats.etheraddress IS NOT NULL AND system_data.system_model NOT IN ('Latitude 7400', 'Latitude 5289') THEN jobstats.etheraddress 
      WHEN jobstats.etheraddress IS NOT NULL AND system_data.system_model IN ('Latitude 7400', 'Latitude 5289') THEN NULL 
      ELSE NULL
      END) AS etheraddress_formatted, 
    system_data.wifi_mac, 
    system_data.chassis_type, 
    (CASE
      WHEN system_data.system_manufacturer IS NOT NULL AND system_data.system_model IS NOT NULL THEN CONCAT(system_data.system_manufacturer, ' - ', system_data.system_model)
      WHEN system_data.system_manufacturer IS NULL AND system_data.system_model IS NOT NULL THEN system_data.system_model
      WHEN system_data.system_manufacturer IS NOT NULL AND system_data.system_model IS NULL THEN system_data.system_manufacturer
      ELSE NULL
      END) AS system_model_formatted,
    system_data.cpu_model,
    (CASE 
      WHEN system_data.cpu_maxspeed IS NOT NULL THEN CONCAT('(Max ', ROUND((system_data.cpu_maxspeed / 1000), 2), ' Ghz)') 
      ELSE NULL 
      END) AS cpu_maxspeed_formatted, 
    (CASE 
      WHEN system_data.cpu_threads > system_data.cpu_cores THEN CONCAT(system_data.cpu_cores, ' cores/', system_data.cpu_threads, ' threads (Multithreaded)') 
      WHEN system_data.cpu_threads = system_data.cpu_cores THEN CONCAT(system_data.cpu_cores, ' cores (Not Multithreaded)')
      ELSE NULL
      END) AS multithreaded_formatted, 
    (CASE 
    WHEN t8.ram_capacity IS NOT NULL AND t8.ram_speed IS NOT NULL THEN CONCAT(t8.ram_capacity, ' GB (', t8.ram_speed, ' MHz)')
    WHEN t8.ram_capacity IS NOT NULL AND t8.ram_speed IS NULL THEN CONCAT(t8.ram_capacity, ' GB')
    END) AS ram_capacity_formatted,
    t4.disk_model, t4.disk_size, t4.disk_type, t4.disk_serial, 
    t5.identifier, t5.recovery_key, 
    (CASE WHEN client_health.battery_health IS NOT NULL THEN client_health.battery_health ELSE NULL END) AS battery_health, client_health.disk_health, 
    (CASE 
      WHEN client_health.avg_erase_time IS NOT NULL THEN client_health.avg_erase_time
      ELSE NULL 
      END) AS avg_erase_time, 
    (CASE 
      WHEN client_health.avg_clone_time IS NOT NULL THEN client_health.avg_clone_time
      ELSE NULL
      END) AS avg_clone_time,
    client_health.all_jobs, 
    (CASE 
      WHEN remote.network_speed IS NOT NULL 
      THEN CONCAT(remote.network_speed, ' mbps') 
      ELSE NULL 
      END) AS network_speed_formatted, 
    client_health.bios_updated, 
    (CASE 
      WHEN client_health.bios_updated = TRUE AND client_health.bios_version IS NOT NULL THEN CONCAT('Updated ', '(', client_health.bios_version, ')') 
      WHEN client_health.bios_updated = FALSE AND client_health.bios_version IS NOT NULL THEN CONCAT('Out of date ', '(', client_health.bios_version, ')') 
      ELSE 'Unknown BIOS Version' 
      END) AS bios_updated_formatted, 
    t4.disk_writes, t4.disk_reads, t4.disk_power_on_hours,
    t4.disk_power_cycles, t4.disk_errors, locations.domain, (CASE WHEN locations.domain IS NOT NULL THEN static_domains.domain_readable ELSE 'Not Joined' END) AS domain_readable,
    (CASE 
      WHEN client_health.os_installed = FALSE AND client_health.os_name IS NOT NULL AND client_health.os_name = 'Image Job Failed' THEN CONCAT(client_health.os_name, ' (Failed on ', TO_CHAR(t6.time, 'MM/DD/YY HH12:MI:SS AM'), ')')
      WHEN client_health.os_installed = TRUE AND client_health.os_name IS NOT NULL AND NOT client_health.os_name = 'Unknown OS' THEN CONCAT(client_health.os_name, ' (Imaged on ', TO_CHAR(t6.time, 'MM/DD/YY HH12:MI:SS AM'), ')') 
      WHEN client_health.os_installed = TRUE AND NOT client_health.os_name = 'Unknown OS' THEN client_health.os_name 
      ELSE client_health.os_name 
      END) AS os_installed_formatted,
    checkouts.customer_name, checkouts.checkout_date, checkouts.checkout_bool, client_health.tpm_version
    FROM locations
    LEFT JOIN client_health ON locations.tagnumber = client_health.tagnumber
    LEFT JOIN jobstats ON (locations.tagnumber = jobstats.tagnumber AND jobstats.time IN (SELECT MAX(time) AS time FROM jobstats WHERE tagnumber IS NOT NULL AND system_serial IS NOT NULL AND (host_connected = TRUE OR (uuid LIKE 'techComm-%' AND etheraddress IS NOT NULL)) GROUP BY tagnumber))
    LEFT JOIN system_data ON locations.tagnumber = system_data.tagnumber
    LEFT JOIN remote ON locations.tagnumber = remote.tagnumber
    LEFT JOIN (SELECT department, department_readable FROM static_departments) t2
    ON locations.department = t2.department
    LEFT JOIN (SELECT tagnumber, time, note FROM locations WHERE time IN (SELECT MAX(time) FROM locations WHERE note IS NOT NULL GROUP BY tagnumber)) t3
    ON locations.tagnumber = t3.tagnumber
    LEFT JOIN (SELECT tagnumber, disk_model, disk_serial, disk_size, disk_type, disk_writes, disk_reads, disk_power_on_hours, disk_power_cycles, (CASE WHEN disk_errors IS NOT NULL THEN disk_errors ELSE 0 END) AS disk_errors FROM jobstats WHERE time IN (SELECT MAX(time) FROM jobstats WHERE disk_type IS NOT NULL AND tagnumber IS NOT NULL GROUP BY tagnumber)) t4 
    ON locations.tagnumber = t4.tagnumber
    LEFT JOIN (SELECT tagnumber, identifier, recovery_key FROM bitlocker) t5 
    ON locations.tagnumber = t5.tagnumber
    LEFT JOIN (SELECT time, tagnumber, clone_image, row_nums FROM (SELECT time, tagnumber, clone_image, ROW_NUMBER() OVER (PARTITION BY tagnumber ORDER BY time DESC) AS row_nums FROM jobstats WHERE tagnumber IS NOT NULL AND clone_completed = TRUE AND clone_image IS NOT NULL) s6 WHERE s6.row_nums = 1) t6
    ON locations.tagnumber = t6.tagnumber
    LEFT JOIN static_image_names ON t6.clone_image = static_image_names.image_name
    LEFT JOIN (SELECT tagnumber, ram_capacity, ram_speed FROM jobstats WHERE time IN (SELECT MAX(time) FROM jobstats WHERE ram_capacity IS NOT NULL AND ram_speed IS NOT NULL AND tagnumber IS NOT NULL GROUP BY tagnumber)) t8
    ON locations.tagnumber = t8.tagnumber
    INNER JOIN (SELECT MAX(time) AS time FROM locations WHERE tagnumber IS NOT NULL AND system_serial IS NOT NULL GROUP BY tagnumber) t10
    ON locations.time = t10.time
    LEFT JOIN static_domains ON locations.domain = static_domains.domain
    LEFT JOIN checkouts ON locations.tagnumber = checkouts.tagnumber AND checkouts.time IN (SELECT s11.time FROM (SELECT time, ROW_NUMBER() OVER (PARTITION BY tagnumber ORDER BY time DESC) AS row_nums FROM checkouts) s11 WHERE s11.row_nums = 1)
    WHERE locations.tagnumber IS NOT NULL and locations.system_serial IS NOT NULL
    AND locations.tagnumber = $1`

	rows, err = db.QueryContext(ctx, sqlCode, tagnumber)
	if err != nil {
		return "", errors.New("Timeout error: " + err.Error())
	}
	defer rows.Close()

	for rows.Next() {
		row := &TagnumberData{}
		if err = rows.Err(); err != nil {
			return "", errors.New("Query error: " + err.Error())
		}
		if err = ctx.Err(); err != nil {
			return "", errors.New("Context error: " + err.Error())
		}

		err = rows.Scan(
			&row.LocationTimeFormatted,
			&row.PlaceholderBool,
			&row.JobstatsTime,
			&row.Tagnumber,
			&row.SystemSerial,
			&row.Department,
			&row.Location,
			&row.RemoteStatus,
			&row.LocationsStatus,
			&row.DepartmentReadable,
			&row.MostRecentNote,
			&row.Note,
			&row.NoteTime,
			&row.DiskRemovedFormatted,
			&row.DiskRemoved,
			&row.Etheraddress,
			&row.WifiMac,
			&row.ChassicType,
			&row.SystemModel,
			&row.CpuModel,
			&row.CpuMaxSpeed,
			&row.MultithreadedFormatted,
			&row.RamCapacityFormatted,
			&row.DiskModel,
			&row.DiskSize,
			&row.DiskType,
			&row.DiskSerial,
			&row.Identifier,
			&row.RecoveryKey,
			&row.BatteryHealth,
			&row.DiskHealth,
			&row.AvgEraseTime,
			&row.AvgCloneTime,
			&row.AllJobs,
			&row.NetworkSpeedFormatted,
			&row.BiosUpdated,
			&row.BiosUpdatedFormatted,
			&row.DiskWrites,
			&row.DiskReads,
			&row.DiskPowerOnHours,
			&row.DiskPowerCycles,
			&row.DiskErrors,
			&row.Domain,
			&row.DomainReadable,
			&row.OsInstalledFormatted,
			&row.CustomerName,
			&row.CheckoutDate,
			&row.CheckoutBool,
			&row.TpmVersion,
		)
		if err != nil && err != sql.ErrNoRows {
			return "", errors.New("Error scanning rows: " + err.Error())
		}
		results = append(results, row)
	}

	resultsJson, err = CreateJson(results)
	if err != nil {
		return "", errors.New("JSON error: " + err.Error())
	}
	return resultsJson, nil
}

func CreateJson(results any) (string, error) {
	var jsonData []byte
	var jsonDataStr string
	var err error

	jsonData, err = json.Marshal(results)
	if err != nil {
		return "", errors.New("Error creating JSON data: " + err.Error())
	}

	// Convert jsonData to string
	if len(jsonData) > 0 {
		jsonDataStr = string(jsonData)
	} else {
		return "", errors.New("length of JSON is zero")
	}

	return jsonDataStr, nil
}

func UpdateDB(ctx context.Context, db *sql.DB, sqlCode string, value string, tagnumber int) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return errors.New("Cannot begin DB transaction: " + err.Error())
	}
	defer tx.Rollback()

	fail := func(err error) error {
		tx.Rollback()
		return errors.New("Error while updating DB (rollback): " + err.Error() + " \n\tSQL: " + sqlCode + " Value: " + value + " Tagnumber: " + strconv.Itoa(tagnumber))
	}

	result, err := tx.ExecContext(ctx, sqlCode, value, tagnumber)
	if err != nil {
		return fail(err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fail(err)
	}

	if rowsAffected != 1 {
		return fail(errors.New("rows affected are not exactly 1"))
	}

	if err = tx.Commit(); err != nil {
		return fail(err)
	}

	return nil
}
