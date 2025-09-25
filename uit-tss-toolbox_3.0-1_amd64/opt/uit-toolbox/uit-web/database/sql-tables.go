package database

type RemoteTable struct {
	Tagnumber         *int    `json:"tagnumber"`
	JobQueued         *string `json:"job_queued"`
	JobQueuedPosition *int    `json:"job_queued_position"`
	JobActive         *bool   `json:"job_active"`
	CloneMode         *string `json:"clone_mode"`
	EraseMode         *string `json:"erase_mode"`
	LastJobTime       *string `json:"last_job_time"`
	Present           *string `json:"present"`
	PresentBool       *bool   `json:"present_bool"`
	Status            *string `json:"status"`
	KernelUpdated     *bool   `json:"kernel_updated"`
	BatteryCharge     *int    `json:"battery_charge"`
	BatteryStatus     *string `json:"battery_status"`
	Uptime            *int    `json:"uptime"`
	CpuTemp           *int    `json:"cpu_temp"`
	DiskTemp          *int    `json:"disk_temp"`
	MaxDiskTemp       *int    `json:"max_disk_temp"`
	WattsNow          *int    `json:"watts_now"`
	NetworkSpeed      *int    `json:"network_speed"`
}
