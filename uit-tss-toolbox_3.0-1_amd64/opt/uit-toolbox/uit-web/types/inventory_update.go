package types

import (
	"fmt"
	"strings"
	"time"

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
	if err := IsTagnumberInt64Valid(updateRequest.Tagnumber); err != nil {
		return nil, CreateInvalidFieldError("tagnumber", err)
	}

	// System serial
	if err := IsSystemSerialValid(updateRequest.SystemSerial); err != nil {
		return nil, CreateInvalidFieldError("system_serial", err)
	}

	// Location
	if err := ValidatePrintableStrLen(updateRequest.Location, 1, 128); err != nil {
		return nil, CreateInvalidFieldError("location", err)
	}

	// Building (optional)
	if err := ValidatePrintableStrLen(updateRequest.Building, 0, 128); err != nil {
		return nil, CreateInvalidFieldError("building", err)
	}

	// Room (optional)
	if err := ValidatePrintableStrLen(updateRequest.Room, 0, 128); err != nil {
		return nil, CreateInvalidFieldError("room", err)
	}

	// System manufacturer (optional)
	if err := ValidatePrintableStrLen(updateRequest.SystemManufacturer, 0, 128); err != nil {
		return nil, CreateInvalidFieldError("system_manufacturer", err)
	}

	// System model (optional)
	if err := ValidateASCIIStrLen(updateRequest.SystemModel, 0, 128); err != nil {
		return nil, CreateInvalidFieldError("system_model", err)
	}

	// Department (required)
	if err := ValidateASCIIStrLen(updateRequest.Department, 1, 64); err != nil {
		return nil, CreateInvalidFieldError("department_name", err)
	}

	// ADDomain (required)
	if err := ValidateASCIIStrLen(updateRequest.ADDomain, 1, 64); err != nil {
		return nil, CreateInvalidFieldError("ad_domain", err)
	}

	// Property custodian (optional, min 1 char, max 64 Unicode chars)
	if err := ValidatePrintableStrLen(updateRequest.PropertyCustodian, 0, 128); err != nil {
		return nil, CreateInvalidFieldError("property_custodian", err)
	}

	// Acquired date, optional, process as UTC
	if updateRequest.AcquiredDate != nil && !updateRequest.AcquiredDate.IsZero() {
		if updateRequest.AcquiredDate.After(time.Now().UTC()) {
			return nil, fmt.Errorf("acquired_date cannot be in the future")
		}
	}

	// Retired date, optional, process as UTC
	if updateRequest.RetiredDate != nil && !updateRequest.RetiredDate.IsZero() {
		if updateRequest.AcquiredDate != nil && updateRequest.RetiredDate.UTC().Before(updateRequest.AcquiredDate.UTC()) {
			return nil, fmt.Errorf("retired_date cannot be before acquired_date")
		}
	}

	// Last hardware check (optional, process as UTC)
	if updateRequest.LastHardwareCheck != nil && !updateRequest.LastHardwareCheck.IsZero() {
		lastHardwareCheckUTC := copyTimePtrToUTC(updateRequest.LastHardwareCheck)
		if lastHardwareCheckUTC.After(time.Now().UTC()) {
			return nil, fmt.Errorf("last_hardware_check cannot be in the future")
		}
	}

	// Client status (required, min 1, max 24, ASCII printable chars only)
	if err := ValidateASCIIStrLen(updateRequest.ClientStatus, 1, 24); err != nil {
		return nil, CreateInvalidFieldError("status (client_status)", err)
	}

	// Customer name (optional, min 1 char, max 64 Unicode chars)
	if err := ValidatePrintableStrLen(updateRequest.CustomerName, 0, 128); err != nil {
		return nil, CreateInvalidFieldError("customer_name", err)
	}

	// Note (optional)
	if err := ValidatePrintableStrLen(updateRequest.Note, 0, 1024); err != nil {
		return nil, CreateInvalidFieldError("note (client_note)", err)
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
	SessionID  *string `json:"session_id"`
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
	AdminUsers          []string                           `json:"admin_users"`
	IsIntuneJoined      *bool                              `json:"is_intune_joined"`
	DomainFormatted     *string                            `json:"ad_domain_formatted"`
	OsInstalled         *bool                              `json:"os_installed"`
	OsName              *string                            `json:"os_name"`
	OsVersion           *string                            `json:"os_version"`
	LatestOsVersion     *string                            `json:"latest_os_version"`
	IsDiskEncrypted     *bool                              `json:"windows_bitlocker_enabled"`
	SecureBootEnabled   *bool                              `json:"secure_boot_enabled"`
	Has2023SecureBootCA *bool                              `json:"has_2023_secure_boot_ca"`
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

type InventoryFormPrefillRow struct {
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
