package types

import "time"

type ClientLookup struct {
	Tagnumber    *int64  `json:"tagnumber"`
	SystemSerial *string `json:"system_serial"`
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
