package types

import (
	"fmt"
	"strings"
	"time"
)

type ClientInfoResponse struct {
	AcquiredDate              *time.Time              `json:"AcquiredDate"`
	AdminUsers                []string                `json:"AdminUsers"`
	Building                  *string                 `json:"Building"`
	IsCheckedOut              *bool                   `json:"IsCheckedOut"`
	CheckoutDate              *time.Time              `json:"CheckoutDate"`
	CheckoutLog               []CheckoutLogResponse   `json:"CheckoutLog"`
	ClientImages              []ImageManifestResponse `json:"ClientImages"`
	ClientNote                *string                 `json:"ClientNote"`
	ClientStatus              *string                 `json:"ClientStatus"`
	ClientUUID                *string                 `json:"ClientUUID"`
	CloneCompleted            *bool                   `json:"CloneCompleted"`
	CloneImageName            *string                 `json:"CloneImageName"`
	CloneJobDuration          *float64                `json:"CloneJobDuration"`
	ComputerName              *string                 `json:"ComputerName"`
	CustomerName              *string                 `json:"CustomerName"`
	DepartmentName            *string                 `json:"DepartmentName"`
	DiskRemoved               *bool                   `json:"DiskRemoved"`
	SecureBootEnabled         *bool                   `json:"SecureBootEnabled"`
	EraseCompleted            *bool                   `json:"EraseCompleted"`
	EraseJobDuration          *float64                `json:"EraseJobDuration"`
	EraseMode                 *string                 `json:"EraseMode"`
	FileCount                 *int64                  `json:"FileCount"`
	IsDiskEncrypted           *bool                   `json:"IsDiskEncrypted"`
	IsBroken                  *bool                   `json:"IsBroken"`
	IsIntuneJoined            *bool                   `json:"IsIntuneJoined"`
	JobLog                    []JobLogResponse        `json:"JobLog"`
	JobStartTime              *time.Time              `json:"JobStartTime"`
	LastOSEntryTime           *time.Time              `json:"LastOSEntryTime"`
	Location                  *string                 `json:"Location"`
	LocationEntryTime         *time.Time              `json:"LocationEntryTime"`
	LocationLog               []LocationLogResponse   `json:"LocationLog"`
	OSInstalled               *bool                   `json:"OSInstalled"`
	OSName                    *string                 `json:"OSName"`
	OSVersion                 *string                 `json:"OSVersion"`
	OUName                    *string                 `json:"OUName"`
	PropertyCustodian         *string                 `json:"PropertyCustodian"`
	RetiredDate               *time.Time              `json:"RetiredDate"`
	ReturnDate                *time.Time              `json:"ReturnDate"`
	Room                      *string                 `json:"Room"`
	SystemSerial              *string                 `json:"SystemSerial"`
	Tagnumber                 *int64                  `json:"Tagnumber"`
	LastHardwareCheck         *time.Time              `json:"LastHardwareCheck"`
	DiskHealthPcnt            *float64                `json:"DiskHealthPcnt"`
	BatteryHealthPcnt         *float64                `json:"BatteryHealthPcnt"`
	DeviceType                *string                 `json:"DeviceType"`
	BIOSVersion               *string                 `json:"BIOSVersion"`
	BIOSReleaseDate           *time.Time              `json:"BIOSReleaseDate"`
	EthernetMAC               *string                 `json:"EthernetMAC"`
	WiFiMAC                   *string                 `json:"WiFiMAC"`
	TPMVersion                *string                 `json:"TPMVersion"`
	DiskModel                 *string                 `json:"DiskModel"`
	DiskType                  *string                 `json:"DiskType"`
	DiskSizeKB                *int64                  `json:"DiskSizeKB"`
	DiskSerial                *string                 `json:"DiskSerial"`
	DiskWritesKB              *int64                  `json:"DiskWritesKB"`
	DiskReadsKB               *int64                  `json:"DiskReadsKB"`
	DiskPowerOnHours          *int64                  `json:"DiskPowerOnHours"`
	DiskErrors                *int64                  `json:"DiskErrors"`
	DiskPowerCycles           *int64                  `json:"DiskPowerCycles"`
	DiskFirmware              *string                 `json:"DiskFirmware"`
	BatteryManufacturer       *string                 `json:"BatteryManufacturer"`
	BatteryModel              *string                 `json:"BatteryModel"`
	BatterySerial             *string                 `json:"BatterySerial"`
	BatteryManufactureDate    *time.Time              `json:"BatteryManufactureDate"`
	BatteryDesignCapacity     *float64                `json:"BatteryDesignCapacity"`
	BatteryCurrentMaxCapacity *float64                `json:"BatteryCurrentMaxCapacity"`
	BatteryChargeCycles       *int64                  `json:"BatteryChargeCycles"`
	MemorySerial              []string                `json:"MemorySerial"`
	MemoryCapacityKB          *int64                  `json:"MemoryCapacityKB"`
	MemorySpeedMHz            *int64                  `json:"MemorySpeedMHz"`
	SystemManufacturer        *string                 `json:"SystemManufacturer"`
	SystemModel               *string                 `json:"SystemModel"`
	SystemSKU                 *string                 `json:"SystemSKU"`
	CPUManufacturer           *string                 `json:"CPUManufacturer"`
	CPUModel                  *string                 `json:"CPUModel"`
	CPUMaxSpeedMhz            *int64                  `json:"CPUMaxSpeedMHz"`
	CPUCoreCount              *int64                  `json:"CPUCoreCount"`
	CPUThreadCount            *int64                  `json:"CPUThreadCount"`
}

type CheckoutLogResponse struct {
	CheckoutDate    *time.Time `json:"checkout_date"`
	CustomerName    *string    `json:"customer_name"`
	IsCheckedOut    *bool      `json:"is_checked_out"`
	ReturnDate      *time.Time `json:"return_date"`
	Tagnumber       *int64     `json:"tagnumber"`
	TransactionUUID *string    `json:"transaction_uuid"`
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

type DiskImageNameRequest struct {
	SystemModel *string `json:"system_model"`
}

func (r *DiskImageNameRequest) Validate() error {
	if r.SystemModel == nil || strings.TrimSpace(*r.SystemModel) == "" {
		return fmt.Errorf("system_model is required in DiskImageNameRequest")
	}
	return nil
}

type DiskImageNameDTO struct {
	SystemModel string `json:"system_model"`
	ImageName   string `json:"disk_image_name"`
}

type DiskImageNameResponse struct {
	SystemModel *string `json:"system_model"`
	ImageName   *string `json:"disk_image_name"`
}
