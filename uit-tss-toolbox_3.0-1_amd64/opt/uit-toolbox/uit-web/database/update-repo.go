package database

import (
	"context"
	"errors"
	"time"
)

func (repo *Repo) InsertNewNote(ctx context.Context, time time.Time, noteType, note string) error {
	sqlCode := `INSERT INTO notes (time, note_type, note) VALUES ($1, $2, $3);`

	rowsAffected, err := repo.DB.ExecContext(ctx, sqlCode, time, noteType, note)
	if rowsAffected == nil {
		return errors.New("no rows affected when inserting new note")
	}

	return err
}

func (repo *Repo) InsertInventory(ctx context.Context, tagnumber int64, systemSerial string, location string, systemManufacturer *string, systemModel *string, department *string, domain *string, working bool, status *string, note *string, image *string) error {
	sqlCode := `INSERT INTO locations (time, tagnumber, system_serial, location, system_manufacturer, system_model, department, domain, working, status, note, image) 
		VALUES 
	(CURRENT_TIMESTAMP, $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11);`

	_, err := repo.DB.ExecContext(ctx, sqlCode, tagnumber, systemSerial, location, systemManufacturer, systemModel, department, domain, working, status, note, image)
	return err
}
