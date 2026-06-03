package types

import "time"

type ClientInfoResponse struct {
	AcquiredDate              *time.Time             `json:"AcquiredDate"`
	AdminUsers                *[]string              `json:"AdminUsers"`
	Building                  *string                `json:"Building"`
	CheckoutDate              *time.Time             `json:"CheckoutDate"`
	CheckoutLog               *[]CheckoutLogResponse `json:"CheckoutLog"`
	ClientImages              *[]ImageManifestResponse   `json:"ClientImages"`
	ClientNote                *string                `json:"ClientNote"`
	ClientStatus              *string                `json:"ClientStatus"`
	ClientUUID                *string                `json:"ClientUUID"`
	CloneCompleted            *bool                  `json:"CloneCompleted"`
	CloneImageName            *string                `json:"CloneImageName"`
	CloneJobDuration          *float64               `json:"CloneJobDuration"`
	ComputerName              *string                `json:"ComputerName"`
	CustomerName              *string                `json:"CustomerName"`
	DepartmentName            *string                `json:"DepartmentName"`
	DiskRemoved               *bool                  `json:"DiskRemoved"`
	EraseCompleted            *bool                  `json:"EraseCompleted"`
	EraseJobDuration          *float64               `json:"EraseJobDuration"`
	EraseMode                 *string                `json:"EraseMode"`
	FileCount                 *int64                 `json:"FileCount"`
	IsDiskEncrypted           *bool                  `json:"IsDiskEncrypted"`
	IsBroken                  *bool                  `json:"IsBroken"`
	IsCheckedOut              *bool                  `json:"IsCheckedOut"`
	IsIntuneJoined            *bool                  `json:"IsIntuneJoined"`
	JobLog                    *[]JobLogResponse      `json:"JobLog"`
	JobStartTime              *time.Time             `json:"JobStartTime"`
	LastOSEntryTime           *time.Time             `json:"LastOSEntryTime"`
	Location                  *string                `json:"Location"`
	LocationEntryTime         *time.Time             `json:"LocationEntryTime"`
	LocationLog               *[]LocationLogResponse `json:"LocationLog"`
	OSInstalled               *bool                  `json:"OSInstalled"`
	OSName                    *string                `json:"OSName"`
	OSVersion                 *string                `json:"OSVersion"`
	OUName                    *string                `json:"OUName"`
	PropertyCustodian         *string                `json:"PropertyCustodian"`
	RetiredDate               *time.Time             `json:"RetiredDate"`
	ReturnDate                *time.Time             `json:"ReturnDate"`
	Room                      *string                `json:"Room"`
	SystemSerial              *string                `json:"SystemSerial"`
	Tagnumber                 *int64                 `json:"Tagnumber"`
	LastHardwareCheck         *time.Time             `json:"LastHardwareCheck"`
	DiskHealthPcnt            *float64               `json:"DiskHealthPcnt"`
	BatteryHealthPcnt         *float64               `json:"BatteryHealthPcnt"`
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
	BatteryManufactureDate    *time.Time             `json:"BatteryManufactureDate"`
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
