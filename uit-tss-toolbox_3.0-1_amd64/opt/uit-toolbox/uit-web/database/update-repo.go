package database

// All UPDATE/INSERT/DELETE queries should check for:
// 1. Basic input constraints/validation (type conversion should be done prior to)
// 2. Get database connection from app state
// 3. Use transactions instead of ExecContext
// 4. Check rows affected when appropriate (especially for updates/deletes where a specific number of rows should be modified)
// 5. Return errors and cancel transactions (defer rollback) on error

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"
	"uit-toolbox/config"
	"uit-toolbox/types"

	"github.com/google/uuid"
)

type Update interface {
	UpdateClientNetworkUsage(ctx context.Context, networkData *types.NetworkData) (err error)
	UpdateClientAppUptime(ctx context.Context, tag int64, appUptime int64) (err error)
	UpdateClientSystemUptime(ctx context.Context, tag int64, systemUptime int64) (err error)
	UpdateClientHardwareData(ctx context.Context, hardwareData *types.ClientHardwareView) (err error)
	UpdateJobQueuedAt(ctx context.Context, jobQueue *types.JobQueueTableRowView) (err error)
	UpdateClientLastHeard(ctx context.Context, tag *int64, lastHeard *time.Time) (err error)
	UpdateClientBatteryChargePcnt(ctx context.Context, tag *int64, percent *float64) (err error)
	BulkUpdateClientLocation(ctx context.Context, transactionUUID *string, tag *int64, location *string) (err error)
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

func InsertNewNote(ctx context.Context, timestamp *time.Time, noteType *string, noteContent *string) (err error) {
	if timestamp == nil || timestamp.IsZero() {
		return fmt.Errorf("%w: %s", types.MissingFieldError, "time")
	}
	if noteType == nil || strings.TrimSpace(*noteType) == "" {
		return fmt.Errorf("%w: %s", types.MissingFieldError, "note type")
	}

	dbConn, err := config.GetDatabaseConn()
	if err != nil {
		return fmt.Errorf("%w: %w", types.DatabaseConnError, err)
	}

	tx, err := dbConn.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("%w: %w", types.DatabaseTransactionError, err)
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
		ptrToNullTime(timestamp),
		ptrToNullString(noteType),
		ptrToNullString(noteContent),
	)
	if err != nil {
		return fmt.Errorf("%w: %w", types.DatabaseUpdateError, err)
	}
	if err := VerifyRowsAffected(sqlResult, 1); err != nil {
		return fmt.Errorf("%w: %w", types.DatabaseAffectedRowsError, err)
	}
	return err
}

func UpdateClientHealthUpdate(ctx context.Context, transactionUUID uuid.UUID, clientHealthData *types.ClientHealthDTO) (err error) {
	if transactionUUID == uuid.Nil || strings.TrimSpace(transactionUUID.String()) == "" {
		return fmt.Errorf("%w: %s", types.MissingFieldError, "transaction UUID")
	}
	if clientHealthData == nil {
		return fmt.Errorf("%w: %s", types.InvalidFieldError, "ClientHealthDTO")
	}
	if err := types.IsTagnumberInt64Valid(&clientHealthData.Tagnumber); err != nil {
		return fmt.Errorf("%w: %s (%w)", types.InvalidFieldError, "tagnumber", err)
	}

	dbConn, err := config.GetDatabaseConn()
	if err != nil {
		return fmt.Errorf("%w: %w", types.DatabaseConnError, err)
	}

	tx, err := dbConn.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("%w: %w", types.DatabaseTransactionError, err)
	}

	defer func() {
		if err != nil {
			_ = tx.Rollback()
		} else {
			err = tx.Commit()
		}
	}()

	// Insert/update client_health table
	const clientHealthSql = `
		INSERT INTO client_health
			(
				time, 
				client_uuid,
				tagnumber, 
				last_hardware_check, 
				transaction_uuid
			) 
		VALUES (
			CURRENT_TIMESTAMP, 
			(SELECT uuid FROM ids WHERE tagnumber = $1 ORDER BY time DESC LIMIT 1),
			$1, 
			$2, 
			$3
		)
		ON CONFLICT (tagnumber)
			DO UPDATE SET
				time = CURRENT_TIMESTAMP,
				client_uuid = EXCLUDED.client_uuid,
				last_hardware_check = EXCLUDED.last_hardware_check,
				transaction_uuid = EXCLUDED.transaction_uuid
	;`

	clientHealthResult, err := tx.ExecContext(ctx, clientHealthSql,
		clientHealthData.Tagnumber,
		ptrToNullTime(clientHealthData.LastHardwareCheck),
		transactionUUID,
	)
	if err != nil {
		return fmt.Errorf("%w: %w", types.DatabaseUpdateError, err)
	}
	if err := VerifyRowsAffected(clientHealthResult, 1); err != nil {
		return err
	}

	return nil
}

func InsertClientCheckoutsUpdate(ctx context.Context, transactionUUID uuid.UUID, checkoutData *types.InventoryCheckoutWriteModel) (err error) {
	if transactionUUID == uuid.Nil || strings.TrimSpace(transactionUUID.String()) == "" {
		return fmt.Errorf("%w: %s", types.MissingFieldError, "transaction UUID")
	}
	if checkoutData == nil {
		return fmt.Errorf("%w: %s", types.InvalidStructureError, "checkoutData")
	}
	if err := types.IsTagnumberInt64Valid(&checkoutData.Tagnumber); err != nil {
		return fmt.Errorf("%w: %s (%w)", types.InvalidFieldError, "tagnumber", err)
	}
	if checkoutData.CheckoutDate == nil &&
		checkoutData.ReturnDate == nil &&
		(checkoutData.CheckoutBool != nil && !*checkoutData.CheckoutBool) {
		return nil
	}

	dbConn, err := config.GetDatabaseConn()
	if err != nil {
		return fmt.Errorf("%w: %w", types.DatabaseConnError, err)
	}

	tx, err := dbConn.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("%w: %w", types.DatabaseTransactionError, err)
	}

	defer func() {
		if err != nil {
			_ = tx.Rollback()
		} else {
			err = tx.Commit()
		}
	}()

	// Insert into checkout_log table if necessary fields are present
	const checkoutSql = `
		INSERT INTO checkout_log
			(
				log_entry_time, 
				client_uuid,
				transaction_uuid, 
				tagnumber, 
				checkout_date, 
				return_date, 
				checkout_bool
			)
		VALUES 
			(
				CURRENT_TIMESTAMP, 
				(SELECT uuid FROM ids WHERE tagnumber = $2 ORDER BY time DESC LIMIT 1),
				$1, 
				$2, 
				$3, 
				$4, 
				$5
			)
	;`

	checkoutLogResult, err := tx.ExecContext(ctx, checkoutSql,
		transactionUUID,
		checkoutData.Tagnumber,
		ptrToNullTime(checkoutData.CheckoutDate),
		ptrToNullTime(checkoutData.ReturnDate),
		ptrToNullBool(checkoutData.CheckoutBool),
	)
	if err != nil {
		return err
	}
	if err := VerifyRowsAffected(checkoutLogResult, 1); err != nil {
		return err
	}
	return nil
}

func UpdateInventoryHardwareData(ctx context.Context, transactionUUID uuid.UUID, hardwareData *types.InventoryHardwareWriteModel) (err error) {
	if transactionUUID == uuid.Nil || strings.TrimSpace(transactionUUID.String()) == "" {
		return fmt.Errorf("%w: %s", types.MissingFieldError, "transaction UUID")
	}
	if hardwareData == nil {
		return fmt.Errorf("%w: %s (%s)", types.InvalidStructureError, "InventoryHardwareWriteModel", "nil")
	}
	if err := types.IsTagnumberInt64Valid(&hardwareData.Tagnumber); err != nil {
		return fmt.Errorf("%w: %s (%w)", types.InvalidFieldError, "tagnumber", err)
	}

	dbConn, err := config.GetDatabaseConn()
	if err != nil {
		return fmt.Errorf("%w: %w", types.DatabaseConnError, err)
	}

	tx, err := dbConn.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("%w: %w", types.DatabaseTransactionError, err)
	}

	defer func() {
		if err != nil {
			_ = tx.Rollback()
		} else {
			err = tx.Commit()
		}
	}()

	// Insert/update hardware_data table
	const hardwareDataSql = `
		INSERT INTO hardware_data
			(
				time, 
				client_uuid,
				transaction_uuid, 
				tagnumber, 
				system_manufacturer, 
				system_model, 
				device_type
			) 
		VALUES
			(
				CURRENT_TIMESTAMP, 
				(SELECT uuid FROM ids WHERE tagnumber = $2 ORDER BY time DESC LIMIT 1),
				$1, 
				$2, 
				$3, 
				$4, 
				$5
			)
		ON CONFLICT (tagnumber)
		DO UPDATE SET
			time = CURRENT_TIMESTAMP,
			client_uuid = EXCLUDED.client_uuid,
			transaction_uuid = EXCLUDED.transaction_uuid,
			system_manufacturer = EXCLUDED.system_manufacturer,
			system_model = EXCLUDED.system_model,
			device_type = EXCLUDED.device_type
	;`

	var hardwareDataResult sql.Result
	hardwareDataResult, err = tx.ExecContext(ctx, hardwareDataSql,
		transactionUUID,
		hardwareData.Tagnumber,
		ptrToNullString(hardwareData.SystemManufacturer),
		ptrToNullString(hardwareData.SystemModel),
		ptrToNullString(hardwareData.DeviceType),
	)
	if err != nil {
		return fmt.Errorf("db error: %w", err)
	}
	if err := VerifyRowsAffected(hardwareDataResult, 1); err != nil {
		return err
	}

	return nil
}

