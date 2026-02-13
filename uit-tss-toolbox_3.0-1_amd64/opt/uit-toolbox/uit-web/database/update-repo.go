package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"
	"uit-toolbox/config"

	"github.com/google/uuid"
)

type Update interface {
	InsertNewNote(ctx context.Context, time *time.Time, noteType *string, note *string) (err error)
	InsertInventoryUpdateForm(ctx context.Context, transactionUUID uuid.UUID, inventoryUpdateForm *InventoryUpdateForm) (err error)
	UpdateHardwareData(ctx context.Context, tagnumber *int64, systemManufacturer *string, systemModel *string) (err error)
	UpdateClientImages(ctx context.Context, transactionUUID uuid.UUID, manifest *ImageManifest) (err error)
	HideClientImageByUUID(ctx context.Context, tagnumber *int64, uuid *string) (err error)
	TogglePinImage(ctx context.Context, tagnumber *int64, uuid *string) (err error)
	SetClientBatteryHealth(ctx context.Context, uuid *string, healthPcnt *int64) (err error)
	SetAllOnlineClientJobs(ctx context.Context, allJobs *AllJobs) (err error)
}

type UpdateRepo struct {
	DB *sql.DB
}

func NewUpdateRepo() (Update, error) {
	db, err := config.GetDatabaseConn()
	if err != nil {
		return nil, fmt.Errorf("error getting database connection in NewUpdateRepo: %w", err)
	}
	return &UpdateRepo{DB: db}, nil
}

var _ Update = (*UpdateRepo)(nil)

func (updateRepo *UpdateRepo) InsertNewNote(ctx context.Context, time *time.Time, noteType *string, note *string) (err error) {
	if time == nil {
		return errors.New("time is required in InsertNewNote")
	}
	if noteType == nil || strings.TrimSpace(*noteType) == "" {
		return errors.New("note type is required in InsertNewNote")
	}

	if ctx.Err() != nil {
		return fmt.Errorf("context error in InsertNewNote: %w", ctx.Err())
	}

	tx, err := updateRepo.DB.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("error beginning transaction in InsertNewNote: %w", err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		} else {
			err = tx.Commit()
		}
	}()

	sqlCode := `INSERT INTO notes (time, note_type, note) VALUES ($1, $2, $3);`
	sqlResult, err := tx.ExecContext(ctx, sqlCode,
		ToNullTime(time),
		ToNullString(noteType),
		ToNullString(note),
	)
	if err != nil {
		return fmt.Errorf("error inserting new note: %w", err)
	}
	if err := VerifyRowsAffected(sqlResult, 1); err != nil {
		return fmt.Errorf("error while checking rows affected when inserting new note: %w", err)
	}
	return err
}

