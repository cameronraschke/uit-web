package types

import (
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
)

// Request model for ingress of form data
type InventoryUpdateRequest struct {
	Tagnumber          *int64     `json:"tagnumber"`
	SystemSerial       *string    `json:"system_serial"`
	Location           *string    `json:"location"`
	Building           *string    `json:"building"`
	Room               *string    `json:"room"`
	SystemManufacturer *string    `json:"system_manufacturer"`
	SystemModel        *string    `json:"system_model"`
	DeviceType         *string    `json:"device_type"`
	Department         *string    `json:"department_name"`
	ADDomain           *string    `json:"ad_domain"`
	PropertyCustodian  *string    `json:"property_custodian"`
	AcquiredDate       *time.Time `json:"acquired_date"`
	RetiredDate        *time.Time `json:"retired_date"`
	IsBroken           *bool      `json:"is_broken"`
	DiskRemoved        *bool      `json:"disk_removed"`
	LastHardwareCheck  *time.Time `json:"last_hardware_check"`
	ClientStatus       *string    `json:"status"`
	CheckoutBool       *bool      `json:"checkout_bool"`
	CheckoutDate       *time.Time `json:"checkout_date"`
	ReturnDate         *time.Time `json:"return_date"`
	CustomerName       *string    `json:"customer_name"`
	Note               *string    `json:"note"`
}

// InventoryUpdateDTO for inventory update operations after ingress
type InventoryUpdateDTO struct {
	Tagnumber          int64
	SystemSerial       string
	Location           string
	Building           *string
	Room               *string
	SystemManufacturer *string
	SystemModel        *string
	DeviceType         *string
	Department         string
	ADDomain           string
	PropertyCustodian  *string
	AcquiredDate       *time.Time
	RetiredDate        *time.Time
	IsBroken           *bool
	DiskRemoved        *bool
	LastHardwareCheck  *time.Time
	ClientStatus       string
	CheckoutBool       *bool
	CheckoutDate       *time.Time
	ReturnDate         *time.Time
	CustomerName       *string
	Note               *string
}

// Write models for database operations, splits by table
type InventoryLocationWriteModel struct {
	TransactionUUID   uuid.UUID
	Tagnumber         int64
	SystemSerial      string
	Location          string
	Building          *string
	Room              *string
	Department        string
	ADDomain          string
	PropertyCustodian *string
	AcquiredDate      *time.Time
	RetiredDate       *time.Time
	IsBroken          *bool
	DiskRemoved       *bool
	ClientStatus      string
	Note              *string
}

type InventoryHardwareWriteModel struct {
	TransactionUUID    uuid.UUID
	Tagnumber          int64
	SystemManufacturer *string
	SystemModel        *string
	DeviceType         *string
}

type InventoryCheckoutWriteModel struct {
	TransactionUUID uuid.UUID
	Tagnumber       int64
	CheckoutDate    *time.Time
	ReturnDate      *time.Time
	CustomerName    *string
	CheckoutBool    *bool
}

