package types

import (
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

type RequestMetadata struct {
	TimeStamp          *time.Time `json:"timestamp"`
	TransactionUUID    *uuid.UUID `json:"transaction_uuid"`
	UpdatedFromWindows *bool      `json:"updated_from_windows"`
	Tagnumber          *int64     `json:"tagnumber"`
	SystemSerial       *string    `json:"system_serial"`
}

type WindowsUpdateRequest struct {
	RequestMetadata           *RequestMetadata `json:"request_metadata"`
	BatteryManufactureDate    *string          `json:"battery_manufacture_date"`
	BatteryManufacturer       *string          `json:"battery_manufacturer"`
	BatteryModel              *string          `json:"battery_model"`
	BatterySerial             *string          `json:"battery_serial"`
	BatteryCurrentMaxCapacity *int64           `json:"battery_current_max_capacity"`
	BatteryDesignCapacity     *int64           `json:"battery_design_capacity"`
	BatteryHealthPcnt         *float64         `json:"battery_health_pct"`
	BatteryChargeCycleCount   *int64           `json:"battery_charge_cycles"`
	SystemUUID                *string          `json:"system_uuid"`
	SystemManufacturer        *string          `json:"system_manufacturer"`
	SystemModel               *string          `json:"system_model"`
	SystemSKU                 *string          `json:"system_sku"`
	ChassisType               *string          `json:"chassis_type"`
	BIOSVersion               *string          `json:"bios_version"`
	BIOSReleaseDate           *string          `json:"bios_release_date"` // Converted later
	TPMVersion                *string          `json:"tpm_version"`
	SecureBootEnabled         *bool            `json:"secure_boot_enabled"`
	OSInstalledAt             *string          `json:"os_installed_at"` // Converted later
	OSVendor                  *string          `json:"os_vendor"`
	OSPlatform                *string          `json:"os_platform"`
	OSArchitecture            *string          `json:"os_architecture"`
	OSName                    *string          `json:"os_name"`
	OSVersion                 *string          `json:"os_version"`
	WindowsDisplayVersion     *string          `json:"windows_display_version"`
	WindowsBuildNumber        *int64           `json:"windows_build_number"`
	WindowsUBR                *int64           `json:"windows_ubr"`
	IsDiskEncrypted           *bool            `json:"windows_bitlocker_enabled"`
	ComputerName              *string          `json:"computer_name"`
	AdminUsers                *string          `json:"ad_admin_users"`
	ADDomain                  *string          `json:"ad_domain"`
	ADComputerName            *string          `json:"ad_computer_name"`
	ADDistinguishedName       *string          `json:"ad_distinguished_name"`
	IsIntuneJoined            *bool            `json:"is_intune_joined"`
	MemorySerial              *string          `json:"memory_serial"`
	MemoryCapacityKB          *int64           `json:"memory_capacity_kb"`
	MemorySpeedMHz            *int64           `json:"memory_speed_mhz"`
	CPUModel                  *string          `json:"cpu_model"`
	CPUCoreCount              *int64           `json:"cpu_core_count"`
	CPUThreadCount            *int64           `json:"cpu_thread_count"`
	DiskModel                 *string          `json:"disk_model"`
	DiskType                  *string          `json:"disk_type"`
	DiskSizeKB                *int64           `json:"disk_size_kb"`
	DiskFreeSpaceKB           *int64           `json:"disk_free_space_kb"`
	EthernetMACAddr           *string          `json:"ethernet_mac_addr"`
	WifiMACAddr               *string          `json:"wifi_mac_addr"`
	InstalledApps             *string          `json:"installed_apps"`
	Has2023CA                 *bool            `json:"has_2023_ca"`
}

type WindowsUpdateDTO struct {
	RequestMetadata           *RequestMetadata
	BatteryManufactureDate    *time.Time
	BatteryManufacturer       *string
	BatteryModel              *string
	BatterySerial             *string
	BatteryCurrentMaxCapacity *int64
	BatteryDesignCapacity     *int64
	BatteryHealthPcnt         *float64
	BatteryChargeCycleCount   *int64
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
	MemorySerial              []string
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
	InstalledApps             []string
	Has2023CA                 *bool
}

type WindowsUpdateResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
}