func (updateRepo *UpdateRepo) InsertInventoryUpdateForm(ctx context.Context, transactionUUID uuid.UUID, inventoryUpdateForm *InventoryUpdateForm) (err error) {
	if transactionUUID == uuid.Nil || strings.TrimSpace(transactionUUID.String()) == "" {
		return fmt.Errorf("generated transaction UUID is nil")
	}
	if inventoryUpdateForm == nil || inventoryUpdateForm.Tagnumber == nil {
		return fmt.Errorf("inventoryUpdateForm is invalid")
	}

	if ctx.Err() != nil {
		return fmt.Errorf("context error: %w", ctx.Err())
	}

	tx, err := updateRepo.DB.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("error beginning transaction: %w", err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		} else {
			err = tx.Commit()
		}
	}()

	// Update locations table
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
		return fmt.Errorf("error inserting location data: %w", err)
	}
	if err := VerifyRowsAffected(locationsResult, 1); err != nil {
		return fmt.Errorf("error while checking rows affected on locations table insert: %w", err)
	}

	// Insert/update hardware_data table
	const hardwareDataSql = `INSERT INTO hardware_data
		(time, transaction_uuid, tagnumber, system_manufacturer, system_model, device_type) 
		VALUES (CURRENT_TIMESTAMP, $1, $2, $3, $4, $5)
		ON CONFLICT (tagnumber)
		DO UPDATE SET
			time = CURRENT_TIMESTAMP,
			transaction_uuid = EXCLUDED.transaction_uuid,
			tagnumber = EXCLUDED.tagnumber,
			system_manufacturer = EXCLUDED.system_manufacturer,
			system_model = EXCLUDED.system_model,
			device_type = EXCLUDED.device_type;`

	var hardwareDataResult sql.Result
	hardwareDataResult, err = tx.ExecContext(ctx, hardwareDataSql,
		transactionUUID,
		ToNullInt64(inventoryUpdateForm.Tagnumber),
		ToNullString(inventoryUpdateForm.SystemManufacturer),
		ToNullString(inventoryUpdateForm.SystemModel),
		ToNullString(inventoryUpdateForm.DeviceType),
	)
	if err != nil {
		return fmt.Errorf("error inserting/updating hardware data: %w", err)
	}
	if err := VerifyRowsAffected(hardwareDataResult, 1); err != nil {
		return fmt.Errorf("error while checking rows affected on hardware_data table insert/update: %w", err)
	}

	// Insert/update client_health table
	const clientHealthSql = `INSERT INTO client_health
		(time, tagnumber, last_hardware_check, transaction_uuid) VALUES
		(CURRENT_TIMESTAMP, $1, $2, $3)
		ON CONFLICT (tagnumber)
		DO UPDATE SET
			time = CURRENT_TIMESTAMP,
			tagnumber = EXCLUDED.tagnumber,
			last_hardware_check = EXCLUDED.last_hardware_check,
			transaction_uuid = EXCLUDED.transaction_uuid;`

	var clientHealthResult sql.Result
	clientHealthResult, err = tx.ExecContext(ctx, clientHealthSql,
		ToNullInt64(inventoryUpdateForm.Tagnumber),
		ToNullTime(inventoryUpdateForm.LastHardwareCheck),
		transactionUUID,
	)
	if err != nil {
		return fmt.Errorf("error inserting/updating client health data: %w", err)
	}
	if err := VerifyRowsAffected(clientHealthResult, 1); err != nil {
		return fmt.Errorf("error while checking rows affected on client_health table insert/update: %w", err)
	}

	// Insert into checkout_log table if necessary fields are present
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
			return fmt.Errorf("error inserting into checkout_log: %w", err)
		}
		if err := VerifyRowsAffected(checkoutLogResult, 1); err != nil {
			return fmt.Errorf("error while checking rows affected on checkout_log table insert: %w", err)
		}
	}
	return nil
}

func (updateRepo *UpdateRepo) UpdateHardwareData(ctx context.Context, tagnumber *int64, systemManufacturer *string, systemModel *string) (err error) {
	if tagnumber == nil {
		return fmt.Errorf("tagnumber is nil")
	}
	if systemManufacturer == nil && systemModel == nil {
		return fmt.Errorf("either system manufacturer or system model must be specified")
	}

	if ctx.Err() != nil {
		return fmt.Errorf("context error: %w", ctx.Err())
	}

	tx, err := updateRepo.DB.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("error beginning transaction: %w", err)
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

	var sqlResult sql.Result
	sqlResult, err = tx.ExecContext(ctx, sqlCode,
		ToNullInt64(tagnumber),
		ToNullString(systemManufacturer),
		ToNullString(systemModel),
	)
	if err != nil {
		return fmt.Errorf("error updating hardware data: %w", err)
	}
	if err := VerifyRowsAffected(sqlResult, 1); err != nil {
		return fmt.Errorf("error while checking rows affected on hardware_data table update: %w", err)
	}
	return nil
}

func (updateRepo *UpdateRepo) UpdateClientImages(ctx context.Context, transactionUUID uuid.UUID, manifest *ImageManifest) (err error) {
	if transactionUUID == uuid.Nil || strings.TrimSpace(transactionUUID.String()) == "" {
		return fmt.Errorf("transaction UUID is nil")
	}

	if manifest == nil ||
		manifest.UUID == nil || strings.TrimSpace(*manifest.UUID) == "" ||
		manifest.Tagnumber == nil ||
		manifest.FileName == nil || strings.TrimSpace(*manifest.FileName) == "" ||
		manifest.FilePath == nil || strings.TrimSpace(*manifest.FilePath) == "" {
		return fmt.Errorf("invalid manifest: %v", manifest)
	}

	if ctx.Err() != nil {
		return fmt.Errorf("context error: %w", ctx.Err())
	}

	tx, err := updateRepo.DB.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("error beginning transaction: %w", err)
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
	var sqlResult sql.Result
	sqlResult, err = tx.ExecContext(ctx, sqlCode,
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
		return fmt.Errorf("error inserting client image: %w", err)
	}
	if err := VerifyRowsAffected(sqlResult, 1); err != nil {
		return fmt.Errorf("error while checking rows affected on client_images table insert: %w", err)
	}
	return nil
}