func (updateRequest *InventoryUpdateRequest) ToDTO(htmlFormConstraints *HTMLFormConstraints) (*InventoryUpdateDTO, error) {
	if updateRequest == nil {
		return nil, fmt.Errorf("inventory update request is nil")
	}

	// Tagnumber
	if updateRequest.Tagnumber == nil {
		return nil, fmt.Errorf("tagnumber is required")
	}
	if err := IsTagnumberInt64Valid(updateRequest.Tagnumber); err != nil {
		return nil, fmt.Errorf("tagnumber is invalid: %v", err)
	}

	// System serial
	if updateRequest.SystemSerial == nil || strings.TrimSpace(*updateRequest.SystemSerial) == "" {
		return nil, fmt.Errorf("system_serial is required")
	}
	if utf8.RuneCountInString(strings.TrimSpace(*updateRequest.SystemSerial)) < htmlFormConstraints.InventoryForm.SystemSerialMinChars || utf8.RuneCountInString(*updateRequest.SystemSerial) > htmlFormConstraints.InventoryForm.SystemSerialMaxChars {
		return nil, fmt.Errorf("system_serial must be between %d and %d characters", htmlFormConstraints.InventoryForm.SystemSerialMinChars, htmlFormConstraints.InventoryForm.SystemSerialMaxChars)
	}
	if !IsASCIIStringPrintable(*updateRequest.SystemSerial) {
		return nil, fmt.Errorf("non-printable ASCII characters in system serial field")
	}

	// Location
	if updateRequest.Location == nil || strings.TrimSpace(*updateRequest.Location) == "" {
		return nil, fmt.Errorf("location is required")
	}
	if utf8.RuneCountInString(strings.TrimSpace(*updateRequest.Location)) < htmlFormConstraints.InventoryForm.LocationMinChars || utf8.RuneCountInString(*updateRequest.Location) > htmlFormConstraints.InventoryForm.LocationMaxChars {
		return nil, fmt.Errorf("location must be between %d and %d characters", htmlFormConstraints.InventoryForm.LocationMinChars, htmlFormConstraints.InventoryForm.LocationMaxChars)
	}
	if !IsPrintableUnicodeString(*updateRequest.Location) {
		return nil, fmt.Errorf("invalid UTF-8 in location field for inventory update")
	}

	// Building (optional)
	if updateRequest.Building != nil && strings.TrimSpace(*updateRequest.Building) != "" {
		if utf8.RuneCountInString(strings.TrimSpace(*updateRequest.Building)) < htmlFormConstraints.InventoryForm.BuildingMinChars || utf8.RuneCountInString(*updateRequest.Building) > htmlFormConstraints.InventoryForm.BuildingMaxChars {

			return nil, fmt.Errorf("building must be between %d and %d characters", htmlFormConstraints.InventoryForm.BuildingMinChars, htmlFormConstraints.InventoryForm.BuildingMaxChars)
		}
		if !IsPrintableUnicodeString(*updateRequest.Building) {
			return nil, fmt.Errorf("invalid UTF-8 in building field")
		}
	}

	// Room (optional)
	if updateRequest.Room != nil {
		if utf8.RuneCountInString(strings.TrimSpace(*updateRequest.Room)) < htmlFormConstraints.InventoryForm.RoomMinChars || utf8.RuneCountInString(*updateRequest.Room) > htmlFormConstraints.InventoryForm.RoomMaxChars {
			return nil, fmt.Errorf("room must be between %d and %d characters", htmlFormConstraints.InventoryForm.RoomMinChars, htmlFormConstraints.InventoryForm.RoomMaxChars)
		}
		if !IsPrintableUnicodeString(*updateRequest.Room) {
			return nil, fmt.Errorf("invalid UTF-8 in room field for inventory update")
		}
	}

	// System manufacturer
	if updateRequest.SystemManufacturer != nil && strings.TrimSpace(*updateRequest.SystemManufacturer) != "" {
		if utf8.RuneCountInString(strings.TrimSpace(*updateRequest.SystemManufacturer)) < htmlFormConstraints.InventoryForm.ManufacturerMinChars || utf8.RuneCountInString(*updateRequest.SystemManufacturer) > htmlFormConstraints.InventoryForm.ManufacturerMaxChars {
			return nil, fmt.Errorf("system manufacturer must be between %d and %d characters", htmlFormConstraints.InventoryForm.ManufacturerMinChars, htmlFormConstraints.InventoryForm.ManufacturerMaxChars)
		}
		if !IsPrintableUnicodeString(*updateRequest.SystemManufacturer) {
			return nil, fmt.Errorf("Non-printable Unicode characters in system manufacturer field")
		}
	}

	// System model (optional, min 1 char, max 64 Unicode chars)
	if updateRequest.SystemModel != nil && strings.TrimSpace(*updateRequest.SystemModel) != "" {
		if utf8.RuneCountInString(strings.TrimSpace(*updateRequest.SystemModel)) < htmlFormConstraints.InventoryForm.SystemModelMinChars || utf8.RuneCountInString(*updateRequest.SystemModel) > htmlFormConstraints.InventoryForm.SystemModelMaxChars {
			return nil, fmt.Errorf("system model must be between %d and %d characters", htmlFormConstraints.InventoryForm.SystemModelMinChars, htmlFormConstraints.InventoryForm.SystemModelMaxChars)
		}
		if !IsPrintableUnicodeString(*updateRequest.SystemModel) {
			return nil, fmt.Errorf("Non-printable Unicode characters in system model field")
		}
	}

	// Department (required, min 1 char, max 64 chars, printable ASCII only)
	if updateRequest.Department == nil || strings.TrimSpace(*updateRequest.Department) == "" {
		return nil, fmt.Errorf("department_name is required")
	}
	if utf8.RuneCountInString(strings.TrimSpace(*updateRequest.Department)) < htmlFormConstraints.InventoryForm.DepartmentMinChars || utf8.RuneCountInString(*updateRequest.Department) > htmlFormConstraints.InventoryForm.DepartmentMaxChars {
		return nil, fmt.Errorf("department_name must be between %d and %d characters", htmlFormConstraints.InventoryForm.DepartmentMinChars, htmlFormConstraints.InventoryForm.DepartmentMaxChars)
	}
	if !IsASCIIStringPrintable(*updateRequest.Department) {
		return nil, fmt.Errorf("non-printable ASCII characters in department_name field")
	}

	// ADDomain (required, min 1 char, max 64 chars)
	if updateRequest.ADDomain == nil || strings.TrimSpace(*updateRequest.ADDomain) == "" {
		return nil, fmt.Errorf("ad_domain is required")
	}
	if utf8.RuneCountInString(strings.TrimSpace(*updateRequest.ADDomain)) < htmlFormConstraints.InventoryForm.DomainMinChars || utf8.RuneCountInString(*updateRequest.ADDomain) > htmlFormConstraints.InventoryForm.DomainMaxChars {
		return nil, fmt.Errorf("ad_domain must be between %d and %d characters", htmlFormConstraints.InventoryForm.DomainMinChars, htmlFormConstraints.InventoryForm.DomainMaxChars)
	}
	if !IsASCIIStringPrintable(*updateRequest.ADDomain) {
		return nil, fmt.Errorf("non-printable ASCII characters in domain field")
	}

	// Property custodian (optional, min 1 char, max 64 Unicode chars)
	if updateRequest.PropertyCustodian != nil && strings.TrimSpace(*updateRequest.PropertyCustodian) != "" {
		if utf8.RuneCountInString(strings.TrimSpace(*updateRequest.PropertyCustodian)) < htmlFormConstraints.InventoryForm.PropertyCustodianMinChars || utf8.RuneCountInString(*updateRequest.PropertyCustodian) > htmlFormConstraints.InventoryForm.PropertyCustodianMaxChars {
			return nil, fmt.Errorf("property custodian must be between %d and %d characters", htmlFormConstraints.InventoryForm.PropertyCustodianMinChars, htmlFormConstraints.InventoryForm.PropertyCustodianMaxChars)
		}
		if !IsPrintableUnicodeString(*updateRequest.PropertyCustodian) {
			return nil, fmt.Errorf("non-printable Unicode characters in property custodian field")
		}
	}

	// Acquired date, optional, process as UTC
	if updateRequest.AcquiredDate != nil {
		if updateRequest.AcquiredDate.After(time.Now().UTC()) {
			return nil, fmt.Errorf("acquired_date cannot be in the future")
		}
	}

	// Retired date, optional, process as UTC
	if updateRequest.RetiredDate != nil {
		if updateRequest.AcquiredDate != nil && updateRequest.RetiredDate.UTC().Before(updateRequest.AcquiredDate.UTC()) {
			return nil, fmt.Errorf("retired_date cannot be before acquired_date")
		}
	}

	// IsBroken (optional, bool)
	if updateRequest.IsBroken == nil {
		// return nil, fmt.Errorf("is_broken is required")
	}

	// Disk removed (optional, bool)
	if updateRequest.DiskRemoved == nil {
		// return nil, fmt.Errorf("disk_removed is required")
	}

	// Last hardware check (optional, process as UTC)
	if updateRequest.LastHardwareCheck != nil && !updateRequest.LastHardwareCheck.IsZero() {
		lastHardwareCheckUTC := copyTimePtrToUTC(updateRequest.LastHardwareCheck)
		if lastHardwareCheckUTC.After(time.Now().UTC()) {
			return nil, fmt.Errorf("last_hardware_check cannot be in the future")
		}
	}

	// Status (required, min 1, max 24, ASCII printable chars only)
	if updateRequest.ClientStatus == nil || strings.TrimSpace(*updateRequest.ClientStatus) == "" {
		return nil, fmt.Errorf("status is required")
	}
	if utf8.RuneCountInString(strings.TrimSpace(*updateRequest.ClientStatus)) < htmlFormConstraints.InventoryForm.ClientStatusMinChars || utf8.RuneCountInString(*updateRequest.ClientStatus) > htmlFormConstraints.InventoryForm.ClientStatusMaxChars {
		return nil, fmt.Errorf("status must be between %d and %d characters", htmlFormConstraints.InventoryForm.ClientStatusMinChars, htmlFormConstraints.InventoryForm.ClientStatusMaxChars)
	}
	if !IsASCIIStringPrintable(*updateRequest.ClientStatus) {
		return nil, fmt.Errorf("non-printable ASCII characters in status field")
	}

	// Checkout bool (optional, bool)
	if htmlFormConstraints.InventoryForm.CheckoutBoolIsMandatory && updateRequest.CheckoutBool == nil {
		return nil, fmt.Errorf("checkout_bool is required")
	}
	// Checkout date (optional, process as UTC)
	if htmlFormConstraints.InventoryForm.CheckoutBoolIsMandatory && updateRequest.CheckoutDate == nil {
		return nil, fmt.Errorf("checkout_date is required when checkout_bool is mandatory")
	}
	// Return date (optional, process as UTC)
	if htmlFormConstraints.InventoryForm.ReturnDateIsMandatory && updateRequest.ReturnDate == nil {
		return nil, fmt.Errorf("return_date is required")
	}

	// Customer name (optional, min 1 char, max 64 Unicode chars)
	if updateRequest.CustomerName != nil && strings.TrimSpace(*updateRequest.CustomerName) != "" {
		if utf8.RuneCountInString(strings.TrimSpace(*updateRequest.CustomerName)) < 1 || utf8.RuneCountInString(*updateRequest.CustomerName) > 128 {
			return nil, fmt.Errorf("customer name must be between %d and %d characters", 1, 128)
		}
		if !IsPrintableUnicodeString(*updateRequest.CustomerName) {
			return nil, fmt.Errorf("non-printable Unicode characters in customer name field")
		}
	}

	// Note (optional)
	if updateRequest.Note != nil && strings.TrimSpace(*updateRequest.Note) != "" {
		if utf8.RuneCountInString(strings.TrimSpace(*updateRequest.Note)) < htmlFormConstraints.InventoryForm.ClientNoteMinChars || utf8.RuneCountInString(*updateRequest.Note) > htmlFormConstraints.InventoryForm.ClientNoteMaxChars {
			return nil, fmt.Errorf("note must be between %d and %d characters", htmlFormConstraints.InventoryForm.ClientNoteMinChars, htmlFormConstraints.InventoryForm.ClientNoteMaxChars)
		}
		if !IsPrintableUnicodeString(*updateRequest.Note) {
			return nil, fmt.Errorf("non-printable Unicode characters in note field")
		}
	}

	domain := &InventoryUpdateDTO{
		Tagnumber:          *updateRequest.Tagnumber,
		SystemSerial:       strings.TrimSpace(*updateRequest.SystemSerial),
		Location:           strings.TrimSpace(*updateRequest.Location),
		Building:           copyTrimmedStringPtr(updateRequest.Building),
		Room:               copyTrimmedStringPtr(updateRequest.Room),
		SystemManufacturer: copyTrimmedStringPtr(updateRequest.SystemManufacturer),
		SystemModel:        copyTrimmedStringPtr(updateRequest.SystemModel),
		DeviceType:         copyTrimmedStringPtr(updateRequest.DeviceType),
		Department:         strings.TrimSpace(*updateRequest.Department),
		ADDomain:           strings.TrimSpace(*updateRequest.ADDomain),
		PropertyCustodian:  copyTrimmedStringPtr(updateRequest.PropertyCustodian),
		AcquiredDate:       copyTimePtrToUTC(updateRequest.AcquiredDate),
		RetiredDate:        copyTimePtrToUTC(updateRequest.RetiredDate),
		IsBroken:           copyBoolPtr(updateRequest.IsBroken),
		DiskRemoved:        copyBoolPtr(updateRequest.DiskRemoved),
		LastHardwareCheck:  copyTimePtrToUTC(updateRequest.LastHardwareCheck),
		ClientStatus:       strings.TrimSpace(*updateRequest.ClientStatus),
		CheckoutBool:       copyBoolPtr(updateRequest.CheckoutBool),
		CheckoutDate:       copyTimePtrToUTC(updateRequest.CheckoutDate),
		ReturnDate:         copyTimePtrToUTC(updateRequest.ReturnDate),
		CustomerName:       copyTrimmedStringPtr(updateRequest.CustomerName),
		Note:               copyTrimmedStringPtr(updateRequest.Note),
	}

	return domain, nil
}

