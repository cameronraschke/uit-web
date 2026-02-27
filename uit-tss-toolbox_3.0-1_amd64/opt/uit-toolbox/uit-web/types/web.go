package types

import "time"

type AuthStatusResponse struct {
	Status    string        `json:"status"`
	ExpiresAt time.Time     `json:"expires_at"`
	TTL       time.Duration `json:"ttl"`
}

type ClientLookup struct {
	Tagnumber    *int64  `json:"tagnumber"`
	SystemSerial *string `json:"system_serial"`
}

type GeneralNoteRow struct {
	Time     *time.Time `json:"time"`
	NoteType *string    `json:"note_type"`
	Note     *string    `json:"note"`
	ToDo     *string    `json:"todo"`
	Projects *string    `json:"projects"`
	Misc     *string    `json:"misc"`
	Bugs     *string    `json:"bugs"`
}