func (updateRepo *UpdateRepo) HideClientImageByUUID(ctx context.Context, tagnumber *int64, uuid *string) (err error) {
	if tagnumber == nil || uuid == nil || strings.TrimSpace(*uuid) == "" {
		return fmt.Errorf("tagnumber and uuid are both required")
	}

	if ctx.Err() != nil {
		return fmt.Errorf("context error: %w", ctx.Err())
	}

	tx, err := updateRepo.DB.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("error beginning transaction: %w", err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		} else {
			err = tx.Commit()
		}
	}()

	const sqlQuery = `UPDATE client_images SET hidden = TRUE WHERE tagnumber = $1 AND uuid = $2;`
	var sqlResult sql.Result
	sqlResult, err = tx.ExecContext(ctx, sqlQuery,
		ToNullInt64(tagnumber),
		ToNullString(uuid),
	)
	if err != nil {
		return fmt.Errorf("error hiding client image: %w", err)
	}
	if err := VerifyRowsAffected(sqlResult, 1); err != nil {
		return fmt.Errorf("error while checking rows affected on client_images table update: %w", err)
	}
	return nil
}

func (updateRepo *UpdateRepo) TogglePinImage(ctx context.Context, tagnumber *int64, uuid *string) (err error) {
	if tagnumber == nil || uuid == nil || strings.TrimSpace(*uuid) == "" {
		return fmt.Errorf("tagnumber and uuid are both required")
	}

	if ctx.Err() != nil {
		return fmt.Errorf("context error: %w", ctx.Err())
	}

	tx, err := updateRepo.DB.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("error beginning transaction: %w", err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		} else {
			err = tx.Commit()
		}
	}()

	const sqlQuery = `UPDATE client_images SET primary_image = NOT COALESCE(primary_image, FALSE) WHERE uuid = $1 AND tagnumber = $2;`
	var sqlResult sql.Result
	sqlResult, err = tx.ExecContext(ctx, sqlQuery,
		ToNullString(uuid),
		ToNullInt64(tagnumber),
	)
	if err != nil {
		return fmt.Errorf("error toggling pin on client image: %w", err)
	}
	if err := VerifyRowsAffected(sqlResult, 1); err != nil {
		return fmt.Errorf("error while checking rows affected on client_images table update: %w", err)
	}
	return nil
}

func (updateRepo *UpdateRepo) SetClientBatteryHealth(ctx context.Context, uuid *string, healthPcnt *int64) (err error) {
	if uuid == nil || strings.TrimSpace(*uuid) == "" {
		return fmt.Errorf("UUID is required")
	}
	if healthPcnt == nil {
		return fmt.Errorf("health percentage is required")
	}

	if ctx.Err() != nil {
		return fmt.Errorf("context error: %w", ctx.Err())
	}

	tx, err := updateRepo.DB.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("error beginning transaction: %w", err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		} else {
			err = tx.Commit()
		}
	}()

	const sqlCode = `UPDATE jobstats SET battery_health = $1 WHERE uuid = $2;`
	var sqlResult sql.Result
	sqlResult, err = tx.ExecContext(ctx, sqlCode,
		ToNullInt64(healthPcnt),
		ToNullString(uuid),
	)
	if err != nil {
		return fmt.Errorf("error updating jobstats battery health: %w", err)
	}
	if err := VerifyRowsAffected(sqlResult, 1); err != nil {
		return fmt.Errorf("error while checking rows affected on jobstats table update: %w", err)
	}
	return nil
}

func (updateRepo *UpdateRepo) SetAllOnlineClientJobs(ctx context.Context, allJobs *AllJobs) (err error) {
	if allJobs == nil {
		return fmt.Errorf("allJobs structure is nil")
	}

	if allJobs.JobName == nil || strings.TrimSpace(*allJobs.JobName) == "" {
		return fmt.Errorf("job name is required")
	}

	if ctx.Err() != nil {
		return fmt.Errorf("context error: %w", ctx.Err())
	}

	tx, err := updateRepo.DB.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("error initializing transaction: %w", err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		} else {
			err = tx.Commit()
		}
	}()

	const sqlCode = `UPDATE job_queue SET job_queued = $1 WHERE NOW() - present < INTERVAL '30 SECONDS';`
	var sqlResult sql.Result
	sqlResult, err = tx.ExecContext(ctx, sqlCode, ptrStringToString(allJobs.JobName))
	if err != nil {
		return fmt.Errorf("error while updating job queue: %w", err)
	}
	if err := VerifyRowsAffected(sqlResult, 1); err != nil {
		return fmt.Errorf("error while checking rows affected on job_queue table update: %w", err)
	}
	return nil
}