func (dto *InventoryUpdateDTO) ToLocationWriteModel(transactionUUID uuid.UUID) *InventoryLocationWriteModel {
	if dto == nil {
		return nil
	}
	return &InventoryLocationWriteModel{
		TransactionUUID:   transactionUUID,
		Tagnumber:         dto.Tagnumber,
		SystemSerial:      dto.SystemSerial,
		Location:          dto.Location,
		Building:          copyTrimmedStringPtr(dto.Building),
		Room:              copyTrimmedStringPtr(dto.Room),
		Department:        dto.Department,
		ADDomain:          dto.ADDomain,
		PropertyCustodian: copyTrimmedStringPtr(dto.PropertyCustodian),
		AcquiredDate:      copyTimePtrToUTC(dto.AcquiredDate),
		RetiredDate:       copyTimePtrToUTC(dto.RetiredDate),
		IsBroken:          copyBoolPtr(dto.IsBroken),
		DiskRemoved:       copyBoolPtr(dto.DiskRemoved),
		ClientStatus:      dto.ClientStatus,
		Note:              copyTrimmedStringPtr(dto.Note),
	}
}

func (dto *InventoryUpdateDTO) ToHardwareWriteModel(transactionUUID uuid.UUID) *InventoryHardwareWriteModel {
	if dto == nil {
		return nil
	}
	return &InventoryHardwareWriteModel{
		TransactionUUID:    transactionUUID,
		Tagnumber:          dto.Tagnumber,
		SystemManufacturer: copyTrimmedStringPtr(dto.SystemManufacturer),
		SystemModel:        copyTrimmedStringPtr(dto.SystemModel),
		DeviceType:         copyTrimmedStringPtr(dto.DeviceType),
	}
}

