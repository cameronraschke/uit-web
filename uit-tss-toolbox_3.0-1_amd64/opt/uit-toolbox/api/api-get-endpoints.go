package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// Helper functions

func jsonEncode(v any) (jsonStr string, err error) {
	jsonBytes, err := json.Marshal(v)
	if err != nil {
		return "", err
	}
	return string(jsonBytes), nil
}

// Per-client functions

type serverTime struct {
	Time string `json:"server_time"`
}

func getServerTime(w http.ResponseWriter, r *http.Request) {
	requestIP, ok := GetRequestIP(r)
	if !ok {
		log.Warning("no IP address stored in context")
		http.Error(w, formatHttpError("Internal server error"), http.StatusInternalServerError)
		return
	}
	requestURL, ok := GetRequestURL(r)
	if !ok {
		log.Warning("no URL stored in context")
		http.Error(w, formatHttpError("Internal server error"), http.StatusInternalServerError)
		return
	}

	jsonData, err := jsonEncode(serverTime{
		Time: time.Now().Format("2006-01-02 15:04:05.000"),
	})
	if err != nil || strings.TrimSpace(jsonData) == "" {
		log.Error("Cannot parse JSON from " + requestIP + " (" + requestURL + "): " + err.Error())
		http.Error(w, formatHttpError("Internal server error"), http.StatusInternalServerError)
		return
	}
	io.WriteString(w, jsonData)
}

type clientLookup struct {
	Tagnumber    int    `json:"tagnumber"`
	SystemSerial string `json:"system_serial"`
}

func dbSelClientLookup(ctx context.Context, db *sql.DB, tagnumber int, serial string) (string, error) {
	var sqlQuery string
	var results []*clientLookup

	if strings.TrimSpace(serial) != "" {
		sqlQuery = "SELECT tagnumber, system_serial FROM locations WHERE system_serial = $1 ORDER BY time DESC LIMIT 1;"
	} else if tagnumber > 0 {
		sqlQuery = "SELECT tagnumber, system_serial FROM locations WHERE tagnumber = $1 ORDER BY time DESC LIMIT 1;"
	} else {
		return "", errors.New("no tagnumber or system_serial provided")
	}

	rows, err := db.QueryContext(ctx, sqlQuery, tagnumber)
	if err != nil {
		return "", errors.New("Context error: " + err.Error())
	}
	defer rows.Close()

	for rows.Next() {
		row := &clientLookup{}
		if err := rows.Scan(
			&row.Tagnumber,
			&row.SystemSerial,
		); err != nil {
			return "", errors.New("Error scanning rows: " + err.Error())
		}
		results = append(results, row)
	}
	if err := rows.Err(); err != nil {
		return "", errors.New("Query error: " + err.Error())
	}
	if err := ctx.Err(); err != nil {
		return "", errors.New("Context error: " + err.Error())
	}
	if len(results) == 0 {
		return "", errors.New("no results found")
	}

	jsonStr, err := jsonEncode(results)
	if err != nil {
		return "", errors.New("Error encoding results to JSON: " + err.Error())
	}

	return jsonStr, nil
}

func getClientLookup(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	requestIP, ok := GetRequestIP(r)
	if !ok {
		log.Warning("no IP address stored in context")
		http.Error(w, formatHttpError("Internal server error"), http.StatusInternalServerError)
		return
	}
	requestURL, ok := GetRequestURL(r)
	if !ok {
		log.Warning("no URL stored in context")
		http.Error(w, formatHttpError("Internal server error"), http.StatusInternalServerError)
		return
	}

	tag := strings.TrimSpace(r.URL.Query().Get("tagnumber"))
	systemSerial := strings.TrimSpace(r.URL.Query().Get("system_serial"))
	if tag == "" && systemSerial == "" {
		log.Warning("No tagnumber or system_serial provided in request from: " + requestIP + " (" + requestURL + ")")
		http.Error(w, formatHttpError("Bad request"), http.StatusBadRequest)
		return
	}

	var tagnumber, err = strconv.Atoi(tag)
	if err != nil || tagnumber <= 0 {
		tagnumber = 0
	}
	jsonStr, err := dbSelClientLookup(ctx, db, tagnumber, systemSerial)
	if err != nil {
		log.Warning("Database lookup failed for: " + requestIP + " (" + requestURL + "): " + err.Error())
		http.Error(w, formatHttpError("Internal server error"), http.StatusInternalServerError)
		return
	}
	io.WriteString(w, jsonStr)
}

