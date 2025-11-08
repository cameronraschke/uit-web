package database

import "time"

type ClientLookup struct {
	Tagnumber    int    `json:"tagnumber"`
	SystemSerial string `json:"system_serial"`
}

type HardwareData struct {
	Tagnumber               int    `json:"tagnumber"`
	SystemSerial            string `json:"system_serial"`
	EthernetMAC             string `json:"ethernet_mac"`
	WifiMac                 string `json:"wifi_mac"`
	SystemModel             string `json:"system_model"`
	SystemUUID              string `json:"system_uuid"`
	SystemSKU               string `json:"system_sku"`
	ChassisType             string `json:"chassis_type"`
	MotherboardManufacturer string `json:"motherboard_manufacturer"`
	MotherboardSerial       string `json:"motherboard_serial"`
	SystemManufacturer      string `json:"system_manufacturer"`
}

type BiosData struct {
	Tagnumber   int    `json:"tagnumber"`
	BiosVersion string `json:"bios_version"`
	BiosUpdated bool   `json:"bios_updated"`
	BiosDate    string `json:"bios_date"`
	TpmVersion  string `json:"tpm_version"`
}

type OsData struct {
	Tagnumber       int           `json:"tagnumber"`
	OsInstalled     bool          `json:"os_installed"`
	OsName          string        `json:"os_name"`
	OsInstalledTime time.Time     `json:"os_installed_time"`
	TPMversion      string        `json:"tpm_version"`
	BootTime        time.Duration `json:"boot_time"`
}

type ActiveJobs struct {
	Tagnumber     int    `json:"tagnumber"`
	QueuedJob     string `json:"job_queued"`
	JobActive     bool   `json:"job_active"`
	QueuePosition int    `json:"queue_position"`
}

type AvailableJobs struct {
	Tagnumber    int  `json:"tagnumber"`
	JobAvailable bool `json:"job_available"`
}

type JobQueueOverview struct {
	TotalQueuedJobs         int `json:"total_queued_jobs"`
	TotalActiveJobs         int `json:"total_active_jobs"`
	TotalActiveBlockingJobs int `json:"total_active_blocking_jobs"`
}

type DashboardInventorySummary struct {
	SystemModel          string `json:"system_model"`
	SystemModelCount     int    `json:"system_model_count"`
	TotalCheckedOut      int    `json:"total_checked_out"`
	AvailableForCheckout int    `json:"available_for_checkout"`
}

type AllTags struct {
	Tagnumber int `json:"tagnumber"`
}

type InventoryFormAutofill struct {
	Time               *time.Time `json:"time"`
	Tagnumber          *int       `json:"tagnumber"`
	SystemSerial       *string    `json:"system_serial"`
	Location           *string    `json:"location"`
	SystemManufacturer *string    `json:"system_manufacturer"`
	SystemModel        *string    `json:"system_model"`
	Status             *string    `json:"status"`
	Broken             *bool      `json:"broken"`
	DiskRemoved        *bool      `json:"disk_removed"`
	Department         *string    `json:"department"`
	Domain             *string    `json:"domain"`
	Note               *string    `json:"note"`
}

type InventoryUpdateFormInput struct {
	Time               *time.Time `json:"time"`
	Tagnumber          *int64     `json:"tagnumber"`
	SystemSerial       *string    `json:"system_serial"`
	Location           *string    `json:"location"`
	Broken             *bool      `json:"is_broken"`
	DiskRemoved        *bool      `json:"disk_removed"`
	Department         *string    `json:"department"`
	Domain             *string    `json:"domain"`
	Note               *string    `json:"note"`
	Status             *string    `json:"status"`
	SystemManufacturer *string    `json:"system_manufacturer"`
	SystemModel        *string    `json:"system_model"`
}

type ImageManifest struct {
	Time              *time.Time `json:"time"`
	Tagnumber         *int64     `json:"tagnumber"`
	Name              *string    `json:"name"`
	UUID              *string    `json:"uuid"`
	Filepath          *string    `json:"filepath"`
	ThumbnailFilepath *string    `json:"thumbnail_filepath"`
	URL               *string    `json:"url"`
	Width             *int       `json:"width"`
	Height            *int       `json:"height"`
	Size              *int64     `json:"size"`
	Hidden            *bool      `json:"hidden"`
	PrimaryImage      *bool      `json:"primary_image"`
	Note              *string    `json:"note"`
	FileType          *string    `json:"file_type"`
}

type InventoryTableData struct {
	Tagnumber           *int64     `json:"tagnumber"`
	SystemSerial        *string    `json:"system_serial"`
	Location            *string    `json:"location"`
	LocationFormatted   *string    `json:"location_formatted"`
	SystemManufacturer  *string    `json:"system_manufacturer"`
	SystemModel         *string    `json:"system_model"`
	Department          *string    `json:"department"`
	DepartmentFormatted *string    `json:"department_formatted"`
	Domain              *string    `json:"domain"`
	DomainFormatted     *string    `json:"domain_formatted"`
	OsInstalled         *bool      `json:"os_installed"`
	OsName              *string    `json:"os_name"`
	Status              *string    `json:"status"`
	Broken              *bool      `json:"broken"`
	Note                *string    `json:"note"`
	LastUpdated         *time.Time `json:"last_updated"`
}

type InventoryFilterOptions struct {
	Tagnumber          *int64  `json:"tagnumber"`
	SystemSerial       *string `json:"system_serial"`
	Location           *string `json:"location"`
	SystemManufacturer *string `json:"system_manufacturer"`
	SystemModel        *string `json:"system_model"`
	Department         *string `json:"department"`
	Domain             *string `json:"domain"`
	Status             *string `json:"status"`
	Broken             *bool   `json:"broken"`
	HasImages          *bool   `json:"has_images"`
}

type ManufacturersAndModels struct {
	SystemModel                 string `json:"system_model"`
	SystemModelFormatted        string `json:"system_model_formatted"`
	SystemManufacturer          string `json:"system_manufacturer"`
	SystemManufacturerFormatted string `json:"system_manufacturer_formatted"`
}
