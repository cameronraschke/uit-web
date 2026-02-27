package types

import (
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
	Domain             *string    `json:"ad_domain"`
	PropertyCustodian  *string    `json:"property_custodian"`
	AcquiredDate       *time.Time `json:"acquired_date"`
	RetiredDate        *time.Time `json:"retired_date"`
	Broken             *bool      `json:"is_broken"`
	DiskRemoved        *bool      `json:"disk_removed"`
	LastHardwareCheck  *time.Time `json:"last_hardware_check"`
	ClientStatus       *string    `json:"status"`
	CheckoutBool       *bool      `json:"checkout_bool"`
	CheckoutDate       *time.Time `json:"checkout_date"`
	ReturnDate         *time.Time `json:"return_date"`
	Note               *string    `json:"note"`
}

// Domain model for inventory update operations after ingress
type InventoryUpdateDomain struct {
	Tagnumber          int64
	SystemSerial       string
	Location           string
	Building           *string
	Room               *string
	SystemManufacturer *string
	SystemModel        *string
	DeviceType         *string
	Department         string
	Domain             string
	PropertyCustodian  *string
	AcquiredDate       *time.Time
	RetiredDate        *time.Time
	Broken             *bool
	DiskRemoved        *bool
	LastHardwareCheck  *time.Time
	ClientStatus       string
	CheckoutBool       *bool
	CheckoutDate       *time.Time
	ReturnDate         *time.Time
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
	Domain            string
	PropertyCustodian *string
	AcquiredDate      *time.Time
	RetiredDate       *time.Time
	Broken            *bool
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

type InventoryClientHealthWriteModel struct {
	TransactionUUID   uuid.UUID
	Tagnumber         int64
	LastHardwareCheck *time.Time
}

type InventoryCheckoutWriteModel struct {
	TransactionUUID uuid.UUID
	Tagnumber       int64
	CheckoutDate    *time.Time
	ReturnDate      *time.Time
	CheckoutBool    *bool
}
