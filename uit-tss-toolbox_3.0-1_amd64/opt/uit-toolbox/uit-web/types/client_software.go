package types

import "time"

type ClientSoftwareView struct {
	Tagnumber       *int64     `json:"tagnumber"`
	OsInstalled     *bool      `json:"os_installed"`
	OsName          *string    `json:"os_name"`
	OsInstalledTime *time.Time `json:"os_installed_time"`
	TPMversion      *string    `json:"tpm_version"`
	BIOSVersion     *string    `json:"bios_version"`
	BIOSUpdated     *bool      `json:"bios_updated"`
	BIOSDate        *string    `json:"bios_date"`
}

type OsData struct {
	Tagnumber       int64         `json:"tagnumber"`
	OsInstalled     *bool         `json:"os_installed"`
	OsName          string        `json:"os_name"`
	OsInstalledTime time.Time     `json:"os_installed_time"`
	TPMversion      string        `json:"tpm_version"`
	BootTime        time.Duration `json:"boot_time"`
}