func InsertInventoryUpdate(ctx context.Context, transactionUUID uuid.UUID, inventoryUpdate *types.InventoryLocationWriteModel) (err error) {
	if transactionUUID == uuid.Nil || strings.TrimSpace(transactionUUID.String()) == "" {
		return fmt.Errorf("%w: %s", types.MissingFieldError, "transaction UUID")
	}
	if inventoryUpdate == nil || inventoryUpdate.Tagnumber == 0 {
		return fmt.Errorf("inventoryUpdate is invalid")
	}

	dbConn, err := config.GetDatabaseConn()
	if err != nil {
		return fmt.Errorf("%w: %w", types.DatabaseConnError, err)
	}

	tx, err := dbConn.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("%w: %w", types.DatabaseTransactionError, err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		} else {
			err = tx.Commit()
		}
	}()

	// Insert/update ids table
	const idsSql = `
		INSERT INTO 
			ids (
				uuid, 
				time, 
				tagnumber, 
				system_serial
			)
		VALUES (
			uuidv7(), 
			CURRENT_TIMESTAMP, 
			$1, 
			$2 
		)
		ON CONFLICT (tagnumber) DO NOTHING
	;`

	_, err = tx.ExecContext(ctx, idsSql,
		inventoryUpdate.Tagnumber,
		inventoryUpdate.SystemSerial,
	)
	if err != nil {
		return fmt.Errorf("%w: %w", types.DatabaseUpdateError, err)
	}

	// Update locations table
	const locationsLogSql = `
	INSERT INTO locations_log (
		time, 
		transaction_uuid,
		client_uuid,
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
		note
	) 
	VALUES (
		CURRENT_TIMESTAMP,
	 	$1, 
		(SELECT uuid FROM ids WHERE tagnumber = $2 ORDER BY time DESC LIMIT 1), 
		$2, 
		$3, 
		$4, 
		$5, 
		$6, 
		$7, 
		$8, 
		$9, 
		$10, 
		$11, 
		$12, 
		$13, 
		$14, 
		$15
	)
	;`

	var locationsLogResult sql.Result
	locationsLogResult, err = tx.ExecContext(ctx, locationsLogSql,
		transactionUUID,
		inventoryUpdate.Tagnumber,
		inventoryUpdate.SystemSerial,
		inventoryUpdate.Location,
		ptrToNullString(inventoryUpdate.Building),
		ptrToNullString(inventoryUpdate.Room),
		inventoryUpdate.Department,
		inventoryUpdate.ADDomain,
		ptrToNullString(inventoryUpdate.PropertyCustodian),
		ptrToNullTime(inventoryUpdate.AcquiredDate),
		ptrToNullTime(inventoryUpdate.RetiredDate),
		ptrToNullBool(inventoryUpdate.IsBroken),
		ptrToNullBool(inventoryUpdate.DiskRemoved),
		inventoryUpdate.ClientStatus,
		ptrToNullString(inventoryUpdate.Note),
	)
	if err != nil {
		return fmt.Errorf("%w: %w", types.DatabaseUpdateError, err)
	}
	if err := VerifyRowsAffected(locationsLogResult, 1); err != nil {
		return err
	}

	const locationsSql = `
	INSERT INTO locations (
		time, 
		transaction_uuid,
		client_uuid,
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
		note
	) 
	VALUES (
		CURRENT_TIMESTAMP,
	 	$1, 
		(SELECT uuid FROM ids WHERE tagnumber = $2 ORDER BY time DESC LIMIT 1), 
		$2, 
		$3, 
		$4, 
		$5, 
		$6, 
		$7, 
		$8, 
		$9, 
		$10, 
		$11, 
		$12, 
		$13, 
		$14, 
		$15
	) ON CONFLICT (client_uuid) DO UPDATE SET
		time = CURRENT_TIMESTAMP,
		transaction_uuid = EXCLUDED.transaction_uuid,
		tagnumber = EXCLUDED.tagnumber,
		system_serial = EXCLUDED.system_serial,
		location = EXCLUDED.location,
		building = EXCLUDED.building,
		room = EXCLUDED.room,
		department_name = EXCLUDED.department_name,
		ad_domain = EXCLUDED.ad_domain,
		property_custodian = EXCLUDED.property_custodian,
		acquired_date = EXCLUDED.acquired_date,
		retired_date = EXCLUDED.retired_date,
		is_broken = EXCLUDED.is_broken,
		disk_removed = EXCLUDED.disk_removed,
		client_status = EXCLUDED.client_status,
		note = EXCLUDED.note
	;`

	var locationsResult sql.Result
	locationsResult, err = tx.ExecContext(ctx, locationsSql,
		transactionUUID,
		inventoryUpdate.Tagnumber,
		inventoryUpdate.SystemSerial,
		inventoryUpdate.Location,
		ptrToNullString(inventoryUpdate.Building),
		ptrToNullString(inventoryUpdate.Room),
		inventoryUpdate.Department,
		inventoryUpdate.ADDomain,
		ptrToNullString(inventoryUpdate.PropertyCustodian),
		ptrToNullTime(inventoryUpdate.AcquiredDate),
		ptrToNullTime(inventoryUpdate.RetiredDate),
		ptrToNullBool(inventoryUpdate.IsBroken),
		ptrToNullBool(inventoryUpdate.DiskRemoved),
		inventoryUpdate.ClientStatus,
		ptrToNullString(inventoryUpdate.Note),
	)
	if err != nil {
		return fmt.Errorf("%w: %w", types.DatabaseUpdateError, err)
	}
	if err := VerifyRowsAffected(locationsResult, 1); err != nil {
		return err
	}

	return nil
}

func UpdateClientImages(ctx context.Context, transactionUUID uuid.UUID, manifest *types.ImageManifestDTO) (err error) {
	if transactionUUID == uuid.Nil || strings.TrimSpace(transactionUUID.String()) == "" {
		return fmt.Errorf("%w: %s", types.MissingFieldError, "transaction UUID")
	}

	if err := types.IsTagnumberInt64Valid(&manifest.Tagnumber); err != nil {
		return fmt.Errorf("%w: %s (%w)", types.InvalidFieldError, "tagnumber", err)
	}

	if manifest == nil ||
		strings.TrimSpace(manifest.UUID) == "" ||
		manifest.Time.IsZero() ||
		strings.TrimSpace(manifest.FileName) == "" ||
		strings.TrimSpace(manifest.FilePath) == "" ||
		manifest.FileSize <= 0 ||
		len(manifest.SHA256Hash) == 0 ||
		strings.TrimSpace(manifest.MimeType) == "" {
		return fmt.Errorf("%w: invalid manifest: %s", types.InvalidStructureError, "ImageManifestDTO")
	}

	dbConn, err := config.GetDatabaseConn()
	if err != nil {
		return fmt.Errorf("%w: %w", types.DatabaseConnError, err)
	}

	tx, err := dbConn.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("%w: %w", types.DatabaseTransactionError, err)
	}

	defer func() {
		if err != nil {
			_ = tx.Rollback()
		} else {
			err = tx.Commit()
		}
	}()

	const sqlCode = `
		INSERT INTO client_images 
			(
				uuid, 
				time, 
				client_uuid,
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
				pinned
			)
		VALUES 
			(
				$1, 
				$2, 
				(SELECT uuid FROM ids WHERE tagnumber = $3 ORDER BY time DESC LIMIT 1),
				$3, 
				$4, 
				$5, 
				$6, 
				$7, 
				$8, 
				$9, 
				$10, 
				$11, 
				$12, 
				$13, 
				$14, 
				$15
			)
	;`

	sqlResult, err := tx.ExecContext(ctx, sqlCode,
		manifest.UUID,
		manifest.Time,
		manifest.Tagnumber,
		manifest.FileName,
		manifest.FilePath,
		ptrToNullString(manifest.ThumbnailFilePath),
		manifest.FileSize,
		manifest.SHA256Hash,
		manifest.MimeType,
		ptrToNullTime(manifest.ExifTimestamp),
		ptrToNullInt64(manifest.ResolutionX),
		ptrToNullInt64(manifest.ResolutionY),
		ptrToNullString(manifest.Caption),
		manifest.Hidden,
		manifest.Pinned,
	)
	if err != nil {
		return err
	}
	if err := VerifyRowsAffected(sqlResult, 1); err != nil {
		return err
	}
	return nil
}

