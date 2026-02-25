package types

import "time"

type BiosData struct {
	Tagnumber   *int64  `json:"tagnumber"`
	BiosVersion *string `json:"bios_version"`
	BiosUpdated *bool   `json:"bios_updated"`
	BiosDate    *string `json:"bios_date"`
	TpmVersion  *string `json:"tpm_version"`
}

type OsData struct {
	Tagnumber       *int64         `json:"tagnumber"`
	OsInstalled     *bool          `json:"os_installed"`
	OsName          *string        `json:"os_name"`
	OsInstalledTime *time.Time     `json:"os_installed_time"`
	TPMversion      *string        `json:"tpm_version"`
	BootTime        *time.Duration `json:"boot_time"`
}
