package types

import (
	"errors"
)

var (
	DatabaseConnError         = errors.New("error getting database connection from app state")
	EmptyTransactionUUIDError = errors.New("transaction UUID is nil")
)
