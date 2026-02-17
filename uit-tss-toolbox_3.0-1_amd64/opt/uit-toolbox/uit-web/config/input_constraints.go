package config

import (
	"fmt"
)

type InputFieldConstraints struct {
	usernameMinChars             int
	usernameMaxChars             int
	passwordMinChars             int
	passwordMaxChars             int
	tagnumberMinChars            int
	tagnumberMaxChars            int
	systemSerialMinChars         int
	systemSerialMaxChars         int
	locationMinChars             int
	locationMaxChars             int
	buildingMinChars             int
	buildingMaxChars             int
	roomMinChars                 int
	roomMaxChars                 int
	manufacturerMinChars         int
	manufacturerMaxChars         int
	systemModelMinChars          int
	systemModelMaxChars          int
	deviceTypeMinChars           int
	deviceTypeMaxChars           int
	departmentMinChars           int
	departmentMaxChars           int
	domainMinChars               int
	domainMaxChars               int
	propertyCustodianMinChars    int
	propertyCustodianMaxChars    int
	acquiredDateIsMandatory      bool
	retiredDateIsMandatory       bool
	isFunctionalIsMandatory      bool
	diskRemovedIsMandatory       bool
	lastHardwareCheckIsMandatory bool
	clientStatusMinChars         int
	clientStatusMaxChars         int
	checkoutBoolIsMandatory      bool
	checkoutDateIsMandatory      bool
	returnDateIsMandatory        bool
	clientNoteMinChars           int
	clientNoteMaxChars           int
	noteTypeMinChars             int
	noteTypeMaxChars             int
	noteContentMinChars          int
	noteContentMaxChars          int
}

type HTMLFormConstraints struct {
	maxLoginFormSizeBytes           int64
	noteMaxBytes                    int64
	inventoryUpdateFormMaxJsonBytes int64
	fileUploadAllowedFileExtensions []string
	fileUploadAllowedFileRegex      string
	fileUploadMaxTotalBytes         int64
	fileUploadMaxFileCount          int
	fileUploadMinFileBytes          int64
	fileUploadMaxFileBytes          int64
}

func (as *AppState) GetTagnumberConstraints() (min int, max int, err error) {
	if as == nil {
		return 0, 0, fmt.Errorf("app state is nil in GetTagnumberConstraints")
	}
	ic := as.appConfig.Load().inputConstraints.Load()
	if ic == nil {
		return 0, 0, fmt.Errorf("input constraints not set in GetTagnumberConstraints")
	}
	return ic.tagnumberMinChars, ic.tagnumberMaxChars, nil
}

func (as *AppState) GetSystemSerialConstraints() (min int, max int, err error) {
	if as == nil {
		return 0, 0, fmt.Errorf("app state is nil in GetSystemSerialConstraints")
	}
	ic := as.appConfig.Load().inputConstraints.Load()
	if ic == nil {
		return 0, 0, fmt.Errorf("input constraints not set in GetSystemSerialConstraints")
	}
	return ic.systemSerialMinChars, ic.systemSerialMaxChars, nil
}

func (as *AppState) GetLocationConstraints() (min int, max int, err error) {
	if as == nil {
		return 0, 0, fmt.Errorf("app state is nil in GetLocationConstraints")
	}
	ic := as.appConfig.Load().inputConstraints.Load()
	if ic == nil {
		return 0, 0, fmt.Errorf("input constraints not set in GetLocationConstraints")
	}
	return ic.locationMinChars, ic.locationMaxChars, nil
}

func (as *AppState) GetBuildingConstraints() (min int, max int, err error) {
	if as == nil {
		return 0, 0, fmt.Errorf("app state is nil in GetBuildingConstraints")
	}
	ic := as.appConfig.Load().inputConstraints.Load()
	if ic == nil {
		return 0, 0, fmt.Errorf("input constraints not set in GetBuildingConstraints")
	}
	return ic.buildingMinChars, ic.buildingMaxChars, nil
}