func HideClientImageByUUID(ctx context.Context, tagnumber *int64, uuid *string) (err error) {
	if err := types.IsTagnumberInt64Valid(tagnumber); err != nil {
		return fmt.Errorf("%w: %s (%w)", types.InvalidFieldError, "tagnumber", err)
	}
	if uuid == nil || strings.TrimSpace(*uuid) == "" {
		return fmt.Errorf("%w: %s", types.MissingFieldError, "image UUID")
	}

	dbConn, err := config.GetDatabaseConn()
	if err != nil {
		return fmt.Errorf("%w: %w", types.DatabaseConnError, err)
	}

	tx, err := dbConn.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("%w: %w", types.DatabaseTransactionError, err)
	}

	defer func() {
		if err != nil {
			_ = tx.Rollback()
		} else {
			err = tx.Commit()
		}
	}()

	const sqlQuery = `
		UPDATE 
			client_images 
		SET 
			hidden = TRUE 
		WHERE 
			client_uuid = (SELECT uuid FROM ids WHERE tagnumber = $1 ORDER BY time DESC LIMIT 1)
			AND uuid = $2
	;`

	sqlResult, err := tx.ExecContext(ctx, sqlQuery,
		ptrToNullInt64(tagnumber),
		ptrToNullString(uuid),
	)
	if err != nil {
		return err
	}
	if err := VerifyRowsAffected(sqlResult, 1); err != nil {
		return err
	}
	return nil
}

func DeleteClientImageByUUID(ctx context.Context, tag *int64, imageUUID *string) (err error) {
	if err := types.IsTagnumberInt64Valid(tag); err != nil {
		return fmt.Errorf("%w: %s (%w)", types.InvalidFieldError, "tagnumber", err)
	}
	if imageUUID == nil || strings.TrimSpace(*imageUUID) == "" {
		return fmt.Errorf("%w: %s", types.MissingFieldError, "image UUID")
	}

	dbConn, err := config.GetDatabaseConn()
	if err != nil {
		return fmt.Errorf("%w: %w", types.DatabaseConnError, err)
	}

	tx, err := dbConn.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("%w: %w", types.DatabaseTransactionError, err)
	}

	defer func() {
		if err != nil {
			_ = tx.Rollback()
		} else {
			err = tx.Commit()
		}
	}()

	const sqlQuery = `
		DELETE FROM 
			client_images 
		WHERE 
			client_uuid = (SELECT uuid FROM ids WHERE tagnumber = $1 ORDER BY time DESC LIMIT 1)
			AND uuid = $2
	;`
	sqlResult, err := tx.ExecContext(ctx, sqlQuery,
		ptrToNullInt64(tag),
		ptrToNullString(imageUUID),
	)
	if err != nil {
		return err
	}
	if err := VerifyRowsAffected(sqlResult, 1); err != nil {
		return err
	}
	return nil
}

func TogglePinImage(ctx context.Context, tagnumber *int64, uuid *string) (err error) {
	if tagnumber == nil {
		return fmt.Errorf("%w: %s", types.MissingFieldError, "tagnumber")
	}
	if err := types.IsTagnumberInt64Valid(tagnumber); err != nil {
		return fmt.Errorf("%w: %s (%w)", types.InvalidFieldError, "tagnumber", err)
	}
	if uuid == nil || strings.TrimSpace(*uuid) == "" {
		return fmt.Errorf("%w: %s", types.MissingFieldError, "image UUID")
	}

	dbConn, err := config.GetDatabaseConn()
	if err != nil {
		return fmt.Errorf("%w: %w", types.DatabaseConnError, err)
	}

	tx, err := dbConn.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("%w: %w", types.DatabaseTransactionError, err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		} else {
			err = tx.Commit()
		}
	}()

	const sqlQuery = `UPDATE client_images SET pinned = NOT COALESCE(pinned, FALSE) WHERE uuid = $1 AND client_uuid = (SELECT uuid FROM ids WHERE tagnumber = $2 ORDER BY time DESC LIMIT 1);`
	sqlResult, err := tx.ExecContext(ctx, sqlQuery,
		ptrToNullString(uuid),
		ptrToNullInt64(tagnumber),
	)
	if err != nil {
		return fmt.Errorf("%w: %w", types.DatabaseUpdateError, err)
	}
	if err := VerifyRowsAffected(sqlResult, 1); err != nil {
		return err
	}
	return nil
}

func SetAllOnlineClientJobs(ctx context.Context, clientJob string) (err error) {
	if strings.TrimSpace(clientJob) == "" {
		return fmt.Errorf("%w: %s", types.MissingFieldError, "job name")
	}

	dbConn, err := config.GetDatabaseConn()
	if err != nil {
		return fmt.Errorf("%w: %w", types.DatabaseConnError, err)
	}

	tx, err := dbConn.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("%w: %w", types.DatabaseTransactionError, err)
	}

	defer func() {
		if err != nil {
			_ = tx.Rollback()
		} else {
			err = tx.Commit()
		}
	}()

	const sqlCode = `
		UPDATE 
			job_queue 
		SET 
			job_name = $1 
		WHERE 
			CURRENT_TIMESTAMP - last_heard < INTERVAL '30 SECONDS' 
			AND job_active = FALSE 
			AND job_queued = FALSE
	;`

	_, err = tx.ExecContext(ctx, sqlCode,
		clientJob,
	)
	if err != nil {
		return fmt.Errorf("%w: %w", types.DatabaseUpdateError, err)
	}
	return nil
}

func SetClientJob(ctx context.Context, tag int64, clientJob string) (err error) {
	if err := types.IsTagnumberInt64Valid(&tag); err != nil {
		return fmt.Errorf("%w: %s (%w)", types.InvalidFieldError, "tagnumber", err)
	}

	if strings.TrimSpace(clientJob) == "" {
		return fmt.Errorf("%w: %s", types.MissingFieldError, "client job name")
	}

	dbConn, err := config.GetDatabaseConn()
	if err != nil {
		return fmt.Errorf("%w: %w", types.DatabaseConnError, err)
	}

	tx, err := dbConn.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("%w: %w", types.DatabaseTransactionError, err)
	}

	defer func() {
		if err != nil {
			_ = tx.Rollback()
		} else {
			err = tx.Commit()
		}
	}()

	const sqlCode = `
		UPDATE 
			job_queue 
		SET 
			job_queued = TRUE, 
			job_name = $1, 
			job_active = FALSE 
		WHERE 
			client_uuid = (SELECT uuid FROM ids WHERE tagnumber = $2 ORDER BY time DESC LIMIT 1)
	;`
	sqlResult, err := tx.ExecContext(ctx, sqlCode,
		clientJob,
		tag,
	)
	if err != nil {
		return fmt.Errorf("%w: %w", types.DatabaseUpdateError, err)
	}
	if err := VerifyRowsAffected(sqlResult, 1); err != nil {
		return err
	}
	return nil
}

func UpsertClientMemoryUsageKB(ctx context.Context, memInfo types.MemoryDataDTO) (err error) {
	if memInfo.Tagnumber == 0 {
		return fmt.Errorf("%w: %w", types.InvalidFieldError, fmt.Errorf("tagnumber is required in memory data"))
	}
	if memInfo.TotalUsageKB <= 0 {
		return fmt.Errorf("%w: %w", types.InvalidFieldError, fmt.Errorf("total memory usage must be greater than 0"))
	}

	if ctx.Err() != nil {
		return ctx.Err()
	}

	dbConn, err := config.GetDatabaseConn()
	if err != nil {
		return fmt.Errorf("%w: %w", types.DatabaseConnError, err)
	}

	tx, err := dbConn.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("%w: %w", types.DatabaseTransactionError, err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		} else {
			err = tx.Commit()
		}
	}()

	const sqlCode = `
		INSERT INTO 
			job_queue (
				client_uuid, 
				tagnumber,
				memory_usage_kb
			) 
		VALUES 
			(
				(SELECT uuid FROM ids WHERE tagnumber = $1 ORDER BY time DESC LIMIT 1),
				$1, 
				$2
			)
		ON CONFLICT (tagnumber) DO UPDATE SET 
			client_uuid = EXCLUDED.client_uuid,
			memory_usage_kb = EXCLUDED.memory_usage_kb
	;`
	var sqlResult sql.Result
	sqlResult, err = tx.ExecContext(ctx, sqlCode,
		toNullInt64(memInfo.Tagnumber),
		toNullInt64(memInfo.TotalUsageKB),
	)
	if err != nil {
		return fmt.Errorf("%w: %w", types.DatabaseUpdateError, err)
	}
	if err := VerifyRowsAffected(sqlResult, 1); err != nil {
		return err
	}
	return nil
}

func UpsertClientMemoryCapacityKB(ctx context.Context, memInfo types.MemoryDataDTO) (err error) {
	if memInfo.Tagnumber == 0 {
		return fmt.Errorf("%w: %w", types.InvalidFieldError, fmt.Errorf("tagnumber is required in memory data"))
	}
	if memInfo.TotalCapacityKB <= 0 {
		return fmt.Errorf("%w: %w", types.InvalidFieldError, fmt.Errorf("memory capacity must be greater than 0"))
	}

	if ctx.Err() != nil {
		return ctx.Err()
	}

	dbConn, err := config.GetDatabaseConn()
	if err != nil {
		return fmt.Errorf("%w: %w", types.DatabaseConnError, err)
	}

	tx, err := dbConn.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("%w: %w", types.DatabaseTransactionError, err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		} else {
			err = tx.Commit()
		}
	}()

	const sqlCode = `
		INSERT INTO 
			job_queue (
				client_uuid, 
				tagnumber,
				memory_capacity_kb
			) 
		VALUES 
			(
				(SELECT uuid FROM ids WHERE tagnumber = $1 ORDER BY time DESC LIMIT 1),
				$1, 
				$2
			)
		ON CONFLICT (tagnumber) DO UPDATE SET 
			client_uuid = EXCLUDED.client_uuid,
			memory_capacity_kb = EXCLUDED.memory_capacity_kb
	;`
	var sqlResult sql.Result
	sqlResult, err = tx.ExecContext(ctx, sqlCode,
		toNullInt64(memInfo.Tagnumber),
		toNullInt64(memInfo.TotalCapacityKB),
	)
	if err != nil {
		return fmt.Errorf("%w: %w", types.DatabaseUpdateError, err)
	}
	if err := VerifyRowsAffected(sqlResult, 1); err != nil {
		return err
	}
	return nil
}

