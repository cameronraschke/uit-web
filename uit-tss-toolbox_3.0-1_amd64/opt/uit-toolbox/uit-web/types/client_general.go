package types

import "time"

type ClientInfoResponse struct {
	Tagnumber                 *int64                 `json:"Tagnumber"`
	SystemSerial              *string                `json:"SystemSerial"`
	ClientUUID                *string                `json:"ClientUUID"`
	LocationEntryTime         *time.Time             `json:"LocationEntryTime"`
	Location                  *string                `json:"Location"`
	Building                  *string                `json:"Building"`
	Room                      *string                `json:"Room"`
	DepartmentName            *string                `json:"DepartmentName"`
	PropertyCustodian         *string                `json:"PropertyCustodian"`
	AcquiredDate              *time.Time             `json:"AcquiredDate"`
	RetiredDate               *time.Time             `json:"RetiredDate"`
	ClientStatus              *string                `json:"ClientStatus"`
	IsBroken                  *bool                  `json:"IsBroken"`
	DiskRemoved               *bool                  `json:"DiskRemoved"`
	ClientNote                *string                `json:"ClientNote"`
	LocationLog               *[]LocationLogResponse `json:"LocationLog"`
	JobStartTime              *time.Time             `json:"JobStartTime"`
	CloneCompleted            *bool                  `json:"CloneCompleted"`
	CloneJobDuration          *float64               `json:"CloneJobDuration"`
	CloneImageName            *string                `json:"CloneImageName"`
	EraseCompleted            *bool                  `json:"EraseCompleted"`
	EraseJobDuration          *float64               `json:"EraseJobDuration"`
	EraseMode                 *string                `json:"EraseMode"`
	JobLog                    *[]JobLogResponse      `json:"JobLog"`
	IsCheckedOut              *bool                  `json:"IsCheckedOut"`
	CheckoutDate              *time.Time             `json:"CheckoutDate"`
	ReturnDate                *time.Time             `json:"ReturnDate"`
	CustomerName              *string                `json:"CustomerName"`
	CheckoutLog               *[]CheckoutLogResponse `json:"CheckoutLog"`
	FileCount                 *int64                 `json:"FileCount"`
	ClientImages              *[]ImageManifestView   `json:"ClientImages"`
	LastOSEntryTime           *time.Time             `json:"LastOSEntryTime"`
	OSInstalled               *bool                  `json:"OSInstalled"`
	OSName                    *string                `json:"OSName"`
	OSVersion                 *string                `json:"OSVersion"`
	ComputerName              *string                `json:"ComputerName"`
	OUName                    *string                `json:"OUName"`
	ADAdminUsers              *[]string              `json:"ADAdminUsers"`
	IsIntuneJoined            *bool                  `json:"IsIntuneJoined"`
	IsBitlockerEnabled        *bool                  `json:"IsBitlockerEnabled"`
	LastHardwareCheck         *time.Time             `json:"LastHardwareCheck"`
	DiskHealthPcnt            *string                `json:"DiskHealthPcnt"`
	BatteryHealthPcnt         *string                `json:"BatteryHealthPcnt"`
	DeviceType                *string                `json:"DeviceType"`
	BIOSVersion               *string                `json:"BIOSVersion"`
	BIOSReleaseDate           *string                `json:"BIOSReleaseDate"`
	EthernetMAC               *string                `json:"EthernetMAC"`
	WiFiMAC                   *string                `json:"WiFiMAC"`
	TPMVersion                *string                `json:"TPMVersion"`
	DiskModel                 *string                `json:"DiskModel"`
	DiskType                  *string                `json:"DiskType"`
	DiskSizeKB                *int64                 `json:"DiskSizeKB"`
	DiskSerial                *string                `json:"DiskSerial"`
	DiskWritesKB              *int64                 `json:"DiskWritesKB"`
	DiskReadsKB               *int64                 `json:"DiskReadsKB"`
	DiskPowerOnHours          *int64                 `json:"DiskPowerOnHours"`
	DiskErrors                *int64                 `json:"DiskErrors"`
	DiskPowerCycles           *int64                 `json:"DiskPowerCycles"`
	DiskFirmware              *string                `json:"DiskFirmware"`
	BatteryManufacturer       *string                `json:"BatteryManufacturer"`
	BatteryModel              *string                `json:"BatteryModel"`
	BatterySerial             *string                `json:"BatterySerial"`
	BatteryManufactureDate    *string                `json:"BatteryManufactureDate"`
	BatteryDesignCapacity     *float64               `json:"BatteryDesignCapacity"`
	BatteryCurrentMaxCapacity *float64               `json:"BatteryCurrentMaxCapacity"`
	BatteryChargeCycles       *int64                 `json:"BatteryChargeCycles"`
	MemorySerial              *string                `json:"MemorySerial"`
	MemoryCapacityKB          *int64                 `json:"MemoryCapacityKB"`
	MemorySpeedMHz            *int64                 `json:"MemorySpeedMHz"`
	SystemManufacturer        *string                `json:"SystemManufacturer"`
	SystemModel               *string                `json:"SystemModel"`
	SystemSKU                 *string                `json:"SystemSKU"`
	CPUManufacturer           *string                `json:"CPUManufacturer"`
	CPUModel                  *string                `json:"CPUModel"`
	CPUMaxSpeedMhz            *int64                 `json:"CPUMaxSpeedMHz"`
	CPUCoreCount              *int64                 `json:"CPUCoreCount"`
	CPUThreadCount            *int64                 `json:"CPUThreadCount"`
}

type CheckoutLogResponse struct {
	TransactionUUID *string    `json:"transaction_uuid"`
	Tagnumber       *int64     `json:"tagnumber"`
	IsCheckedOut    *bool      `json:"is_checked_out"`
	CheckoutDate    *time.Time `json:"checkout_date"`
	ReturnDate      *time.Time `json:"return_date"`
	CustomerName    *string    `json:"customer_name"`
}

type LocationLogResponse struct {
	TransactionUUID *string    `json:"transaction_uuid"`
	Time            *time.Time `json:"time"`
	Location        *string    `json:"location"`
}

type JobLogResponse struct {
	TransactionUUID  *string    `json:"transaction_uuid"`
	JobTime          *time.Time `json:"job_time"`
	CloneCompleted   *bool      `json:"clone_completed"`
	CloneJobDuration *float64   `json:"clone_job_duration"`
	CloneImageName   *string    `json:"clone_image_name"`
	EraseCompleted   *bool      `json:"erase_completed"`
	EraseJobDuration *float64   `json:"erase_job_duration"`
	EraseMode        *string    `json:"erase_mode"`
}
