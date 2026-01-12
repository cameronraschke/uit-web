package database

import "time"

type ClientLookup struct {
	Tagnumber    int64  `json:"tagnumber"`
	SystemSerial string `json:"system_serial"`
}

type HardwareData struct {
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

type BiosData struct {
	Tagnumber   int    `json:"tagnumber"`
	BiosVersion string `json:"bios_version"`
	BiosUpdated bool   `json:"bios_updated"`
	BiosDate    string `json:"bios_date"`
	TpmVersion  string `json:"tpm_version"`
}

type OsData struct {
	Tagnumber       int           `json:"tagnumber"`
	OsInstalled     bool          `json:"os_installed"`
	OsName          string        `json:"os_name"`
	OsInstalledTime time.Time     `json:"os_installed_time"`
	TPMversion      string        `json:"tpm_version"`
	BootTime        time.Duration `json:"boot_time"`
}

type ActiveJobs struct {
	Tagnumber     int    `json:"tagnumber"`
	QueuedJob     string `json:"job_queued"`
	JobActive     bool   `json:"job_active"`
	QueuePosition int    `json:"queue_position"`
}

type AvailableJobs struct {
	Tagnumber    int  `json:"tagnumber"`
	JobAvailable bool `json:"job_available"`
}

type JobQueueOverview struct {
	TotalQueuedJobs         int `json:"total_queued_jobs"`
	TotalActiveJobs         int `json:"total_active_jobs"`
	TotalActiveBlockingJobs int `json:"total_active_blocking_jobs"`
}

type DashboardInventorySummary struct {
	SystemModel          string `json:"system_model"`
	SystemModelCount     int    `json:"system_model_count"`
	TotalCheckedOut      int    `json:"total_checked_out"`
	AvailableForCheckout int    `json:"available_for_checkout"`
}

type AllTags struct {
	Tagnumber int64 `json:"tagnumber"`
}

type InventoryFormAutofill struct {
	Time               *time.Time `json:"last_update_time"`
	Tagnumber          *int       `json:"tagnumber"`
	SystemSerial       *string    `json:"system_serial"`
	Location           *string    `json:"location"`
	Building           *string    `json:"building"`
	Room               *string    `json:"room"`
	SystemManufacturer *string    `json:"system_manufacturer"`
	SystemModel        *string    `json:"system_model"`
	Status             *string    `json:"status"`
	Broken             *bool      `json:"is_broken"`
	DiskRemoved        *bool      `json:"disk_removed"`
	Department         *string    `json:"department_name"`
	PropertyCustodian  *string    `json:"property_custodian"`
	Domain             *string    `json:"ad_domain"`
	Note               *string    `json:"note"`
	AcquiredDate       *time.Time `json:"acquired_date"`
}

type InventoryUpdateFormInput struct {
	Time               *time.Time `json:"time"`
	Tagnumber          *int64     `json:"tagnumber"`
	SystemSerial       *string    `json:"system_serial"`
	Location           *string    `json:"location"`
	Broken             *bool      `json:"is_broken"`
	DiskRemoved        *bool      `json:"disk_removed"`
	Department         *string    `json:"department_name"`
	Domain             *string    `json:"ad_domain"`
	Note               *string    `json:"note"`
	Status             *string    `json:"status"`
	SystemManufacturer *string    `json:"system_manufacturer"`
	SystemModel        *string    `json:"system_model"`
	Building           *string    `json:"building"`
	Room               *string    `json:"room"`
	PropertyCustodian  *string    `json:"property_custodian"`
	AcquiredDate       *time.Time `json:"acquired_date"`
}

type ImageManifest struct {
	Time              *time.Time `json:"time"`
	Tagnumber         *int64     `json:"tagnumber"`
	UUID              *string    `json:"uuid"`
	SHA256Hash        *string    `json:"sha256_hash"`
	FileName          *string    `json:"filename"`
	FilePath          *string    `json:"filepath"`
	ThumbnailFilePath *string    `json:"thumbnail_filepath"`
	FileSize          *int64     `json:"file_size"`
	MimeType          *string    `json:"mime_type"`
	ExifTimestamp     *time.Time `json:"exif_timestamp"`
	ResolutionX       *int       `json:"resolution_x"`
	ResolutionY       *int       `json:"resolution_y"`
	URL               *string    `json:"url"`
	Hidden            *bool      `json:"hidden"`
	PrimaryImage      *bool      `json:"primary_image"`
	Note              *string    `json:"note"`
	FileType          *string    `json:"file_type"`
}

type InventoryTableData struct {
	Tagnumber           *int64     `json:"tagnumber"`
	SystemSerial        *string    `json:"system_serial"`
	Location            *string    `json:"location"`
	LocationFormatted   *string    `json:"location_formatted"`
	SystemManufacturer  *string    `json:"system_manufacturer"`
	SystemModel         *string    `json:"system_model"`
	Department          *string    `json:"department_name"`
	DepartmentFormatted *string    `json:"department_formatted"`
	Domain              *string    `json:"domain"`
	DomainFormatted     *string    `json:"domain_formatted"`
	OsInstalled         *bool      `json:"os_installed"`
	OsName              *string    `json:"os_name"`
	Status              *string    `json:"status"`
	Broken              *bool      `json:"is_broken"`
	Note                *string    `json:"note"`
	LastUpdated         *time.Time `json:"last_updated"`
}

type InventoryFilterOptions struct {
	Tagnumber          *int64  `json:"tagnumber"`
	SystemSerial       *string `json:"system_serial"`
	Location           *string `json:"location"`
	SystemManufacturer *string `json:"system_manufacturer"`
	SystemModel        *string `json:"system_model"`
	Department         *string `json:"department_name"`
	Domain             *string `json:"ad_domain"`
	Status             *string `json:"status"`
	Broken             *bool   `json:"is_broken"`
	HasImages          *bool   `json:"has_images"`
}

type ManufacturersAndModels struct {
	SystemModel                 string `json:"system_model"`
	SystemModelFormatted        string `json:"system_model_formatted"`
	SystemManufacturer          string `json:"system_manufacturer"`
	SystemManufacturerFormatted string `json:"system_manufacturer_formatted"`
}

type Domain struct {
	DomainName          string `json:"domain_name"`
	DomainNameFormatted string `json:"domain_name_formatted"`
	DomainSortOrder     int64  `json:"domain_sort_order"`
}

type Department struct {
	DepartmentName          string `json:"department_name"`
	DepartmentNameFormatted string `json:"department_name_formatted"`
	DepartmentSortOrder     int64  `json:"department_sort_order"`
}

type JobQueueTableRow struct {
	Tagnumber          *int64         `json:"tagnumber"`
	SystemSerial       *string        `json:"system_serial"`
	OSInstalled        *string        `json:"os_installed"`
	OSName             *string        `json:"os_name"`
	KernelUpdated      *bool          `json:"kernel_updated"`
	BIOSUpdated        *bool          `json:"bios_updated"`
	BIOSVersion        *string        `json:"bios_version"`
	SystemManufacturer *string        `json:"system_manufacturer"`
	SystemModel        *string        `json:"system_model"`
	BatteryCharge      *int64         `json:"battery_charge"`
	BatteryStatus      *string        `json:"battery_status"`
	CPUTemp            *float64       `json:"cpu_temp"`
	DiskTemp           *float64       `json:"disk_temp"`
	MaxDiskTemp        *float64       `json:"max_disk_temp"`
	PowerUsage         *float64       `json:"power_usage"`
	NetworkUsage       *float64       `json:"network_usage"`
	ClientStatus       *string        `json:"client_status"`
	IsBroken           *bool          `json:"is_broken"`
	JobQueued          *bool          `json:"job_queued"`
	QueuePosition      *int64         `json:"queue_position"`
	JobActive          *bool          `json:"job_active"`
	JobName            *string        `json:"job_name"`
	JobStatus          *string        `json:"job_status"`
	JobCloneMode       *string        `json:"job_clone_mode"`
	JobEraseMode       *string        `json:"job_erase_mode"`
	LastJobTime        *time.Time     `json:"last_job_time"`
	Location           *string        `json:"location"`
	LastHeard          *time.Time     `json:"last_heard"`
	Uptime             *time.Duration `json:"uptime"`
	Online             *bool          `json:"online"`
}

type ClientBatteryHealth struct {
	Time                *time.Time `json:"time"`
	Tagnumber           *int64     `json:"tagnumber"`
	JobstatsBattery     *string    `json:"jobstatsHealthPcnt"`
	ClientHealthBattery *string    `json:"clientHealthPcnt"`
	BatteryChargeCycles *int64     `json:"chargeCycles"`
}

type ClientReport struct {
	Tagnumber              *int64     `json:"tagnumber"`
	BatteryHealthStdDev    *float64   `json:"battery_health_std_dev"`
	BatteryHealthTimestamp *time.Time `json:"battery_health_timestamp"`
}
