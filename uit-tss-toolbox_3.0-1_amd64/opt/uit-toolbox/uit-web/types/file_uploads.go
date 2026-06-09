package types

import "time"

type ImageManifestDTO struct {
	Time              time.Time  `json:"time"`
	Tagnumber         int64      `json:"tagnumber"`
	FileUUID          string     `json:"file_uuid"`
	SHA256Hash        []uint8    `json:"sha256_hash"`
	FileName          string     `json:"filename"`
	ThumbnailFileName *string    `json:"thumbnail_filename"`
	FileSize          int64      `json:"file_size"`
	MimeType          string     `json:"mime_type"`
	ExifTimestamp     *time.Time `json:"exif_timestamp"`
	ResolutionX       *int64     `json:"resolution_x"`
	ResolutionY       *int64     `json:"resolution_y"`
	URL               string     `json:"url"`
	Hidden            bool       `json:"hidden"`
	Pinned            bool       `json:"pinned"`
	Caption           *string    `json:"caption"`
	FileType          string     `json:"file_type"`
}

type ImageManifestResponse struct {
	Time              *time.Time `json:"time"`
	ClientUUID        *string    `json:"client_uuid"`
	Tagnumber         *int64     `json:"tagnumber"`
	FileUUID          *string    `json:"file_uuid"`
	SHA256Hash        *[]uint8   `json:"sha256_hash"`
	FileName          *string    `json:"filename"`
	ThumbnailFileName *string    `json:"thumbnail_filename"`
	FileSize          *int64     `json:"file_size"`
	MimeType          *string    `json:"mime_type"`
	ExifTimestamp     *time.Time `json:"exif_timestamp"`
	ResolutionX       *int64     `json:"resolution_x"`
	ResolutionY       *int64     `json:"resolution_y"`
	URL               *string    `json:"url"`
	Hidden            *bool      `json:"hidden"`
	Pinned            *bool      `json:"pinned"`
	Caption           *string    `json:"caption"`
	FileType          *string    `json:"file_type"`
}

type ImageUploadRequest struct {
	Tagnumber  *int64  `json:"tagnumber"`
	FileName   *string `json:"filename"`
	Caption    *string `json:"caption"`
	ImageBytes []byte
}
