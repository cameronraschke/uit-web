package types

import (
	"time"

	"github.com/google/uuid"
)

type ClientStatusView struct {
	Tagnumber         *int64     `json:"tagnumber"`
	ClientStatus      *string    `json:"client_status"`
	BatteryHealthPcnt *string    `json:"battery_health_pcnt"`
	LastHardwareCheck *time.Time `json:"last_hardware_check"`
}

type ClientUptime struct {
	Tagnumber       int64         `json:"tagnumber"`
	ClientAppUptime time.Duration `json:"client_app_uptime"`
	SystemUptime    time.Duration `json:"system_uptime"`
}

type ClientStatus struct {
	Status          string `json:"status"`
	StatusFormatted string `json:"status_formatted"`
	SortOrder       int64  `json:"status_sort_order"`
}

type ActiveJobs struct {
	Tagnumber     int64  `json:"tagnumber"`
	QueuedJob     string `json:"job_queued"`
	JobActive     *bool  `json:"job_active"`
	QueuePosition int64  `json:"queue_position"`
}

type AvailableJobs struct {
	Tagnumber    int64 `json:"tagnumber"`
	JobAvailable *bool `json:"job_available"`
}

type ClientBatteryHealth struct {
	Time                time.Time `json:"time"`
	Tagnumber           int64     `json:"tagnumber"`
	JobstatsBattery     string    `json:"jobstatsHealthPcnt"`
	ClientHealthBattery string    `json:"clientHealthPcnt"`
	BatteryChargeCycles int64     `json:"chargeCycles"`
}

type ClientHealth struct {
	Time               *time.Time `json:"time"`
	Tagnumber          int64      `json:"tagnumber"`
	SystemSerial       *string    `json:"system_serial"`
	TPMVersion         *string    `json:"tpm_version"`
	BIOSUpdated        *bool      `json:"bios_updated"`
	OSName             *string    `json:"os_name"`
	OSInstalled        *bool      `json:"os_installed"`
	DiskHealth         *float64   `json:"disk_health"`
	BatteryHealthPcnt  *float64   `json:"battery_health_pcnt"`
	AvgEraseTime       *float64   `json:"avg_erase_time"`
	AvgCloneTime       *float64   `json:"avg_clone_time"`
	LastCloneJobTime   *time.Time `json:"last_clone_job_time"`
	LastEraseJobTime   *time.Time `json:"last_erase_job_time"`
	TotalJobsCompleted *int64     `json:"total_jobs_completed"`
	LastHardwareCheck  *time.Time `json:"last_hardware_check"`
	TransactionUUID    *uuid.UUID `json:"transaction_uuid"`
}
