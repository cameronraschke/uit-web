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

func (repo *Repo) InsertInventory(ctx context.Context, tagnumber int64, systemSerial string, location string, department *string, domain *string, working bool, status *string, note *string) error {
	sqlCode := `INSERT INTO locations (time, tagnumber, system_serial, location, department, domain, working, status, note) 
		VALUES 
	(CURRENT_TIMESTAMP, $1, $2, $3, $4, $5, $6, $7, $8, $9);`

	_, err := repo.DB.ExecContext(ctx, sqlCode, tagnumber, systemSerial, location, department, domain, working, status, note)
	return err
}

func (repo *Repo) UpdateSystemData(ctx context.Context, tagnumber int64, systemManufacturer *string, systemModel *string) error {
	sqlCode := `UPDATE system_data SET  system_manufacturer = $2, system_model = $3 WHERE tagnumber = $1;`
	_, err := repo.DB.ExecContext(ctx, sqlCode, tagnumber, systemManufacturer, systemModel)
	return err
}
