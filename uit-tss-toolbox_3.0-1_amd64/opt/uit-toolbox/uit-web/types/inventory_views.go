package types

import (
	"time"
)

// Error levels
type ConfigurationErrorLevel int

const (
	Info ConfigurationErrorLevel = iota
	Warning
	Error
)

var ConfigurationErrorLevels = map[ConfigurationErrorLevel]string{
	Info:    "info",
	Warning: "warning",
	Error:   "error",
}

func (cel ConfigurationErrorLevel) String() string {
	if errorLevel, ok := ConfigurationErrorLevels[cel]; ok {
		return errorLevel
	}
	return ""
}

// Error types
type ConfigurationErrorType int

const (
	HardwareIssueType ConfigurationErrorType = iota
	FirmwareIssueType
	SoftwareIssueType
	OtherIssueType
)

var ClientConfigurationErrorTypes = map[ConfigurationErrorType]string{
	HardwareIssueType: "hardware",
	SoftwareIssueType: "software",
	FirmwareIssueType: "firmware",
	OtherIssueType:    "other",
}

func (t ConfigurationErrorType) String() string {
	if errorType, ok := ClientConfigurationErrorTypes[t]; ok {
		return errorType
	}
	return ""
}

// Error strings
type ClientConfigurationError int

const (
	IsBroken ClientConfigurationError = iota
	DiskNotRemoved
	DomainNotJoined
	IntuneNotEnrolled
	AdminUsersMissing
	BIOSOutdated
	OSNotInstalled
	OSInvalidData
	DiskNotEncrypted
	OSOutdated
	NeedsHardwareCheck
	NeedsErasing
	MissingRequiredHardwareInfo
	MissingRequiredSoftwareInfo
	MissingImages
)

var ClientConfigurationErrorStrings = map[ClientConfigurationError]string{
	IsBroken:                    "Client hardware is broken",
	DiskNotRemoved:              "Disk needs to be removed",
	DomainNotJoined:             "Not joined to domain",
	IntuneNotEnrolled:           "Not enrolled in Intune",
	AdminUsersMissing:           "Missing admin users",
	BIOSOutdated:                "BIOS is outdated",
	OSNotInstalled:              "OS is not installed",
	OSOutdated:                  "OS is outdated",
	OSInvalidData:               "OS data is invalid",
	DiskNotEncrypted:            "Disk is not fully encrypted",
	NeedsHardwareCheck:          "Needs hardware check",
	NeedsErasing:                "Needs erasing",
	MissingRequiredHardwareInfo: "Missing required hardware information",
	MissingRequiredSoftwareInfo: "Missing required OS information",
	MissingImages:               "Missing images",
}

func (s ClientConfigurationError) String() string {
	if errString, ok := ClientConfigurationErrorStrings[s]; ok {
		return errString
	}
	return ""
}

func (cce ClientConfigurationError) ToConfigErrorResponse() ClientConfigErrorMessageResponse {
	if response, ok := ClientConfigurationErrors[cce]; ok {
		return response
	}
	return ClientConfigErrorMessageResponse{}
}

type ClientConfigErrorMessageResponse struct {
	ErrorLevel   string `json:"error_level"`
	ErrorType    string `json:"error_type"`
	ErrorMessage string `json:"error_message"`
}