type hardwareData struct {
	Tagnumber               int    `json:"tagnumber"`
	SystemSerial            string `json:"system_serial"`
	EthernetMAC             string `json:"ethernet_mac"`
	WifiMac                 string `json:"wifi_mac"`
	SystemModel             string `json:"system_model"`
	SystemUUID              string `json:"system_uuid"`
	SystemSKU               string `json:"system_sku"`
	ChassisType             string `json:"chassis_type"`
	MotherboardManufacturer string `json:"motherboard_manufacturer"`
	MotherboardSerial       string `json:"motherboard_serial"`
	SystemManufacturer      string `json:"system_manufacturer"`
}

func dbSelHardwareData(ctx context.Context, db *sql.DB, tagnumber int) (string, error) {
	var sqlQuery string
	var results []*hardwareData

	if tagnumber <= 0 {
		return "", errors.New("no tagnumber provided")
	}

	sqlQuery = `SELECT locations.tagnumber, locations.system_serial, jobstats.etheraddress, system_data.wifi_mac,
	system_data.system_model, system_data.system_uuid, system_data.system_sku, system_data.chassis_type, 
	system_data.motherboard_manufacturer, system_data.motherboard_serial, system_data.system_manufacturer
	FROM locations
	LEFT JOIN jobstats ON locations.tagnumber = jobstats.tagnumber AND jobstats.time IN (SELECT MAX(time) FROM jobstats GROUP BY tagnumber)
	LEFT JOIN system_data ON locations.tagnumber = system_data.tagnumber
	WHERE locations.time IN (SELECT MAX(time) FROM locations GROUP BY tagnumber)
	AND locations.tagnumber = $1`

	rows, err := db.QueryContext(ctx, sqlQuery, tagnumber)
	if err != nil {
		return "", errors.New("Context error: " + err.Error())
	}
	defer rows.Close()

	for rows.Next() {
		row := &hardwareData{}
		if err := rows.Scan(
			&row.Tagnumber,
			&row.SystemSerial,
			&row.EthernetMAC,
			&row.WifiMac,
			&row.SystemModel,
			&row.SystemUUID,
			&row.SystemSKU,
			&row.ChassisType,
			&row.MotherboardManufacturer,
			&row.MotherboardSerial,
			&row.SystemManufacturer,
		); err != nil {
			return "", errors.New("Error scanning rows: " + err.Error())
		}
		results = append(results, row)
	}
	if err := rows.Err(); err != nil {
		return "", errors.New("Query error: " + err.Error())
	}
	if err := ctx.Err(); err != nil {
		return "", errors.New("Context error: " + err.Error())
	}
	if len(results) == 0 {
		return "", errors.New("no results found")
	}

	jsonStr, err := jsonEncode(results)
	if err != nil {
		return "", errors.New("Error encoding results to JSON: " + err.Error())
	}

	return jsonStr, nil
}

func getHardwareData(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	requestIP, ok := GetRequestIP(r)
	if !ok {
		log.Warning("no IP address stored in context")
		http.Error(w, formatHttpError("Internal server error"), http.StatusInternalServerError)
		return
	}
	requestURL, ok := GetRequestURL(r)
	if !ok {
		log.Warning("no URL stored in context")
		http.Error(w, formatHttpError("Internal server error"), http.StatusInternalServerError)
		return
	}

	tag := strings.TrimSpace(r.URL.Query().Get("tagnumber"))
	if tag == "" {
		log.Warning("No tagnumber provided in request from: " + requestIP + " (" + requestURL + ")")
		http.Error(w, formatHttpError("Bad request"), http.StatusBadRequest)
		return
	}

	var tagnumber, err = strconv.Atoi(tag)
	if err != nil || tagnumber <= 0 {
		tagnumber = 0
	}
	jsonStr, err := dbSelHardwareData(ctx, db, tagnumber)
	if err != nil {
		log.Warning("Database lookup failed for: " + requestIP + " (" + requestURL + "): " + err.Error())
		http.Error(w, formatHttpError("Internal server error"), http.StatusInternalServerError)
		return
	}
	io.WriteString(w, jsonStr)
}

