package types

import (
	"fmt"
	"strings"
	"time"
)

type WindowsUpdateRequest struct {
	LastHardwareCheck         *time.Time `json:"last_hardware_check"`
	Tagnumber                 *int64     `json:"tagnumber"`
	SystemSerial              *string    `json:"system_serial"`
	SystemUUID                *string    `json:"system_uuid"`
	SystemManufacturer        *string    `json:"system_manufacturer"`
	SystemModel               *string    `json:"system_model"`
	SystemSKU                 *string    `json:"system_sku"`
	ChassisType               *string    `json:"chassis_type"`
	BIOSVersion               *string    `json:"bios_version"`
	BIOSReleaseDate           *string    `json:"bios_release_date"` // Converted later
	TPMVersion                *string    `json:"tpm_version"`
	SecureBootEnabled         *bool      `json:"secure_boot_enabled"`
	OSInstalledAt             *string    `json:"os_installed_at"` // Converted later
	OSVendor                  *string    `json:"os_vendor"`
	OSPlatform                *string    `json:"os_platform"`
	OSArchitecture            *string    `json:"os_architecture"`
	OSName                    *string    `json:"os_name"`
	OSVersion                 *string    `json:"os_version"`
	WindowsDisplayVersion     *string    `json:"windows_display_version"`
	WindowsBuildNumber        *int64     `json:"windows_build_number"`
	WindowsUBR                *int64     `json:"windows_ubr"`
	IsDiskEncrypted           *bool      `json:"windows_bitlocker_enabled"`
	ComputerName              *string    `json:"computer_name"`
	AdminUsers                *string    `json:"ad_admin_users"`
	ADDomain                  *string    `json:"ad_domain"`
	ADComputerName            *string    `json:"ad_computer_name"`
	ADDistinguishedName       *string    `json:"ad_distinguished_name"`
	IsIntuneJoined            *bool      `json:"is_intune_joined"`
	MemoryCapacityKB          *int64     `json:"memory_capacity_kb"`
	MemorySpeedMHz            *int64     `json:"memory_speed_mhz"`
	CPUModel                  *string    `json:"cpu_model"`
	CPUCoreCount              *int64     `json:"cpu_core_count"`
	CPUThreadCount            *int64     `json:"cpu_thread_count"`
	DiskModel                 *string    `json:"disk_model"`
	DiskType                  *string    `json:"disk_type"`
	DiskSizeKB                *int64     `json:"disk_size_kb"`
	DiskFreeSpaceKB           *int64     `json:"disk_free_space_kb"`
	EthernetMACAddr           *string    `json:"ethernet_mac_addr"`
	WifiMACAddr               *string    `json:"wifi_mac_addr"`
	BatteryManufacturer       *string    `json:"battery_manufacturer"`
	BatterySerial             *string    `json:"battery_serial"`
	BatteryCurrentMaxCapacity *int64     `json:"battery_current_max_capacity"`
	BatteryDesignCapacity     *int64     `json:"battery_design_capacity"`
	BatteryHealthPcnt         *float64   `json:"battery_health_pct"`
	BatteryChargeCycleCount   *int64     `json:"battery_charge_cycle_count"`
	UpdatedFromWindows        bool       `json:"updated_from_windows"`
}

