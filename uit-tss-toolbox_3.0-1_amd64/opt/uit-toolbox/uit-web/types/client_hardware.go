package types

import (
	"fmt"
	"time"
)

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

type ClientHealthCheck struct {
	TransactionUUID   string     `json:"transaction_uuid"`
	Tagnumber         int64      `json:"tagnumber"`
	SystemSerial      *string    `json:"health_system_serial"`
	BIOSVersion       *string    `json:"bios_version"`
	TPMVersion        *string    `json:"health_tpm_version"`
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
	Tagnumber       *int64  `json:"tagnumber"`
	TotalUsageKB    *int64  `json:"memory_usage_kb"`
	TotalCapacityKB *int64  `json:"memory_capacity_kb"`
	Type            *string `json:"type"`
	SpeedMHz        *int64  `json:"speed_mhz"`
}

type MemoryDataDTO struct {
	Tagnumber       int64
	TotalUsageKB    int64
	TotalCapacityKB int64
	Type            string
	SpeedMHz        int64
}

func (m *MemoryDataRequest) ToDTO() (*MemoryDataDTO, error) {
	if m == nil {
		return nil, fmt.Errorf("memory data request is nil")
	}
	if m.Tagnumber == nil || *m.Tagnumber == 0 {
		return nil, fmt.Errorf("tag number is required")
	}
	var usageKB int64
	if m.TotalUsageKB != nil {
		usageKB = *m.TotalUsageKB
	}
	var capacityKB int64
	if m.TotalCapacityKB != nil {
		capacityKB = *m.TotalCapacityKB
	}
	var memType string
	if m.Type != nil {
		memType = *m.Type
	}
	var speedMHz int64
	if m.SpeedMHz != nil {
		speedMHz = *m.SpeedMHz
	}
	return &MemoryDataDTO{
		Tagnumber:       *m.Tagnumber,
		TotalUsageKB:    usageKB,
		TotalCapacityKB: capacityKB,
		Type:            memType,
		SpeedMHz:        speedMHz,
	}, nil
}

type CPUDataRequest struct {
	Tagnumber     *int64   `json:"tagnumber"`
	UsagePercent  *float64 `json:"cpu_current_usage"`
	MHz           *float64 `json:"cpu_current_mhz"`
	MillidegreesC *float64 `json:"cpu_millidegrees_c"`
}

type CPUDataDTO struct {
	Tagnumber     int64
	UsagePercent  float64
	MHz           float64
	MillidegreesC float64
}

func (c *CPUDataRequest) ToDTO() (*CPUDataDTO, error) {
	if c == nil {
		return nil, fmt.Errorf("CPU data request is nil")
	}
	if c.Tagnumber == nil || *c.Tagnumber == 0 {
		return nil, fmt.Errorf("tag number is required")
	}
	var usagePercent float64
	if c.UsagePercent != nil {
		if *c.UsagePercent < 0 || *c.UsagePercent > 100 {
			return nil, fmt.Errorf("%w: CPU usage percent must be between 0 and 100", InvalidFieldError)
		}
		usagePercent = *c.UsagePercent
	}
	var mhz float64
	if c.MHz != nil {
		if *c.MHz <= 0 {
			return nil, fmt.Errorf("%w: CPU MHz must be greater than 0", InvalidFieldError)
		}
		mhz = *c.MHz
	}
	var millidegreesC float64
	if c.MillidegreesC != nil {
		if *c.MillidegreesC <= 0 {
			return nil, fmt.Errorf("%w: CPU temperature must be greater than 0", InvalidFieldError)
		}
		millidegreesC = *c.MillidegreesC
	}
	return &CPUDataDTO{
		Tagnumber:     *c.Tagnumber,
		UsagePercent:  usagePercent,
		MHz:           mhz,
		MillidegreesC: millidegreesC,
	}, nil
}

type NetworkData struct {
	Tagnumber    int64  `json:"tagnumber"`
	NetworkUsage *int64 `json:"network_usage"`
	LinkSpeed    *int64 `json:"link_speed"`
}

type BatteryData struct {
	Tagnumber int64    `json:"tagnumber"`
	Percent   *float64 `json:"battery_charge_pcnt"`
}