func (request *WindowsUpdateRequest) ToDTO() (*WindowsUpdateDTO, error) {
	if request == nil {
		return nil, fmt.Errorf("request cannot be nil")
	}
	if request.RequestMetadata == nil {
		return nil, fmt.Errorf("%w for '%s': %v", InvalidFieldError, "request_metadata", "request_metadata is required")
	}

	if request.RequestMetadata.TimeStamp == nil || request.RequestMetadata.TimeStamp.IsZero() {
		return nil, fmt.Errorf("%w for '%s': %v", InvalidFieldError, "timestamp", "timestamp is required")
	}

	if err := IsTagnumberInt64Valid(request.RequestMetadata.Tagnumber); err != nil {
		return nil, fmt.Errorf("%w for '%s': %v", InvalidFieldError, "tagnumber", err)
	}

	if request.RequestMetadata.SystemSerial == nil || strings.TrimSpace(*request.RequestMetadata.SystemSerial) == "" {
		return nil, fmt.Errorf("%w for '%s': %v", InvalidFieldError, "system_serial", "system_serial is required")
	}

	if request.OSInstalledAt == nil || strings.TrimSpace(*request.OSInstalledAt) == "" {
		return nil, fmt.Errorf("%w for '%s': %v", InvalidFieldError, "os_installed_at", "os_installed_at is required")
	}
	convertedTime, err := time.Parse(time.RFC3339, strings.TrimSpace(*request.OSInstalledAt))
	if err != nil {
		return nil, fmt.Errorf("%w for '%s': invalid format: %v", InvalidFieldError, "os_installed_at", err)
	}

	if request.BIOSReleaseDate == nil || strings.TrimSpace(*request.BIOSReleaseDate) == "" {
		return nil, fmt.Errorf("%w for '%s': %v", InvalidFieldError, "bios_release_date", "bios_release_date is required")
	}

	var convertedBatteryManufactureDate *time.Time
	if request.BatteryManufactureDate != nil && strings.TrimSpace(*request.BatteryManufactureDate) != "" {
		tmpBatteryManufactureDate, err := time.Parse(time.RFC3339, strings.TrimSpace(*request.BatteryManufactureDate))
		if err != nil || tmpBatteryManufactureDate.IsZero() {
			// return nil, fmt.Errorf("invalid battery_manufacture_date format: %w", err)
			convertedBatteryManufactureDate = nil
		} else {
			convertedBatteryManufactureDate = &tmpBatteryManufactureDate
		}
	}

	var convertedBIOSReleaseDate *time.Time
	if request.BIOSReleaseDate != nil && strings.TrimSpace(*request.BIOSReleaseDate) != "" {
		tmpBIOSReleaseDate, err := time.Parse(time.RFC3339, strings.TrimSpace(*request.BIOSReleaseDate))
		if err != nil || tmpBIOSReleaseDate.IsZero() {
			// return nil, fmt.Errorf("invalid bios_release_date format: %w", err)
			convertedBIOSReleaseDate = nil
		} else {
			convertedBIOSReleaseDate = &tmpBIOSReleaseDate
		}
	}

	adAdminUsersArr := make([]string, 0)
	if request.AdminUsers != nil && strings.TrimSpace(*request.AdminUsers) != "" {
		adAdminUsersArr = strings.Split(*request.AdminUsers, ";")
	}

	memorySerialArr := make([]string, 0)
	if request.MemorySerial != nil && strings.TrimSpace(*request.MemorySerial) != "" {
		memorySerial := strings.TrimSpace(*request.MemorySerial)
		if memorySerial != "" {
			memorySerialArr = strings.Split(memorySerial, ";")
		}
	}

	installedAppsArr := make([]string, 0)
	if request.InstalledApps != nil && strings.TrimSpace(*request.InstalledApps) != "" {
		installedAppsArr = strings.Split(*request.InstalledApps, ";")
	}

	return &WindowsUpdateDTO{
		RequestMetadata:           request.RequestMetadata,
		BatteryManufactureDate:    convertedBatteryManufactureDate,
		BatteryManufacturer:       request.BatteryManufacturer,
		BatteryModel:              request.BatteryModel,
		BatterySerial:             request.BatterySerial,
		BatteryCurrentMaxCapacity: request.BatteryCurrentMaxCapacity,
		BatteryDesignCapacity:     request.BatteryDesignCapacity,
		BatteryHealthPcnt:         request.BatteryHealthPcnt,
		BatteryChargeCycleCount:   request.BatteryChargeCycleCount,
		SystemUUID:                request.SystemUUID,
		SystemManufacturer:        request.SystemManufacturer,
		SystemModel:               request.SystemModel,
		SystemSKU:                 request.SystemSKU,
		ChassisType:               request.ChassisType,
		BIOSVersion:               request.BIOSVersion,
		BIOSReleaseDate:           convertedBIOSReleaseDate,
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
		MemorySerial:              memorySerialArr,
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
		InstalledApps:             installedAppsArr,
		Has2023CA:                 request.Has2023CA,
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
		return nil, fmt.Errorf("%w for '%s': %v", InvalidFieldError, "request", "request cannot be nil")
	}
	if err := IsTagnumberInt64Valid(req.Tagnumber); err != nil {
		return nil, fmt.Errorf("%w for '%s': %v", InvalidFieldError, "tagnumber", err)
	}
	if err := IsSystemSerialValid(req.SystemSerial); err != nil {
		return nil, fmt.Errorf("%w for '%s': %v", InvalidFieldError, "system_serial", err)
	}
	if req.TransactionUUID == nil || strings.TrimSpace(*req.TransactionUUID) == "" {
		return nil, fmt.Errorf("%w for '%s': %v", InvalidFieldError, "transaction_uuid", "transaction UUID is required")
	}

	return &ClientInitDTO{
		Tagnumber:       *req.Tagnumber,
		SystemSerial:    *req.SystemSerial,
		TransactionUUID: *req.TransactionUUID,
	}, nil
}
