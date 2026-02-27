package types

import "time"

type InventoryAdvSearchOptions struct {
	Tagnumber          *int64  `json:"tagnumber"`
	SystemSerial       *string `json:"system_serial"`
	Location           *string `json:"location"`
	SystemManufacturer *string `json:"system_manufacturer"`
	SystemModel        *string `json:"system_model"`
	DeviceType         *string `json:"device_type"`
	Department         *string `json:"department_name"`
	Domain             *string `json:"ad_domain"`
	Status             *string `json:"status"`
	Broken             *bool   `json:"is_broken"`
	HasImages          *bool   `json:"has_images"`
}

type JobQueueTableRow struct {
	Tagnumber            *int64         `json:"tagnumber"`
	SystemSerial         *string        `json:"system_serial"`
	SystemManufacturer   *string        `json:"system_manufacturer"`
	SystemModel          *string        `json:"system_model"`
	Location             *string        `json:"location"`
	Department           *string        `json:"department_name"`
	ClientStatus         *string        `json:"client_status"`
	IsBroken             *bool          `json:"is_broken"`
	DiskRemoved          *bool          `json:"disk_removed"`
	TempWarning          *bool          `json:"temp_warning"`
	BatteryHealthWarning *bool          `json:"battery_health_warning"`
	CheckoutBool         *bool          `json:"checkout_bool"`
	KernelUpdated        *bool          `json:"kernel_updated"`
	LastHeard            *time.Time     `json:"last_heard"`
	SystemUptime         *time.Duration `json:"system_uptime"`
	Online               *bool          `json:"online"`
	JobActive            *bool          `json:"job_active"`
	JobQueued            *bool          `json:"job_queued"`
	QueuePosition        *int64         `json:"queue_position"`
	JobName              *string        `json:"job_name"`
	JobNameReadable      *string        `json:"job_name_readable"`
	JobCloneMode         *string        `json:"job_clone_mode"`
	JobEraseMode         *string        `json:"job_erase_mode"`
	JobStatus            *string        `json:"job_status"`
	LastJobTime          *time.Time     `json:"last_job_time"`
	OSInstalled          *string        `json:"os_installed"`
	OSName               *string        `json:"os_name"`
	OSUpdated            *bool          `json:"os_updated"`
	DomainJoined         *bool          `json:"domain_joined"`
	DomainName           *string        `json:"ad_domain"`
	DomainNameFormatted  *string        `json:"ad_domain_formatted"`
	BIOSUpdated          *bool          `json:"bios_updated"`
	BIOSVersion          *string        `json:"bios_version"`
	CPUUsage             *float64       `json:"cpu_usage"`
	CPUTemp              *float64       `json:"cpu_temp"`
	CPUTempWarning       *bool          `json:"cpu_temp_warning"`
	MemoryUsage          *float64       `json:"memory_usage"`
	MemoryCapacity       *float64       `json:"memory_capacity"`
	DiskUsage            *float64       `json:"disk_usage"`
	DiskTemp             *float64       `json:"disk_temp"`
	DiskType             *string        `json:"disk_type"`
	DiskSize             *float64       `json:"disk_size"`
	MaxDiskTemp          *float64       `json:"max_disk_temp"`
	DiskTempWarning      *bool          `json:"disk_temp_warning"`
	NetworkLinkStatus    *string        `json:"network_link_status"`
	NetworkLinkSpeed     *float64       `json:"network_link_speed"`
	NetworkUsage         *float64       `json:"network_usage"`
	BatteryCharge        *int64         `json:"battery_charge"`
	BatteryStatus        *string        `json:"battery_status"`
	BatteryHealth        *float64       `json:"battery_health"`
	PluggedIn            *bool          `json:"plugged_in"`
	PowerUsage           *float64       `json:"power_usage"`
}

type Note struct {
	NoteType *string `json:"note_type"`
	Content  *string `json:"note"`
}
