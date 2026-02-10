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

func (repo *Repo) InsertNewNote(ctx context.Context, time *time.Time, noteType *string, note *string) (err error) {
	if time == nil {
		return errors.New("time is required in InsertNewNote")
	}
	if noteType == nil || strings.TrimSpace(*noteType) == "" {
		return errors.New("note type is required in InsertNewNote")
	}

	if ctx.Err() != nil {
		return fmt.Errorf("context error in InsertNewNote: %w", ctx.Err())
	}

	if repo.DB == nil {
		return fmt.Errorf("database connection is nil in InsertNewNote")
	}
	tx, err := repo.DB.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("error beginning transaction in InsertNewNote: %w", err)
	}
	if tx == nil {
		return errors.New("transaction is nil in InsertNewNote")
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		} else {
			err = tx.Commit()
		}
	}()

	sqlCode := `INSERT INTO notes (time, note_type, note) VALUES ($1, $2, $3);`
	rowsAffected, err := tx.ExecContext(ctx, sqlCode,
		ToNullTime(time),
		ToNullString(noteType),
		ToNullString(note),
	)
	if rowsAffected == nil {
		return fmt.Errorf("no rows affected when inserting new note")
	}

	return err
}

func (repo *Repo) InsertInventoryUpdateForm(ctx context.Context, transactionUUID uuid.UUID, inventoryUpdateForm *InventoryUpdateForm) (err error) {
	if transactionUUID == uuid.Nil || strings.TrimSpace(transactionUUID.String()) == "" {
		return fmt.Errorf("generated transaction UUID is nil in InsertInventoryUpdateForm")
	}
	if inventoryUpdateForm == nil {
		return fmt.Errorf("inventoryUpdateForm is nil in InsertInventoryUpdateForm")
	}

	if ctx.Err() != nil {
		return fmt.Errorf("context error in InsertInventoryUpdateForm: %w", ctx.Err())
	}

	if repo.DB == nil {
		return fmt.Errorf("database connection is nil in InsertInventoryUpdateForm")
	}
	tx, err := repo.DB.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("error beginning transaction in InsertInventoryUpdateForm: %w", err)
	}
	if tx == nil {
		return fmt.Errorf("transaction is nil in InsertInventoryUpdateForm")
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		} else {
			err = tx.Commit()
		}
	}()

	// Update locations table
	if ctx.Err() != nil {
		return fmt.Errorf("context error in InsertInventoryUpdateForm: %w", ctx.Err())
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
		ToNullInt64(inventoryUpdateForm.Tagnumber),
		ToNullString(inventoryUpdateForm.SystemSerial),
		ToNullString(inventoryUpdateForm.Location),
		ToNullString(inventoryUpdateForm.Building),
		ToNullString(inventoryUpdateForm.Room),
		ToNullString(inventoryUpdateForm.Department),
		ToNullString(inventoryUpdateForm.Domain),
		ToNullString(inventoryUpdateForm.PropertyCustodian),
		ToNullTime(inventoryUpdateForm.AcquiredDate),
		ToNullTime(inventoryUpdateForm.RetiredDate),
		ToNullBool(inventoryUpdateForm.Broken),
		ToNullBool(inventoryUpdateForm.DiskRemoved),
		ToNullString(inventoryUpdateForm.ClientStatus),
		ToNullString(inventoryUpdateForm.Note),
	)
	if err != nil {
		return fmt.Errorf("error inserting location data in InsertInventoryUpdateForm: %w", err)
	}
	locationRowsAffected, rowsAffectedErr := locationsResult.RowsAffected()
	if rowsAffectedErr != nil {
		return fmt.Errorf("Error getting number of rows affected on locations table insert (InsertInventoryUpdateForm): %w", rowsAffectedErr)
	}
	if locationRowsAffected != 1 {
		return fmt.Errorf("During locations update, %d rows were affected on insert (InsertInventoryUpdateForm)", locationRowsAffected)
	}

	// Update hardware_data table
	if ctx.Err() != nil {
		return fmt.Errorf("context error in InsertInventoryUpdateForm: %w", ctx.Err())
	}
	var hardwareDataResult sql.Result
	const hardwareDataSql = `INSERT INTO hardware_data
		(time, transaction_uuid, tagnumber, system_manufacturer, system_model) 
		VALUES (CURRENT_TIMESTAMP, $1, $2, $3, $4)
		ON CONFLICT (tagnumber)
		DO UPDATE SET
			time = CURRENT_TIMESTAMP,
			transaction_uuid = EXCLUDED.transaction_uuid,
			tagnumber = EXCLUDED.tagnumber,
			system_manufacturer = EXCLUDED.system_manufacturer,
			system_model = EXCLUDED.system_model;`
	hardwareDataResult, err = tx.ExecContext(ctx, hardwareDataSql,
		transactionUUID,
		ToNullInt64(inventoryUpdateForm.Tagnumber),
		ToNullString(inventoryUpdateForm.SystemManufacturer),
		ToNullString(inventoryUpdateForm.SystemModel),
	)
	if err != nil {
		return fmt.Errorf("error inserting/updating hardware data in InsertInventoryUpdateForm: %w", err)
	}
	hardwareDataRowsAffected, rowsAffectedErr := hardwareDataResult.RowsAffected()
	if rowsAffectedErr != nil {
		return fmt.Errorf("error getting number of rows affected on hardware_data table insert (InsertInventoryUpdateForm): %w", rowsAffectedErr)
	}
	if hardwareDataRowsAffected != 1 {
		return fmt.Errorf("during hardware_data update, %d rows were affected on insert (InsertInventoryUpdateForm)", hardwareDataRowsAffected)
	}

	// Insert/update into client_health table
	if ctx.Err() != nil {
		return fmt.Errorf("context error in InsertInventoryUpdateForm: %w", ctx.Err())
	}
	var clientHealthResult sql.Result
	const clientHealthSql = `INSERT INTO client_health
		(time, tagnumber, last_hardware_check, transaction_uuid) VALUES
		(CURRENT_TIMESTAMP, $1, $2, $3)
		ON CONFLICT (tagnumber)
		DO UPDATE SET
			time = CURRENT_TIMESTAMP,
			tagnumber = EXCLUDED.tagnumber,
			last_hardware_check = EXCLUDED.last_hardware_check,
			transaction_uuid = EXCLUDED.transaction_uuid;`

	clientHealthResult, err = tx.ExecContext(ctx, clientHealthSql,
		ToNullInt64(inventoryUpdateForm.Tagnumber),
		ToNullTime(inventoryUpdateForm.LastHardwareCheck),
		transactionUUID,
	)
	if err != nil {
		return fmt.Errorf("error inserting/updating client health data in InsertInventoryUpdateForm: %w", err)
	}
	clientHealthRowsAffected, rowsAffectedErr := clientHealthResult.RowsAffected()
	if rowsAffectedErr != nil {
		return fmt.Errorf("error getting number of rows affected on client_health table insert/update (InsertInventoryUpdateForm): %w", rowsAffectedErr)
	}
	if clientHealthRowsAffected != 1 {
		return fmt.Errorf("during client_health update, %d rows were affected on insert/update (InsertInventoryUpdateForm)", clientHealthRowsAffected)
	}

	// Insert into checkout_log table
	if ctx.Err() != nil {
		return fmt.Errorf("context error in InsertInventoryUpdateForm: %w", ctx.Err())
	}
	if inventoryUpdateForm.CheckoutDate != nil || inventoryUpdateForm.ReturnDate != nil || (inventoryUpdateForm.CheckoutBool != nil && *inventoryUpdateForm.CheckoutBool) {
		var checkoutLogResult sql.Result
		const checkoutSql = `INSERT INTO checkout_log
			(log_entry_time, transaction_uuid, tagnumber, checkout_date, return_date, checkout_bool)
			VALUES (CURRENT_TIMESTAMP, $1, $2, $3, $4, $5);`

		checkoutLogResult, err = tx.ExecContext(ctx, checkoutSql,
			transactionUUID,
			ToNullInt64(inventoryUpdateForm.Tagnumber),
			ToNullTime(inventoryUpdateForm.CheckoutDate),
			ToNullTime(inventoryUpdateForm.ReturnDate),
			ToNullBool(inventoryUpdateForm.CheckoutBool),
		)
		if err != nil {
			return fmt.Errorf("error inserting into checkout_log in InsertInventoryUpdateForm: %w", err)
		}

		checkoutLogRowsAffected, rowsAffectedErr := checkoutLogResult.RowsAffected()
		if rowsAffectedErr != nil {
			return fmt.Errorf("error getting number of rows affected on checkout_log table insert (InsertInventoryUpdateForm): %w", rowsAffectedErr)
		}
		if checkoutLogRowsAffected != 1 {
			return fmt.Errorf("during checkout_log update, %d rows were affected on insert (InsertInventoryUpdateForm)", checkoutLogRowsAffected)
		}
	}

	return nil
}

