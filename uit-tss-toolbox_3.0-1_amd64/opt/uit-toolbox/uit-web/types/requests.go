package types

type WindowsUpdateRequest struct {
	Tagnumber    *int64  `json:"tagnumber"`
	SystemSerial *string `json:"system_serial"`
}