func UpsertClientCPUUsage(ctx context.Context, cpuData *types.CPUDataDTO) (err error) {
	if cpuData == nil {
		return fmt.Errorf("CPU data is required")
	}

	if cpuData.Tagnumber == 0 {
		return fmt.Errorf("%w: %s", types.InvalidFieldError, "tagnumber is missing")
	}

	if cpuData.UsagePercent < 0 || cpuData.UsagePercent > 110 {
		return fmt.Errorf("%w: %s must be between 0 and 100", types.InvalidFieldError, "CPU usage percent")
	}

	if ctx.Err() != nil {
		return fmt.Errorf("context error: %w", ctx.Err())
	}

	dbConn, err := config.GetDatabaseConn()
	if err != nil {
		return fmt.Errorf("%w: %w", types.DatabaseConnError, err)
	}

	tx, err := dbConn.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("%w: %w", types.DatabaseTransactionError, err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		} else {
			err = tx.Commit()
		}
	}()

	const sqlCode = `
		INSERT INTO 
			job_queue (
				client_uuid,
				tagnumber, 
				cpu_usage
			) 
		VALUES (
			(SELECT uuid FROM ids WHERE tagnumber = $1 ORDER BY time DESC LIMIT 1),
			$1, 
			$2
		)
		ON CONFLICT (tagnumber) DO UPDATE SET 
			client_uuid = EXCLUDED.client_uuid,
			cpu_usage = EXCLUDED.cpu_usage
	;`
	var sqlResult sql.Result
	sqlResult, err = tx.ExecContext(ctx, sqlCode,
		toNullInt64(cpuData.Tagnumber),
		cpuData.UsagePercent,
	)
	if err != nil {
		return fmt.Errorf("%w: %w", types.DatabaseUpdateError, err)
	}
	if err := VerifyRowsAffected(sqlResult, 1); err != nil {
		return err
	}
	return nil
}

func UpsertClientCPUMHz(ctx context.Context, cpuData *types.CPUDataDTO) (err error) {
	if cpuData == nil {
		return fmt.Errorf("CPU data is required")
	}
	if cpuData.Tagnumber == 0 {
		return fmt.Errorf("%w: %s", types.InvalidFieldError, "tagnumber is missing")
	}
	if cpuData.MHz <= 0 {
		return fmt.Errorf("%w: %s must be greater than 0", types.InvalidFieldError, "CPU MHz")
	}

	if ctx.Err() != nil {
		return ctx.Err()
	}

	dbConn, err := config.GetDatabaseConn()
	if err != nil {
		return fmt.Errorf("%w: %w", types.DatabaseConnError, err)
	}

	tx, err := dbConn.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("%w: %w", types.DatabaseTransactionError, err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		} else {
			err = tx.Commit()
		}
	}()

	const sqlCode = `
		INSERT INTO 
			job_queue (
				client_uuid, 
				tagnumber, 
				cpu_mhz
			) 
		VALUES (
			(SELECT uuid FROM ids WHERE tagnumber = $1 ORDER BY time DESC LIMIT 1),
			$1, 
			$2
		)
		ON CONFLICT (tagnumber) DO UPDATE SET 
			client_uuid = EXCLUDED.client_uuid,
			cpu_mhz = EXCLUDED.cpu_mhz
	;`
	var sqlResult sql.Result
	sqlResult, err = tx.ExecContext(ctx, sqlCode,
		toNullInt64(cpuData.Tagnumber),
		toNullFloat64(cpuData.MHz),
	)
	if err != nil {
		return fmt.Errorf("%w: %w", types.DatabaseUpdateError, err)
	}
	if err := VerifyRowsAffected(sqlResult, 1); err != nil {
		return err
	}
	return nil
}

func (updateRepo *UpdateRepo) UpdateClientNetworkUsage(ctx context.Context, networkData *types.NetworkData) (err error) {
	if networkData == nil {
		return fmt.Errorf("network data is required")
	}
	if networkData.Tagnumber == 0 || networkData.NetworkUsage == nil || networkData.LinkSpeed == nil {
		return fmt.Errorf("tagnumber, network usage, and link speed are required")
	}
	if ctx.Err() != nil {
		return fmt.Errorf("context error: %w", ctx.Err())
	}

	tx, err := updateRepo.DB.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("error beginning DB transaction: %w", err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		} else {
			err = tx.Commit()
		}
	}()
	const sqlCode = `INSERT INTO job_queue (client_uuid, tagnumber, network_usage, link_speed) VALUES (
			(SELECT uuid FROM ids WHERE tagnumber = $1 ORDER BY time DESC LIMIT 1),
			$1, 
			$2,
			$3
		)
		ON CONFLICT (tagnumber) DO UPDATE SET 
			client_uuid = EXCLUDED.client_uuid,
			network_usage = EXCLUDED.network_usage,
			link_speed = EXCLUDED.link_speed;`
	var sqlResult sql.Result
	sqlResult, err = tx.ExecContext(ctx, sqlCode,
		toNullInt64(networkData.Tagnumber),
		ptrToNullInt64(networkData.NetworkUsage),
		ptrToNullInt64(networkData.LinkSpeed),
	)
	if err != nil {
		return fmt.Errorf("%w: %w", types.DatabaseUpdateError, err)
	}
	if err := VerifyRowsAffected(sqlResult, 1); err != nil {
		return err
	}
	return nil
}

func UpsertClientCPUTemperature(ctx context.Context, cpuTempData *types.CPUDataDTO) (err error) {
	if cpuTempData == nil {
		return fmt.Errorf("CPU data is required")
	}
	if cpuTempData.Tagnumber == 0 {
		return fmt.Errorf("both tagnumber and temperature are required")
	}
	if cpuTempData.MillidegreesC <= 0 {
		return fmt.Errorf("%w: %s must be greater than 0", types.InvalidFieldError, "CPU temperature")
	}

	if ctx.Err() != nil {
		return ctx.Err()
	}

	dbConn, err := config.GetDatabaseConn()
	if err != nil {
		return fmt.Errorf("%w: %w", types.DatabaseConnError, err)
	}

	tx, err := dbConn.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("%w: %w", types.DatabaseTransactionError, err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		} else {
			err = tx.Commit()
		}
	}()

	degreesC := float64(cpuTempData.MillidegreesC / 1000)

	const sqlCode = `INSERT INTO job_queue (client_uuid, tagnumber, cpu_temp) VALUES (
			(SELECT uuid FROM ids WHERE tagnumber = $1 ORDER BY time DESC LIMIT 1),
			$1, 
			$2
		)
		ON CONFLICT (tagnumber) DO UPDATE SET 
			client_uuid = EXCLUDED.client_uuid,
			cpu_temp = EXCLUDED.cpu_temp;`
	var sqlResult sql.Result
	sqlResult, err = tx.ExecContext(ctx, sqlCode,
		toNullInt64(cpuTempData.Tagnumber),
		ptrToNullFloat64(&degreesC),
	)
	if err != nil {
		return fmt.Errorf("%w: %w", types.DatabaseUpdateError, err)
	}
	if err := VerifyRowsAffected(sqlResult, 1); err != nil {
		return err
	}
	return nil
}

func (updateRepo *UpdateRepo) UpdateClientSystemUptime(ctx context.Context, tag int64, systemUptime int64) (err error) {
	if tag == 0 {
		return fmt.Errorf("tagnumber is required")
	}
	if systemUptime <= 0 {
		return fmt.Errorf("system uptime cannot be negative or zero")
	}
	if ctx.Err() != nil {
		return fmt.Errorf("context error: %w", ctx.Err())
	}

	tx, err := updateRepo.DB.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("error beginning DB transaction: %w", err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		} else {
			err = tx.Commit()
		}
	}()

	const sqlCode = `
		INSERT INTO 
			job_queue (
				client_uuid,
				tagnumber, 
				system_uptime
			) VALUES (
				(SELECT uuid FROM ids WHERE tagnumber = $1 ORDER BY time DESC LIMIT 1),
				$1, 
			 	$2
			)
		ON CONFLICT (tagnumber) DO UPDATE SET 
		client_uuid = EXCLUDED.client_uuid,
		system_uptime = COALESCE(EXCLUDED.system_uptime, job_queue.system_uptime)
	;`
	var sqlResult sql.Result
	sqlResult, err = tx.ExecContext(ctx, sqlCode,
		toNullInt64(tag),
		toNullInt64(systemUptime),
	)
	if err != nil {
		return fmt.Errorf("%w: %w", types.DatabaseUpdateError, err)
	}
	if err := VerifyRowsAffected(sqlResult, 1); err != nil {
		return err
	}
	return nil
}

