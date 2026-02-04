package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

func (repo *Repo) InsertNewNote(ctx context.Context, time time.Time, noteType, note string) error {
	sqlCode := `INSERT INTO notes (time, note_type, note) VALUES ($1, $2, $3);`

	rowsAffected, err := repo.DB.ExecContext(ctx, sqlCode, time, noteType, note)
	if rowsAffected == nil {
		return errors.New("no rows affected when inserting new note")
	}

	return err
}

func (repo *Repo) InsertInventoryUpdateForm(ctx context.Context, inventoryUpdateForm *InventoryUpdateForm) error {
	if repo.DB == nil {
		return fmt.Errorf("database connection is nil in InsertInventory")
	}

	tx, err := repo.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		} else {
			err = tx.Commit()
		}
	}()

	transactionUUID, err := uuid.NewUUID()
	if err != nil {
		return fmt.Errorf("error generating transaction UUID in InsertInventoryUpdateForm: %w", err)
	}

	const locationsSql = `INSERT INTO locations 
		(time, 
		transaction_uuid,
		tagnumber, 
		system_serial, 
		location, 
		building, 
		room, 
		department_name, 
		ad_domain, 
		property_custodian,  
		acquired_date,
		retired_date,
		is_broken, 
		disk_removed, 
		client_status,
		note) 
		VALUES 
	(CURRENT_TIMESTAMP, $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15);`

	var locationsResult sql.Result
	locationsResult, err = tx.ExecContext(ctx, locationsSql,
		transactionUUID,
		toNullInt64(inventoryUpdateForm.Tagnumber),
		toNullString(inventoryUpdateForm.SystemSerial),
		toNullString(inventoryUpdateForm.Location),
		toNullString(inventoryUpdateForm.Building),
		toNullString(inventoryUpdateForm.Room),
		toNullString(inventoryUpdateForm.Department),
		toNullString(inventoryUpdateForm.Domain),
		toNullString(inventoryUpdateForm.PropertyCustodian),
		toNullTime(inventoryUpdateForm.AcquiredDate),
		toNullTime(inventoryUpdateForm.RetiredDate),
		toNullBool(inventoryUpdateForm.Broken),
		toNullBool(inventoryUpdateForm.DiskRemoved),
		toNullString(inventoryUpdateForm.ClientStatus),
		toNullString(inventoryUpdateForm.Note),
	)
	if err != nil {
		return err
	}
	locationRowsAffected, rowsAffectedErr := locationsResult.RowsAffected()
	if rowsAffectedErr != nil {
		return fmt.Errorf("Error getting number of rows affected on locations table insert (InsertInventoryUpdateForm): %s", err.Error())
	}
	if locationRowsAffected != 1 {
		return fmt.Errorf("During locations update, %d rows were affected on insert (InsertInventoryUpdateForm)", locationRowsAffected)
	}

	var hardwareDataResult sql.Result
	const hardwareDataSql = `INSERT INTO hardware_data
		(time, transaction_uuid, tagnumber, system_manufacturer, system_model) 
		VALUES (CURRENT_TIMESTAMP, $1, $2, $3, $4)
		ON CONFLICT (tagnumber)
		DO UPDATE SET
			time = CURRENT_TIMESTAMP,
			transaction_uuid = $1,
			tagnumber = $2,
			system_manufacturer = $3,
			system_model = $4;`
	hardwareDataResult, err = tx.ExecContext(ctx, hardwareDataSql,
		transactionUUID,
		toNullInt64(inventoryUpdateForm.Tagnumber),
		toNullString(inventoryUpdateForm.SystemManufacturer),
		toNullString(inventoryUpdateForm.SystemModel),
	)
	if err != nil {
		return err
	}
	hardwareDataRowsAffected, rowsAffectedErr := hardwareDataResult.RowsAffected()
	if rowsAffectedErr != nil {
		return fmt.Errorf("Error getting number of rows affected on hardware_data table insert (InsertInventoryUpdateForm): %s", err.Error())
	}
	if hardwareDataRowsAffected != 1 {
		return fmt.Errorf("During locations update, %d rows were affected on insert (InsertInventoryUpdateForm)", hardwareDataRowsAffected)
	}

	var checkoutLogResult sql.Result
	const checkoutSql = `INSERT INTO checkout_log
		(log_entry_time, transaction_uuid, tagnumber, checkout_date, return_date, checkout_bool)
		VALUES (CURRENT_TIMESTAMP, $1, $2, $3, $4, $5);`

	checkoutLogResult, err = tx.ExecContext(ctx, checkoutSql,
		transactionUUID,
		toNullInt64(inventoryUpdateForm.Tagnumber),
		toNullTime(inventoryUpdateForm.CheckoutDate),
		toNullTime(inventoryUpdateForm.ReturnDate),
		toNullBool(inventoryUpdateForm.CheckoutBool),
	)
	if err != nil {
		return err
	}

	checkoutLogRowsAffected, rowsAffectedErr := checkoutLogResult.RowsAffected()
	if rowsAffectedErr != nil {
		return fmt.Errorf("Error getting number of rows affected on checkout_log table insert (InsertInventoryUpdateForm): %s", err.Error())
	}
	if checkoutLogRowsAffected != 1 {
		return fmt.Errorf("During checkout_log update, %d rows were affected on insert (InsertInventoryUpdateForm)", checkoutLogRowsAffected)
	}

	return nil
}

