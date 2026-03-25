package types

import "time"

const MaxLiveImageBytes = 512 << 20

type ClientRealtimeView struct {
	TotalMemoryUsageKB    *int64         `json:"memory_usage_kb"`
	TotalMemoryCapacityKB *int64         `json:"memory_capacity_kb"`
	TotalCPUUsagePercent  *float64       `json:"cpu_usage"`
	CPUSpeedMHz           *int64         `json:"cpu_speed_mhz"`
	CPUMillidegreesC      *float64       `json:"cpu_millidegrees_c"`
	NetworkUsage          *int64         `json:"network_usage"`
	NetworkLinkSpeed      *int64         `json:"network_link_speed"`
	SystemUptime          *time.Duration `json:"uptime"`
	AppUptime             *time.Duration `json:"client_app_uptime"`
	JobQueued             *bool          `json:"job_queued"`
	JobActive             *bool          `json:"job_active"`
	JobName               *string        `json:"job_name"`
	BatteryChargePcnt     *float64       `json:"battery_charge_pcnt"`
}