func (updateRepo *UpdateRepo) UpdateClientAppUptime(ctx context.Context, tag int64, appUptime int64) (err error) {
	if tag == 0 {
		return fmt.Errorf("tagnumber is required")
	}
	if appUptime <= 0 {
		return fmt.Errorf("app uptime cannot be negative or zero")
	}
	if ctx.Err() != nil {
		return fmt.Errorf("context error: %w", ctx.Err())
	}

	tx, err := updateRepo.DB.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("error beginning DB transaction: %w", err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		} else {
			err = tx.Commit()
		}
	}()

	const sqlCode = `
		INSERT INTO 
			job_queue (
				client_uuid,
				tagnumber, 
				client_app_uptime
			) VALUES (
				(SELECT uuid FROM ids WHERE tagnumber = $1 ORDER BY time DESC LIMIT 1),
				$1, 
			 	$2
			)
		ON CONFLICT (tagnumber) DO UPDATE SET 
		client_uuid = EXCLUDED.client_uuid,
		client_app_uptime = COALESCE(EXCLUDED.client_app_uptime, job_queue.client_app_uptime)
	;`
	var sqlResult sql.Result
	sqlResult, err = tx.ExecContext(ctx, sqlCode,
		toNullInt64(tag),
		toNullInt64(appUptime),
	)
	if err != nil {
		return fmt.Errorf("%w: %w", types.DatabaseUpdateError, err)
	}
	if err := VerifyRowsAffected(sqlResult, 1); err != nil {
		return err
	}
	return nil
}

func UpsertClientHealthCheck(ctx context.Context, healthCheck *types.ClientHealthCheck) (err error) {
	if healthCheck == nil {
		return fmt.Errorf("healthCheck data is required")
	}
	if healthCheck.Tagnumber == 0 {
		return fmt.Errorf("tagnumber is required in healthCheck data")
	}
	if healthCheck.TransactionUUID == "" {
		return fmt.Errorf("transaction UUID is required in healthCheck data")
	}
	if ctx.Err() != nil {
		return fmt.Errorf("context error: %w", ctx.Err())
	}

	dbConn, err := config.GetDatabaseConn()
	if err != nil {
		return fmt.Errorf("%w: %w", types.DatabaseConnError, err)
	}

	tx, err := dbConn.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("error beginning DB transaction: %w", err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		} else {
			err = tx.Commit()
		}
	}()
	const clientHealthCheckSQL = `
		INSERT INTO 
			client_health (
				transaction_uuid,
				client_uuid,
				tagnumber, 
				system_serial,
				tpm_version,
				last_hardware_check
			) 
		VALUES (
			$1,
			(SELECT uuid FROM ids WHERE tagnumber = $2 ORDER BY time DESC LIMIT 1),
			$2,
			$3,
			$4,
			$5
		)
		ON CONFLICT (tagnumber) DO UPDATE SET 
			transaction_uuid = EXCLUDED.transaction_uuid,
			client_uuid = COALESCE(EXCLUDED.client_uuid, client_health.client_uuid), 
			system_serial = COALESCE(EXCLUDED.system_serial, client_health.system_serial),
			tpm_version = COALESCE(EXCLUDED.tpm_version, client_health.tpm_version),
			last_hardware_check = EXCLUDED.last_hardware_check
		;`
	var sqlResult sql.Result
	sqlResult, err = tx.ExecContext(ctx, clientHealthCheckSQL,
		ptrToNullString(&healthCheck.TransactionUUID),
		ptrToNullInt64(&healthCheck.Tagnumber),
		ptrToNullString(healthCheck.SystemSerial),
		ptrToNullString(healthCheck.TPMVersion),
		ptrToNullTime(healthCheck.LastHardwareCheck),
	)
	if err != nil {
		return fmt.Errorf("%w: %w", types.DatabaseUpdateError, err)
	}
	if err := VerifyRowsAffected(sqlResult, 1); err != nil {
		return err
	}

	const clientHealthCheckHistorySQL = `
		INSERT INTO 
			historical_hardware_data (
				transaction_uuid,
				time, 
				client_uuid, 
				tagnumber, 
				bios_version
			) 
		VALUES (
			$1,
			CURRENT_TIMESTAMP,
			(SELECT uuid FROM ids WHERE tagnumber = $2 ORDER BY time DESC LIMIT 1),
			$2,
			$3
		) ON CONFLICT (transaction_uuid) DO UPDATE SET
		 	time = CURRENT_TIMESTAMP,
			client_uuid = COALESCE(EXCLUDED.client_uuid, historical_hardware_data.client_uuid), 
			tagnumber = EXCLUDED.tagnumber, 
			bios_version = COALESCE(EXCLUDED.bios_version, historical_hardware_data.bios_version)
	;`

	sqlResult, err = tx.ExecContext(ctx, clientHealthCheckHistorySQL,
		ptrToNullString(&healthCheck.TransactionUUID),
		ptrToNullInt64(&healthCheck.Tagnumber),
		ptrToNullString(healthCheck.BIOSVersion),
	)
	if err != nil {
		return fmt.Errorf("%w: %w", types.DatabaseUpdateError, err)
	}
	if err := VerifyRowsAffected(sqlResult, 1); err != nil {
		return err
	}

	return nil
}