func (dto *InventoryUpdateDTO) ToClientHealthWriteModel(transactionUUID uuid.UUID) *ClientHealthDTO {
	if dto == nil {
		return nil
	}
	return &ClientHealthDTO{
		TransactionUUID:   transactionUUID.String(),
		Tagnumber:         dto.Tagnumber,
		LastHardwareCheck: copyTimePtrToUTC(dto.LastHardwareCheck),
	}
}

func (dto *InventoryUpdateDTO) ToCheckoutWriteModel(transactionUUID uuid.UUID) *InventoryCheckoutWriteModel {
	if dto == nil {
		return nil
	}
	// if err := IsTagnumberInt64Valid(&domain.Tagnumber); err != nil {
	// 	return nil
	// }

	return &InventoryCheckoutWriteModel{
		TransactionUUID: transactionUUID,
		Tagnumber:       dto.Tagnumber,
		CheckoutBool:    copyBoolPtr(dto.CheckoutBool),
		CheckoutDate:    copyTimePtrToUTC(dto.CheckoutDate),
		ReturnDate:      copyTimePtrToUTC(dto.ReturnDate),
		CustomerName:    copyTrimmedStringPtr(dto.CustomerName),
	}
}

type BulkUpdateRequest struct {
	Location   *string `json:"bulk_location"`
	Tagnumbers []int64 `json:"bulk_tagnumbers"`
}

