package config

import (
	"fmt"
	"regexp"
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

func (as *AppState) GetInventoryUpdateJsonConstraints() (maxFormJsonBytes int64, err error) {
	if as == nil {
		return 0, fmt.Errorf("app state is nil in GetInventoryUpdateJsonConstraints")
	}
	fc := as.appConfig.Load().formConstraints.Load()
	if fc == nil {
		return 0, fmt.Errorf("form constraints not set in GetInventoryUpdateJsonConstraints")
	}
	return fc.inventoryUpdateFormMaxJsonBytes, nil
}

func (as *AppState) GetFileUploadDefaultConstraints() (fileUploadRegex *regexp.Regexp, minFileSize int64, maxFileSize int64, err error) {
	if as == nil {
		return nil, 0, 0, fmt.Errorf("app state is nil in GetFileUploadDefaultConstraints")
	}
	fc := as.appConfig.Load().fileConstraints.Load()
	if fc == nil {
		return nil, 0, 0, fmt.Errorf("file constraints not set in GetFileUploadDefaultConstraints")
	}
	if fc.defaultAllowedFileRegex == nil {
		return nil, 0, 0, fmt.Errorf("default file constraints not set in GetFileUploadDefaultConstraints")
	}
	if fc.defaultMaxFileSize <= 0 {
		return nil, 0, 0, fmt.Errorf("default max file size constraint is not set or invalid in GetFileUploadDefaultConstraints")
	}
	if fc.defaultMinFileSize <= 0 {
		return nil, 0, 0, fmt.Errorf("default min file size constraint is not set or invalid in GetFileUploadDefaultConstraints")
	}
	return fc.defaultAllowedFileRegex, fc.defaultMinFileSize, fc.defaultMaxFileSize, nil
}

func (as *AppState) GetFileUploadImageConstraints() (minFileSize int64, maxFileSize int64, maxFileCount int, acceptedImageExtensionsAndMimeTypes map[string]string, err error) {
	if as == nil {
		return 0, 0, 0, nil, fmt.Errorf("app state is nil in GetFileUploadImageConstraints")
	}
	fc := as.appConfig.Load().fileConstraints.Load()
	if fc == nil {
		return 0, 0, 0, nil, fmt.Errorf("file constraints not set in GetFileUploadImageConstraints")
	}
	if fc.imageConstraints == nil {
		return 0, 0, 0, nil, fmt.Errorf("image constraints not set in GetFileUploadImageConstraints")
	}
	return fc.imageConstraints.minFileSize, fc.imageConstraints.maxFileSize, fc.imageConstraints.maxFileCount, fc.imageConstraints.acceptedImageExtensionsAndMimeTypes, nil
}

func (as *AppState) GetFileUploadVideoConstraints() (minFileSize int64, maxFileSize int64, maxFileCount int, acceptedVideoExtensionsAndMimeTypes map[string]string, err error) {
	if as == nil {
		return 0, 0, 0, nil, fmt.Errorf("app state is nil in GetFileUploadVideoConstraints")
	}
	fc := as.appConfig.Load().fileConstraints.Load()
	if fc == nil {
		return 0, 0, 0, nil, fmt.Errorf("file constraints not set in GetFileUploadVideoConstraints")
	}
	if fc.videoConstraints == nil {
		return 0, 0, 0, nil, fmt.Errorf("video constraints not set in GetFileUploadVideoConstraints")
	}
	return fc.videoConstraints.minFileSize, fc.videoConstraints.maxFileSize, fc.videoConstraints.maxFileCount, fc.videoConstraints.acceptedVideoExtensionsAndMimeTypes, nil
}
