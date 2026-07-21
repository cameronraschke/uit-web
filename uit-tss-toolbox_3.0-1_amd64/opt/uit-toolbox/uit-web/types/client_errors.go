package types

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
	DiskNotRemoved ClientConfigurationError = iota
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
	MissingOptionalInfo
	MissingRequiredGeneralInfo
	MissingRequiredHardwareInfo
	MissingRequiredSoftwareInfo
	MissingImages
	SecureBootNotEnabled
	Missing2023SecureBootCA
)

var ClientConfigurationErrorStrings = map[ClientConfigurationError]string{
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
	MissingOptionalInfo:         "Missing optional information",
	MissingRequiredGeneralInfo:  "Missing required general information",
	MissingRequiredHardwareInfo: "Missing required hardware information",
	MissingRequiredSoftwareInfo: "Missing required OS information",
	MissingImages:               "Missing images",
	SecureBootNotEnabled:        "Secure Boot is not enabled",
	Missing2023SecureBootCA:     "2023 Microsoft Secure Boot CA is missing",
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
	DiskNotRemoved:              {ErrorLevel: Error.String(), ErrorType: HardwareIssueType.String(), ErrorMessage: DiskNotRemoved.String()},
	DomainNotJoined:             {ErrorLevel: Error.String(), ErrorType: SoftwareIssueType.String(), ErrorMessage: DomainNotJoined.String()},
	IntuneNotEnrolled:           {ErrorLevel: Warning.String(), ErrorType: SoftwareIssueType.String(), ErrorMessage: IntuneNotEnrolled.String()},
	AdminUsersMissing:           {ErrorLevel: Warning.String(), ErrorType: SoftwareIssueType.String(), ErrorMessage: AdminUsersMissing.String()},
	BIOSOutdated:                {ErrorLevel: Warning.String(), ErrorType: FirmwareIssueType.String(), ErrorMessage: BIOSOutdated.String()},
	OSNotInstalled:              {ErrorLevel: Info.String(), ErrorType: SoftwareIssueType.String(), ErrorMessage: OSNotInstalled.String()},
	OSOutdated:                  {ErrorLevel: Info.String(), ErrorType: SoftwareIssueType.String(), ErrorMessage: OSOutdated.String()},
	OSInvalidData:               {ErrorLevel: Error.String(), ErrorType: SoftwareIssueType.String(), ErrorMessage: OSInvalidData.String()},
	DiskNotEncrypted:            {ErrorLevel: Warning.String(), ErrorType: FirmwareIssueType.String(), ErrorMessage: DiskNotEncrypted.String()},
	NeedsHardwareCheck:          {ErrorLevel: Warning.String(), ErrorType: HardwareIssueType.String(), ErrorMessage: NeedsHardwareCheck.String()},
	NeedsErasing:                {ErrorLevel: Warning.String(), ErrorType: SoftwareIssueType.String(), ErrorMessage: NeedsErasing.String()},
	MissingOptionalInfo:         {ErrorLevel: Info.String(), ErrorType: OtherIssueType.String(), ErrorMessage: MissingOptionalInfo.String()},
	MissingRequiredGeneralInfo:  {ErrorLevel: Error.String(), ErrorType: OtherIssueType.String(), ErrorMessage: MissingRequiredGeneralInfo.String()},
	MissingRequiredHardwareInfo: {ErrorLevel: Error.String(), ErrorType: HardwareIssueType.String(), ErrorMessage: MissingRequiredHardwareInfo.String()},
	MissingRequiredSoftwareInfo: {ErrorLevel: Error.String(), ErrorType: SoftwareIssueType.String(), ErrorMessage: MissingRequiredSoftwareInfo.String()},
	MissingImages:               {ErrorLevel: Info.String(), ErrorType: OtherIssueType.String(), ErrorMessage: MissingImages.String()},
	SecureBootNotEnabled:        {ErrorLevel: Warning.String(), ErrorType: FirmwareIssueType.String(), ErrorMessage: SecureBootNotEnabled.String()},
	Missing2023SecureBootCA:     {ErrorLevel: Warning.String(), ErrorType: FirmwareIssueType.String(), ErrorMessage: Missing2023SecureBootCA.String()},
}
