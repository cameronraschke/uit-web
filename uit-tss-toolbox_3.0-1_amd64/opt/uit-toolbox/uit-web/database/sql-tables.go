package database

import "time"

type RemoteTable struct {
	Tagnumber         *int    `json:"tagnumber"`
	JobQueued         *string `json:"job_queued"`
	JobQueuedPosition *int    `json:"job_queued_position"`
	JobActive         *bool   `json:"job_active"`
	CloneMode         *string `json:"clone_mode"`
	EraseMode         *string `json:"erase_mode"`
	LastJobTime       *string `json:"last_job_time"`
	Present           *string `json:"present"`
	PresentBool       *bool   `json:"present_bool"`
	Status            *string `json:"status"`
	KernelUpdated     *bool   `json:"kernel_updated"`
	BatteryCharge     *int    `json:"battery_charge"`
	BatteryStatus     *string `json:"battery_status"`
	Uptime            *int    `json:"uptime"`
	CpuTemp           *int    `json:"cpu_temp"`
	DiskTemp          *int    `json:"disk_temp"`
	MaxDiskTemp       *int    `json:"max_disk_temp"`
	WattsNow          *int    `json:"watts_now"`
	NetworkSpeed      *int    `json:"network_speed"`
}

type LocationsTable struct {
	Time          *time.Time `json:"time"`
	Tagnumber     *int       `json:"tagnumber"`
	SystemSerial  *string    `json:"system_serial"`
	Location      *string    `json:"location"`
	Status        *bool      `json:"status"`
	StatusWorking *bool      `json:"status_working"`
	DiskRemoved   *bool      `json:"disk_removed"`
	Department    *string    `json:"department"`
	Domain        *string    `json:"domain"`
	Note          *string    `json:"note"`
}

type NotesTable struct {
	Time     *time.Time `json:"time"`
	NoteType *string    `json:"note_type"`
	Note     *string    `json:"note"`
	ToDo     *string    `json:"todo"`
	Projects *string    `json:"projects"`
	Misc     *string    `json:"misc"`
	Bugs     *string    `json:"bugs"`
}

type LoginsTable struct {
	Username   *string `json:"username"`
	Password   *string `json:"password"`
	Email      *string `json:"email"`
	FirstName  *string `json:"first_name"`
	LastName   *string `json:"last_name"`
	CommonName *string `json:"common_name"`
	Role       *string `json:"role"`
	IsAdmin    *bool   `json:"is_admin"`
	Enabled    *bool   `json:"enabled"`
}

type ClientImagesTable struct {
	UUID              *string    `json:"uuid"`
	Time              *time.Time `json:"time"`
	Tagnumber         *int       `json:"tagnumber"`
	Filename          *string    `json:"filename"`
	FilePath          *string    `json:"file_path"`
	ThumbnailFilePath *string    `json:"thumbnail_filepath"`
	Filesize          *float64   `json:"filesize"`
	SHA256Hash        *[]byte    `json:"sha256_hash"`
	MimeType          *string    `json:"mime_type"`
	ExifTimestamp     *time.Time `json:"exif_timestamp"`
	ResolutionX       *int       `json:"resolution_x"`
	ResolutionY       *int       `json:"resolution_y"`
	Note              *string    `json:"note"`
	Hidden            *bool      `json:"hidden"`
	PrimaryImage      *bool      `json:"primary_image"`
}