type biosData struct {
	Tagnumber   int    `json:"tagnumber"`
	BiosVersion string `json:"bios_version"`
	BiosUpdated bool   `json:"bios_updated"`
	BiosDate    string `json:"bios_date"`
	TpmVersion  string `json:"tpm_version"`
}

func dbSelBiosData(ctx context.Context, db *sql.DB, tagnumber int) (string, error) {
	var sqlQuery string
	var results []*biosData

	if tagnumber <= 0 {
		return "", errors.New("no tagnumber provided")
	}

	sqlQuery = `SELECT client_health.tagnumber, client_health.bios_version, client_health.bios_updated, 
	client_health.bios_date, client_health.tpm_version 
	FROM client_health WHERE client_health.tagnumber = $1`

	rows, err := db.QueryContext(ctx, sqlQuery, tagnumber)
	if err != nil {
		return "", errors.New("Context error: " + err.Error())
	}
	defer rows.Close()

	for rows.Next() {
		row := &biosData{}
		if err := rows.Scan(
			&row.Tagnumber,
			&row.BiosVersion,
			&row.BiosUpdated,
			&row.BiosDate,
			&row.TpmVersion,
		); err != nil {
			return "", errors.New("Error scanning rows: " + err.Error())
		}
		results = append(results, row)
	}
	if err := rows.Err(); err != nil {
		return "", errors.New("Query error: " + err.Error())
	}
	if err := ctx.Err(); err != nil {
		return "", errors.New("Context error: " + err.Error())
	}
	if len(results) == 0 {
		return "", errors.New("no results found")
	}

	jsonStr, err := jsonEncode(results)
	if err != nil {
		return "", errors.New("Error encoding results to JSON: " + err.Error())
	}

	return jsonStr, nil
}

func getBiosData(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	requestIP, ok := GetRequestIP(r)
	if !ok {
		log.Warning("no IP address stored in context")
		http.Error(w, formatHttpError("Internal server error"), http.StatusInternalServerError)
		return
	}
	requestURL, ok := GetRequestURL(r)
	if !ok {
		log.Warning("no URL stored in context")
		http.Error(w, formatHttpError("Internal server error"), http.StatusInternalServerError)
		return
	}

	tag := strings.TrimSpace(r.URL.Query().Get("tagnumber"))
	if tag == "" {
		log.Warning("No tagnumber provided in request from: " + requestIP + " (" + requestURL + ")")
		http.Error(w, formatHttpError("Bad request"), http.StatusBadRequest)
		return
	}

	var tagnumber, err = strconv.Atoi(tag)
	if err != nil || tagnumber <= 0 {
		tagnumber = 0
	}
	jsonStr, err := dbSelBiosData(ctx, db, tagnumber)
	if err != nil {
		log.Warning("Database lookup failed for: " + requestIP + " (" + requestURL + "): " + err.Error())
		http.Error(w, formatHttpError("Internal server error"), http.StatusInternalServerError)
		return
	}
	io.WriteString(w, jsonStr)
}

type osData struct {
	Tagnumber       int           `json:"tagnumber"`
	OsInstalled     bool          `json:"os_installed"`
	OsName          string        `json:"os_name"`
	OsInstalledTime time.Time     `json:"os_installed_time"`
	TPMversion      string        `json:"tpm_version"`
	BootTime        time.Duration `json:"boot_time"`
}

