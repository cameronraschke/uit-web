package types

import "time"

type ClientRealtimeView struct {
	TotalMemoryUsage     *int64         `json:"memory_usage"`
	TotalMemoryCapacity  *int64         `json:"memory_capacity"`
	TotalCPUUsagePercent *float64       `json:"cpu_usage"`
	CPUSpeedMHz          *int64         `json:"cpu_speed_mhz"`
	CPUMillidegreesC     *float64       `json:"cpu_millidegrees_c"`
	NetworkUsage         *int64         `json:"network_usage"`
	NetworkLinkSpeed     *int64         `json:"network_link_speed"`
	SystemUptime         *time.Duration `json:"uptime"`
	AppUptime            *time.Duration `json:"app_uptime"`
	JobQueued            *bool          `json:"job_queued"`
	JobActive            *bool          `json:"job_active"`
	JobName              *string        `json:"job_name"`
	BatteryChargePcnt    *float64       `json:"battery_charge_pcnt"`
}
