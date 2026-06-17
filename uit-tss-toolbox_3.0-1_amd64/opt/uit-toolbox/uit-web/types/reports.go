package types

import "time"

type ClientLookupRow struct {
	Tagnumber          *int64     `json:"tagnumber"`
	SystemSerial       *string    `json:"system_serial"`
	ClientUUID         *string    `json:"client_uuid"`
	LastInventoryEntry *time.Time `json:"last_inventory_entry,omitempty"`
}

type AllDeviceTypesRow struct {
	DeviceType          string `json:"device_type"`
	DeviceTypeFormatted string `json:"device_type_formatted"`
	DeviceMetaCategory  string `json:"device_meta_category"`
	DeviceTypeCount     int64  `json:"device_type_count"`
	SortOrder           int64  `json:"sort_order"`
}

type AllBuildingsAndRooms struct {
	BuildingName *string `json:"building_name"`
	RoomName     *string `json:"room_name"`
	ClientCount  *int64  `json:"client_count"`
}

type AllManufacturersAndModelsRow struct {
	SystemManufacturer      *string `json:"system_manufacturer"`
	SystemManufacturerCount *int64  `json:"system_manufacturer_count"`
	SystemModel             *string `json:"system_model"`
	SystemModelCount        *int64  `json:"system_model_count"`
}

type AllDomainsRow struct {
	DomainName          *string `json:"ad_domain"`
	DomainNameFormatted *string `json:"ad_domain_formatted"`
	DomainSortOrder     *int64  `json:"domain_sort_order"`
	ClientCount         *int64  `json:"client_count"`
}

type AllDepartmentsRow struct {
	DepartmentName            *string `json:"department_name"`
	DepartmentNameFormatted   *string `json:"department_name_formatted"`
	DepartmentSortOrder       *int64  `json:"department_sort_order"`
	OrganizationName          *string `json:"organization_name"`
	OrganizationNameFormatted *string `json:"organization_name_formatted"`
	OrganizationSortOrder     *int64  `json:"organization_sort_order"`
	ClientCount               *int64  `json:"client_count"`
}

type AllJobsRow struct {
	JobName         *string `json:"job_name"`
	JobNameReadable *string `json:"job_name_readable"`
	JobSortOrder    *int64  `json:"job_sort_order"`
	JobHidden       *bool   `json:"job_hidden"`
}

type AllLocationsRow struct {
	Timestamp         *time.Time `json:"timestamp"`
	Location          *string    `json:"location"`
	LocationFormatted *string    `json:"location_formatted"`
	LocationCount     *int64     `json:"location_count"`
}