func dbSelOsData(ctx context.Context, db *sql.DB, tagnumber int) (string, error) {
	var sqlQuery string
	var results []*osData

	if tagnumber <= 0 {
		return "", errors.New("no tagnumber provided")
	}

	sqlQuery = `SELECT locations.tagnumber, client_health.os_installed, client_health.os_name,
	client_health.last_imaged_time, client_health.tpm_version, jobstats.boot_time
	FROM locations
	LEFT JOIN client_health ON locations.tagnumber = client_health.tagnumber
	LEFT JOIN jobstats ON locations.tagnumber = jobstats.tagnumber AND jobstats.time IN (SELECT MAX(time) FROM jobstats GROUP BY tagnumber)
	WHERE locations.time IN (SELECT MAX(time) FROM locations GROUP BY tagnumber)
	AND locations.tagnumber = $1`

	rows, err := db.QueryContext(ctx, sqlQuery, tagnumber)
	if err != nil {
		return "", errors.New("Context error: " + err.Error())
	}
	defer rows.Close()

	for rows.Next() {
		row := &osData{}
		if err := rows.Scan(
			&row.Tagnumber,
			&row.OsInstalled,
			&row.OsName,
			&row.OsInstalledTime,
			&row.TPMversion,
			&row.BootTime,
		); err != nil {
			return "", errors.New("Error scanning rows: " + err.Error())
		}
		results = append(results, row)
	}
	if err := rows.Err(); err != nil {
		return "", errors.New("Query error: " + err.Error())
	}
	if err := ctx.Err(); err != nil {
		return "", errors.New("Context error: " + err.Error())
	}
	if len(results) == 0 {
		return "", errors.New("no results found")
	}

	jsonStr, err := jsonEncode(results)
	if err != nil {
		return "", errors.New("Error encoding results to JSON: " + err.Error())
	}

	return jsonStr, nil
}

func getOSData(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	requestIP, ok := GetRequestIP(r)
	if !ok {
		log.Warning("no IP address stored in context")
		http.Error(w, formatHttpError("Internal server error"), http.StatusInternalServerError)
		return
	}
	requestURL, ok := GetRequestURL(r)
	if !ok {
		log.Warning("no URL stored in context")
		http.Error(w, formatHttpError("Internal server error"), http.StatusInternalServerError)
		return
	}

	tag := strings.TrimSpace(r.URL.Query().Get("tagnumber"))
	if tag == "" {
		log.Warning("No tagnumber provided in request from: " + requestIP + " (" + requestURL + ")")
		http.Error(w, formatHttpError("Bad request"), http.StatusBadRequest)
		return
	}

	var tagnumber, err = strconv.Atoi(tag)
	if err != nil || tagnumber <= 0 {
		tagnumber = 0
	}
	jsonStr, err := dbSelOsData(ctx, db, tagnumber)
	if err != nil {
		log.Warning("Database lookup failed for: " + requestIP + " (" + requestURL + "): " + err.Error())
		http.Error(w, formatHttpError("Internal server error"), http.StatusInternalServerError)
		return
	}
	io.WriteString(w, jsonStr)
}

type activeJobs struct {
	Tagnumber     int    `json:"tagnumber"`
	QueuedJob     string `json:"job_queued"`
	JobActive     bool   `json:"job_active"`
	QueuePosition int    `json:"queue_position"`
}

func dbSelQueuedJobData(ctx context.Context, db *sql.DB, tagnumber int) (string, error) {
	var sqlQuery string
	var results []*activeJobs

	if tagnumber <= 0 {
		return "", errors.New("no tagnumber provided")
	}

	sqlQuery = `SELECT remote.tagnumber, remote.job_queued, remote.job_active, t1.queue_position
	FROM remote
	LEFT JOIN (SELECT tagnumber, ROW_NUMBER() OVER (PARTITION BY tagnumber ORDER BY time DESC) AS queue_position FROM job_queue) AS t1 
		ON remote.tagnumber = t1.tagnumber
	WHERE remote.tagnumber = $1`

	rows, err := db.QueryContext(ctx, sqlQuery, tagnumber)
	if err != nil {
		return "", errors.New("Context error: " + err.Error())
	}
	defer rows.Close()

	for rows.Next() {
		row := &activeJobs{}
		if err := rows.Scan(
			&row.Tagnumber,
			&row.QueuedJob,
			&row.JobActive,
			&row.QueuePosition,
		); err != nil {
			return "", errors.New("Error scanning rows: " + err.Error())
		}
		results = append(results, row)
	}
	if err := rows.Err(); err != nil {
		return "", errors.New("Query error: " + err.Error())
	}
	if err := ctx.Err(); err != nil {
		return "", errors.New("Context error: " + err.Error())
	}
	if len(results) == 0 {
		return "", errors.New("no results found")
	}

	jsonStr, err := jsonEncode(results)
	if err != nil {
		return "", errors.New("Error encoding results to JSON: " + err.Error())
	}

	return jsonStr, nil
}

