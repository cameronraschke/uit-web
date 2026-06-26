package types

import (
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

type ClientHardwareView struct {
	TransactionUUID           string     `json:"transaction_uuid"`
	Tagnumber                 *int64     `json:"tagnumber"`
	SystemSerial              *string    `json:"system_serial"`
	SystemUUID                *string    `json:"system_uuid"`
	SystemManufacturer        *string    `json:"system_manufacturer"`
	SystemModel               *string    `json:"system_model"`
	SystemSKU                 *string    `json:"system_sku"`
	ProductFamily             *string    `json:"product_family,omitempty"`
	ProductName               *string    `json:"product_name,omitempty"`
	DeviceType                *string    `json:"device_type"`
	ChassisType               *string    `json:"chassis_type"`
	MotherboardSerial         *string    `json:"motherboard_serial"`
	MotherboardManufacturer   *string    `json:"motherboard_manufacturer"`
	CPUManufacturer           *string    `json:"cpu_manufacturer"`
	CPUModel                  *string    `json:"cpu_model"`
	CPUMaxSpeedMhz            *int64     `json:"cpu_max_speed_mhz"`
	CPUCoreCount              *int64     `json:"cpu_core_count"`
	CPUThreadCount            *int64     `json:"cpu_thread_count"`
	EthernetMAC               *string    `json:"ethernet_mac"`
	WiFiMAC                   *string    `json:"wifi_mac"`
	TPMVersion                *string    `json:"tpm_version"`
	DiskModel                 *string    `json:"disk_model"`
	DiskType                  *string    `json:"disk_type"`
	DiskSize                  *int64     `json:"disk_size_kb"`
	DiskSerial                *string    `json:"disk_serial"`
	DiskWritesKB              *int64     `json:"disk_writes_kb"`
	DiskReadsKB               *int64     `json:"disk_reads_kb"`
	DiskPowerOnHours          *int64     `json:"disk_power_on_hours"`
	DiskErrors                *int64     `json:"disk_errors"`
	DiskPowerCycles           *int64     `json:"disk_power_cycles"`
	DiskFirmware              *string    `json:"disk_firmware"`
	BatteryModel              *string    `json:"battery_model"`
	BatterySerial             *string    `json:"battery_serial"`
	BatteryChargeCycles       *int64     `json:"battery_charge_cycles"`
	BatteryCurrentMaxCapacity *float64   `json:"battery_current_max_capacity"`
	BatteryDesignCapacity     *float64   `json:"battery_design_capacity"`
	BatteryManufacturer       *string    `json:"battery_manufacturer"`
	BatteryManufactureDate    *string    `json:"battery_manufacture_date"`
	BiosVersion               *string    `json:"bios_version"`
	BiosReleaseDate           *time.Time `json:"bios_release_date"`
	BiosFirmware              *string    `json:"bios_firmware"`
	MemorySerial              []string   `json:"memory_serial"`
	MemoryCapacityKB          *int64     `json:"memory_capacity_kb"`
	MemorySpeedMHz            *int64     `json:"memory_speed_mhz"`
}

type ClientHealthCheck struct {
	TransactionUUID   string     `json:"transaction_uuid"`
	Tagnumber         int64      `json:"tagnumber"`
	SystemSerial      *string    `json:"health_system_serial"`
	BIOSVersion       *string    `json:"bios_version"`
	BIOSReleaseDate   *time.Time `json:"bios_release_date"`
	TPMVersion        *string    `json:"health_tpm_version"`
	LastHardwareCheck *time.Time `json:"last_hardware_check"`
}

type MemoryDataUpdateRequest struct {
	Tagnumber       *int64  `json:"tagnumber"`
	TotalUsageKB    *int64  `json:"memory_usage_kb"`
	TotalCapacityKB *int64  `json:"memory_capacity_kb"`
	Type            *string `json:"type"`
	SpeedMHz        *int64  `json:"speed_mhz"`
}

type MemoryDataUpdateDTO struct {
	Tagnumber       int64
	TotalUsageKB    int64
	TotalCapacityKB int64
	Type            string
	SpeedMHz        int64
}

func (m *MemoryDataUpdateRequest) ToDTO() (*MemoryDataUpdateDTO, error) {
	if m == nil {
		return nil, fmt.Errorf("memory data request is nil")
	}
	if err := IsTagnumberInt64Valid(m.Tagnumber); err != nil {
		return nil, fmt.Errorf("%w for '%s': %v", InvalidFieldError, "tagnumber", err)
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
	return &MemoryDataUpdateDTO{
		Tagnumber:       *m.Tagnumber,
		TotalUsageKB:    usageKB,
		TotalCapacityKB: capacityKB,
		Type:            memType,
		SpeedMHz:        speedMHz,
	}, nil
}

type CPUDataUpdateRequest struct {
	Tagnumber     *int64   `json:"tagnumber"`
	UsagePercent  *float64 `json:"cpu_current_usage"`
	MHz           *float64 `json:"cpu_current_mhz"`
	MillidegreesC *float64 `json:"cpu_millidegrees_c"`
}

type CPUDataUpdateDTO struct {
	Tagnumber     int64
	UsagePercent  float64
	MHz           float64
	MillidegreesC float64
}

func (c *CPUDataUpdateRequest) ToDTO() (*CPUDataUpdateDTO, error) {
	if c == nil {
		return nil, fmt.Errorf("CPU data request is nil")
	}
	if err := IsTagnumberInt64Valid(c.Tagnumber); err != nil {
		return nil, fmt.Errorf("%w for '%s': %v", InvalidFieldError, "tagnumber", err)
	}
	var usagePercent float64
	if c.UsagePercent != nil {
		if *c.UsagePercent < 0 || *c.UsagePercent > 110 {
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
	return &CPUDataUpdateDTO{
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

type BatteryDataRequest struct {
	TransactionUUID           *string    `json:"transaction_uuid"`
	UpdatedFromWindows        *bool      `json:"updated_from_windows"`
	TimeStamp                 *time.Time `json:"timestamp"`
	Tagnumber                 *int64     `json:"tagnumber"`
	SystemSerial              *string    `json:"system_serial"`
	BatteryChargeCycles       *int64     `json:"battery_charge_cycles"`
	BatteryChargePcnt         *float64   `json:"battery_charge_pcnt"`
	BatteryCurrentMaxCapacity *float64   `json:"battery_current_max_capacity"`
	BatteryDesignCapacity     *float64   `json:"battery_design_capacity"`
	BatteryManufactureDate    *string    `json:"battery_manufacture_date"`
	BatteryManufacturer       *string    `json:"battery_manufacturer"`
	BatteryModel              *string    `json:"battery_model"`
	BatterySerial             *string    `json:"battery_serial"`
}

type BatteryDataDTO struct {
	TransactionUUID           uuid.UUID
	UpdatedFromWindows        bool
	TimeStamp                 time.Time
	Tagnumber                 int64
	SystemSerial              string
	BatteryChargeCycles       *int64
	BatteryChargePcnt         *float64
	BatteryCurrentMaxCapacity *float64
	BatteryDesignCapacity     *float64
	BatteryManufactureDate    *string
	BatteryManufacturer       *string
	BatteryModel              *string
	BatterySerial             *string
}

func (b *BatteryDataRequest) ToDTO() (dto *BatteryDataDTO, err error) {
	if b == nil {
		return nil, fmt.Errorf("battery data request is nil")
	}

	// transactionUUID is optional, generate a new one if not provided
	var transactionUUID uuid.UUID
	if b.TransactionUUID == nil {
		transactionUUID, err = uuid.NewV7()
		if err != nil {
			return nil, fmt.Errorf("%w: failed to generate transaction UUID", InvalidFieldError)
		}
	} else {
		transactionUUID, err = uuid.Parse(*b.TransactionUUID)
		if err != nil {
			return nil, fmt.Errorf("%w: invalid transaction UUID", InvalidFieldError)
		}
	}

	// updatedFromWindows is optional, default to false if not provided
	updatedFromWindows := false
	if b.UpdatedFromWindows != nil && *b.UpdatedFromWindows {
		updatedFromWindows = *b.UpdatedFromWindows
	}

	// timestamp is optional, use current UTC time if not provided
	var timestamp time.Time
	if b.TimeStamp == nil {
		timestamp = time.Now().UTC()
	} else {
		if b.TimeStamp.IsZero() {
			return nil, fmt.Errorf("%w: timestamp cannot be zero", InvalidFieldError)
		}
		timestamp = b.TimeStamp.UTC()
	}

	// tagnumber
	if err := IsTagnumberInt64Valid(b.Tagnumber); err != nil {
		return nil, fmt.Errorf("%w for '%s': %v", InvalidFieldError, "tagnumber", err)
	}

	// systemSerial is optional, but if provided, it must not be empty
	if b.SystemSerial != nil && len(strings.TrimSpace(*b.SystemSerial)) > 0 {
		if err := IsSystemSerialValid(b.SystemSerial); err != nil {
			return nil, fmt.Errorf("%w for '%s': %v", InvalidFieldError, "system_serial", err)
		}
	}

	dto = &BatteryDataDTO{
		TransactionUUID:           transactionUUID,
		UpdatedFromWindows:        updatedFromWindows,
		TimeStamp:                 timestamp,
		Tagnumber:                 *b.Tagnumber,
		SystemSerial:              *b.SystemSerial,
		BatteryChargeCycles:       b.BatteryChargeCycles,
		BatteryChargePcnt:         b.BatteryChargePcnt,
		BatteryCurrentMaxCapacity: b.BatteryCurrentMaxCapacity,
		BatteryDesignCapacity:     b.BatteryDesignCapacity,
		BatteryManufactureDate:    b.BatteryManufactureDate,
		BatteryManufacturer:       b.BatteryManufacturer,
		BatteryModel:              b.BatteryModel,
		BatterySerial:             b.BatterySerial,
	}

	return dto, nil
}