func (as *AppState) GetRoomConstraints() (min int, max int, err error) {
	if as == nil {
		return 0, 0, fmt.Errorf("app state is nil in GetRoomConstraints")
	}
	ic := as.appConfig.Load().inputConstraints.Load()
	if ic == nil {
		return 0, 0, fmt.Errorf("input constraints not set in GetRoomConstraints")
	}
	return ic.roomMinChars, ic.roomMaxChars, nil
}

func (as *AppState) GetManufacturerConstraints() (min int, max int, err error) {
	if as == nil {
		return 0, 0, fmt.Errorf("app state is nil in GetManufacturerConstraints")
	}
	ic := as.appConfig.Load().inputConstraints.Load()
	if ic == nil {
		return 0, 0, fmt.Errorf("input constraints not set in GetManufacturerConstraints")
	}
	return ic.manufacturerMinChars, ic.manufacturerMaxChars, nil
}

func (as *AppState) GetSystemModelConstraints() (min int, max int, err error) {
	if as == nil {
		return 0, 0, fmt.Errorf("app state is nil in GetSystemModelConstraints")
	}
	ic := as.appConfig.Load().inputConstraints.Load()
	if ic == nil {
		return 0, 0, fmt.Errorf("input constraints not set in GetSystemModelConstraints")
	}
	return ic.systemModelMinChars, ic.systemModelMaxChars, nil
}

func (as *AppState) GetDeviceTypeConstraints() (min int, max int, err error) {
	if as == nil {
		return 0, 0, fmt.Errorf("app state is nil in GetDeviceTypeConstraints")
	}
	ic := as.appConfig.Load().inputConstraints.Load()
	if ic == nil {
		return 0, 0, fmt.Errorf("input constraints not set in GetDeviceTypeConstraints")
	}
	return ic.deviceTypeMinChars, ic.deviceTypeMaxChars, nil
}

func (as *AppState) GetDepartmentConstraints() (min int, max int, err error) {
	if as == nil {
		return 0, 0, fmt.Errorf("app state is nil in GetDepartmentConstraints")
	}
	ic := as.appConfig.Load().inputConstraints.Load()
	if ic == nil {
		return 0, 0, fmt.Errorf("input constraints not set in GetDepartmentConstraints")
	}
	return ic.departmentMinChars, ic.departmentMaxChars, nil
}

func (as *AppState) GetDomainConstraints() (min int, max int, err error) {
	if as == nil {
		return 0, 0, fmt.Errorf("app state is nil in GetDomainConstraints")
	}
	ic := as.appConfig.Load().inputConstraints.Load()
	if ic == nil {
		return 0, 0, fmt.Errorf("input constraints not set in GetDomainConstraints")
	}
	return ic.domainMinChars, ic.domainMaxChars, nil
}

func (as *AppState) GetPropertyCustodianConstraints() (min int, max int, err error) {
	if as == nil {
		return 0, 0, fmt.Errorf("app state is nil in GetPropertyCustodianConstraints")
	}
	ic := as.appConfig.Load().inputConstraints.Load()
	if ic == nil {
		return 0, 0, fmt.Errorf("input constraints not set in GetPropertyCustodianConstraints")
	}
	return ic.propertyCustodianMinChars, ic.propertyCustodianMaxChars, nil
}

func (as *AppState) GetClientStatusConstraints() (min int, max int, err error) {
	if as == nil {
		return 0, 0, fmt.Errorf("app state is nil in GetClientStatusConstraints")
	}
	ic := as.appConfig.Load().inputConstraints.Load()
	if ic == nil {
		return 0, 0, fmt.Errorf("input constraints not set in GetClientStatusConstraints")
	}
	return ic.clientStatusMinChars, ic.clientStatusMaxChars, nil
}

func (as *AppState) GetCheckoutConstraints() (checkoutDateIsMandatory, returnDateMandatory, checkoutBoolIsMandatory bool, err error) {
	if as == nil {
		return false, false, false, fmt.Errorf("app state is nil in GetCheckoutConstraints")
	}
	ic := as.appConfig.Load().inputConstraints.Load()
	if ic == nil {
		return false, false, false, fmt.Errorf("input constraints not set in GetCheckoutConstraints")
	}
	return ic.checkoutDateIsMandatory, ic.returnDateIsMandatory, ic.checkoutBoolIsMandatory, nil
}