func (repo *Repo) UpdateSystemData(ctx context.Context, tagnumber int64, systemManufacturer *string, systemModel *string) error {
	sqlCode := `INSERT INTO hardware_data (tagnumber, system_manufacturer, system_model) 
			VALUES ($1, $2, $3)
			ON CONFLICT (tagnumber) DO 
			UPDATE SET 
				system_manufacturer = EXCLUDED.system_manufacturer, 
				system_model = EXCLUDED.system_model;`
	_, err := repo.DB.ExecContext(ctx, sqlCode, tagnumber, systemManufacturer, systemModel)
	if err != nil {
		return err
	}
	return nil
}

func (repo *Repo) UpdateClientImages(ctx context.Context, manifest ImageManifest) (err error) {
	if repo.DB == nil {
		return fmt.Errorf("database connection is nil in UpdateClientImages")
	}
	sqlCode := `INSERT INTO client_images (uuid, 
		time, 
		tagnumber, 
		filename, 
		filepath, 
		thumbnail_filepath, 
		filesize, 
		sha256_hash, 
		mime_type, 
		exif_timestamp, 
		resolution_x, 
		resolution_y, 
		note, 
		hidden, 
		primary_image)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15);`

	tx, err := repo.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		} else {
			err = tx.Commit()
		}
	}()

	_, err = tx.ExecContext(ctx, sqlCode,
		toNullString(manifest.UUID),
		toNullTime(manifest.Time),
		toNullInt64(manifest.Tagnumber),
		toNullString(manifest.FileName),
		toNullString(manifest.FilePath),
		toNullString(manifest.ThumbnailFilePath),
		toNullInt64(manifest.FileSize),
		toNullString(manifest.SHA256Hash),
		toNullString(manifest.MimeType),
		toNullTime(manifest.ExifTimestamp),
		toNullInt64(manifest.ResolutionX),
		toNullInt64(manifest.ResolutionY),
		toNullString(manifest.Note),
		toNullBool(manifest.Hidden),
		toNullBool(manifest.PrimaryImage),
	)
	if err != nil {
		return err
	}
	return nil
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

func (repo *Repo) SetClientBatteryHealth(ctx context.Context, uuid string, healthPcnt *int64) (err error) {
	if repo.DB == nil {
		return fmt.Errorf("database connection is nil in SetClientBatteryHealth")
	}

	tx, err := repo.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		} else {
			err = tx.Commit()
		}
	}()

	if strings.TrimSpace(uuid) == "" {
		err = fmt.Errorf("UUID is empty in SetClientBatteryHealth")
		return err
	}
	if healthPcnt == nil {
		err = fmt.Errorf("health percentage is nil in SetClientBatteryHealth")
		return err
	}
	sql := `UPDATE jobstats SET battery_health = $1 WHERE uuid = $2;`
	result, err := tx.ExecContext(ctx, sql, healthPcnt, uuid)
	if err != nil {
		return err
	}
	if result == nil {
		err = errors.New("no result returned when updating battery health")
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows != 1 {
		err = errors.New("unexpected number of rows affected when updating battery health")
		return err
	}

	return nil
}

func (repo *Repo) SetAllJobs(ctx context.Context, allJobs AllJobs) (err error) {
	if repo.DB == nil {
		return errors.New("database connection is nil in SetAllJobs")
	}
	var job = allJobs.JobName

	sqlCode := `UPDATE job_queue SET job_queued = $1 WHERE NOW() - present < INTERVAL '30 SECONDS';`

	tx, err := repo.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		} else {
			err = tx.Commit()
		}
	}()
	// Don't check rows affected - could be no clients online
	_, err = tx.ExecContext(ctx, sqlCode, job)
	if err != nil {
		return err
	}

	return nil
}