func (updateRepo *UpdateRepo) UpdateClientHardwareData(ctx context.Context, hardwareData *types.ClientHardwareView) (err error) {
	if hardwareData == nil || hardwareData.Tagnumber == nil || strings.TrimSpace(hardwareData.TransactionUUID) == "" {
		return fmt.Errorf("hardwareData is invalid")
	}
	if ctx.Err() != nil {
		return fmt.Errorf("context error: %w", ctx.Err())
	}
	tx, err := updateRepo.DB.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("error beginning DB transaction: %w", err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		} else {
			err = tx.Commit()
		}
	}()

	const hardwareDataTable = `INSERT INTO hardware_data
		(
			transaction_uuid,
			time,
			client_uuid,
			tagnumber,
			system_serial,
			system_uuid,
			system_manufacturer,
			system_model,
			system_sku,
			device_type,
			chassis_type,
			motherboard_serial,
			motherboard_manufacturer,
			cpu_manufacturer,
			cpu_model,
			cpu_max_speed_mhz,
			cpu_core_count,
			cpu_thread_count,
			ethernet_mac,
			wifi_mac
		) VALUES (
			$1,
			CURRENT_TIMESTAMP,
			(SELECT uuid FROM ids WHERE tagnumber = $2 ORDER BY time DESC LIMIT 1),
			$2,
			$3,
			$4,
			$5,
			$6,
			$7,
			$8,
			$9,
			$10,
			$11,
			$12,
			$13,
			$14,
			$15,
			$16,
			$17,
			$18
		) ON CONFLICT (tagnumber)
		 DO UPDATE SET
		 	transaction_uuid = COALESCE(EXCLUDED.transaction_uuid, hardware_data.transaction_uuid),
			time = CURRENT_TIMESTAMP,
			client_uuid = COALESCE(EXCLUDED.client_uuid, hardware_data.client_uuid),
			system_serial = COALESCE(EXCLUDED.system_serial, hardware_data.system_serial),
			system_uuid = COALESCE(EXCLUDED.system_uuid, hardware_data.system_uuid),
			system_manufacturer = COALESCE(EXCLUDED.system_manufacturer, hardware_data.system_manufacturer),
			system_model = COALESCE(EXCLUDED.system_model, hardware_data.system_model),
			system_sku = COALESCE(EXCLUDED.system_sku, hardware_data.system_sku),
			device_type = COALESCE(EXCLUDED.device_type, hardware_data.device_type),
			chassis_type = COALESCE(EXCLUDED.chassis_type, hardware_data.chassis_type),
			motherboard_serial = COALESCE(EXCLUDED.motherboard_serial, hardware_data.motherboard_serial),
			motherboard_manufacturer = COALESCE(EXCLUDED.motherboard_manufacturer, hardware_data.motherboard_manufacturer),
			cpu_manufacturer = COALESCE(EXCLUDED.cpu_manufacturer, hardware_data.cpu_manufacturer),
			cpu_model = COALESCE(EXCLUDED.cpu_model, hardware_data.cpu_model),
			cpu_max_speed_mhz = COALESCE(EXCLUDED.cpu_max_speed_mhz, hardware_data.cpu_max_speed_mhz),
			cpu_core_count = COALESCE(EXCLUDED.cpu_core_count, hardware_data.cpu_core_count),
			cpu_thread_count = COALESCE(EXCLUDED.cpu_thread_count, hardware_data.cpu_thread_count),
			ethernet_mac = COALESCE(EXCLUDED.ethernet_mac, hardware_data.ethernet_mac),
			wifi_mac = COALESCE(EXCLUDED.wifi_mac, hardware_data.wifi_mac)
			;
	`
	var hardwareResult sql.Result
	hardwareResult, err = tx.ExecContext(ctx, hardwareDataTable,
		ptrToNullString(&hardwareData.TransactionUUID),
		ptrToNullInt64(hardwareData.Tagnumber),
		ptrToNullString(hardwareData.SystemSerial),
		ptrToNullString(hardwareData.SystemUUID),
		ptrToNullString(hardwareData.SystemManufacturer),
		ptrToNullString(hardwareData.SystemModel),
		ptrToNullString(hardwareData.SystemSKU),
		ptrToNullString(hardwareData.DeviceType),
		ptrToNullString(hardwareData.ChassisType),
		ptrToNullString(hardwareData.MotherboardSerial),
		ptrToNullString(hardwareData.MotherboardManufacturer),
		ptrToNullString(hardwareData.CPUManufacturer),
		ptrToNullString(hardwareData.CPUModel),
		ptrToNullInt64(hardwareData.CPUMaxSpeedMhz),
		ptrToNullInt64(hardwareData.CPUCoreCount),
		ptrToNullInt64(hardwareData.CPUThreadCount),
		ptrToNullString(hardwareData.EthernetMAC),
		ptrToNullString(hardwareData.WiFiMAC),
	)
	if err != nil {
		return err
	}
	if err := VerifyRowsAffected(hardwareResult, 1); err != nil {
		return err
	}

	const historicalHardwareDataTable = `INSERT INTO historical_hardware_data 
		(
			transaction_uuid,
			time,
			client_uuid,
			tagnumber, 
			system_serial, 
			ethernet_mac, 
			wifi_mac, 
			disk_model,
			disk_type,
			disk_size_kb,
			disk_serial,
			disk_writes_kb,
			disk_reads_kb,
			disk_power_on_hours,
			disk_errors,
			disk_power_cycles,
			disk_firmware,
			battery_model,
			battery_serial,
			battery_charge_cycles,
			battery_current_max_capacity,
			battery_design_capacity,
			battery_manufacturer,
			battery_manufacture_date,
			bios_version,
			bios_release_date,
			bios_firmware,
			memory_serial,
			memory_capacity_kb,
			memory_speed_mhz
		) VALUES (
			$1,
			CURRENT_TIMESTAMP,
			(SELECT uuid FROM ids WHERE tagnumber = $2 ORDER BY time DESC LIMIT 1),
			$2,
			$3,
			$4,
			$5,
			$6,
			$7,
			$8,
			$9,
			$10,
			$11,
			$12,
			$13,
			$14,
			$15,
			$16,
			$17,
			$18,
			$19,
			$20,
			$21,
			$22,
			$23,
			$24,
			$25,
			$26,
			$27,
			$28
		) ON CONFLICT (transaction_uuid) 
		DO UPDATE SET
			time = CURRENT_TIMESTAMP,
			client_uuid = COALESCE(EXCLUDED.client_uuid, historical_hardware_data.client_uuid),
			tagnumber =  COALESCE(EXCLUDED.tagnumber, historical_hardware_data.tagnumber),
			system_serial = COALESCE(EXCLUDED.system_serial, historical_hardware_data.system_serial),
			ethernet_mac = COALESCE(EXCLUDED.ethernet_mac, historical_hardware_data.ethernet_mac),
			wifi_mac =  COALESCE(EXCLUDED.wifi_mac, historical_hardware_data.wifi_mac),
			disk_model = COALESCE(EXCLUDED.disk_model, historical_hardware_data.disk_model),
			disk_type = COALESCE(EXCLUDED.disk_type, historical_hardware_data.disk_type),
			disk_size_kb = COALESCE(EXCLUDED.disk_size_kb, historical_hardware_data.disk_size_kb),
			disk_serial = COALESCE(EXCLUDED.disk_serial, historical_hardware_data.disk_serial),
			disk_writes_kb = COALESCE(EXCLUDED.disk_writes_kb, historical_hardware_data.disk_writes_kb),
			disk_reads_kb = COALESCE(EXCLUDED.disk_reads_kb, historical_hardware_data.disk_reads_kb),
			disk_power_on_hours = COALESCE(EXCLUDED.disk_power_on_hours, historical_hardware_data.disk_power_on_hours),
			disk_errors = COALESCE(EXCLUDED.disk_errors, historical_hardware_data.disk_errors),
			disk_power_cycles = COALESCE(EXCLUDED.disk_power_cycles, historical_hardware_data.disk_power_cycles),
			disk_firmware = COALESCE(EXCLUDED.disk_firmware, historical_hardware_data.disk_firmware),
			battery_model = COALESCE(EXCLUDED.battery_model, historical_hardware_data.battery_model),
			battery_serial = COALESCE(EXCLUDED.battery_serial, historical_hardware_data.battery_serial),
			battery_charge_cycles = COALESCE(EXCLUDED.battery_charge_cycles, historical_hardware_data.battery_charge_cycles),
			battery_current_max_capacity = COALESCE(EXCLUDED.battery_current_max_capacity, historical_hardware_data.battery_current_max_capacity),
			battery_design_capacity = COALESCE(EXCLUDED.battery_design_capacity, historical_hardware_data.battery_design_capacity),
			battery_manufacturer = COALESCE(EXCLUDED.battery_manufacturer, historical_hardware_data.battery_manufacturer),
			battery_manufacture_date = COALESCE(EXCLUDED.battery_manufacture_date, historical_hardware_data.battery_manufacture_date),
			bios_version = COALESCE(EXCLUDED.bios_version, historical_hardware_data.bios_version),
			bios_release_date = COALESCE(EXCLUDED.bios_release_date, historical_hardware_data.bios_release_date),
			bios_firmware = COALESCE(EXCLUDED.bios_firmware, historical_hardware_data.bios_firmware),
			memory_serial = COALESCE(EXCLUDED.memory_serial, historical_hardware_data.memory_serial),
			memory_capacity_kb = COALESCE(EXCLUDED.memory_capacity_kb, historical_hardware_data.memory_capacity_kb),
			memory_speed_mhz = COALESCE(EXCLUDED.memory_speed_mhz, historical_hardware_data.memory_speed_mhz)
	;`

	var hardwareHistoryResult sql.Result
	hardwareHistoryResult, err = tx.ExecContext(ctx, historicalHardwareDataTable,
		ptrToNullString(&hardwareData.TransactionUUID),
		ptrToNullInt64(hardwareData.Tagnumber),
		ptrToNullString(hardwareData.SystemSerial),
		ptrToNullString(hardwareData.EthernetMAC),
		ptrToNullString(hardwareData.WiFiMAC),
		ptrToNullString(hardwareData.DiskModel),
		ptrToNullString(hardwareData.DiskType),
		ptrToNullInt64(hardwareData.DiskSize),
		ptrToNullString(hardwareData.DiskSerial),
		ptrToNullInt64(hardwareData.DiskWritesKB),
		ptrToNullInt64(hardwareData.DiskReadsKB),
		ptrToNullInt64(hardwareData.DiskPowerOnHours),
		ptrToNullInt64(hardwareData.DiskErrors),
		ptrToNullInt64(hardwareData.DiskPowerCycles),
		ptrToNullString(hardwareData.DiskFirmware),
		ptrToNullString(hardwareData.BatteryModel),
		ptrToNullString(hardwareData.BatterySerial),
		ptrToNullInt64(hardwareData.BatteryChargeCycles),
		ptrToNullFloat64(hardwareData.BatteryCurrentMaxCapacity),
		ptrToNullFloat64(hardwareData.BatteryDesignCapacity),
		ptrToNullString(hardwareData.BatteryManufacturer),
		ptrToNullDate(hardwareData.BatteryManufactureDate),
		ptrToNullString(hardwareData.BiosVersion),
		ptrToNullString(hardwareData.BiosReleaseDate),
		ptrToNullString(hardwareData.BiosFirmware),
		ptrToNullString(hardwareData.MemorySerial),
		ptrToNullInt64(hardwareData.MemoryCapacityKB),
		ptrToNullInt64(hardwareData.MemorySpeedMHz),
	)
	if err != nil {
		return err
	}
	if err := VerifyRowsAffected(hardwareHistoryResult, 1); err != nil {
		return err
	}
	return nil
}

func (updateRepo *UpdateRepo) UpdateJobQueuedAt(ctx context.Context, jobQueue *types.JobQueueTableRowView) (err error) {
	if jobQueue == nil {
		return fmt.Errorf("required info is nil")
	}
	if jobQueue.Tagnumber == nil || *jobQueue.Tagnumber == 0 {
		return fmt.Errorf("tagnumber is nil")
	}

	if ctx.Err() != nil {
		return fmt.Errorf("context error: %w", ctx.Err())
	}
	tx, err := updateRepo.DB.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("error beginning DB transaction: %w", err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		} else {
			err = tx.Commit()
		}
	}()

	const sqlCode = `
		UPDATE
			job_queue
		SET
			job_queued_at = $2
		WHERE
			client_uuid = (SELECT uuid FROM ids WHERE tagnumber = $1 ORDER BY time DESC LIMIT 1)
	;`

	var res sql.Result
	res, err = tx.ExecContext(ctx, sqlCode,
		ptrToNullInt64(jobQueue.Tagnumber),
		ptrToNullTime(jobQueue.JobQueuedAt),
	)
	if err != nil {
		return err
	}
	if err := VerifyRowsAffected(res, 1); err != nil {
		return err
	}

	return nil
}

