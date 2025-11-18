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

func (repo *Repo) InsertInventory(ctx context.Context, inventoryUpdateFormInput *InventoryUpdateFormInput) error {
	sqlCode := `INSERT INTO locations (time, tagnumber, system_serial, location, building, room, is_broken, disk_removed, department_name, property_custodian, ad_domain, note, client_status, acquired_date) 
		VALUES 
	(CURRENT_TIMESTAMP, $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13);`

	timeInputLayout := "2006-01-02"
	if inventoryUpdateFormInput.AcquiredDateString != nil && *inventoryUpdateFormInput.AcquiredDateString != "" {
		parsedTime, err := time.Parse(timeInputLayout, *inventoryUpdateFormInput.AcquiredDateString)
		if err != nil {
			return errors.New("failed to parse acquired date: " + err.Error())
		}
		inventoryUpdateFormInput.AcquiredDate = &parsedTime
	} else {
		inventoryUpdateFormInput.AcquiredDate = nil
	}
	_, err := repo.DB.ExecContext(ctx, sqlCode,
		toNullInt64(inventoryUpdateFormInput.Tagnumber),
		toNullString(inventoryUpdateFormInput.SystemSerial),
		toNullString(inventoryUpdateFormInput.Location),
		toNullString(inventoryUpdateFormInput.Building),
		toNullString(inventoryUpdateFormInput.Room),
		toNullBool(inventoryUpdateFormInput.Broken),
		toNullBool(inventoryUpdateFormInput.DiskRemoved),
		toNullString(inventoryUpdateFormInput.Department),
		toNullString(inventoryUpdateFormInput.PropertyCustodian),
		toNullString(inventoryUpdateFormInput.Domain),
		toNullString(inventoryUpdateFormInput.Note),
		toNullString(inventoryUpdateFormInput.Status),
		toNullTime(inventoryUpdateFormInput.AcquiredDate),
	)
	return err
}

func (repo *Repo) UpdateSystemData(ctx context.Context, tagnumber int64, systemManufacturer *string, systemModel *string) error {
	sqlCode := `INSERT INTO system_data (tagnumber, system_manufacturer, system_model) 
			VALUES ($1, $2, $3)
			ON CONFLICT (tagnumber) DO 
			UPDATE SET 
				system_manufacturer = EXCLUDED.system_manufacturer, 
				system_model = EXCLUDED.system_model;`
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

func (repo *Repo) TogglePinImage(ctx context.Context, uuid string, tagnumber int64) (err error) {
	sqlQuery := `UPDATE client_images SET primary_image = NOT COALESCE(primary_image, FALSE) WHERE uuid = $1 AND tagnumber = $2;`
	_, err = repo.DB.ExecContext(ctx, sqlQuery, uuid, tagnumber)
	return err
}
