package types

import (
	"fmt"
	"strings"
	"time"
)

type WindowsUpdateRequest struct {
	Tagnumber            *string  `json:"tagnumber"` // Converted later
	SystemSerial         *string  `json:"system_serial"`
	ChassisType          *string  `json:"chassis_type"`
	ADDomain             *string  `json:"ad_domain"`
	ADDomainJoined       *bool    `json:"ad_domain_joined"`
	SystemManufacturer   *string  `json:"system_manufacturer"`
	SystemModel          *string  `json:"system_model"`
	BIOSVersion          *string  `json:"bios_version"`
	OSName               *string  `json:"os_name"`
	OSInstalledAt        *string  `json:"os_installed_at"` // Converted later
	OSVersion            *string  `json:"os_version"`
	UBR                  *string  `json:"ubr"`
	MemoryCapacityKB     *int64   `json:"memory_capacity_kb"`
	MemorySpeedMHz       *int64   `json:"memory_speed_mhz"`
	CPUModel             *string  `json:"cpu_model"`
	CPUCoreCount         *int64   `json:"cpu_core_count"`
	CPUThreadCount       *int64   `json:"cpu_thread_count"`
	DiskSizeKB           *int64   `json:"disk_size_kb"`
	EthernetMACAddr      *string  `json:"ethernet_mac_addr"`
	WifiMACAddr          *string  `json:"wifi_mac_addr"`
	DiskModel            *string  `json:"disk_model"`
	BatteryChargePercent *float64 `json:"battery_charge_percent"`
}

type WindowsUpdateDTO struct {
	Tagnumber            int64      `json:"tagnumber"`
	SystemSerial         string     `json:"system_serial"`
	ChassisType          *string    `json:"chassis_type"`
	ADDomain             *string    `json:"ad_domain"`
	ADDomainJoined       *bool      `json:"ad_domain_joined"`
	SystemManufacturer   *string    `json:"system_manufacturer"`
	SystemModel          *string    `json:"system_model"`
	BIOSVersion          *string    `json:"bios_version"`
	OSName               *string    `json:"os_name"`
	OSInstalledAt        *time.Time `json:"os_installed_at"`
	OSVersion            *string    `json:"os_version"`
	UBR                  *string    `json:"ubr"`
	MemoryCapacityKB     *int64     `json:"memory_capacity_kb"`
	MemorySpeedMHz       *int64     `json:"memory_speed_mhz"`
	CPUModel             *string    `json:"cpu_model"`
	CPUCoreCount         *int64     `json:"cpu_core_count"`
	CPUThreadCount       *int64     `json:"cpu_thread_count"`
	DiskSizeKB           *int64     `json:"disk_size_kb"`
	EthernetMACAddr      *string    `json:"ethernet_mac_addr"`
	WifiMACAddr          *string    `json:"wifi_mac_addr"`
	DiskModel            *string    `json:"disk_model"`
	BatteryChargePercent *float64   `json:"battery_charge_percent"`
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

	return &WindowsUpdateDTO{
		Tagnumber:            *convertedTag,
		SystemSerial:         *request.SystemSerial,
		ChassisType:          request.ChassisType,
		ADDomain:             request.ADDomain,
		ADDomainJoined:       request.ADDomainJoined,
		SystemManufacturer:   request.SystemManufacturer,
		SystemModel:          request.SystemModel,
		BIOSVersion:          request.BIOSVersion,
		OSName:               request.OSName,
		OSInstalledAt:        &convertedTime,
		OSVersion:            request.OSVersion,
		UBR:                  request.UBR,
		MemoryCapacityKB:     request.MemoryCapacityKB,
		MemorySpeedMHz:       request.MemorySpeedMHz,
		CPUModel:             request.CPUModel,
		CPUCoreCount:         request.CPUCoreCount,
		CPUThreadCount:       request.CPUThreadCount,
		DiskSizeKB:           request.DiskSizeKB,
		EthernetMACAddr:      request.EthernetMACAddr,
		WifiMACAddr:          request.WifiMACAddr,
		DiskModel:            request.DiskModel,
		BatteryChargePercent: request.BatteryChargePercent,
	}, nil
}
