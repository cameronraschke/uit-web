package types

import "time"

type JobQueueOverview struct {
	TotalQueuedJobs         *int64 `json:"total_queued_jobs"`
	TotalActiveJobs         *int64 `json:"total_active_jobs"`
	TotalActiveBlockingJobs *int64 `json:"total_active_blocking_jobs"`
}

type DashboardInventorySummary struct {
	SystemModel          *string `json:"system_model"`
	SystemModelCount     *int64  `json:"system_model_count"`
	TotalCheckedOut      *int64  `json:"total_checked_out"`
	AvailableForCheckout *int64  `json:"available_for_checkout"`
}

type AllTagsRow struct {
	Tagnumber          *int64     `json:"tagnumber"`
	LastInventoryEntry *time.Time `json:"last_inventory_entry"`
	LastSeen           *time.Time `json:"last_seen"`
}

type ManufacturersAndModels struct {
	SystemManufacturer      *string `json:"system_manufacturer"`
	SystemManufacturerCount *int64  `json:"system_manufacturer_count"`
	SystemModel             *string `json:"system_model"`
	SystemModelCount        *int64  `json:"system_model_count"`
}

type Domain struct {
	DomainName          *string `json:"ad_domain"`
	DomainNameFormatted *string `json:"ad_domain_formatted"`
	DomainSortOrder     *int64  `json:"domain_sort_order"`
}

type Department struct {
	DepartmentName            *string `json:"department_name"`
	DepartmentNameFormatted   *string `json:"department_name_formatted"`
	DepartmentSortOrder       *int64  `json:"department_sort_order"`
	OrganizationName          *string `json:"organization_name"`
	OrganizationNameFormatted *string `json:"organization_name_formatted"`
	OrganizationSortOrder     *int64  `json:"organization_sort_order"`
}

type AllJobs struct {
	JobName         *string `json:"job_name"`
	JobNameReadable *string `json:"job_name_readable"`
	JobSortOrder    *int64  `json:"job_sort_order"`
	JobHidden       *bool   `json:"job_hidden"`
}

type AllLocations struct {
	Timestamp         *time.Time `json:"timestamp"`
	Location          *string    `json:"location"`
	LocationFormatted *string    `json:"location_formatted"`
	LocationCount     *int64     `json:"location_count"`
}

type ClientReport struct {
	Tagnumber              int64     `json:"tagnumber"`
	BatteryHealthPcnt      float64   `json:"battery_health_pcnt"`
	BatteryHealthStdDev    float64   `json:"battery_health_stddev"`
	BatteryHealthTimestamp time.Time `json:"battery_health_timestamp"`
}