func getClientQueuedJobs(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	requestIP, ok := GetRequestIP(r)
	if !ok {
		log.Warning("no IP address stored in context")
		http.Error(w, formatHttpError("Internal server error"), http.StatusInternalServerError)
		return
	}
	requestURL, ok := GetRequestURL(r)
	if !ok {
		log.Warning("no URL stored in context")
		http.Error(w, formatHttpError("Internal server error"), http.StatusInternalServerError)
		return
	}

	tag := strings.TrimSpace(r.URL.Query().Get("tagnumber"))
	if tag == "" {
		log.Warning("No tagnumber provided in request from: " + requestIP + " (" + requestURL + ")")
		http.Error(w, formatHttpError("Bad request"), http.StatusBadRequest)
		return
	}

	var tagnumber, err = strconv.Atoi(tag)
	if err != nil || tagnumber <= 0 {
		tagnumber = 0
	}
	jsonStr, err := dbSelQueuedJobData(ctx, db, tagnumber)
	if err != nil {
		log.Warning("Database lookup failed for: " + requestIP + " (" + requestURL + "): " + err.Error())
		http.Error(w, formatHttpError("Internal server error"), http.StatusInternalServerError)
		return
	}
	io.WriteString(w, jsonStr)
}

type availableJobs struct {
	Tagnumber    int  `json:"tagnumber"`
	JobAvailable bool `json:"job_available"`
}

func dbSelAvailableJobs(ctx context.Context, db *sql.DB, tagnumber int) (string, error) {
	var sqlQuery string
	var results []*availableJobs

	if tagnumber <= 0 {
		return "", errors.New("no tagnumber provided")
	}

	sqlQuery = `SELECT 
	remote.tagnumber,
	(CASE 
		WHEN (remote.job_queued IS NULL) THEN TRUE
		ELSE FALSE
	END) AS job_available,
	WHERE remote.tagnumber = $1`

	rows, err := db.QueryContext(ctx, sqlQuery, tagnumber)
	if err != nil {
		return "", errors.New("Context error: " + err.Error())
	}
	defer rows.Close()

	for rows.Next() {
		row := &availableJobs{}
		if err := rows.Scan(
			&row.Tagnumber,
			&row.JobAvailable,
		); err != nil {
			return "", errors.New("Error scanning rows: " + err.Error())
		}
		results = append(results, row)
	}
	if err := rows.Err(); err != nil {
		return "", errors.New("Query error: " + err.Error())
	}
	if err := ctx.Err(); err != nil {
		return "", errors.New("Context error: " + err.Error())
	}
	if len(results) == 0 {
		return "", errors.New("no results found")
	}

	jsonStr, err := jsonEncode(results)
	if err != nil {
		return "", errors.New("Error encoding results to JSON: " + err.Error())
	}

	return jsonStr, nil
}

func getClientAvailableJobs(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	requestIP, ok := GetRequestIP(r)
	if !ok {
		log.Warning("no IP address stored in context")
		http.Error(w, formatHttpError("Internal server error"), http.StatusInternalServerError)
		return
	}
	requestURL, ok := GetRequestURL(r)
	if !ok {
		log.Warning("no URL stored in context")
		http.Error(w, formatHttpError("Internal server error"), http.StatusInternalServerError)
		return
	}

	tag := strings.TrimSpace(r.URL.Query().Get("tagnumber"))
	if tag == "" {
		log.Warning("No tagnumber provided in request from: " + requestIP + " (" + requestURL + ")")
		http.Error(w, formatHttpError("Bad request"), http.StatusBadRequest)
		return
	}

	var tagnumber, err = strconv.Atoi(tag)
	if err != nil || tagnumber <= 0 {
		tagnumber = 0
	}
	jsonStr, err := dbSelAvailableJobs(ctx, db, tagnumber)
	if err != nil {
		log.Warning("Database lookup failed for: " + requestIP + " (" + requestURL + "): " + err.Error())
		http.Error(w, formatHttpError("Internal server error"), http.StatusInternalServerError)
		return
	}
	io.WriteString(w, jsonStr)
}

