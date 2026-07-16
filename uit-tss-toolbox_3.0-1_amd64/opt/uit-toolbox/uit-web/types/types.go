package types

import (
	"encoding/json"
	"fmt"
	"net/netip"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

var (
	ISODateRegex = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)
	USADateRegex = regexp.MustCompile(`^\d{2}/\d{2}/\d{4}$`)
)

const (
	minTagnumberValue     = 100000
	maxTagnumberValue     = 999999
	minTagnumberLength    = 6
	maxTagnumberLength    = 6
	minSystemSerialLength = 1
	maxSystemSerialLength = 256
)

type DurationSeconds time.Duration

func (d *DurationSeconds) UnmarshalJSON(b []byte) error {
	var seconds float64
	if err := json.Unmarshal(b, &seconds); err != nil {
		return err
	}
	*d = DurationSeconds(time.Duration(seconds * float64(time.Second)))
	return nil
}

func (d DurationSeconds) MarshalJSON() ([]byte, error) {
	return json.Marshal(time.Duration(d).Seconds())
}

// Helper to get underlying time.Duration
func (d DurationSeconds) Duration() time.Duration {
	return time.Duration(d)
}

func ValidateASCIIStrLen(s *string, minLen int, maxLen int) error {
	if err := ValidateStrLen(s, minLen, maxLen); err != nil {
		return err
	}
	if s != nil && !IsPrintableASCII([]byte(*s)) {
		return fmt.Errorf("string contains non-printable ASCII characters")
	}
	return nil
}

func ValidatePrintableStrLen(s *string, minLen int, maxLen int) error {
	if err := ValidateStrLen(s, minLen, maxLen); err != nil {
		return err
	}
	if s != nil && !IsPrintableUnicodeString(*s) {
		return fmt.Errorf("string contains non-printable Unicode characters")
	}
	return nil
}

func ValidateStrLen(s *string, minLen int, maxLen int) error {
	if s == nil {
		if minLen == 0 {
			return nil
		}
		return fmt.Errorf("string is nil")
	}
	if utf8.RuneCountInString(*s) < minLen || utf8.RuneCountInString(*s) > maxLen {
		return fmt.Errorf("string length must be between %d and %d characters", minLen, maxLen)
	}
	return nil
}

func copyTrimmedStringPtr(value *string) *string {
	if value == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*value)
	return &trimmed
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

func copyTimePtrToUTC(value *time.Time) *time.Time {
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

func IsSystemSerialValid(s *string) error {
	if s == nil || strings.TrimSpace(*s) == "" {
		return fmt.Errorf("system serial is empty or nil")
	}
	// Check length constraints, max length check does not trim spaces
	if utf8.RuneCountInString(*s) > maxSystemSerialLength ||
		utf8.RuneCountInString(strings.TrimSpace(*s)) < minSystemSerialLength {
		return fmt.Errorf("system serial must be between %d and %d characters", minSystemSerialLength, maxSystemSerialLength)
	}
	if !IsPrintableASCII([]byte(*s)) {
		return fmt.Errorf("non-printable ASCII characters in system serial field")
	}
	return nil
}

func IsTagnumberInt64Valid(i *int64) error {
	if i == nil {
		return fmt.Errorf("tagnumber is nil")
	}
	if *i < minTagnumberValue || *i > maxTagnumberValue {
		return fmt.Errorf("tagnumber is out of valid numeric range")
	}
	return nil
}

func IsTagnumberStringValid(str string) error {
	if len(str) == 0 || strings.TrimSpace(str) == "" {
		return fmt.Errorf("tagnumber is empty")
	}
	if !IsNumericAscii([]byte(str)) {
		return fmt.Errorf("tagnumber contains non-numeric ASCII characters")
	}
	if utf8.RuneCountInString(str) > maxTagnumberLength ||
		utf8.RuneCountInString(strings.TrimSpace(str)) < minTagnumberLength {
		return fmt.Errorf("tagnumber does not contain exactly %d characters", minTagnumberLength)
	}
	parsedInt, err := strconv.ParseInt(str, 10, 64)
	if err != nil {
		return fmt.Errorf("tagnumber cannot be parsed as int64: %v", err)
	}
	if err := IsTagnumberInt64Valid(&parsedInt); err != nil {
		return fmt.Errorf("tagnumber is not valid after parsing as int64: %v", err)
	}
	return nil
}

func ConvertAndVerifyTagnumber(tagStr string) (*int64, error) {
	trimmedTagStr := strings.TrimSpace(tagStr)
	validStringErr := IsTagnumberStringValid(trimmedTagStr)
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
	for i := range b {
		char := b[i]
		if char < '0' || char > '9' {
			return false
		}
	}
	return true
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
