package types

import (
	"errors"
	"fmt"
)

var (
	// General errors
	ContextError              = errors.New("context error")
	CannotGetAppStateError    = errors.New("cannot retrieve app state")
	NilAppStateError          = errors.New("app state is nil")
	MissingFieldError         = errors.New("required field is empty/nil")
	InvalidFieldError         = errors.New("invalid field value")
	InvalidStructureError     = errors.New("invalid structure")
	ErrFailedToUpdateAppState = errors.New("failed to update app state")

	// Client errors
	ErrFailedToUpdateRealtimeData = errors.New("failed to update realtime data")
	ErrClientUUIDMissingError     = errors.New("client UUID is missing from AppState for the given tag")
	ErrClientNotFound             = errors.New("client UUID not found")
	ClientLastHeardMissingError   = errors.New("client last heard not found for the given tag")
	ErrNoOnlineClients            = errors.New("no online clients were found")
	ErrNoRealtimeClientData       = errors.New("no realtime client data found for the given tag")

	// JSON parsing errors
	JSONParseError     = errors.New("error parsing JSON")
	JSONUnmarshalError = errors.New("error unmarshaling JSON")

	// Database errors
	DatabaseConnNilError      = errors.New("database connection is nil/uninitialized")
	DatabaseConnError         = errors.New("cannot get DB connection from app state")
	DatabaseQueryError        = errors.New("error executing DB query")
	DatabaseTransactionError  = errors.New("cannot begin DB transaction")
	DatabaseUpdateError       = errors.New("cannot update DB")
	DatabaseAffectedRowsError = errors.New("unexpected number of rows affected")
	DatabaseRowScanError      = errors.New("error scanning DB row")
	DatabaseRowIterationError = errors.New("error during DB row iteration")
	DatabaseRowNotFoundError  = errors.New("row not found in DB")
	DatabaseDeletionError     = errors.New("error deleting row from DB")
	ErrInvalidClientUUID      = errors.New("invalid client UUID")
	ErrClientUUIDNotFoundInDB = errors.New("client UUID is not found in the database for the given tag")

	// Web endpoint errors
	EndpointNotFoundError    = errors.New("endpoint not found in config")
	InvalidEndpointDataError = errors.New("invalid/missing endpoint data")
	NilEndpointDataError     = errors.New("endpoint data is nil")
	LiveImageMissingError    = errors.New("live image not found for the given tag")

	// Web request errors
	InvalidRequestError              = errors.New("invalid request")
	InvalidRequestFieldError         = errors.New("invalid request field value")
	FailedToUpdateDatabaseValueError = errors.New("failed to update value in DB")
)

func CreateInvalidFieldErrorStr(fieldName string, err error) string {
	return CreateInvalidFieldError(fieldName, err).Error()
}

func CreateInvalidFieldError(fieldName string, err error) error {
	return fmt.Errorf("%v for '%s': %v", InvalidFieldError, fieldName, err)
}
