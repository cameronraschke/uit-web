package types

import (
	"fmt"
	"time"
)

type ConfigurationErrorCode int

const (
	IsBroken ConfigurationErrorCode = iota
	DiskNotRemoved
	DomainNotJoined
	BIOSOutdated
	OSNotInstalled
	OSOutdated
	NeedsHardwareCheck
	NeedsErasing
	MissingRequiredInfo
	MissingImages
)

var ClientConfigurationErrorCodeToString = map[ConfigurationErrorCode]string{
	IsBroken:            "Client is broken",
	DiskNotRemoved:      "Disk is not removed",
	DomainNotJoined:     "AD domain is not joined",
	BIOSOutdated:        "BIOS is outdated",
	OSNotInstalled:      "OS is not installed",
	OSOutdated:          "OS is outdated",
	NeedsHardwareCheck:  "Needs hardware check",
	NeedsErasing:        "Needs erasing",
	MissingRequiredInfo: "Missing required information",
	MissingImages:       "Missing images",
}

func (c ConfigurationErrorCode) String() string {
	if str, ok := ClientConfigurationErrorCodeToString[c]; ok {
		return str
	}
	return "Unknown error code"
}

func (c *ConfigurationErrorCode) MarshalJSON() ([]byte, error) {
	if c == nil {
		return nil, fmt.Errorf("cannot marshal nil ConfigurationErrorCode")
	}
	return []byte(`"` + c.String() + `"`), nil
}

func (c *ConfigurationErrorCode) ToErrorCode() *int {
	if c == nil {
		return nil
	}
	for code, str := range ClientConfigurationErrorCodeToString {
		if str == c.String() {
			codeCopy := int(code)
			return &codeCopy
		}
	}
	return nil
}

type InventoryTableRow struct {
	Tagnumber           *int64     `json:"tagnumber"`
	SystemSerial        *string    `json:"system_serial"`
	Location            *string    `json:"location"`
	LocationFormatted   *string    `json:"location_formatted"`
	Building            *string    `json:"building"`
	Room                *string    `json:"room"`
	SystemManufacturer  *string    `json:"system_manufacturer"`
	SystemModel         *string    `json:"system_model"`
	DeviceType          *string    `json:"device_type"`
	DeviceTypeFormatted *string    `json:"device_type_formatted"`
	Department          *string    `json:"department_name"`
	DepartmentFormatted *string    `json:"department_formatted"`
	ADDomain            *string    `json:"ad_domain"`
	DomainFormatted     *string    `json:"ad_domain_formatted"`
	OsInstalled         *bool      `json:"os_installed"`
	OsName              *string    `json:"os_name"`
	LastHardwareCheck   *time.Time `json:"last_hardware_check"`
	BIOSUpdated         *bool      `json:"bios_updated"`
	BIOSVersion         *string    `json:"bios_version"`
	Status              *string    `json:"status"`
	IsBroken            *bool      `json:"is_broken"`
	DiskRemoved         *bool      `json:"disk_removed"`
	Note                *string    `json:"note"`
	LastUpdated         *time.Time `json:"last_updated"`
	FileCount           *int64     `json:"file_count"`
	ClientErrors        []string   `json:"client_configuration_errors"`
}

type InventoryFormPrefill struct {
	Time               *time.Time `json:"time"`
	Tagnumber          *int64     `json:"tagnumber"`
	SystemSerial       *string    `json:"system_serial"`
	Location           *string    `json:"location"`
	Building           *string    `json:"building"`
	Room               *string    `json:"room"`
	SystemManufacturer *string    `json:"system_manufacturer"`
	SystemModel        *string    `json:"system_model"`
	DeviceType         *string    `json:"device_type"`
	Department         *string    `json:"department_name"`
	ADDomain           *string    `json:"ad_domain"`
	PropertyCustodian  *string    `json:"property_custodian"`
	AcquiredDate       *time.Time `json:"acquired_date"`
	RetiredDate        *time.Time `json:"retired_date"`
	IsBroken           *bool      `json:"is_broken"`
	DiskRemoved        *bool      `json:"disk_removed"`
	LastHardwareCheck  *time.Time `json:"last_hardware_check"`
	ClientStatus       *string    `json:"status"`
	CheckoutBool       *bool      `json:"checkout_bool"`
	CheckoutDate       *time.Time `json:"checkout_date"`
	ReturnDate         *time.Time `json:"return_date"`
	Note               *string    `json:"note"`
	FileCount          *int64     `json:"file_count"`
}