type WindowsUpdateDTO struct {
	LastHardwareCheck         time.Time
	Tagnumber                 int64
	SystemSerial              string
	SystemUUID                *string
	SystemManufacturer        *string
	SystemModel               *string
	SystemSKU                 *string
	ChassisType               *string
	BIOSVersion               *string
	BIOSReleaseDate           *time.Time
	TPMVersion                *string
	SecureBootEnabled         *bool
	OSInstalledAt             *time.Time
	OSVendor                  *string
	OSPlatform                *string
	OSArchitecture            *string
	OSName                    *string
	OSVersion                 *string
	WindowsDisplayVersion     *string
	WindowsBuildNumber        *int64
	WindowsUBR                *int64
	IsDiskEncrypted           *bool
	ComputerName              *string
	AdminUsers                []string
	ADDomain                  *string
	ADComputerName            *string
	ADDistinguishedName       *string
	IsIntuneJoined            *bool
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

func (request *WindowsUpdateRequest) ToDTO() (*WindowsUpdateDTO, error) {
	if request.Tagnumber == nil {
		return nil, fmt.Errorf("tagnumber is required")
	}

	if request.LastHardwareCheck == nil || request.LastHardwareCheck.IsZero() {
		return nil, fmt.Errorf("last_hardware_check is required")
	}

	if err := IsTagnumberInt64Valid(request.Tagnumber); err != nil {
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

	adAdminUsersArr := make([]string, 0)
	if request.AdminUsers != nil && strings.TrimSpace(*request.AdminUsers) != "" {
		adAdminUsersArr = strings.Split(*request.AdminUsers, ";")
	}

	return &WindowsUpdateDTO{
		LastHardwareCheck:         *request.LastHardwareCheck,
		Tagnumber:                 *request.Tagnumber,
		SystemSerial:              *request.SystemSerial,
		SystemUUID:                request.SystemUUID,
		SystemManufacturer:        request.SystemManufacturer,
		SystemModel:               request.SystemModel,
		SystemSKU:                 request.SystemSKU,
		ChassisType:               request.ChassisType,
		BIOSVersion:               request.BIOSVersion,
		BIOSReleaseDate:           &convertedBIOSReleaseDate,
		TPMVersion:                request.TPMVersion,
		SecureBootEnabled:         request.SecureBootEnabled,
		OSInstalledAt:             &convertedTime,
		OSVendor:                  request.OSVendor,
		OSPlatform:                request.OSPlatform,
		OSArchitecture:            request.OSArchitecture,
		OSName:                    request.OSName,
		OSVersion:                 request.OSVersion,
		WindowsDisplayVersion:     request.WindowsDisplayVersion,
		WindowsBuildNumber:        request.WindowsBuildNumber,
		WindowsUBR:                request.WindowsUBR,
		IsDiskEncrypted:           request.IsDiskEncrypted,
		ComputerName:              request.ComputerName,
		AdminUsers:                adAdminUsersArr,
		ADDomain:                  request.ADDomain,
		ADComputerName:            request.ADComputerName,
		ADDistinguishedName:       request.ADDistinguishedName,
		IsIntuneJoined:            request.IsIntuneJoined,
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
		UpdatedFromWindows:        request.UpdatedFromWindows,
	}, nil
}

type ClientInit interface {
	ToDTO(*ClientInitRequest) (*ClientInitDTO, error)
}

type ClientInitRequest struct {
	Tagnumber       *int64  `json:"tagnumber"`
	SystemSerial    *string `json:"system_serial"`
	TransactionUUID *string `json:"transaction_uuid,omitempty"`
}

type ClientInitDTO struct {
	Tagnumber       int64  `json:"tagnumber"`
	SystemSerial    string `json:"system_serial"`
	TransactionUUID string `json:"transaction_uuid,omitempty"`
}

type ClientInitResponse struct {
	ClientUUID string `json:"client_uuid"`
}

func (req *ClientInitRequest) ToDTO() (*ClientInitDTO, error) {
	if req == nil {
		return nil, fmt.Errorf("request cannot be nil")
	}
	if req.Tagnumber == nil {
		return nil, fmt.Errorf("tagnumber is required")
	}
	if err := IsTagnumberInt64Valid(req.Tagnumber); err != nil {
		return nil, fmt.Errorf("invalid tagnumber: %w", err)
	}
	if req.SystemSerial == nil || strings.TrimSpace(*req.SystemSerial) == "" {
		return nil, fmt.Errorf("system serial is required")
	}
	if req.TransactionUUID == nil || strings.TrimSpace(*req.TransactionUUID) == "" {
		return nil, fmt.Errorf("transaction UUID is required")
	}

	return &ClientInitDTO{
		Tagnumber:       *req.Tagnumber,
		SystemSerial:    *req.SystemSerial,
		TransactionUUID: *req.TransactionUUID,
	}, nil
}