func (updateRepo *UpdateRepo) UpdateClientLastHeard(ctx context.Context, tag *int64, lastHeard *time.Time) (err error) {
	if tag == nil || *tag == 0 {
		return fmt.Errorf("tagnumber is required")
	}
	if lastHeard == nil || lastHeard.IsZero() {
		return fmt.Errorf("last heard time is required")
	}
	if ctx.Err() != nil {
		return fmt.Errorf("context error: %w", ctx.Err())
	}
	tx, err := updateRepo.DB.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("error beginning DB transaction: %w", err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		} else {
			err = tx.Commit()
		}
	}()

	const sqlCode = `
		UPDATE 
			job_queue 
		SET 
			last_heard = COALESCE($2, CURRENT_TIMESTAMP) 
		WHERE client_uuid = (SELECT uuid FROM ids WHERE tagnumber = $1 ORDER BY time DESC LIMIT 1);`
	var sqlResult sql.Result
	sqlResult, err = tx.ExecContext(ctx, sqlCode,
		ptrToNullInt64(tag),
		ptrToNullTime(lastHeard),
	)
	if err != nil {
		return err
	}
	if err := VerifyRowsAffected(sqlResult, 1); err != nil {
		return err
	}
	return nil
}

func (updateRepo *UpdateRepo) UpdateClientBatteryChargePcnt(ctx context.Context, tag *int64, percent *float64) (err error) {
	if err := types.IsTagnumberInt64Valid(tag); err != nil {
		return err
	}
	if percent == nil || *percent < 0 || *percent > 100 {
		return fmt.Errorf("percent must be between 0 and 100")
	}
	tx, err := updateRepo.DB.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("error beginning DB transaction: %w", err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		} else {
			err = tx.Commit()
		}
	}()
	const sqlCode = `
		UPDATE
			job_queue
		SET
			battery_charge_pcnt = $2
		WHERE
			client_uuid = (SELECT uuid FROM ids WHERE tagnumber = $1 ORDER BY time DESC LIMIT 1)
	;`
	var sqlResult sql.Result
	sqlResult, err = tx.ExecContext(ctx, sqlCode,
		ptrToNullInt64(tag),
		ptrToNullFloat64(percent),
	)
	if err != nil {
		return fmt.Errorf("error updating client's battery charge percent: %w", err)
	}
	if err := VerifyRowsAffected(sqlResult, 1); err != nil {
		return fmt.Errorf("error while checking rows affected on job_queue table update: %w", err)
	}
	return nil
}

func (updateRepo *UpdateRepo) BulkUpdateClientLocation(ctx context.Context, transactionUUID *string, tag *int64, location *string) (err error) {
	if transactionUUID == nil || strings.TrimSpace(*transactionUUID) == "" {
		return fmt.Errorf("transactionUUID is required")
	}
	if err := types.IsTagnumberInt64Valid(tag); err != nil {
		return err
	}
	if location == nil || strings.TrimSpace(*location) == "" {
		return fmt.Errorf("location is required")
	}
	tx, err := updateRepo.DB.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("error beginning DB transaction: %w", err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		} else {
			err = tx.Commit()
		}
	}()
	const locationsLogSql = `
	INSERT INTO locations_log (
		time, 
		client_uuid,
		tagnumber,
		system_serial,
		location,
		is_broken,
		disk_removed,
		department_name,
		ad_domain,
		note,
		client_status,
		building,
		room,
		property_custodian,
		acquired_date,
		retired_date,
		transaction_uuid,
		bulk_update
	)
	SELECT 
		CURRENT_TIMESTAMP, 
		(SELECT uuid FROM ids WHERE tagnumber = $1 ORDER BY time DESC LIMIT 1),
		$1,
		locations.system_serial,
		$2,
		locations.is_broken,
		locations.disk_removed,
		locations.department_name,
		locations.ad_domain,
		locations.note,
		locations.client_status,
		locations.building,
		locations.room,
		locations.property_custodian,
		locations.acquired_date,
		locations.retired_date,
		$3,
		TRUE
	FROM 
		locations
	WHERE 
		locations.client_uuid = (SELECT uuid FROM ids WHERE tagnumber = $1 ORDER BY time DESC LIMIT 1)
	ORDER BY 
		time DESC NULLS LAST
	LIMIT 1
	;`
	var locationsLogSqlResult sql.Result
	locationsLogSqlResult, err = tx.ExecContext(ctx, locationsLogSql,
		ptrToNullInt64(tag),
		ptrToNullString(location),
		ptrToNullString(transactionUUID),
	)
	if err != nil {
		return fmt.Errorf("error while bulk updating a client's ('%d') location: %w", *tag, err)
	}
	if err := VerifyRowsAffected(locationsLogSqlResult, 1); err != nil {
		return fmt.Errorf("error while checking rows affected on a locations bulk update: %w", err)
	}

	const locationsSQL = `
	INSERT INTO locations (
		time, 
		client_uuid,
		tagnumber,
		system_serial,
		location,
		is_broken,
		disk_removed,
		department_name,
		ad_domain,
		note,
		client_status,
		building,
		room,
		property_custodian,
		acquired_date,
		retired_date,
		transaction_uuid,
		bulk_update
	)
	SELECT 
		CURRENT_TIMESTAMP, 
		(SELECT uuid FROM ids WHERE tagnumber = $1 ORDER BY time DESC LIMIT 1),
		$1,
		locations.system_serial,
		$2,
		locations.is_broken,
		locations.disk_removed,
		locations.department_name,
		locations.ad_domain,
		locations.note,
		locations.client_status,
		locations.building,
		locations.room,
		locations.property_custodian,
		locations.acquired_date,
		locations.retired_date,
		$3,
		TRUE
	FROM 
		locations
	WHERE 
		locations.client_uuid = (SELECT uuid FROM ids WHERE tagnumber = $1 ORDER BY time DESC LIMIT 1)
	ORDER BY 
		time DESC NULLS LAST
	LIMIT 1
	ON CONFLICT (client_uuid) DO UPDATE SET
	 time = EXCLUDED.time,
	 client_uuid = EXCLUDED.client_uuid,
	 tagnumber = EXCLUDED.tagnumber,
	 system_serial = COALESCE(EXCLUDED.system_serial, locations.system_serial),
	 location = EXCLUDED.location,
	 is_broken = COALESCE(EXCLUDED.is_broken, locations.is_broken),
	 disk_removed = COALESCE(EXCLUDED.disk_removed, locations.disk_removed),
	 department_name = COALESCE(EXCLUDED.department_name, locations.department_name),
	 ad_domain = COALESCE(EXCLUDED.ad_domain, locations.ad_domain),
	 note = COALESCE(EXCLUDED.note, locations.note),
	 building = COALESCE(EXCLUDED.building, locations.building),
	 room = COALESCE(EXCLUDED.room, locations.room),
	 property_custodian = COALESCE(EXCLUDED.property_custodian, locations.property_custodian),
	 acquired_date = COALESCE(EXCLUDED.acquired_date, locations.acquired_date),
	 retired_date = COALESCE(EXCLUDED.retired_date, locations.retired_date),
	 transaction_uuid = COALESCE(EXCLUDED.transaction_uuid, locations.transaction_uuid),
	 bulk_update = TRUE
	;`
	var locationsSQLResult sql.Result
	locationsSQLResult, err = tx.ExecContext(ctx, locationsSQL,
		ptrToNullInt64(tag),
		ptrToNullString(location),
		ptrToNullString(transactionUUID),
	)
	if err != nil {
		return fmt.Errorf("error while bulk updating a client's ('%d') location: %w", *tag, err)
	}
	if err := VerifyRowsAffected(locationsSQLResult, 1); err != nil {
		return fmt.Errorf("error while checking rows affected on a locations bulk update: %w", err)
	}
	return nil
}

