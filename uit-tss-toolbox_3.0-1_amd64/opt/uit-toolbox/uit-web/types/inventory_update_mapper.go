package types

import (
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
)

func MapInventoryUpdateRequestToDomain(updateRequest *InventoryUpdateRequest, htmlFormConstraints *HTMLFormConstraints) (*InventoryUpdateDomain, error) {
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

	// Domain (required, min 1 char, max 64 chars)
	if updateRequest.Domain == nil || strings.TrimSpace(*updateRequest.Domain) == "" {
		return nil, fmt.Errorf("ad_domain is required")
	}
	if utf8.RuneCountInString(strings.TrimSpace(*updateRequest.Domain)) < htmlFormConstraints.InventoryForm.DomainMinChars || utf8.RuneCountInString(*updateRequest.Domain) > htmlFormConstraints.InventoryForm.DomainMaxChars {
		return nil, fmt.Errorf("ad_domain must be between %d and %d characters", htmlFormConstraints.InventoryForm.DomainMinChars, htmlFormConstraints.InventoryForm.DomainMaxChars)
	}
	if !IsASCIIStringPrintable(*updateRequest.Domain) {
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

	// Broken (optional, bool)
	if updateRequest.Broken == nil {
		// return nil, fmt.Errorf("is_broken is required")
	}

	// Disk removed (optional, bool)
	if updateRequest.DiskRemoved == nil {
		// return nil, fmt.Errorf("disk_removed is required")
	}

	// Last hardware check (optional, process as UTC)
	if updateRequest.LastHardwareCheck != nil && !updateRequest.LastHardwareCheck.IsZero() {
		lastHardwareCheckUTC := timePtrToUTC(updateRequest.LastHardwareCheck)
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

	// Note (optional)
	if updateRequest.Note != nil && strings.TrimSpace(*updateRequest.Note) != "" {
		if utf8.RuneCountInString(strings.TrimSpace(*updateRequest.Note)) < htmlFormConstraints.InventoryForm.ClientNoteMinChars || utf8.RuneCountInString(*updateRequest.Note) > htmlFormConstraints.InventoryForm.ClientNoteMaxChars {
			return nil, fmt.Errorf("note must be between %d and %d characters", htmlFormConstraints.InventoryForm.ClientNoteMinChars, htmlFormConstraints.InventoryForm.ClientNoteMaxChars)
		}
		if !IsPrintableUnicodeString(*updateRequest.Note) {
			return nil, fmt.Errorf("non-printable Unicode characters in note field")
		}
	}

	domain := &InventoryUpdateDomain{
		Tagnumber:          *updateRequest.Tagnumber,
		SystemSerial:       strings.TrimSpace(*updateRequest.SystemSerial),
		Location:           strings.TrimSpace(*updateRequest.Location),
		Building:           copyTrimmedStringPtr(updateRequest.Building),
		Room:               copyTrimmedStringPtr(updateRequest.Room),
		SystemManufacturer: copyTrimmedStringPtr(updateRequest.SystemManufacturer),
		SystemModel:        copyTrimmedStringPtr(updateRequest.SystemModel),
		DeviceType:         copyTrimmedStringPtr(updateRequest.DeviceType),
		Department:         strings.TrimSpace(*updateRequest.Department),
		Domain:             strings.TrimSpace(*updateRequest.Domain),
		PropertyCustodian:  copyTrimmedStringPtr(updateRequest.PropertyCustodian),
		AcquiredDate:       timePtrToUTC(updateRequest.AcquiredDate),
		RetiredDate:        timePtrToUTC(updateRequest.RetiredDate),
		Broken:             copyBoolPtr(updateRequest.Broken),
		DiskRemoved:        copyBoolPtr(updateRequest.DiskRemoved),
		LastHardwareCheck:  timePtrToUTC(updateRequest.LastHardwareCheck),
		ClientStatus:       strings.TrimSpace(*updateRequest.ClientStatus),
		CheckoutBool:       copyBoolPtr(updateRequest.CheckoutBool),
		CheckoutDate:       timePtrToUTC(updateRequest.CheckoutDate),
		ReturnDate:         timePtrToUTC(updateRequest.ReturnDate),
		Note:               copyTrimmedStringPtr(updateRequest.Note),
	}

	return domain, nil
}

func MapInventoryUpdateDomainToLocationWriteModel(transactionUUID uuid.UUID, domain *InventoryUpdateDomain) *InventoryLocationWriteModel {
	if domain == nil {
		return nil
	}
	return &InventoryLocationWriteModel{
		TransactionUUID:   transactionUUID,
		Tagnumber:         domain.Tagnumber,
		SystemSerial:      domain.SystemSerial,
		Location:          domain.Location,
		Building:          copyStringPtr(domain.Building),
		Room:              copyStringPtr(domain.Room),
		Department:        domain.Department,
		Domain:            domain.Domain,
		PropertyCustodian: copyStringPtr(domain.PropertyCustodian),
		AcquiredDate:      copyTimePtr(domain.AcquiredDate),
		RetiredDate:       copyTimePtr(domain.RetiredDate),
		Broken:            copyBoolPtr(domain.Broken),
		DiskRemoved:       copyBoolPtr(domain.DiskRemoved),
		ClientStatus:      domain.ClientStatus,
		Note:              copyStringPtr(domain.Note),
	}
}

func MapInventoryUpdateDomainToHardwareWriteModel(transactionUUID uuid.UUID, domain *InventoryUpdateDomain) *InventoryHardwareWriteModel {
	if domain == nil {
		return nil
	}
	return &InventoryHardwareWriteModel{
		TransactionUUID:    transactionUUID,
		Tagnumber:          domain.Tagnumber,
		SystemManufacturer: copyStringPtr(domain.SystemManufacturer),
		SystemModel:        copyStringPtr(domain.SystemModel),
		DeviceType:         copyStringPtr(domain.DeviceType),
	}
}

func MapInventoryUpdateDomainToClientHealthWriteModel(transactionUUID uuid.UUID, domain *InventoryUpdateDomain) *InventoryClientHealthWriteModel {
	if domain == nil {
		return nil
	}
	return &InventoryClientHealthWriteModel{
		TransactionUUID:   transactionUUID,
		Tagnumber:         domain.Tagnumber,
		LastHardwareCheck: copyTimePtr(domain.LastHardwareCheck),
	}
}

func MapInventoryUpdateDomainToCheckoutWriteModel(transactionUUID uuid.UUID, domain *InventoryUpdateDomain) *InventoryCheckoutWriteModel {
	if domain == nil {
		return nil
	}
	if domain.CheckoutDate == nil && domain.ReturnDate == nil && (domain.CheckoutBool == nil || !*domain.CheckoutBool) {
		return nil
	}
	return &InventoryCheckoutWriteModel{
		TransactionUUID: transactionUUID,
		Tagnumber:       domain.Tagnumber,
		CheckoutDate:    copyTimePtr(domain.CheckoutDate),
		ReturnDate:      copyTimePtr(domain.ReturnDate),
		CheckoutBool:    copyBoolPtr(domain.CheckoutBool),
	}
}

func copyTrimmedStringPtr(value *string) *string {
	if value == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*value)
	return &trimmed
}

func copyStringPtr(value *string) *string {
	if value == nil {
		return nil
	}
	v := *value
	return &v
}

func copyTimePtr(value *time.Time) *time.Time {
	if value == nil {
		return nil
	}
	v := *value
	return &v
}

func timePtrToUTC(value *time.Time) *time.Time {
	if value == nil {
		return nil
	}
	utc := value.UTC()
	return &utc
}

func copyBoolPtr(value *bool) *bool {
	if value == nil {
		return nil
	}
	v := *value
	return &v
}

func int64ToPtr(value int64) *int64 {
	v := value
	return &v
}

func stringToPtr(value string) *string {
	v := value
	return &v
}
