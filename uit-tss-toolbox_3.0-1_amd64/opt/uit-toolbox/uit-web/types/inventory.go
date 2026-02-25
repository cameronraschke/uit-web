package types

import "time"

type ImageManifest struct {
	Time              *time.Time `json:"time"`
	Tagnumber         *int64     `json:"tagnumber"`
	UUID              *string    `json:"uuid"`
	SHA256Hash        *[]uint8   `json:"sha256_hash"`
	FileName          *string    `json:"filename"`
	FilePath          *string    `json:"filepath"`
	ThumbnailFilePath *string    `json:"thumbnail_filepath"`
	FileSize          *int64     `json:"file_size"`
	MimeType          *string    `json:"mime_type"`
	ExifTimestamp     *time.Time `json:"exif_timestamp"`
	ResolutionX       *int64     `json:"resolution_x"`
	ResolutionY       *int64     `json:"resolution_y"`
	URL               *string    `json:"url"`
	Hidden            *bool      `json:"hidden"`
	Pinned            *bool      `json:"pinned"`
	Note              *string    `json:"note"`
	FileType          *string    `json:"file_type"`
}

type InventoryTableData struct {
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
	Domain              *string    `json:"ad_domain"`
	DomainFormatted     *string    `json:"ad_domain_formatted"`
	OsInstalled         *bool      `json:"os_installed"`
	OsName              *string    `json:"os_name"`
	Status              *string    `json:"status"`
	Broken              *bool      `json:"is_broken"`
	Note                *string    `json:"note"`
	LastUpdated         *time.Time `json:"last_updated"`
}
