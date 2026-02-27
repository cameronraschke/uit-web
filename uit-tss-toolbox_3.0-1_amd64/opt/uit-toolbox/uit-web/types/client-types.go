package types

import (
	"encoding/json"
	"fmt"
	"net/netip"
	"strings"
	"time"

	"github.com/google/uuid"
)

type Client interface {
	GetAuth() *ClientAuth
	GetData() *ClientData
	SetAuth(auth *ClientAuth)
	SetData(data *ClientData)
}

type ClientAuth struct {
	AuthToken string `json:"auth_token,omitempty"`
}

type ClientData struct {
	Tagnumber          *int64              `json:"tagnumber,omitempty"`
	Serial             *string             `json:"serial,omitempty"`
	Manufacturer       *string             `json:"manufacturer,omitempty"`
	Model              *string             `json:"model,omitempty"`
	ProductFamily      *string             `json:"product_family,omitempty"`
	ProductName        *string             `json:"product_name,omitempty"`
	SKU                *string             `json:"sku,omitempty"`
	UUID               *string             `json:"uuid,omitempty"`
	ConnectedToHost    *bool               `json:"connected_to_host,omitempty"`
	NTPSynced          *bool               `json:"ntp_synced,omitempty"`
	Hardware           *ClientHardwareData `json:"hardware_data,omitempty"`
	Software           *ClientSoftwareData `json:"software_data,omitempty"`
	RealtimeSystemData *RealtimeSystemData `json:"realtime_system_data,omitempty"`
	JobData            *JobData            `json:"job_data,omitempty"`
}

type RealtimeSystemData struct {
	LastHeardTimestamp *time.Time     `json:"last_heard_timestamp,omitempty"`
	SystemUptime       time.Duration  `json:"system_uptime,omitempty"`
	AppUptime          *time.Duration `json:"app_uptime,omitempty"`
	KernelUpdated      *bool          `json:"kernel_updated,omitempty"`
	CPU                *CPUUsage      `json:"cpu_usage,omitempty"`
	Memory             *MemoryUsage   `json:"memory_usage,omitempty"`
	Network            *NetworkUsage  `json:"network_usage,omitempty"`
	Energy             *EnergyUsage   `json:"energy_usage,omitempty"`
}

type MemoryUsage struct {
	TotalUsage    *int64 `json:"memory_usage"`
	TotalCapacity int64  `json:"memory_capacity"`
	Type          string `json:"type"`
	SpeedMHz      int64  `json:"speed_mhz"`
}

type NetworkUsage struct {
	NetworkUsage *int64 `json:"network_usage"`
	LinkSpeed    *int64 `json:"link_speed"`
}

type CPUUsage struct {
	UsagePercent  *float64 `json:"cpu_usage"`
	MillidegreesC *float64 `json:"cpu_millidegrees_c"`
}

type EnergyUsage struct {
	Watts *float64 `json:"watts"`
}

type JobData struct {
	UUID               uuid.UUID     `json:"uuid,omitempty"`
	QueuedRemotely     *bool         `json:"queued_remotely,omitempty"`
	Type               *JobType      `json:"type,omitempty"` // e.g. clone/erase/erase+clone
	SelectedDisk       *string       `json:"selected_disk,omitempty"`
	EraseQueued        *bool         `json:"erase_queued,omitempty"`
	EraseMode          *string       `json:"erase_mode,omitempty"`
	SecureEraseCapable *bool         `json:"secure_erase_capable,omitempty"`
	UsedSecureErase    *bool         `json:"used_secure_erase,omitempty"`
	EraseVerified      *bool         `json:"erase_verified,omitempty"`
	EraseVerifyPcnt    *float64      `json:"erase_verify_percent,omitempty"`
	EraseCompleted     *bool         `json:"erase_completed,omitempty"`
	CloneQueued        *bool         `json:"clone_queued,omitempty"`
	CloneMode          *string       `json:"clone_mode,omitempty"`
	CloneImageName     *string       `json:"clone_image_name,omitempty"`
	CloneSourceHost    *string       `json:"clone_source_host,omitempty"`
	CloneCompleted     *bool         `json:"clone_completed,omitempty"`
	Failed             *bool         `json:"job_failed,omitempty"`
	FailedMessage      *string       `json:"job_failed_message,omitempty"`
	StartTime          *time.Time    `json:"start_time,omitempty"`
	EndTime            *time.Time    `json:"end_time,omitempty"`
	Duration           time.Duration `json:"duration,omitempty"`
	AvgCPUUsage        *CPUUsage     `json:"avg_cpu_usage,omitempty"`
	AvgNetworkUsage    *NetworkUsage `json:"avg_network_usage,omitempty"`
	Hibernated         *bool         `json:"hibernated,omitempty"`
	Realtime           *JobQueueData `json:"realtime_job_data,omitempty"`
}

