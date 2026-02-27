package types

import "time"

type ClientHardwareView struct {
	Tagnumber               *int64  `json:"tagnumber"`
	SystemSerial            *string `json:"system_serial"`
	EthernetMAC             *string `json:"ethernet_mac"`
	WifiMAC                 *string `json:"wifi_mac"`
	SystemManufacturer      *string `json:"system_manufacturer"`
	SystemModel             *string `json:"system_model"`
	ProductFamily           *string `json:"product_family,omitempty"`
	ProductName             *string `json:"product_name,omitempty"`
	SystemUUID              *string `json:"system_uuid"`
	SystemSKU               *string `json:"system_sku"`
	ChassisType             *string `json:"chassis_type"`
	MotherboardManufacturer *string `json:"motherboard_manufacturer"`
	MotherboardSerial       *string `json:"motherboard_serial"`
	DeviceType              *string `json:"device_type"`
	MemorySpeedMHz          *int64  `json:"memory_speed_mhz"`
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
	Tagnumber     int64  `json:"tagnumber"`
	TotalUsage    *int64 `json:"memory_usage"`
	TotalCapacity int64  `json:"memory_capacity"`
	Type          string `json:"type"`
	SpeedMHz      int64  `json:"speed_mhz"`
}

type CPUData struct {
	Tagnumber     int64    `json:"tagnumber"`
	UsagePercent  *float64 `json:"cpu_usage"`
	MillidegreesC *float64 `json:"cpu_millidegrees_c"`
}

type NetworkData struct {
	Tagnumber    int64  `json:"tagnumber"`
	NetworkUsage *int64 `json:"network_usage"`
	LinkSpeed    *int64 `json:"link_speed"`
}

type DiskData struct {
	Tagnumber           int64    `json:"tagnumber"`
	DiskSerial          *string  `json:"disk_serial"`
	DiskModel           *string  `json:"disk_model"`
	DiskType            *string  `json:"disk_type"`
	DiskSizeGB          *float64 `json:"disk_size_gb"`
	DiskTotalWritesMB   *float64 `json:"disk_total_writes_mb"`
	DiskTotalReadsMB    *float64 `json:"disk_total_reads_mb"`
	DiskPowerOnHours    *int64   `json:"disk_power_on_hours"`
	DiskErrors          *int64   `json:"disk_errors"`
	DiskPowerCycleCount *int64   `json:"disk_power_cycle_count"`
	DiskTempC           *float64 `json:"disk_temp_c"`
	DiskFirmwareVersion *string  `json:"disk_firmware_version"`
}
