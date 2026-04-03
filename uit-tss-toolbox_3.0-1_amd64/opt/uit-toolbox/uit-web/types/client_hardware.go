package types

import "time"

type ClientHardwareView struct {
	TransactionUUID           string   `json:"transaction_uuid"`
	Tagnumber                 *int64   `json:"tagnumber"`
	SystemSerial              *string  `json:"system_serial"`
	SystemUUID                *string  `json:"system_uuid"`
	SystemManufacturer        *string  `json:"system_manufacturer"`
	SystemModel               *string  `json:"system_model"`
	SystemSKU                 *string  `json:"system_sku"`
	ProductFamily             *string  `json:"product_family,omitempty"`
	ProductName               *string  `json:"product_name,omitempty"`
	DeviceType                *string  `json:"device_type"`
	ChassisType               *string  `json:"chassis_type"`
	MotherboardSerial         *string  `json:"motherboard_serial"`
	MotherboardManufacturer   *string  `json:"motherboard_manufacturer"`
	CPUManufacturer           *string  `json:"cpu_manufacturer"`
	CPUModel                  *string  `json:"cpu_model"`
	CPUMaxSpeedMhz            *int64   `json:"cpu_max_speed_mhz"`
	CPUCoreCount              *int64   `json:"cpu_core_count"`
	CPUThreadCount            *int64   `json:"cpu_thread_count"`
	EthernetMAC               *string  `json:"ethernet_mac"`
	WiFiMAC                   *string  `json:"wifi_mac"`
	DiskModel                 *string  `json:"disk_model"`
	DiskType                  *string  `json:"disk_type"`
	DiskSize                  *int64   `json:"disk_size_kb"`
	DiskSerial                *string  `json:"disk_serial"`
	DiskWritesKB              *int64   `json:"disk_writes_kb"`
	DiskReadsKB               *int64   `json:"disk_reads_kb"`
	DiskPowerOnHours          *int64   `json:"disk_power_on_hours"`
	DiskErrors                *int64   `json:"disk_errors"`
	DiskPowerCycles           *int64   `json:"disk_power_cycles"`
	DiskFirmware              *string  `json:"disk_firmware"`
	BatteryModel              *string  `json:"battery_model"`
	BatterySerial             *string  `json:"battery_serial"`
	BatteryChargeCycles       *int64   `json:"battery_charge_cycles"`
	BatteryCurrentMaxCapacity *float64 `json:"battery_current_max_capacity"`
	BatteryDesignCapacity     *float64 `json:"battery_design_capacity"`
	BatteryManufacturer       *string  `json:"battery_manufacturer"`
	BatteryManufactureDate    *string  `json:"battery_manufacture_date"`
	BiosVersion               *string  `json:"bios_version"`
	BiosReleaseDate           *string  `json:"bios_release_date"`
	BiosFirmware              *string  `json:"bios_firmware"`
	MemorySerial              *string  `json:"memory_serial"`
	MemoryCapacityKB          *int64   `json:"memory_capacity_kb"`
	MemorySpeedMHz            *int64   `json:"memory_speed_mhz"`
}

type ClientHardwareCheck struct {
	Tagnumber         int64      `json:"tagnumber"`
	LastHardwareCheck *time.Time `json:"last_hardware_check"`
}

type DeviceType struct {
	DeviceType          string `json:"device_type"`
	DeviceTypeFormatted string `json:"device_type_formatted"`
	DeviceMetaCategory  string `json:"device_meta_category"`
	DeviceTypeCount     int64  `json:"device_type_count"`
	SortOrder           int64  `json:"sort_order"`
}

type MemoryDataRequest struct {
	Tagnumber       int64  `json:"tagnumber"`
	TotalUsageKB    *int64 `json:"memory_usage_kb"`
	TotalCapacityKB *int64 `json:"memory_capacity_kb"`
	Type            string `json:"type"`
	SpeedMHz        int64  `json:"speed_mhz"`
}

type CPUData struct {
	Tagnumber     int64    `json:"tagnumber"`
	UsagePercent  *float64 `json:"cpu_usage"`
	MHz           *float64 `json:"cpu_mhz"`
	MillidegreesC *float64 `json:"cpu_millidegrees_c"`
}

type NetworkData struct {
	Tagnumber    int64  `json:"tagnumber"`
	NetworkUsage *int64 `json:"network_usage"`
	LinkSpeed    *int64 `json:"link_speed"`
}