type JobQueueData struct {
	JobName           *string       `json:"job_name,omitempty"`
	JobNameFormatted  *string       `json:"job_name_formatted,omitempty"`
	JobQueued         *bool         `json:"job_queued,omitempty"`
	JobRequiresQueue  *bool         `json:"job_requires_queue,omitempty"`
	JobQueuePosition  *int          `json:"job_queue_position,omitempty"`
	JobQueuedOverride *bool         `json:"job_queue_override,omitempty"`
	JobActive         *bool         `json:"job_active,omitempty"`
	JobProgress       *float64      `json:"job_progress,omitempty"`
	JobDuration       time.Duration `json:"job_duration,omitempty"`
	JobStatusMessage  *string       `json:"job_status_message,omitempty"`
}

type ClientSoftwareData struct {
	OSInstalled          *bool                    `json:"os_installed,omitempty"`
	OSName               *string                  `json:"os_name,omitempty"`
	OSVersion            *string                  `json:"os_version,omitempty"`
	OSInstalledTimestamp *time.Time               `json:"os_installed_timestamp,omitempty"`
	ImageName            *string                  `json:"image_name,omitempty"`
	Motherboard          *MotherboardSoftwareData `json:"motherboard,omitempty"`
}

type MotherboardSoftwareData struct {
	BIOSUpdated          *bool   `json:"bios_updated,omitempty"`
	BIOSVersion          *string `json:"bios_version,omitempty"`
	BIOSDate             *string `json:"bios_date,omitempty"`
	BIOSFirmwareRevision *string `json:"bios_firmware_revision,omitempty"`
	UEFIEnabled          *bool   `json:"uefi_enabled,omitempty"`
	SecureBootEnabled    *bool   `json:"secure_boot_enabled,omitempty"`
	TPMEnabled           *bool   `json:"tpm_enabled,omitempty"`
}

type ClientHardwareData struct {
	CPU         *CPUHardwareData               `json:"cpu,omitempty"`
	Motherboard *MotherboardHardwareData       `json:"motherboard,omitempty"`
	Memory      map[string]MemoryHardwareData  `json:"memory,omitempty"`
	Network     map[string]NetworkHardwareData `json:"network,omitempty"`
	Graphics    *GraphicsHardwareData          `json:"graphics,omitempty"`
	Disks       map[string]DiskHardwareData    `json:"disks,omitempty"`
	Battery     *BatteryHardwareData           `json:"battery,omitempty"`
	Wireless    *WirelessHardwareData          `json:"wireless,omitempty"`
	Chassis     *ChassisHardwareData           `json:"chassis,omitempty"`
	PowerSupply *PowerSupplyHardwareData       `json:"power_supply,omitempty"`
	TPM         *TPMHardwareData               `json:"tpm,omitempty"`
}

