package types

import (
	"fmt"
	"strings"
	"time"
)

type WindowsUpdateRequest struct {
	Tagnumber                 *string  `json:"tagnumber"` // Converted later
	SystemSerial              *string  `json:"system_serial"`
	SystemManufacturer        *string  `json:"system_manufacturer"`
	SystemModel               *string  `json:"system_model"`
	SystemSKU                 *string  `json:"system_sku"`
	ChassisType               *string  `json:"chassis_type"`
	BIOSVersion               *string  `json:"bios_version"`
	BIOSReleaseDate           *string  `json:"bios_release_date"` // Converted later
	TPMVersion                *string  `json:"tpm_version"`
	OSInstalledAt             *string  `json:"os_installed_at"` // Converted later
	OSVendor                  *string  `json:"os_vendor"`
	OSPlatform                *string  `json:"os_platform"`
	OSArchitecture            *string  `json:"os_architecture"`
	OSName                    *string  `json:"os_name"`
	OSVersion                 *string  `json:"os_version"`
	WindowsDisplayVersion     *string  `json:"windows_display_version"`
	WindowsBuildNumber        *int64   `json:"windows_build_number"`
	WindowsUBR                *int64   `json:"windows_ubr"`
	WindowsBitlockerEnabled   *bool    `json:"windows_bitlocker_enabled"`
	ADDomain                  *string  `json:"ad_domain"`
	ADDomainUser              *string  `json:"ad_domain_user"`
	MemoryCapacityKB          *int64   `json:"memory_capacity_kb"`
	MemorySpeedMHz            *int64   `json:"memory_speed_mhz"`
	CPUModel                  *string  `json:"cpu_model"`
	CPUCoreCount              *int64   `json:"cpu_core_count"`
	CPUThreadCount            *int64   `json:"cpu_thread_count"`
	DiskModel                 *string  `json:"disk_model"`
	DiskType                  *string  `json:"disk_type"`
	DiskSizeKB                *int64   `json:"disk_size_kb"`
	DiskFreeSpaceKB           *int64   `json:"disk_free_space_kb"`
	EthernetMACAddr           *string  `json:"ethernet_mac_addr"`
	WifiMACAddr               *string  `json:"wifi_mac_addr"`
	BatteryManufacturer       *string  `json:"battery_manufacturer"`
	BatterySerial             *string  `json:"battery_serial"`
	BatteryCurrentMaxCapacity *int64   `json:"battery_current_max_capacity"`
	BatteryDesignCapacity     *int64   `json:"battery_design_capacity"`
	BatteryHealthPcnt         *float64 `json:"battery_health_pct"`
	BatteryChargeCycleCount   *int64   `json:"battery_charge_cycle_count"`
	UpdatedFromWindows        *bool    `json:"updated_from_windows"`
}

type WindowsUpdateDTO struct {
	Tagnumber                 int64
	SystemSerial              string
	SystemManufacturer        *string
	SystemModel               *string
	SystemSKU                 *string
	ChassisType               *string
	BIOSVersion               *string
	BIOSReleaseDate           *time.Time
	TPMVersion                *string
	OSInstalledAt             *time.Time
	OSVendor                  *string
	OSPlatform                *string
	OSArchitecture            *string
	OSName                    *string
	OSVersion                 *string
	WindowsDisplayVersion     *string
	WindowsBuildNumber        *int64
	WindowsUBR                *int64
	WindowsBitlockerEnabled   *bool
	ADDomain                  *string
	ADDomainUser              *string
	MemoryCapacityKB          *int64
	MemorySpeedMHz            *int64
	CPUModel                  *string
	CPUCoreCount              *int64
	CPUThreadCount            *int64
	DiskModel                 *string
	DiskType                  *string
	DiskSizeKB                *int64
	DiskFreeSpaceKB           *int64
	EthernetMACAddr           *string
	WifiMACAddr               *string
	BatteryManufacturer       *string
	BatterySerial             *string
	BatteryCurrentMaxCapacity *int64
	BatteryDesignCapacity     *int64
	BatteryHealthPcnt         *float64
	BatteryChargeCycleCount   *int64
	UpdatedFromWindows        bool
}

type WindowsUpdateResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
}

func NewWindowsUpdateDTO(request WindowsUpdateRequest) (*WindowsUpdateDTO, error) {
	if request.Tagnumber == nil || strings.TrimSpace(*request.Tagnumber) == "" {
		return nil, fmt.Errorf("tagnumber is required")
	}

	convertedTag, err := ConvertAndVerifyTagnumber(*request.Tagnumber)
	if err != nil {
		return nil, fmt.Errorf("invalid tagnumber: %w", err)
	}

	if request.SystemSerial == nil || strings.TrimSpace(*request.SystemSerial) == "" {
		return nil, fmt.Errorf("system_serial is required")
	}

	if request.OSInstalledAt == nil || strings.TrimSpace(*request.OSInstalledAt) == "" {
		return nil, fmt.Errorf("os_installed_at is required")
	}
	convertedTime, err := time.Parse(time.RFC3339, *request.OSInstalledAt)
	if err != nil {
		return nil, fmt.Errorf("invalid os_installed_at format: %w", err)
	}

	if request.BIOSReleaseDate == nil || strings.TrimSpace(*request.BIOSReleaseDate) == "" {
		return nil, fmt.Errorf("bios_release_date is required")
	}
	convertedBIOSReleaseDate, err := time.Parse(time.RFC3339, *request.BIOSReleaseDate)
	if err != nil {
		return nil, fmt.Errorf("invalid bios_release_date format: %w", err)
	}

	if request.UpdatedFromWindows == nil {
		request.UpdatedFromWindows = new(bool) // default to false if not provided
	}

	return &WindowsUpdateDTO{
		Tagnumber:                 *convertedTag,
		SystemSerial:              *request.SystemSerial,
		SystemManufacturer:        request.SystemManufacturer,
		SystemModel:               request.SystemModel,
		SystemSKU:                 request.SystemSKU,
		ChassisType:               request.ChassisType,
		BIOSVersion:               request.BIOSVersion,
		BIOSReleaseDate:           &convertedBIOSReleaseDate,
		TPMVersion:                request.TPMVersion,
		OSInstalledAt:             &convertedTime,
		OSVendor:                  request.OSVendor,
		OSPlatform:                request.OSPlatform,
		OSArchitecture:            request.OSArchitecture,
		OSName:                    request.OSName,
		OSVersion:                 request.OSVersion,
		WindowsDisplayVersion:     request.WindowsDisplayVersion,
		WindowsBuildNumber:        request.WindowsBuildNumber,
		WindowsUBR:                request.WindowsUBR,
		WindowsBitlockerEnabled:   request.WindowsBitlockerEnabled,
		ADDomain:                  request.ADDomain,
		ADDomainUser:              request.ADDomainUser,
		MemoryCapacityKB:          request.MemoryCapacityKB,
		MemorySpeedMHz:            request.MemorySpeedMHz,
		CPUModel:                  request.CPUModel,
		CPUCoreCount:              request.CPUCoreCount,
		CPUThreadCount:            request.CPUThreadCount,
		DiskModel:                 request.DiskModel,
		DiskType:                  request.DiskType,
		DiskSizeKB:                request.DiskSizeKB,
		DiskFreeSpaceKB:           request.DiskFreeSpaceKB,
		EthernetMACAddr:           request.EthernetMACAddr,
		WifiMACAddr:               request.WifiMACAddr,
		BatteryManufacturer:       request.BatteryManufacturer,
		BatterySerial:             request.BatterySerial,
		BatteryCurrentMaxCapacity: request.BatteryCurrentMaxCapacity,
		BatteryDesignCapacity:     request.BatteryDesignCapacity,
		BatteryHealthPcnt:         request.BatteryHealthPcnt,
		BatteryChargeCycleCount:   request.BatteryChargeCycleCount,
		UpdatedFromWindows:        *request.UpdatedFromWindows,
	}, nil
}
