package types

import (
	"fmt"
	"net/netip"
	"strconv"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

func copyTrimmedStringPtr(value *string) *string {
	if value == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*value)
	return &trimmed
}

func copyStringPtr(value *string) *string {
	if value == nil {
		return nil
	}
	v := *value
	return &v
}

func copyInt64Ptr(value *int64) *int64 {
	if value == nil {
		return nil
	}
	v := *value
	return &v
}

func copyTimePtr(value *time.Time) *time.Time {
	if value == nil {
		return nil
	}
	v := *value
	return &v
}

func timePtrToUTC(value *time.Time) *time.Time {
	if value == nil {
		return nil
	}
	utc := value.UTC()
	return &utc
}

func copyBoolPtr(value *bool) *bool {
	if value == nil {
		return nil
	}
	v := *value
	return &v
}

func int64ToPtr(value int64) *int64 {
	v := value
	return &v
}

func stringToPtr(value string) *string {
	v := value
	return &v
}

func IsTagnumberInt64Valid(i *int64) error {
	if i == nil {
		return fmt.Errorf("tagnumber is nil")
	}
	if *i < 100000 || *i > 999999 {
		return fmt.Errorf("tagnumber is out of valid numeric range")
	}
	return nil
}

func IsTagnumberStringValid(b []byte) error {
	if len(b) == 0 {
		return fmt.Errorf("tagnumber is nil")
	}
	if !IsNumericAscii(b) {
		return fmt.Errorf("tagnumber contains non-numeric ASCII characters")
	}
	if utf8.RuneCount(b) != 6 {
		return fmt.Errorf("tagnumber does not contain exactly 6 characters")
	}
	return nil
}

func ValidateIPAddress(ipAddr *netip.Addr) error {
	if ipAddr == nil {
		return fmt.Errorf("nil IP address")
	}
	if ipAddr.IsUnspecified() || !ipAddr.IsValid() {
		return fmt.Errorf("unspecified or invalid IP address: %s", ipAddr.String())
	}
	if ipAddr.IsInterfaceLocalMulticast() || ipAddr.IsLinkLocalMulticast() || ipAddr.IsMulticast() {
		return fmt.Errorf("multicast IP address not allowed: %s", ipAddr.String())
	}
	return nil
}

func ConvertAndCheckIPStr(ipPtr *string) (ipAddr *netip.Addr, isLoopback bool, isLocal bool, err error) {
	if ipPtr == nil {
		return nil, false, false, fmt.Errorf("nil IP address")
	}

	ipStr := strings.TrimSpace(*ipPtr)
	if ipStr == "" {
		return nil, false, false, fmt.Errorf("empty IP address")
	}

	ip, err := netip.ParseAddr(ipStr)
	if err != nil {
		return nil, false, false, fmt.Errorf("failed to parse IP address: %w", err)
	}

	if err := ValidateIPAddress(&ip); err != nil {
		return nil, false, false, fmt.Errorf("invalid IP address: %w", err)
	}

	return &ip, ip.IsLoopback(), ip.IsPrivate(), nil
}

func IsPrintableASCII(b []byte) bool {
	for i := range b {
		char := b[i]
		if char < 0x20 || char > 0x7E { // Space (0x20) to tilde (0x7E)
			return false
		}
	}
	return true
}

func IsASCIIStringPrintable(s string) bool {
	if !utf8.ValidString(s) {
		return false
	}
	for _, char := range s {
		if char < 32 || char > 126 {
			return false
		}
	}
	return true
}

func IsPrintableUnicodeString(s string) bool {
	if !utf8.ValidString(s) {
		return false
	}
	for _, char := range s {
		if !unicode.IsPrint(char) && !unicode.IsSpace(char) {
			return false
		}
	}
	return true
}

func IsPrintableUnicode(b []byte) bool {
	if !utf8.Valid(b) {
		return false
	}
	for _, char := range string(b) {
		if !unicode.IsPrint(char) && !unicode.IsSpace(char) {
			return false
		}
	}
	return true
}

func IsNumericAscii(b []byte) bool {
	if len(b) == 0 {
		return false
	}
	if !utf8.Valid(b) {
		return false
	}
	for i := range b {
		char := b[i]
		if char < '0' || char > '9' {
			return false
		}
	}
	return true
}

func CountDigits(n int64) int {
	if n == 0 {
		return 1
	}
	count := 0
	for n != 0 {
		n /= 10
		count++
	}
	return count
}

func IsSHA256String(shaStr string) error {
	if len(shaStr) != 64 { // ASCII, 1 byte per char
		return fmt.Errorf("invalid length for SHA256 string: %d chars", len(shaStr))
	}
	for _, char := range shaStr {
		if !((char >= '0' && char <= '9') || (char >= 'a' && char <= 'f') || (char >= 'A' && char <= 'F')) {
			return fmt.Errorf("invalid character found in SHA256 string")
		}
	}
	return nil
}

