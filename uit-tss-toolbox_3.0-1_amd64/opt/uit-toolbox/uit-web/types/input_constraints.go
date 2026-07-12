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
	GeneralNote   *GeneralNoteConstraints
	InventoryForm *InventoryUpdateFormConstraints
}

type InventoryUpdateFormConstraints struct {
	MaxJSONBytes                 int64
	AcquiredDateIsMandatory      bool
	RetiredDateIsMandatory       bool
	IsFunctionalIsMandatory      bool
	DiskRemovedIsMandatory       bool
	LastHardwareCheckIsMandatory bool
	CheckoutBoolIsMandatory      bool
	CheckoutDateIsMandatory      bool
	ReturnDateIsMandatory        bool
}

type GeneralNoteConstraints struct {
	MaxFormBytes        int64
}
