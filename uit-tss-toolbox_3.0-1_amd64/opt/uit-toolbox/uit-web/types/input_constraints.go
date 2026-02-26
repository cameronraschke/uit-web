package types

type FileUploadConstraints struct {
	ImageConstraints       *ImageUploadConstraints
	VideoConstraints       *VideoUploadConstraints
	MaxUploadFileSizeLimit int64
}

type ImageUploadConstraints struct {
	MinFileSize                         int64
	MaxFileSize                         int64
	MaxFileCount                        int
	AcceptedImageExtensionsAndMimeTypes map[string]string
}

type VideoUploadConstraints struct {
	MinFileSize                         int64
	MaxFileSize                         int64
	MaxFileCount                        int
	AcceptedVideoExtensionsAndMimeTypes map[string]string
}

type HTMLFormConstraints struct {
	LoginForm     *LoginFormConstraints
	GeneralNote   *GeneralNoteConstraints
	InventoryForm *InventoryUpdateFormConstraints
}

type LoginFormConstraints struct {
	MaxFormBytes     int64
	UsernameMinChars int
	UsernameMaxChars int
	PasswordMinChars int
	PasswordMaxChars int
}

type InventoryUpdateFormConstraints struct {
	MaxJSONBytes                 int64
	TagnumberMinChars            int
	TagnumberMaxChars            int
	SystemSerialMinChars         int
	SystemSerialMaxChars         int
	LocationMinChars             int
	LocationMaxChars             int
	BuildingMinChars             int
	BuildingMaxChars             int
	RoomMinChars                 int
	RoomMaxChars                 int
	ManufacturerMinChars         int
	ManufacturerMaxChars         int
	SystemModelMinChars          int
	SystemModelMaxChars          int
	DeviceTypeMinChars           int
	DeviceTypeMaxChars           int
	DepartmentMinChars           int
	DepartmentMaxChars           int
	DomainMinChars               int
	DomainMaxChars               int
	PropertyCustodianMinChars    int
	PropertyCustodianMaxChars    int
	AcquiredDateIsMandatory      bool
	RetiredDateIsMandatory       bool
	IsFunctionalIsMandatory      bool
	DiskRemovedIsMandatory       bool
	LastHardwareCheckIsMandatory bool
	ClientStatusMinChars         int
	ClientStatusMaxChars         int
	CheckoutBoolIsMandatory      bool
	CheckoutDateIsMandatory      bool
	ReturnDateIsMandatory        bool
	ClientNoteMinChars           int
	ClientNoteMaxChars           int
}

type GeneralNoteConstraints struct {
	MaxFormBytes        int64
	NoteTypeMinChars    int
	NoteTypeMaxChars    int
	NoteContentMinChars int
	NoteContentMaxChars int
}
