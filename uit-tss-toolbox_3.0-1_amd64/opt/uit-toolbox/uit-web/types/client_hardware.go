package types

type HardwareData struct {
	Tagnumber               *int64  `json:"tagnumber"`
	SystemSerial            *string `json:"system_serial"`
	EthernetMAC             *string `json:"ethernet_mac"`
	WifiMac                 *string `json:"wifi_mac"`
	SystemModel             *string `json:"system_model"`
	SystemUUID              *string `json:"system_uuid"`
	SystemSKU               *string `json:"system_sku"`
	ChassisType             *string `json:"chassis_type"`
	MotherboardManufacturer *string `json:"motherboard_manufacturer"`
	MotherboardSerial       *string `json:"motherboard_serial"`
	SystemManufacturer      *string `json:"system_manufacturer"`
}

type DeviceType struct {
	DeviceType          *string `json:"device_type"`
	DeviceTypeFormatted *string `json:"device_type_formatted"`
	DeviceMetaCategory  *string `json:"device_meta_category"`
	DeviceTypeCount     *int64  `json:"device_type_count"`
	SortOrder           *int64  `json:"sort_order"`
}