type CPUHardwareData struct {
	ID                     *string          `json:"id,omitempty"`
	Signature              *string          `json:"signature,omitempty"`
	Manufacturer           *string          `json:"manufacturer,omitempty"`
	ProductFamily          *string          `json:"product_family,omitempty"`
	Model                  *string          `json:"model,omitempty"`
	Socket                 *string          `json:"socket,omitempty"`
	Version                *string          `json:"version,omitempty"`
	Voltage                *float64         `json:"voltage,omitempty"`
	PhysicalCores          *int64           `json:"physical_cores,omitempty"`
	LogicalCores           *int64           `json:"logical_cores,omitempty"`
	CurrentSpeedMHz        *float64         `json:"current_speed_mhz,omitempty"`
	MaxSpeedMHz            *float64         `json:"max_speed_mhz,omitempty"`
	L1CacheKB              *float64         `json:"l1_cache_kb,omitempty"`
	L2CacheKB              *float64         `json:"l2_cache_kb,omitempty"`
	L3CacheKB              *float64         `json:"l3_cache_kb,omitempty"`
	ThermalProbeWorking    map[string]*bool `json:"thermal_probe_working,omitempty"`
	ThermalProbeResolution *float64         `json:"thermal_probe_resolution,omitempty"`
	Temperature            int64            `json:"temperature,omitempty"`
}

type MotherboardHardwareData struct {
	Serial                 *string          `json:"serial,omitempty"`
	Model                  *string          `json:"model,omitempty"`
	Manufacturer           *string          `json:"manufacturer,omitempty"`
	TotalRAMSlots          *int64           `json:"total_ram_slots,omitempty"`
	PCIELanes              map[string]int64 `json:"pcie_lanes,omitempty"`
	M2Slots                map[string]int64 `json:"m2_slots,omitempty"`
	ThermalProbeWorking    map[string]*bool `json:"thermal_probe_working,omitempty"`
	ThermalProbeResolution *float64         `json:"thermal_probe_resolution,omitempty"`
}

type MemoryHardwareData struct {
	Serial       *string  `json:"serial,omitempty"`
	AssestTag    *string  `json:"asset_tag,omitempty"`
	PartNumber   *string  `json:"part_number,omitempty"`
	Rank         *int64   `json:"rank,omitempty"`
	CapacityGB   *int64   `json:"capacity_gb,omitempty"`
	SpeedMHz     *int64   `json:"speed_mhz,omitempty"`
	Voltage      *float64 `json:"voltage,omitempty"`
	FormFactor   *string  `json:"form_factor,omitempty"`
	Type         *string  `json:"type,omitempty"`
	Manufacturer *string  `json:"manufacturer,omitempty"`
}

type NetworkHardwareData struct {
	MACAddress    *string      `json:"mac_addr,omitempty"`
	Type          *string      `json:"type,omitempty"`
	Wired         *bool        `json:"wired,omitempty"`
	Wireless      *bool        `json:"wireless,omitempty"`
	Model         *string      `json:"model,omitempty"`
	NetworkLinkUp *bool        `json:"network_link_up,omitempty"`
	IPAddress     []netip.Addr `json:"ip_address,omitempty"`
	Netmask       *string      `json:"netmask,omitempty"`
}

type GraphicsHardwareData struct {
	HasBuiltInScreen *bool  `json:"has_built_in_screen,omitempty"`
	HasTouchscreen   *bool  `json:"has_touchscreen,omitempty"`
	HasDedicatedGPU  *bool  `json:"has_dedicated_gpu,omitempty"`
	ScreenWidth      *int64 `json:"screen_width,omitempty"`
	ScreenHeight     *int64 `json:"screen_height,omitempty"`
	TerminalRows     *int64 `json:"terminal_rows,omitempty"`
	TerminalCols     *int64 `json:"terminal_cols,omitempty"`
}