type InventoryTableRow struct {
	Tagnumber           *int64                             `json:"tagnumber"`
	SystemSerial        *string                            `json:"system_serial"`
	Location            *string                            `json:"location"`
	LocationFormatted   *string                            `json:"location_formatted"`
	Building            *string                            `json:"building"`
	Room                *string                            `json:"room"`
	SystemManufacturer  *string                            `json:"system_manufacturer"`
	SystemModel         *string                            `json:"system_model"`
	DeviceType          *string                            `json:"device_type"`
	DeviceTypeFormatted *string                            `json:"device_type_formatted"`
	Department          *string                            `json:"department_name"`
	DepartmentFormatted *string                            `json:"department_formatted"`
	ADDomain            *string                            `json:"ad_domain"`
	AdminUsers          *[]string                          `json:"admin_users"`
	IsIntuneJoined      *bool                              `json:"is_intune_joined"`
	DomainFormatted     *string                            `json:"ad_domain_formatted"`
	OsInstalled         *bool                              `json:"os_installed"`
	OsName              *string                            `json:"os_name"`
	OsVersion           *string                            `json:"os_version"`
	LatestOsVersion     *string                            `json:"latest_os_version"`
	IsDiskEncrypted     *bool                              `json:"windows_bitlocker_enabled"`
	SecureBootEnabled   *bool                              `json:"secure_boot_enabled"`
	LastHardwareCheck   *time.Time                         `json:"last_hardware_check"`
	BIOSUpdated         *bool                              `json:"bios_updated"`
	BIOSVersion         *string                            `json:"bios_version"`
	Status              *string                            `json:"status"`
	StatusFormatted     *string                            `json:"status_formatted"`
	IsBroken            *bool                              `json:"is_broken"`
	DiskRemoved         *bool                              `json:"disk_removed"`
	RetiredDate         *time.Time                         `json:"retired_date"`
	IsCheckedOut        *bool                              `json:"checkout_bool"`
	Note                *string                            `json:"note"`
	LastUpdated         *time.Time                         `json:"last_updated"`
	FileCount           *int64                             `json:"file_count"`
	ClientErrors        []ClientConfigErrorMessageResponse `json:"client_configuration_errors,omitempty"`
}