func (as *AppState) GetClientNoteConstraints() (minChars int, maxChars int, err error) {
	if as == nil {
		return 0, 0, fmt.Errorf("app state is nil in GetClientNoteConstraints")
	}
	ic := as.appConfig.Load().inputConstraints.Load()
	if ic == nil {
		return 0, 0, fmt.Errorf("input constraints not set in GetClientNoteConstraints")
	}
	return ic.clientNoteMinChars, ic.clientNoteMaxChars, nil
}

func (as *AppState) GetNoteConstraints() (noteFormMaxBytes int64, noteTypeMinChars int, noteTypeMaxChars int, noteContentMinChars int, noteContentMaxChars int, err error) {
	if as == nil {
		return 0, 0, 0, 0, 0, fmt.Errorf("app state is nil in GetNoteConstraints")
	}
	ic := as.appConfig.Load().inputConstraints.Load()
	if ic == nil {
		return 0, 0, 0, 0, 0, fmt.Errorf("input constraints not set in GetNoteConstraints")
	}
	fc := as.appConfig.Load().formConstraints.Load()
	if fc == nil {
		return 0, 0, 0, 0, 0, fmt.Errorf("form constraints not set in GetNoteConstraints")
	}
	return fc.noteMaxBytes, ic.noteTypeMinChars, ic.noteTypeMaxChars, ic.noteContentMinChars, ic.noteContentMaxChars, nil
}

func (as *AppState) GetLoginFormSizeConstraint() (maxFormBytes int64, minUsernameChars int, maxUsernameChars int, minPasswordChars int, maxPasswordChars int, err error) {
	if as == nil {
		return 0, 0, 0, 0, 0, fmt.Errorf("app state is nil in GetLoginFormSizeConstraint")
	}
	ic := as.appConfig.Load().inputConstraints.Load()
	if ic == nil {
		return 0, 0, 0, 0, 0, fmt.Errorf("input constraints not set in GetLoginFormSizeConstraint")
	}
	fc := as.appConfig.Load().formConstraints.Load()
	if fc == nil {
		return 0, 0, 0, 0, 0, fmt.Errorf("form constraints not set in GetLoginFormSizeConstraint")
	}
	return fc.maxLoginFormSizeBytes, ic.usernameMinChars, ic.usernameMaxChars, ic.passwordMinChars, ic.passwordMaxChars, nil
}

func (as *AppState) GetInventoryUpdateFormConstraints() (maxFormJsonBytes int64, err error) {
	if as == nil {
		return 0, fmt.Errorf("app state is nil in GetInventoryUpdateFormConstraints")
	}
	fc := as.appConfig.Load().formConstraints.Load()
	if fc == nil {
		return 0, fmt.Errorf("form constraints not set in GetInventoryUpdateFormConstraints")
	}
	return fc.inventoryUpdateFormMaxJsonBytes, nil
}

func (as *AppState) GetFileUploadSizeConstraints() (maxFormBytes int64, minFileSize int64, maxFileSize int64, maxFileCount int, err error) {
	if as == nil {
		return 0, 0, 0, 0, fmt.Errorf("app state is nil in GetFileUploadSizeConstraints")
	}
	fc := as.appConfig.Load().formConstraints.Load()
	if fc == nil {
		return 0, 0, 0, 0, fmt.Errorf("form constraints not set in GetFileUploadSizeConstraints")
	}
	return fc.fileUploadMaxTotalBytes, fc.fileUploadMinFileBytes, fc.fileUploadMaxFileBytes, fc.fileUploadMaxFileCount, nil
}

func (as *AppState) GetFileUploadAllowedExtensionsAndRegex() (allowedExtensions []string, allowedFileNameRegex string, err error) {
	if as == nil {
		return nil, "", fmt.Errorf("app state is nil in GetFileUploadAllowedExtensionsAndRegex")
	}
	fc := as.appConfig.Load().formConstraints.Load()
	if fc == nil {
		return nil, "", fmt.Errorf("form constraints not set in GetFileUploadAllowedExtensionsAndRegex")
	}
	return fc.fileUploadAllowedFileExtensions, fc.fileUploadAllowedFileRegex, nil
}