type DiskHardwareData struct {
	LinuxAlias               *string  `json:"linux_alias,omitempty"`
	Type                     *string  `json:"type,omitempty"`
	LinuxDevicePath          *string  `json:"linux_device_path,omitempty"`
	LinuxMajorNumber         *int64   `json:"linux_major_number,omitempty"`
	LinuxMinorNumber         *int64   `json:"linux_minor_number,omitempty"`
	InterfaceType            *string  `json:"interface_type,omitempty"`
	Serial                   *string  `json:"serial,omitempty"`
	WWID                     *string  `json:"wwid,omitempty"`
	NvmeQualifiedName        *string  `json:"nvme_qualified_name,omitempty"`
	Model                    *string  `json:"model,omitempty"`
	Manufacturer             *string  `json:"manufacturer,omitempty"`
	CapacityMiB              *float64 `json:"capacity_mib,omitempty"`
	LogicalBlockSize         *int64   `json:"logical_block_size,omitempty"`
	PhysicalBlockSize        *int64   `json:"physical_block_size,omitempty"`
	SectorCount              *int64   `json:"sector_count,omitempty"`
	Firmware                 *string  `json:"firmware,omitempty"`
	DeviceState              *string  `json:"device_state,omitempty"`
	Rotating                 *bool    `json:"rotating,omitempty"`
	Removable                *bool    `json:"removable,omitempty"`
	TotalReadsLBAs           *int64   `json:"total_reads_lbas,omitempty"`
	TotalWritesLBAs          *int64   `json:"total_writes_lbas,omitempty"`
	TotalReadsGiB            *float64 `json:"total_reads_gib,omitempty"`
	TotalWritesGiB           *float64 `json:"total_writes_gib,omitempty"`
	TotalUptimeHrs           *float64 `json:"total_uptime_hrs,omitempty"`
	TotalPowerCycles         *int64   `json:"total_power_cycles,omitempty"`
	Temperature              *float64 `json:"temperature,omitempty"`
	MaxTemperature           *float64 `json:"max_temperature,omitempty"`
	SMARTSupported           *bool    `json:"smart_supported,omitempty"`
	SMARTEnabled             *bool    `json:"smart_enabled,omitempty"`
	SMARTCheckCompleted      *bool    `json:"smart_check_completed,omitempty"`
	SMARTTemperature         *float64 `json:"smart_temperature,omitempty"`
	SMARTErrors              *int64   `json:"smart_errors,omitempty"`
	SMARTDataIntegrityErrors *int64   `json:"smart_data_integrity_errors,omitempty"`
	PCIeCurrentLinkSpeed     *string  `json:"pcie_current_link_speed,omitempty"`
	PCIeMaxLinkSpeed         *string  `json:"pcie_max_link_speed,omitempty"`
	PCIeCurrentLinkWidth     *string  `json:"pcie_current_link_width,omitempty"`
	PCIeMaxLinkWidth         *string  `json:"pcie_max_link_width,omitempty"`
}

type BatteryHardwareData struct {
	HasBattery      *bool    `json:"has_battery,omitempty"`
	Manufacturer    *string  `json:"manufacturer,omitempty"`
	ManufactureDate *string  `json:"manufacture_date,omitempty"`
	Serial          *string  `json:"serial,omitempty"`
	ChargeCycles    *int64   `json:"charge_cycles,omitempty"`
	DesignMWh       *float64 `json:"design_mwh,omitempty"`
	FullMWh         *float64 `json:"full_mwh,omitempty"`
	CurrentMWh      *float64 `json:"current_mwh,omitempty"`
	HealthPercent   *float64 `json:"health_percent,omitempty"`
	CurrentCharge   *float64 `json:"current_charge,omitempty"`
	Status          *string  `json:"status,omitempty"`
}

type WirelessHardwareData struct {
	HasWiFi               *bool   `json:"has_wifi,omitempty"`
	HasBluetooth          *bool   `json:"has_bluetooth,omitempty"`
	WiFiVersion           *string `json:"wifi_version,omitempty"`
	WiFiManufacturer      *string `json:"wifi_manufacturer,omitempty"`
	WiFiModel             *string `json:"wifi_model,omitempty"`
	BluetoothVersion      *string `json:"bluetooth_version,omitempty"`
	BluetoothManufacturer *string `json:"bluetooth_manufacturer,omitempty"`
	BluetoothModel        *string `json:"bluetooth_model,omitempty"`
}

