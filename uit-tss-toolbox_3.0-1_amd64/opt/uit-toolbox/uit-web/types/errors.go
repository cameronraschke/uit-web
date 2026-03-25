package types

import (
	"errors"
)

var (
	// General errors
	CannotGetAppStateError = errors.New("cannot retrieve app state")
	NilAppStateError       = errors.New("app state is nil")
	MissingFieldError      = errors.New("required field is empty/nil")
	InvalidFieldError      = errors.New("invalid field value")
	InvalidStructureError  = errors.New("invalid structure")

	// Database errors
	DatabaseConnNilError      = errors.New("database connection is nil/uninitialized")
	DatabaseConnError         = errors.New("cannot get DB connection from app state")
	DatabaseQueryError        = errors.New("error executing DB query")
	DatabaseTransactionError  = errors.New("cannot begin DB transaction")
	DatabaseUpdateError       = errors.New("cannot update DB")
	DatabaseAffectedRowsError = errors.New("unexpected number of rows affected")
	DatabaseRowScanError      = errors.New("error scanning DB row")
	DatabaseRowIterationError = errors.New("error during DB row iteration")

	// Web endpoint errors
	EndpointNotFoundError    = errors.New("endpoint not found in config")
	InvalidEndpointDataError = errors.New("invalid/missing endpoint data")
	NilEndpointDataError     = errors.New("endpoint data is nil")
	LiveImageMissingError    = errors.New("live image not found for the given tag")
)