type InventoryAdvSearchOptions struct {
	Location           *AdvSearchOptionString `json:"filter_location"`
	BuildingAndRoom    *AdvSearchOptionString `json:"filter_building_room"`
	Building           *string                `json:"-"`
	Room               *string                `json:"-"`
	SystemManufacturer *AdvSearchOptionString `json:"filter_system_manufacturer"`
	SystemModel        *AdvSearchOptionString `json:"filter_system_model"`
	DeviceType         *AdvSearchOptionString `json:"filter_device_type"`
	Department         *AdvSearchOptionString `json:"filter_department_name"`
	ADDomain           *AdvSearchOptionString `json:"filter_ad_domain"`
	Status             *AdvSearchOptionString `json:"filter_status"`
	IsBroken           *AdvSearchOptionBool   `json:"filter_is_broken"`
	HasImages          *AdvSearchOptionBool   `json:"filter_has_images"`
}

type AdvSearchOptionString struct {
	ParamValue *string `json:"param_value"`
	Not        *bool   `json:"not"`
}

type AdvSearchOptionBool struct {
	ParamValue *bool `json:"param_value"`
	Not        *bool `json:"not"`
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
	ADDomain           *string    `json:"ad_domain"`
	PropertyCustodian  *string    `json:"property_custodian"`
	AcquiredDate       *time.Time `json:"acquired_date"`
	RetiredDate        *time.Time `json:"retired_date"`
	IsBroken           *bool      `json:"is_broken"`
	DiskRemoved        *bool      `json:"disk_removed"`
	LastHardwareCheck  *time.Time `json:"last_hardware_check"`
	ClientStatus       *string    `json:"status"`
	CheckoutBool       *bool      `json:"checkout_bool"`
	CheckoutDate       *time.Time `json:"checkout_date"`
	ReturnDate         *time.Time `json:"return_date"`
	CustomerName       *string    `json:"customer_name"`
	Note               *string    `json:"note"`
	FileCount          *int64     `json:"file_count"`
}
