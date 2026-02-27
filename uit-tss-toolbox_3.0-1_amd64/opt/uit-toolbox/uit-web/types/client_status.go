package types

import "time"

type ClientStatusView struct {
	Tagnumber         *int64     `json:"tagnumber"`
	ClientStatus      *string    `json:"client_status"`
	BatteryHealth     *string    `json:"battery_health"`
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
