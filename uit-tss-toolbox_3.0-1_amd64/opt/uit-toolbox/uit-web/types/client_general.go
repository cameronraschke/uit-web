package types

import "time"

type ClientInfoResponse struct {
	Tagnumber                 *int64                 `json:"tagnumber"`
	SystemSerial              *string                `json:"system_serial"`
	ClientUUID                *string                `json:"client_uuid"`
	LocationEntryTime         *time.Time             `json:"location_entry_time"`
	Location                  *string                `json:"location"`
	Building                  *string                `json:"building"`
	Room                      *string                `json:"room"`
	DepartmentName            *string                `json:"department_name"`
	PropertyCustodian         *string                `json:"property_custodian"`
	AcquiredDate              *time.Time             `json:"acquired_date"`
	RetiredDate               *time.Time             `json:"retired_date"`
	ClientStatus              *string                `json:"client_status"`
	IsBroken                  *bool                  `json:"is_broken"`
	DiskRemoved               *bool                  `json:"disk_removed"`
	ClientNote                *string                `json:"client_note"`
	LocationLog               *[]LocationLogResponse `json:"location_log"`
	JobStartTime                   *time.Time             `json:"job_time"`
	CloneCompleted            *bool                  `json:"clone_completed"`
	CloneJobDuration          *float64               `json:"clone_job_duration"`
	CloneImageName            *string                `json:"clone_image_name"`
	EraseCompleted            *bool                  `json:"erase_completed"`
	EraseJobDuration          *float64               `json:"erase_job_duration"`
	EraseMode                 *string                `json:"erase_mode"`
	JobLog                    *[]JobLogResponse      `json:"job_log"`
	IsCheckedOut              *bool                  `json:"is_checked_out"`
	CheckoutDate              *time.Time             `json:"checkout_date"`
	ReturnDate                *time.Time             `json:"return_date"`
	CustomerName              *string                `json:"customer_name"`
	CheckoutLog               *[]CheckoutLogResponse `json:"checkout_log"`
	FileCount                 *int64                 `json:"file_count"`
	ClientImages              *[]ImageManifestView   `json:"client_images"`
	LastOSEntryTime           *time.Time             `json:"last_os_entry"`
	OSInstalled               *bool                  `json:"os_installed"`
	OSName                    *string                `json:"os_name"`
	OSVersion                 *string                `json:"os_version"`
	ComputerName              *string                `json:"computer_name"`
	OUName                    *string                `json:"ou_name"`
	ADAdminUsers              *[]string              `json:"ad_admin_users"`
	IsIntuneJoined            *bool                  `json:"is_intune_joined"`
	IsBitlockerEnabled        *bool                  `json:"is_bitlocker_enabled"`
	LastHardwareCheck         *time.Time             `json:"last_hardware_check"`
	DiskHealthPcnt            *string                `json:"disk_health_pcnt"`
	BatteryHealthPcnt         *string                `json:"battery_health_pcnt"`
	DeviceType                *string                `json:"device_type"`
	BIOSVersion               *string                `json:"bios_version"`
	BIOSReleaseDate           *string                `json:"bios_release_date"`
	EthernetMAC               *string                `json:"ethernet_mac"`
	WiFiMAC                   *string                `json:"wifi_mac"`
	TPMVersion                *string                `json:"tpm_version"`
	DiskModel                 *string                `json:"disk_model"`
	DiskType                  *string                `json:"disk_type"`
	DiskSizeKB                *int64                 `json:"disk_size_kb"`
	DiskSerial                *string                `json:"disk_serial"`
	DiskWritesKB              *int64                 `json:"disk_writes_kb"`
	DiskReadsKB               *int64                 `json:"disk_reads_kb"`
	DiskPowerOnHours          *int64                 `json:"disk_power_on_hours"`
	DiskErrors                *int64                 `json:"disk_errors"`
	DiskPowerCycles           *int64                 `json:"disk_power_cycle_count"`
	DiskFirmware              *string                `json:"disk_firmware"`
	BatteryManufacturer       *string                `json:"battery_manufacturer"`
	BatteryModel              *string                `json:"battery_model"`
	BatterySerial             *string                `json:"battery_serial"`
	BatteryManufactureDate    *string                `json:"battery_manufacture_date"`
	BatteryDesignCapacity     *float64               `json:"battery_design_capacity"`
	BatteryCurrentMaxCapacity *float64               `json:"battery_current_max_capacity"`
	BatteryChargeCycles       *int64                 `json:"battery_charge_cycles"`
	MemorySerial              *string                `json:"memory_serial"`
	MemoryCapacityKB          *int64                 `json:"memory_capacity_kb"`
	MemorySpeedMHz            *int64                 `json:"memory_speed_mhz"`
	SystemManufacturer        *string                `json:"system_manufacturer"`
	SystemModel               *string                `json:"system_model"`
	SystemSKU                 *string                `json:"system_sku"`
	CPUManufacturer           *string                `json:"cpu_manufacturer"`
	CPUModel                  *string                `json:"cpu_model"`
	CPUMaxSpeedMhz            *int64                 `json:"cpu_max_speed_mhz"`
	CPUCoreCount              *int64                 `json:"cpu_core_count"`
	CPUThreadCount            *int64                 `json:"cpu_thread_count"`
}

type CheckoutLogResponse struct {
	TransactionUUID *string    `json:"transaction_uuid"`
	Tagnumber       *int64     `json:"tagnumber"`
	IsCheckedOut    *bool      `json:"is_checked_out"`
	CheckoutDate    *time.Time `json:"checkout_date"`
	ReturnDate      *time.Time `json:"return_date"`
	CustomerName    *string    `json:"customer_name"`
}

type LocationLogResponse struct {
	TransactionUUID *string    `json:"transaction_uuid"`
	Time            *time.Time `json:"time"`
	Location        *string    `json:"location"`
}

type JobLogResponse struct {
	TransactionUUID  *string    `json:"transaction_uuid"`
	JobTime          *time.Time `json:"job_time"`
	CloneCompleted   *bool      `json:"clone_completed"`
	CloneJobDuration *float64   `json:"clone_job_duration"`
	CloneImageName   *string    `json:"clone_image_name"`
	EraseCompleted   *bool      `json:"erase_completed"`
	EraseJobDuration *float64   `json:"erase_job_duration"`
	EraseMode        *string    `json:"erase_mode"`
}