type ChassisHardwareData struct {
	Type             *string            `json:"type,omitempty"`
	LockPresent      *bool              `json:"lock_present,omitempty"`
	Serial           *string            `json:"serial,omitempty"`
	AssetTag         *string            `json:"asset_tag,omitempty"`
	BootUpSafe       *bool              `json:"boot_up_safe,omitempty"`
	ThermalSafe      *bool              `json:"thermal_safe,omitempty"`
	HasRJ45          *bool              `json:"has_rj45,omitempty"`
	DisplayPortPorts *int64             `json:"display_port_ports,omitempty"`
	HDMIPorts        *int64             `json:"hdmi_ports,omitempty"`
	VGAPorts         *int64             `json:"vga_ports,omitempty"`
	DVIPorts         *int64             `json:"dvi_ports,omitempty"`
	SerialPorts      *int64             `json:"serial_ports,omitempty"`
	USB1Ports        map[string]int64   `json:"usb1_ports,omitempty"`
	USB2Ports        map[string]int64   `json:"usb2_ports,omitempty"`
	USB3Ports        map[string]int64   `json:"usb3_ports,omitempty"`
	SATAPorts        map[string]int64   `json:"sata_ports,omitempty"`
	InternalFans     map[string]float64 `json:"fan_rpm,omitempty"`
	AudioPorts       map[string]int64   `json:"audio_ports,omitempty"`
}

type PowerSupplyHardwareData struct {
	Manufacturer    *string `json:"manufacturer,omitempty"`
	Model           *string `json:"model,omitempty"`
	Serial          *string `json:"serial,omitempty"`
	Location        *string `json:"location,omitempty"`
	MaxWattage      *int64  `json:"max_wattage,omitempty"`
	PowerSupplySafe *bool   `json:"power_supply_safe,omitempty"`
	Status          *string `json:"status,omitempty"`
	HotPlugCapable  *bool   `json:"hot_plug_capable,omitempty"`
}

type TPMHardwareData struct {
	Present      *bool   `json:"present,omitempty"`
	Manufacturer *string `json:"manufacturer,omitempty"`
}

// --- JobType enum and JSON marshalling ---
type JobType int

const (
	JobModeNone JobType = iota + 1
	JobModeErase
	JobModeClone
	JobModeEraseAndClone
)

var JobModeMap = map[string]JobType{
	"None":          JobModeNone,
	"Erase":         JobModeErase,
	"Clone":         JobModeClone,
	"EraseAndClone": JobModeEraseAndClone,
}

// normalizeJobMode lowercases and removes non-alphanumeric characters.
// This lets us accept variants like "erase_and-clone" or " Erase And Clone ".
func normalizeJobMode(s string) string {
	s = strings.TrimSpace(s)
	s = strings.ToLower(s)
	var b strings.Builder
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// ParseJobMode converts a string to a JobType using JobModeMap with normalization.
// Returns a pointer to the parsed value, or an error if invalid.
func ParseJobMode(modeStr string) (*JobType, error) {
	// Try exact match for canonical keys
	if mode, ok := JobModeMap[modeStr]; ok {
		v := mode
		return &v, nil
	}
	// Next, try normalized match
	normalizedJobMode := normalizeJobMode(modeStr)
	for jmStr, jm := range JobModeMap {
		if normalizeJobMode(jmStr) == normalizedJobMode {
			v := jm
			return &v, nil
		}
	}
	return nil, fmt.Errorf("invalid JobType: %s", modeStr)
}

func (jm JobType) IsValid() bool {
	var jmMin int
	var jmMax int

	for _, val := range JobModeMap {
		if int(val) < jmMin || jmMin == 0 {
			jmMin = int(val)
		}
		if int(val) > jmMax {
			jmMax = int(val)
		}
	}
	if int(jm) < jmMin || int(jm) > jmMax {
		return false
	}
	return true
}

func (jm JobType) String() string {
	if !jm.IsValid() {
		return "Invalid"
	}
	for str, val := range JobModeMap {
		if val == jm {
			return str
		}
	}
	return "Invalid"
}

func (jm JobType) MarshalJSON() ([]byte, error) {
	return json.Marshal(jm.String())
}

func (jm *JobType) UnmarshalJSON(data []byte) error {
	var str string
	if err := json.Unmarshal(data, &str); err != nil {
		return err
	}
	if val, ok := JobModeMap[str]; ok {
		*jm = val
		return nil
	}
	return fmt.Errorf("invalid JobType: %s", str)
}
