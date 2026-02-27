package types

import "time"

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
	Domain              *string    `json:"ad_domain"`
	DomainFormatted     *string    `json:"ad_domain_formatted"`
	OsInstalled         *bool      `json:"os_installed"`
	OsName              *string    `json:"os_name"`
	Status              *string    `json:"status"`
	Broken              *bool      `json:"is_broken"`
	Note                *string    `json:"note"`
	LastUpdated         *time.Time `json:"last_updated"`
	FileCount           *int64     `json:"file_count"`
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
	Domain             *string    `json:"ad_domain"`
	PropertyCustodian  *string    `json:"property_custodian"`
	AcquiredDate       *time.Time `json:"acquired_date"`
	RetiredDate        *time.Time `json:"retired_date"`
	Broken             *bool      `json:"is_broken"`
	DiskRemoved        *bool      `json:"disk_removed"`
	LastHardwareCheck  *time.Time `json:"last_hardware_check"`
	ClientStatus       *string    `json:"status"`
	CheckoutBool       *bool      `json:"checkout_bool"`
	CheckoutDate       *time.Time `json:"checkout_date"`
	ReturnDate         *time.Time `json:"return_date"`
	Note               *string    `json:"note"`
	FileCount          *int64     `json:"file_count"`
}