func UpdateWindowsClientInfo(ctx context.Context, winClientInfo *types.WindowsUpdateDTO, transactionUUID string) (err error) {
	if winClientInfo == nil {
		return fmt.Errorf("%w: %s", types.InvalidStructureError, "WindowsUpdateDTO")
	}
	if err := types.IsTagnumberInt64Valid(&winClientInfo.Tagnumber); err != nil {
		return fmt.Errorf("%w: %s", types.InvalidFieldError, "Tagnumber")
	}
	if strings.TrimSpace(winClientInfo.SystemSerial) == "" {
		return fmt.Errorf("%w: %s", types.MissingFieldError, "SystemSerial")
	}
	if strings.TrimSpace(transactionUUID) == "" {
		return fmt.Errorf("%w: %s", types.MissingFieldError, "TransactionUUID")
	}

	dbConn, err := config.GetDatabaseConn()
	if err != nil {
		return fmt.Errorf("%w: %w", types.DatabaseConnError, err)
	}
	tx, err := dbConn.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("%w: %w", types.DatabaseConnError, err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		} else {
			err = tx.Commit()
		}
	}()

	const clientHealthDataSQLCode = `
		INSERT INTO 
			client_health
		(
			time,
			client_uuid,
			last_hardware_check,
			tagnumber,
			system_serial,
			tpm_version,
			os_name,
			disk_free_space_kb,
			updated_from_windows,
			transaction_uuid
		)
		VALUES
		(
			CURRENT_TIMESTAMP,
			(SELECT uuid FROM ids WHERE tagnumber = $1 ORDER BY time DESC LIMIT 1),
			CURRENT_TIMESTAMP,
			$1,
			$2,
			$3,
			$4,
			$5,
			$6,
			$7,
			$8
		) ON CONFLICT (tagnumber) DO UPDATE SET
			time = CURRENT_TIMESTAMP,
			client_uuid = COALESCE(EXCLUDED.client_uuid, client_health.client_uuid),
			last_hardware_check = CURRENT_TIMESTAMP,
			system_serial = COALESCE(EXCLUDED.system_serial, client_health.system_serial),
			tpm_version = COALESCE(EXCLUDED.tpm_version, client_health.tpm_version),
			os_name = COALESCE(EXCLUDED.os_name, client_health.os_name),
			disk_free_space_kb = COALESCE(EXCLUDED.disk_free_space_kb, client_health.disk_free_space_kb),
			updated_from_windows = TRUE,
			transaction_uuid = COALESCE(EXCLUDED.transaction_uuid, client_health.transaction_uuid)
	;`

	var sqlResult sql.Result
	sqlResult, err = tx.ExecContext(ctx, clientHealthDataSQLCode,
		toNullInt64(winClientInfo.Tagnumber),
		toNullString(winClientInfo.SystemSerial),
		winClientInfo.TPMVersion,
		winClientInfo.OSName,
		winClientInfo.DiskFreeSpaceKB,
		true,
		transactionUUID,
	)
	if err != nil {
		return fmt.Errorf("%w: %w", types.DatabaseUpdateError, err)
	}
	if err := VerifyRowsAffected(sqlResult, 1); err != nil {
		return fmt.Errorf("%w: %w", types.DatabaseUpdateError, err)
	}

	const historicalHardwareDataSQLCode = `
		INSERT INTO
			historical_hardware_data (
				time,
				client_uuid,
				tagnumber,
				system_serial,
				ethernet_mac,
				wifi_mac,
				bios_version,
				disk_model,
				disk_type,
				disk_size_kb,
				memory_capacity_kb,
				memory_speed_mhz,
				battery_manufacturer,
				battery_serial,
				battery_current_max_capacity,
				battery_design_capacity,
				battery_charge_cycles,
				battery_health,
				updated_from_windows,
				transaction_uuid
			)
		VALUES (
			CURRENT_TIMESTAMP,
			(SELECT uuid FROM ids WHERE tagnumber = $1 ORDER BY time DESC LIMIT 1),
			$1,
			$2,
			$3,
			$4,
			$5,
			$6,
			$7,
			$8,
			$9,
			$10,
			$11,
			$12,
			$13,
			$14,
			$15,
			$16,
			TRUE,
			$17
		)
	;`

	historicalHWData, err := tx.ExecContext(ctx, historicalHardwareDataSQLCode,
		toNullInt64(winClientInfo.Tagnumber),
		toNullString(winClientInfo.SystemSerial),
		ptrToNullString(winClientInfo.EthernetMACAddr),
		ptrToNullString(winClientInfo.WifiMACAddr),
		ptrToNullString(winClientInfo.BIOSVersion),
		ptrToNullString(winClientInfo.DiskModel),
		ptrToNullString(winClientInfo.DiskType),
		ptrToNullInt64(winClientInfo.DiskSizeKB),
		ptrToNullInt64(winClientInfo.MemoryCapacityKB),
		ptrToNullInt64(winClientInfo.MemorySpeedMHz),
		ptrToNullString(winClientInfo.BatteryManufacturer),
		ptrToNullString(winClientInfo.BatterySerial),
		ptrToNullInt64(winClientInfo.BatteryCurrentMaxCapacity),
		ptrToNullInt64(winClientInfo.BatteryDesignCapacity),
		ptrToNullInt64(winClientInfo.BatteryChargeCycleCount),
		ptrToNullFloat64(winClientInfo.BatteryHealthPcnt),
		transactionUUID,
	)
	if err != nil {
		return fmt.Errorf("%w: %w", types.DatabaseUpdateError, err)
	}
	if err := VerifyRowsAffected(historicalHWData, 1); err != nil {
		return fmt.Errorf("%w: %w", types.DatabaseUpdateError, err)
	}

	return nil
}

func UpsertOSInfo(ctx context.Context, osInfo *types.WindowsUpdateDTO, transactionUUID string) (err error) {
	if osInfo == nil {
		return fmt.Errorf("%w: %s", types.InvalidStructureError, "WindowsUpdateDTO")
	}

	if err := types.IsTagnumberInt64Valid(&osInfo.Tagnumber); err != nil {
		return fmt.Errorf("%w: %s", types.InvalidFieldError, "Tagnumber")
	}
	if strings.TrimSpace(osInfo.SystemSerial) == "" {
		return fmt.Errorf("%w: %s", types.MissingFieldError, "SystemSerial")
	}
	if strings.TrimSpace(transactionUUID) == "" {
		return fmt.Errorf("%w: %s", types.MissingFieldError, "TransactionUUID")
	}
	dbConn, err := config.GetDatabaseConn()
	if err != nil {
		return fmt.Errorf("%w: %w", types.DatabaseConnError, err)
	}
	tx, err := dbConn.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("%w: %w", types.DatabaseConnError, err)
	}
	defer func() {
		if err != nil {
			if rollbackErr := tx.Rollback(); rollbackErr != nil {
				err = fmt.Errorf("%w: %w", err, rollbackErr)
			}
		} else {
			if commitErr := tx.Commit(); commitErr != nil {
				err = fmt.Errorf("%w: %w", types.DatabaseUpdateError, commitErr)
			}
		}
	}()
	const osInfoSQLCode = `
		INSERT INTO os_info (
			client_uuid,
			transaction_uuid,
			time,
			os_install_date,
			os_vendor,
			os_platform,
			os_architecture,
			os_name,
			os_version,
			windows_display_version,
			windows_build_number,
			windows_ubr,
			windows_bitlocker_enabled,
			ad_admin_users,
			computer_name,
			ad_domain,
			ad_computer_name,
			ad_distinguished_name,
			is_intune_joined
		)
		VALUES (
			(SELECT uuid FROM ids WHERE tagnumber = $2 AND system_serial = $3),
			$1,
			CURRENT_TIMESTAMP,
			$4,
			$5,
			$6,
			$7,
			$8,
			$9,
			$10,
			$11,
			$12,
			$13,
			$14,
			$15,
			$16,
			$17,
			$18,
			$19
		) ON CONFLICT (client_uuid) DO UPDATE SET
			client_uuid = EXCLUDED.client_uuid,
			transaction_uuid = EXCLUDED.transaction_uuid,
			time = CURRENT_TIMESTAMP,
			os_install_date = EXCLUDED.os_install_date,
			os_vendor = EXCLUDED.os_vendor,
			os_platform = EXCLUDED.os_platform,
			os_architecture = EXCLUDED.os_architecture,
			os_name = EXCLUDED.os_name,
			os_version = EXCLUDED.os_version,
			windows_display_version = EXCLUDED.windows_display_version,
			windows_build_number = EXCLUDED.windows_build_number,
			windows_ubr = EXCLUDED.windows_ubr,
			windows_bitlocker_enabled = EXCLUDED.windows_bitlocker_enabled,
			ad_admin_users = EXCLUDED.ad_admin_users,
			computer_name = EXCLUDED.computer_name,
			ad_domain = EXCLUDED.ad_domain,
			ad_computer_name = EXCLUDED.ad_computer_name,
			ad_distinguished_name = EXCLUDED.ad_distinguished_name,
			is_intune_joined = EXCLUDED.is_intune_joined
			;`

	var sqlResult sql.Result
	sqlResult, err = tx.ExecContext(ctx, osInfoSQLCode,
		ptrToNullString(&transactionUUID),
		toNullInt64(osInfo.Tagnumber),
		toNullString(osInfo.SystemSerial),
		ptrToNullTime(osInfo.OSInstalledAt),
		ptrToNullString(osInfo.OSVendor),
		ptrToNullString(osInfo.OSPlatform),
		ptrToNullString(osInfo.OSArchitecture),
		ptrToNullString(osInfo.OSName),
		ptrToNullString(osInfo.OSVersion),
		ptrToNullString(osInfo.WindowsDisplayVersion),
		ptrToNullInt64(osInfo.WindowsBuildNumber),
		ptrToNullInt64(osInfo.WindowsUBR),
		ptrToNullBool(osInfo.WindowsBitlockerEnabled),
		ptrToNullString(osInfo.ADAdminUsers),
		ptrToNullString(osInfo.ComputerName),
		ptrToNullString(osInfo.ADDomain),
		ptrToNullString(osInfo.ADComputerName),
		ptrToNullString(osInfo.ADDistinguishedName),
		ptrToNullBool(osInfo.IsIntuneJoined),
	)

	if err != nil {
		return fmt.Errorf("%w: %w", types.DatabaseUpdateError, err)
	}
	if err := VerifyRowsAffected(sqlResult, 1); err != nil {
		return fmt.Errorf("%w: %w", types.DatabaseUpdateError, err)
	}
	return nil
}