func (repo *Repo) UpdateHardwareData(ctx context.Context, tagnumber *int64, systemManufacturer *string, systemModel *string) (err error) {
	if tagnumber == nil {
		return fmt.Errorf("tagnumber is nil in UpdateHardwareData")
	}

	if ctx.Err() != nil {
		return fmt.Errorf("context error in UpdateHardwareData: %w", ctx.Err())
	}

	if repo.DB == nil {
		return fmt.Errorf("database connection is nil in UpdateHardwareData")
	}
	tx, err := repo.DB.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("error beginning transaction in UpdateHardwareData: %w", err)
	}
	if tx == nil {
		return fmt.Errorf("transaction is nil in UpdateHardwareData")
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		} else {
			err = tx.Commit()
		}
	}()

	const sqlCode = `INSERT INTO hardware_data (tagnumber, system_manufacturer, system_model) 
			VALUES ($1, $2, $3)
			ON CONFLICT (tagnumber) DO 
			UPDATE SET 
				system_manufacturer = EXCLUDED.system_manufacturer, 
				system_model = EXCLUDED.system_model;`
	_, err = tx.ExecContext(ctx, sqlCode,
		ToNullInt64(tagnumber),
		ToNullString(systemManufacturer),
		ToNullString(systemModel),
	)
	if err != nil {
		return fmt.Errorf("error updating hardware data in UpdateHardwareData: %w", err)
	}
	return nil
}