func ConvertAndVerifyTagnumber(tagStr string) (*int64, error) {
	trimmedTagStr := strings.TrimSpace(tagStr)
	if trimmedTagStr == "" {
		return nil, fmt.Errorf("tagnumber string is empty")
	}
	validStringErr := IsTagnumberStringValid([]byte(trimmedTagStr))
	if validStringErr != nil {
		return nil, fmt.Errorf("invalid tagnumber string: %v", validStringErr)
	}
	tag, err := strconv.ParseInt(trimmedTagStr, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("cannot parse tagnumber: %v", err)
	}
	validInt64Err := IsTagnumberInt64Valid(&tag)
	if validInt64Err != nil {
		return nil, fmt.Errorf("invalid tagnumber: %v", validInt64Err)
	}
	return &tag, nil
}

type InventoryAdvSearchOptions struct {
	Tagnumber          *int64  `json:"tagnumber"`
	SystemSerial       *string `json:"system_serial"`
	Location           *string `json:"location"`
	SystemManufacturer *string `json:"system_manufacturer"`
	SystemModel        *string `json:"system_model"`
	DeviceType         *string `json:"device_type"`
	Department         *string `json:"department_name"`
	Domain             *string `json:"ad_domain"`
	Status             *string `json:"status"`
	Broken             *bool   `json:"is_broken"`
	HasImages          *bool   `json:"has_images"`
}

type JobQueueTableRowView struct {
	Tagnumber            *int64         `json:"tagnumber"`
	SystemSerial         *string        `json:"system_serial"`
	SystemManufacturer   *string        `json:"system_manufacturer"`
	SystemModel          *string        `json:"system_model"`
	Location             *string        `json:"location"`
	Department           *string        `json:"department_name"`
	ClientStatus         *string        `json:"client_status"`
	IsBroken             *bool          `json:"is_broken"`
	DiskRemoved          *bool          `json:"disk_removed"`
	TempWarning          *bool          `json:"temp_warning"`
	BatteryHealthWarning *bool          `json:"battery_health_warning"`
	CheckoutBool         *bool          `json:"checkout_bool"`
	KernelUpdated        *bool          `json:"kernel_updated"`
	LastHeard            *time.Time     `json:"last_heard"`
	SystemUptime         *time.Duration `json:"system_uptime"`
	Online               *bool          `json:"online"`
	JobActive            *bool          `json:"job_active"`
	JobQueued            *bool          `json:"job_queued"`
	QueuePosition        *int64         `json:"queue_position"`
	JobName              *string        `json:"job_name"`
	JobNameReadable      *string        `json:"job_name_readable"`
	JobCloneMode         *string        `json:"job_clone_mode"`
	JobEraseMode         *string        `json:"job_erase_mode"`
	JobStatus            *string        `json:"job_status"`
	LastJobTime          *time.Time     `json:"last_job_time"`
	OSInstalled          *string        `json:"os_installed"`
	OSName               *string        `json:"os_name"`
	OSUpdated            *bool          `json:"os_updated"`
	DomainJoined         *bool          `json:"domain_joined"`
	DomainName           *string        `json:"ad_domain"`
	DomainNameFormatted  *string        `json:"ad_domain_formatted"`
	BIOSUpdated          *bool          `json:"bios_updated"`
	BIOSVersion          *string        `json:"bios_version"`
	CPUUsage             *float64       `json:"cpu_usage"`
	CPUTemp              *float64       `json:"cpu_temp"`
	CPUTempWarning       *bool          `json:"cpu_temp_warning"`
	MemoryUsage          *float64       `json:"memory_usage"`
	MemoryCapacity       *float64       `json:"memory_capacity"`
	DiskUsage            *float64       `json:"disk_usage"`
	DiskTemp             *float64       `json:"disk_temp"`
	DiskType             *string        `json:"disk_type"`
	DiskSize             *float64       `json:"disk_size"`
	MaxDiskTemp          *float64       `json:"max_disk_temp"`
	DiskTempWarning      *bool          `json:"disk_temp_warning"`
	NetworkLinkStatus    *string        `json:"network_link_status"`
	NetworkLinkSpeed     *float64       `json:"network_link_speed"`
	NetworkUsage         *float64       `json:"network_usage"`
	BatteryCharge        *int64         `json:"battery_charge"`
	BatteryStatus        *string        `json:"battery_status"`
	BatteryHealth        *float64       `json:"battery_health"`
	PluggedIn            *bool          `json:"plugged_in"`
	PowerUsage           *float64       `json:"power_usage"`
}

type Note struct {
	NoteType *string `json:"note_type"`
	Content  *string `json:"note"`
}
