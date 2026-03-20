package types

import (
	"errors"
)

var (
	CannotGetAppStateError      = errors.New("cannot retrieve app state")
	LiveImageMissingError       = errors.New("live image not found for the given tag, please insert live image first")
	AppStateNilError            = errors.New("app state is nil")
	DatabaseConnNilError        = errors.New("database connection is nil/uninitialized")
	DatabaseConnError           = errors.New("cannot get DB connection from app state")
	MissingFieldError           = errors.New("required field is empty/nil")
	InvalidFieldError           = errors.New("invalid field value")
	CannotBeginTransactionError = errors.New("cannot begin DB transaction")
	DBUpdateError               = errors.New("error while updating DB")
	DBRowsAffectedError         = errors.New("unexpected number of rows affected")
)