var ClientConfigurationErrors = map[ClientConfigurationError]ClientConfigErrorMessageResponse{
	IsBroken:                    {ErrorLevel: Warning.String(), ErrorType: HardwareIssueType.String(), ErrorMessage: IsBroken.String()},
	DiskNotRemoved:              {ErrorLevel: Error.String(), ErrorType: HardwareIssueType.String(), ErrorMessage: DiskNotRemoved.String()},
	DomainNotJoined:             {ErrorLevel: Error.String(), ErrorType: SoftwareIssueType.String(), ErrorMessage: DomainNotJoined.String()},
	IntuneNotEnrolled:           {ErrorLevel: Warning.String(), ErrorType: SoftwareIssueType.String(), ErrorMessage: IntuneNotEnrolled.String()},
	AdminUsersMissing:           {ErrorLevel: Warning.String(), ErrorType: SoftwareIssueType.String(), ErrorMessage: AdminUsersMissing.String()},
	BIOSOutdated:                {ErrorLevel: Warning.String(), ErrorType: FirmwareIssueType.String(), ErrorMessage: BIOSOutdated.String()},
	OSNotInstalled:              {ErrorLevel: Info.String(), ErrorType: SoftwareIssueType.String(), ErrorMessage: OSNotInstalled.String()},
	OSOutdated:                  {ErrorLevel: Warning.String(), ErrorType: SoftwareIssueType.String(), ErrorMessage: OSOutdated.String()},
	OSInvalidData:               {ErrorLevel: Error.String(), ErrorType: SoftwareIssueType.String(), ErrorMessage: OSInvalidData.String()},
	DiskNotEncrypted:            {ErrorLevel: Warning.String(), ErrorType: FirmwareIssueType.String(), ErrorMessage: DiskNotEncrypted.String()},
	NeedsHardwareCheck:          {ErrorLevel: Warning.String(), ErrorType: HardwareIssueType.String(), ErrorMessage: NeedsHardwareCheck.String()},
	NeedsErasing:                {ErrorLevel: Warning.String(), ErrorType: SoftwareIssueType.String(), ErrorMessage: NeedsErasing.String()},
	MissingRequiredHardwareInfo: {ErrorLevel: Error.String(), ErrorType: HardwareIssueType.String(), ErrorMessage: MissingRequiredHardwareInfo.String()},
	MissingRequiredSoftwareInfo: {ErrorLevel: Error.String(), ErrorType: SoftwareIssueType.String(), ErrorMessage: MissingRequiredSoftwareInfo.String()},
	MissingImages:               {ErrorLevel: Info.String(), ErrorType: OtherIssueType.String(), ErrorMessage: MissingImages.String()},
}

type InventoryTableRow struct {
	Tagnumber           *int64                             `json:"tagnumber"`
	SystemSerial        *string                            `json:"system_serial"`
	Location            *string                            `json:"location"`
	LocationFormatted   *string                            `json:"location_formatted"`
	Building            *string                            `json:"building"`
	Room                *string                            `json:"room"`
	SystemManufacturer  *string                            `json:"system_manufacturer"`
	SystemModel         *string                            `json:"system_model"`
	DeviceType          *string                            `json:"device_type"`
	DeviceTypeFormatted *string                            `json:"device_type_formatted"`
	Department          *string                            `json:"department_name"`
	DepartmentFormatted *string                            `json:"department_formatted"`
	ADDomain            *string                            `json:"ad_domain"`
	AdminUsers          *[]string                          `json:"admin_users"`
	IsIntuneJoined      *bool                              `json:"is_intune_joined"`
	DomainFormatted     *string                            `json:"ad_domain_formatted"`
	OsInstalled         *bool                              `json:"os_installed"`
	OsName              *string                            `json:"os_name"`
	OsVersion           *string                            `json:"os_version"`
	LatestOsVersion     *string                            `json:"latest_os_version"`
	IsDiskEncrypted     *bool                              `json:"windows_bitlocker_enabled"`
	LastHardwareCheck   *time.Time                         `json:"last_hardware_check"`
	BIOSUpdated         *bool                              `json:"bios_updated"`
	BIOSVersion         *string                            `json:"bios_version"`
	Status              *string                            `json:"status"`
	StatusFormatted     *string                            `json:"status_formatted"`
	IsBroken            *bool                              `json:"is_broken"`
	DiskRemoved         *bool                              `json:"disk_removed"`
	RetiredDate         *time.Time                         `json:"retired_date"`
	IsCheckedOut        *bool                              `json:"checkout_bool"`
	Note                *string                            `json:"note"`
	LastUpdated         *time.Time                         `json:"last_updated"`
	FileCount           *int64                             `json:"file_count"`
	ClientErrors        []ClientConfigErrorMessageResponse `json:"client_configuration_errors,omitempty"`
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
	CustomerName       *string    `json:"customer_name"`
	Note               *string    `json:"note"`
	FileCount          *int64     `json:"file_count"`
}
