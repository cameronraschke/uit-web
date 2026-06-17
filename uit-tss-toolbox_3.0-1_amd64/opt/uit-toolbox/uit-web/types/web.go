package types

import "time"

type GeneralNoteResponse struct {
	Time        *time.Time `json:"time"`
	NoteType    *string    `json:"note_type"`
	NoteContent *string    `json:"note"`
	ToDo        *string    `json:"todo"`
	Projects    *string    `json:"projects"`
	Misc        *string    `json:"misc"`
	Bugs        *string    `json:"bugs"`
}