// Overview section

type JobQueueOverview struct {
	TotalQueuedJobs         int `json:"total_queued_jobs"`
	TotalActiveJobs         int `json:"total_active_jobs"`
	TotalActiveBlockingJobs int `json:"total_active_blocking_jobs"`
}

func dbSelJobQueueOverview(ctx context.Context, db *sql.DB) (string, error) {
	var sqlQuery string
	var results []*JobQueueOverview

	sqlQuery = `SELECT t1.total_queued_jobs, t2.total_active_jobs, t3.total_active_blocking_jobs
	FROM 
	(SELECT COUNT(*) AS total_queued_jobs FROM remote WHERE job_queued IS NOT NULL AND (NOW() - present < INTERVAL '30 SECOND')) AS t1,
	(SELECT COUNT(*) AS total_active_jobs FROM remote WHERE job_active IS NOT NULL AND job_active = TRUE AND (NOW() - present < INTERVAL '30 SECOND')) AS t2,
	(SELECT COUNT(*) AS total_active_blocking_jobs FROM remote WHERE job_active IS NOT NULL AND job_active = TRUE AND job_queued IS NOT NULL AND job_queued IN ('hpEraseAndClone', 'hpCloneOnly', 'generic-erase+clone', 'generic-clone')) AS t3;`

	rows, err := db.QueryContext(ctx, sqlQuery)
	if err != nil {
		return "", errors.New("Context error: " + err.Error())
	}
	defer rows.Close()

	for rows.Next() {
		row := &JobQueueOverview{}
		if err := rows.Scan(
			&row.TotalQueuedJobs,
			&row.TotalActiveJobs,
			&row.TotalActiveBlockingJobs,
		); err != nil {
			return "", errors.New("Error scanning rows: " + err.Error())
		}
		results = append(results, row)
	}
	if err := rows.Err(); err != nil {
		return "", errors.New("Query error: " + err.Error())
	}
	if err := ctx.Err(); err != nil {
		return "", errors.New("Context error: " + err.Error())
	}
	if len(results) == 0 {
		return "", errors.New("no results found")
	}

	jsonStr, err := jsonEncode(results)
	if err != nil {
		return "", errors.New("Error encoding results to JSON: " + err.Error())
	}

	return jsonStr, nil
}

func getJobQueueOverview(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	requestIP, ok := GetRequestIP(r)
	if !ok {
		log.Warning("no IP address stored in context")
		http.Error(w, formatHttpError("Internal server error"), http.StatusInternalServerError)
		return
	}
	requestURL, ok := GetRequestURL(r)
	if !ok {
		log.Warning("no URL stored in context")
		http.Error(w, formatHttpError("Internal server error"), http.StatusInternalServerError)
		return
	}

	tag := strings.TrimSpace(r.URL.Query().Get("tagnumber"))
	if tag == "" {
		log.Warning("No tagnumber provided in request from: " + requestIP + " (" + requestURL + ")")
		http.Error(w, formatHttpError("Bad request"), http.StatusBadRequest)
		return
	}

	var tagnumber, err = strconv.Atoi(tag)
	if err != nil || tagnumber <= 0 {
		tagnumber = 0
	}
	jsonStr, err := dbSelJobQueueOverview(ctx, db)
	if err != nil {
		log.Warning("Database lookup failed for: " + requestIP + " (" + requestURL + "): " + err.Error())
		http.Error(w, formatHttpError("Internal server error"), http.StatusInternalServerError)
		return
	}
	io.WriteString(w, jsonStr)
}
