package database

import (
	"context"
	"errors"
	"time"
)

func toNullString(p *string) any {
	if p == nil {
		return nil
	}
	return *p
}
func toNullBool(p *bool) any {
	if p == nil {
		return nil
	}
	return *p
}

func (repo *Repo) InsertNewNote(ctx context.Context, time time.Time, noteType, note string) error {
	sqlCode := `INSERT INTO notes (time, note_type, note) VALUES ($1, $2, $3);`

	rowsAffected, err := repo.DB.ExecContext(ctx, sqlCode, time, noteType, note)
	if rowsAffected == nil {
		return errors.New("no rows affected when inserting new note")
	}

	return err
}

func (repo *Repo) InsertInventory(ctx context.Context, tagnumber int64, systemSerial *string, location *string, department *string, domain *string, functional *bool, status *string, note *string) error {
	sqlCode := `INSERT INTO locations (time, tagnumber, system_serial, location, department, domain, functional, status, note) 
		VALUES 
	(CURRENT_TIMESTAMP, $1, $2, $3, $4, $5, $6, $7, $8);`

	_, err := repo.DB.ExecContext(ctx, sqlCode,
		tagnumber,
		toNullString(systemSerial),
		toNullString(location),
		toNullString(department),
		toNullString(domain),
		toNullBool(functional),
		toNullString(status),
		toNullString(note),
	)
	return err
}

func (repo *Repo) UpdateSystemData(ctx context.Context, tagnumber int64, systemManufacturer *string, systemModel *string) error {
	sqlCode := `UPDATE system_data 
			SET 
				system_manufacturer = COALESCE($2, system_manufacturer), 
				system_model = COALESCE($3, system_model)
			WHERE tagnumber = $1;`
	_, err := repo.DB.ExecContext(ctx, sqlCode, tagnumber, systemManufacturer, systemModel)
	return err
}

func (repo *Repo) UpdateClientImages(ctx context.Context, tagnumber int64, uuid string, filename *string, filePath string, thumbnailFilePath *string, filesize *float64, sha256Hash *[]byte, mimeType *string, exifTimestamp *time.Time, resolutionX *int, resolutionY *int, note *string, hidden *bool, primaryImage *bool) error {
	sqlCode := `INSERT INTO client_images (uuid, time, tagnumber, filename, filepath, thumbnail_filepath, filesize, sha256_hash, mime_type, exif_timestamp, resolution_x, resolution_y, note, hidden, primary_image)
		VALUES ($1, CURRENT_TIMESTAMP, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14);`
	_, err := repo.DB.ExecContext(ctx, sqlCode, uuid, tagnumber, filename, filePath, thumbnailFilePath, filesize, sha256Hash, mimeType, exifTimestamp, resolutionX, resolutionY, note, hidden, primaryImage)
	return err
}

func (repo *Repo) HideClientImageByUUID(ctx context.Context, tagnumber int64, uuid string) (err error) {
	sqlQuery := `UPDATE client_images SET hidden = TRUE WHERE tagnumber = $1 AND uuid = $2;`
	_, err = repo.DB.ExecContext(ctx, sqlQuery, tagnumber, uuid)
	return err
}