func (repo *Repo) UpdateClientImages(ctx context.Context, transactionUUID uuid.UUID, manifest ImageManifest) (err error) {
	if ctx.Err() != nil {
		return fmt.Errorf("context error in UpdateClientImages: %w", ctx.Err())
	}

	if repo.DB == nil {
		return fmt.Errorf("database connection is nil in UpdateClientImages")
	}
	tx, err := repo.DB.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("error beginning transaction in UpdateClientImages: %w", err)
	}
	if tx == nil {
		return fmt.Errorf("transaction is nil in UpdateClientImages")
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		} else {
			err = tx.Commit()
		}
	}()

	const sqlCode = `INSERT INTO client_images (uuid, 
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

	_, err = tx.ExecContext(ctx, sqlCode,
		ToNullString(manifest.UUID),
		ToNullTime(manifest.Time),
		ToNullInt64(manifest.Tagnumber),
		ToNullString(manifest.FileName),
		ToNullString(manifest.FilePath),
		ToNullString(manifest.ThumbnailFilePath),
		ToNullInt64(manifest.FileSize),
		ToNullString(manifest.SHA256Hash),
		ToNullString(manifest.MimeType),
		ToNullTime(manifest.ExifTimestamp),
		ToNullInt64(manifest.ResolutionX),
		ToNullInt64(manifest.ResolutionY),
		ToNullString(manifest.Note),
		ToNullBool(manifest.Hidden),
		ToNullBool(manifest.PrimaryImage),
	)
	if err != nil {
		return fmt.Errorf("error inserting client image in UpdateClientImages: %w", err)
	}
	return nil
}

func (repo *Repo) HideClientImageByUUID(ctx context.Context, tagnumber *int64, uuid *string) (err error) {
	if tagnumber == nil || uuid == nil || strings.TrimSpace(*uuid) == "" {
		return fmt.Errorf("tagnumber and uuid are required in HideClientImageByUUID")
	}

	if ctx.Err() != nil {
		return fmt.Errorf("context error in HideClientImageByUUID: %w", ctx.Err())
	}

	if repo.DB == nil {
		return fmt.Errorf("database connection is nil in HideClientImageByUUID")
	}
	tx, err := repo.DB.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("error beginning transaction in HideClientImageByUUID: %w", err)
	}
	if tx == nil {
		return fmt.Errorf("transaction is nil in HideClientImageByUUID")
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		} else {
			err = tx.Commit()
		}
	}()

	const sqlQuery = `UPDATE client_images SET hidden = TRUE WHERE tagnumber = $1 AND uuid = $2;`
	_, err = tx.ExecContext(ctx, sqlQuery,
		ToNullInt64(tagnumber),
		ToNullString(uuid),
	)
	if err != nil {
		return fmt.Errorf("error hiding client image in HideClientImageByUUID: %w", err)
	}
	return nil
}

func (repo *Repo) TogglePinImage(ctx context.Context, tagnumber *int64, uuid *string) (err error) {
	if tagnumber == nil || uuid == nil || strings.TrimSpace(*uuid) == "" {
		return fmt.Errorf("tagnumber and uuid are required in TogglePinImage")
	}

	if ctx.Err() != nil {
		return fmt.Errorf("context error in TogglePinImage: %w", ctx.Err())
	}

	if repo.DB == nil {
		return fmt.Errorf("database connection is nil in TogglePinImage")
	}
	tx, err := repo.DB.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("error beginning transaction in TogglePinImage: %w", err)
	}
	if tx == nil {
		return fmt.Errorf("transaction is nil in TogglePinImage")
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		} else {
			err = tx.Commit()
		}
	}()

	const sqlQuery = `UPDATE client_images SET primary_image = NOT COALESCE(primary_image, FALSE) WHERE uuid = $1 AND tagnumber = $2;`
	_, err = tx.ExecContext(ctx, sqlQuery,
		ToNullString(uuid),
		ToNullInt64(tagnumber),
	)
	if err != nil {
		return fmt.Errorf("error toggling pin on client image in TogglePinImage: %w", err)
	}
	return nil
}

func (repo *Repo) SetClientBatteryHealth(ctx context.Context, uuid *string, healthPcnt *int64) (err error) {
	if uuid == nil || strings.TrimSpace(*uuid) == "" {
		return fmt.Errorf("UUID is required in SetClientBatteryHealth")
	}
	if healthPcnt == nil {
		return fmt.Errorf("health percentage is required in SetClientBatteryHealth")
	}
	if *healthPcnt < 0 || *healthPcnt > 100 {
		return fmt.Errorf("health percentage must be between 0 and 100 in SetClientBatteryHealth")
	}

	if ctx.Err() != nil {
		return fmt.Errorf("context error in SetClientBatteryHealth: %w", ctx.Err())
	}

	if repo.DB == nil {
		return fmt.Errorf("database connection is nil in SetClientBatteryHealth")
	}
	tx, err := repo.DB.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("error beginning transaction in SetClientBatteryHealth: %w", err)
	}
	if tx == nil {
		return fmt.Errorf("transaction is nil in SetClientBatteryHealth")
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		} else {
			err = tx.Commit()
		}
	}()

	const sqlCode = `UPDATE jobstats SET battery_health = $1 WHERE uuid = $2;`
	_, err = tx.ExecContext(ctx, sqlCode,
		ToNullInt64(healthPcnt),
		ToNullString(uuid),
	)
	if err != nil {
		return fmt.Errorf("error updating jobstats battery health in SetClientBatteryHealth: %w", err)
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
