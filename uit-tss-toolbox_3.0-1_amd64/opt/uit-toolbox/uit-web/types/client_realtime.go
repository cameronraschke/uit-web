package types

import (
	"time"

	"github.com/google/uuid"
)

const (
	MaxLiveImageBytes = 512 << 20 // 512 MB
	// If LastHeardTimeout is too low, then the job queue
	// gets messed up because the clients temporarily
	// stop sending last_heard while they get ready for a job
	LastHeardTimeout = 20 * time.Second
)

type JobQueueRealtimeData struct {
	ClientUUID     uuid.UUID
	Tagnumber      int64
	SerialNumber   string
	LastHeard      *time.Time
	LastHeardInDB  *bool
	SystemUptime   time.Duration
	AppUptime      time.Duration
	LiveImageBytes []byte
}

type JobQueueTableRowView struct {
	ClientUUID             *uuid.UUID    `json:"client_uuid"`
	Tagnumber              *int64        `json:"tagnumber"`
	SystemSerial           *string       `json:"system_serial"`
	SystemManufacturer     *string       `json:"system_manufacturer"`
	SystemModel            *string       `json:"system_model"`
	Location               *string       `json:"location"`
	Department             *string       `json:"department_name"`
	ClientStatus           *string       `json:"client_status"`
	IsBroken               *bool         `json:"is_broken"`
	DiskRemoved            *bool         `json:"disk_removed"`
	TempWarning            *bool         `json:"temp_warning"`
	CheckoutBool           *bool         `json:"checkout_bool"`
	KernelUpdated          *bool         `json:"kernel_updated"`
	LastHeard              *time.Time    `json:"last_heard"`
	SystemUptime           time.Duration `json:"system_uptime"`     // seconds
	AppUptime              time.Duration `json:"client_app_uptime"` // seconds
	Online                 *bool         `json:"online"`
	JobActive              *bool         `json:"job_active"`
	JobQueued              *bool         `json:"job_queued"`
	JobQueuedAt            *time.Time    `json:"job_queued_at"`
	QueuePosition          *int64        `json:"job_queue_position"`
	JobName                *string       `json:"job_name"`
	JobNameReadable        *string       `json:"job_name_readable"`
	JobCloneMode           *string       `json:"job_clone_mode"`
	JobEraseMode           *string       `json:"job_erase_mode"`
	JobStatus              *string       `json:"job_status"`
	LastJobTime            *time.Time    `json:"last_job_time"`
	OSInstalled            *bool         `json:"os_installed"`
	OSName                 *string       `json:"os_name"`
	LatestImageInstalled   *bool         `json:"latest_image_installed"`
	DomainJoined           *bool         `json:"domain_joined"`
	DomainName             *string       `json:"ad_domain"`
	DomainNameFormatted    *string       `json:"ad_domain_formatted"`
	BIOSUpdated            *bool         `json:"bios_updated"`
	BIOSVersion            *string       `json:"bios_version"`
	CPUUsage               *float64      `json:"cpu_current_usage"`
	CPUMHz                 *float64      `json:"cpu_mhz"`
	CPUTemp                *float64      `json:"cpu_temp"`
	CPUTempWarning         *bool         `json:"cpu_temp_warning"`
	MemoryUsageKB          *int64        `json:"memory_usage_kb"`
	MemoryCapacityKB       *int64        `json:"memory_capacity_kb"`
	DiskUsage              *float64      `json:"disk_usage"`
	DiskTemp               *float64      `json:"disk_temp"`
	DiskType               *string       `json:"disk_type"`
	DiskSize               *float64      `json:"disk_size_kb"`
	MaxDiskTemp            *float64      `json:"max_disk_temp"`
	DiskTempWarning        *bool         `json:"disk_temp_warning"`
	NetworkLinkStatus      *string       `json:"network_link_status"`
	NetworkLinkSpeed       *float64      `json:"network_link_speed"`
	NetworkUsage           *float64      `json:"network_usage"`
	BatteryCharge          *int64        `json:"battery_charge_pcnt"`
	BatteryStatus          *string       `json:"battery_status"`
	BatteryHealthDeviation *float64      `json:"battery_health_deviation"`
	BatteryHealthPcnt      *float64      `json:"battery_health_pcnt"`
	PluggedIn              *bool         `json:"plugged_in"`
	PowerUsage             *float64      `json:"power_usage"`
}
