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
	"errors"
	"fmt"
	"strings"
	"time"
	"uit-toolbox/config"
	"uit-toolbox/types"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type Update interface {
	UpdateClientNetworkUsage(ctx context.Context, networkData *types.NetworkData) (err error)
	UpdateClientAppUptime(ctx context.Context, tag int64, appUptime int64) (err error)
	UpdateClientSystemUptime(ctx context.Context, tag int64, systemUptime int64) (err error)
	UpdateJobQueuedAt(ctx context.Context, jobQueue *types.JobQueueTableRowView) (err error)
	UpdateClientBatteryChargePcnt(ctx context.Context, tag *int64, percent *float64) (err error)
	BulkUpdateClientLocation(ctx context.Context, transactionUUID *string, tag *int64, location *string) (err error)
}

type UpdateRepo struct {
	DB *sql.DB
}

func lockClientRowByTagnumber(ctx context.Context, tx *sql.Tx, tagnumber int64) (clientUUID uuid.UUID, err error) {
	const sqlCode = `
		SELECT uuid
		FROM ids
		WHERE tagnumber = $1
		FOR UPDATE
	;`

	err = tx.QueryRowContext(ctx, sqlCode, toNullInt64(tagnumber)).Scan(&clientUUID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return uuid.Nil, fmt.Errorf("%w: no client found for tagnumber '%d'", types.DatabaseQueryError, tagnumber)
		}
		return uuid.Nil, fmt.Errorf("%w: %w", types.DatabaseQueryError, err)
	}

	return clientUUID, nil
}

func lockClientRowByTagnumberPGX(ctx context.Context, tx pgx.Tx, tag int64) (uuid.UUID, error) {
	const sqlCode = `
        SELECT uuid
        FROM ids
        WHERE tagnumber = $1
        FOR UPDATE
    ;`
	var clientUUID uuid.UUID
	err := tx.QueryRow(ctx, sqlCode, tag).Scan(&clientUUID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return uuid.Nil, fmt.Errorf("%w: no client found for tagnumber '%d'", types.DatabaseQueryError, tag)
		}
		return uuid.Nil, fmt.Errorf("%w: %w", types.DatabaseQueryError, err)
	}
	return clientUUID, nil
}

func lockClientRowBySystemSerial(ctx context.Context, tx *sql.Tx, systemSerial string) (clientUUID uuid.UUID, err error) {
	const sqlCode = `
		SELECT uuid
		FROM ids
		WHERE system_serial = $1
		FOR UPDATE
	;`

	err = tx.QueryRowContext(ctx, sqlCode, toNullString(systemSerial)).Scan(&clientUUID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return uuid.Nil, fmt.Errorf("%w: no client found for system serial '%s'", types.DatabaseQueryError, systemSerial)
		}
		return uuid.Nil, fmt.Errorf("%w: %w", types.DatabaseQueryError, err)
	}

	return clientUUID, nil
}

func lockClientRowBySystemSerialPGX(ctx context.Context, tx pgx.Tx, systemSerial string) (uuid.UUID, error) {
	const sqlCode = `
        SELECT uuid
        FROM ids
        WHERE system_serial = $1
        FOR UPDATE
    ;`
	var clientUUID uuid.UUID
	err := tx.QueryRow(ctx, sqlCode, systemSerial).Scan(&clientUUID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return uuid.Nil, fmt.Errorf("%w: no client found for system serial '%s'", types.DatabaseQueryError, systemSerial)
		}
		return uuid.Nil, fmt.Errorf("%w: %w", types.DatabaseQueryError, err)
	}
	return clientUUID, nil
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
		return fmt.Errorf("%w: %s", types.InvalidStructureError, "ClientHealthDTO is nil")
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
		INSERT INTO client_health (
			time, 
			client_uuid,
			last_hardware_check, 
			transaction_uuid
		) VALUES (
			CURRENT_TIMESTAMP, 
			(SELECT uuid FROM ids WHERE tagnumber = $1 ORDER BY time DESC LIMIT 1),
			$2, 
			$3
		)
		ON CONFLICT (client_uuid)
			DO UPDATE SET
				time = CURRENT_TIMESTAMP,
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
		return fmt.Errorf("%w: %s", types.InvalidStructureError, "InventoryCheckoutWriteModel is nil")
	}
	if err := types.IsTagnumberInt64Valid(&checkoutData.Tagnumber); err != nil {
		return types.CreateInvalidFieldError("tagnumber", err)
	}
	// if checkoutData.CheckoutDate == nil &&
	// 	checkoutData.ReturnDate == nil &&
	// 	(checkoutData.CheckoutBool != nil && !*checkoutData.CheckoutBool) &&
	// 	(checkoutData.CustomerName == nil || strings.TrimSpace(*checkoutData.CustomerName) == "") {
	// 	return nil
	// }

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

	const checkoutSql = `
		INSERT INTO checkout_log (
			time, 
			client_uuid,
			transaction_uuid, 
			checkout_date, 
			return_date, 
			checkout_bool,
			customer_name
		) VALUES (
			CURRENT_TIMESTAMP, 
			(SELECT uuid FROM ids WHERE tagnumber = $2 ORDER BY time DESC LIMIT 1),
			$1, 
			$3, 
			$4, 
			$5, 
			$6
		) ON CONFLICT (transaction_uuid) DO UPDATE SET
		 	time = EXCLUDED.time, 
			client_uuid = EXCLUDED.client_uuid, 
			checkout_date = EXCLUDED.checkout_date, 
			return_date = EXCLUDED.return_date, 
			checkout_bool = EXCLUDED.checkout_bool, 
			customer_name = EXCLUDED.customer_name 
	;`

	checkoutLogResult, err := tx.ExecContext(ctx, checkoutSql,
		transactionUUID,
		toNullInt64(checkoutData.Tagnumber),
		ptrToNullTime(checkoutData.CheckoutDate),
		ptrToNullTime(checkoutData.ReturnDate),
		ptrToNullBool(checkoutData.CheckoutBool),
		ptrToNullString(checkoutData.CustomerName),
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
		return types.CreateInvalidFieldError("transaction_uuid", types.MissingFieldError)
	}
	if hardwareData == nil {
		return fmt.Errorf("%w: %s (%s)", types.InvalidStructureError, "InventoryHardwareWriteModel", "nil")
	}
	if err := types.IsTagnumberInt64Valid(&hardwareData.Tagnumber); err != nil {
		return types.CreateInvalidFieldError("tagnumber", err)
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
		INSERT INTO hardware_data (
			time, 
			client_uuid, 
			transaction_uuid, 
			system_manufacturer, 
			system_model, 
			device_type 
		) VALUES (
			CURRENT_TIMESTAMP, 
			(SELECT uuid FROM ids WHERE tagnumber = $2 ORDER BY time DESC LIMIT 1), 
			$1, 
			$3, 
			$4, 
			$5 
		)
		ON CONFLICT (client_uuid) DO UPDATE SET
			time = EXCLUDED.time, 
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
		return types.CreateInvalidFieldError("transaction_uuid", types.MissingFieldError)
	}
	if inventoryUpdate == nil {
		return fmt.Errorf("%w: %s (%s)", types.InvalidStructureError, "InventoryLocationWriteModel", "nil")
	}

	if err := types.IsTagnumberInt64Valid(&inventoryUpdate.Tagnumber); err != nil {
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

	// Insert/update ids table
	const idsSql = `
		INSERT INTO ids (
			uuid, 
			time, 
			tagnumber, 
			system_serial
		) VALUES (
			uuidv7(), 
			CURRENT_TIMESTAMP, 
			$1, 
			$2 
		)
		ON CONFLICT (tagnumber) DO NOTHING
	;`

	_, err = tx.ExecContext(ctx, idsSql,
		toNullInt64(inventoryUpdate.Tagnumber),
		toNullString(inventoryUpdate.SystemSerial),
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
		toNullInt64(inventoryUpdate.Tagnumber),
		toNullString(inventoryUpdate.SystemSerial),
		toNullString(inventoryUpdate.Location),
		ptrToNullString(inventoryUpdate.Building),
		ptrToNullString(inventoryUpdate.Room),
		toNullString(inventoryUpdate.Department),
		toNullString(inventoryUpdate.ADDomain),
		ptrToNullString(inventoryUpdate.PropertyCustodian),
		ptrToNullTime(inventoryUpdate.AcquiredDate),
		ptrToNullTime(inventoryUpdate.RetiredDate),
		ptrToNullBool(inventoryUpdate.IsBroken),
		ptrToNullBool(inventoryUpdate.DiskRemoved),
		toNullString(inventoryUpdate.ClientStatus),
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
	) VALUES (
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
		toNullInt64(inventoryUpdate.Tagnumber),
		toNullString(inventoryUpdate.SystemSerial),
		toNullString(inventoryUpdate.Location),
		ptrToNullString(inventoryUpdate.Building),
		ptrToNullString(inventoryUpdate.Room),
		toNullString(inventoryUpdate.Department),
		toNullString(inventoryUpdate.ADDomain),
		ptrToNullString(inventoryUpdate.PropertyCustodian),
		ptrToNullTime(inventoryUpdate.AcquiredDate),
		ptrToNullTime(inventoryUpdate.RetiredDate),
		ptrToNullBool(inventoryUpdate.IsBroken),
		ptrToNullBool(inventoryUpdate.DiskRemoved),
		toNullString(inventoryUpdate.ClientStatus),
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
		strings.TrimSpace(manifest.FileUUID) == "" ||
		manifest.Time.IsZero() ||
		strings.TrimSpace(manifest.FileName) == "" ||
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
		INSERT INTO client_images (
			uuid, 
			time, 
			client_uuid, 
			tagnumber, 
			filename, 
			thumbnail_filename, 
			filesize, 
			sha256_hash, 
			mime_type, 
			exif_timestamp, 
			resolution_x, 
			resolution_y, 
			note, 
			hidden, 
			pinned
		) VALUES (
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
			$14
		) ON CONFLICT (uuid) DO NOTHING
	;`

	sqlResult, err := tx.ExecContext(ctx, sqlCode,
		toNullString(manifest.FileUUID),
		toNullTime(manifest.Time),
		toNullInt64(manifest.Tagnumber),
		toNullString(manifest.FileName),
		ptrToNullString(manifest.ThumbnailFileName),
		toNullInt64(manifest.FileSize),
		manifest.SHA256Hash,
		toNullString(manifest.MimeType),
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

func HideClientImageByUUID(ctx context.Context, fileUUID string) (err error) {
	if strings.TrimSpace(fileUUID) == "" {
		return fmt.Errorf("%w: %s", types.MissingFieldError, "file UUID")
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
			uuid = $1
	;`

	sqlResult, err := tx.ExecContext(ctx, sqlQuery,
		toNullString(fileUUID),
	)
	if err != nil {
		return err
	}
	if err := VerifyRowsAffected(sqlResult, 1); err != nil {
		return err
	}
	return nil
}

func TogglePinImage(ctx context.Context, tagnumber *int64, fileUUID *string) (err error) {
	if err := types.IsTagnumberInt64Valid(tagnumber); err != nil {
		return fmt.Errorf("%w: %s (%w)", types.InvalidFieldError, "tagnumber", err)
	}
	if fileUUID == nil || strings.TrimSpace(*fileUUID) == "" {
		return fmt.Errorf("%w: %s", types.MissingFieldError, "file UUID")
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
			pinned = NOT COALESCE(pinned, FALSE) 
		WHERE 
			uuid = $1 
			AND client_uuid = (SELECT uuid FROM ids WHERE tagnumber = $2)
	;`
	sqlResult, err := tx.ExecContext(ctx, sqlQuery,
		ptrToNullString(fileUUID),
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
		UPDATE job_queue
		SET job_name = $1
		WHERE job_queue.job_active = FALSE
 	 		AND job_queue.job_queued = FALSE
  		AND EXISTS (
    		SELECT 1 
				FROM live_os_data
    		WHERE live_os_data.client_uuid = job_queue.client_uuid
      		AND CURRENT_TIMESTAMP - live_os_data.last_heard < INTERVAL '10 SECONDS'
  		)
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

func UpsertClientMemoryUsageKB(ctx context.Context, memInfo types.MemoryDataUpdateDTO) (err error) {
	if err := types.IsTagnumberInt64Valid(&memInfo.Tagnumber); err != nil {
		return fmt.Errorf("%w: %s (%w)", types.InvalidFieldError, "tagnumber", err)
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

	clientUUID, err := lockClientRowByTagnumber(ctx, tx, memInfo.Tagnumber)
	if err != nil {
		return fmt.Errorf("%w: %w", types.DatabaseQueryError, err)
	}

	const sqlCode = `
		INSERT INTO 
			live_os_data (
				client_uuid, 
				memory_usage_kb
			) 
		VALUES 
			(
				$1, 
				$2
			)
		ON CONFLICT (client_uuid) DO UPDATE SET 
			memory_usage_kb = EXCLUDED.memory_usage_kb
	;`
	var sqlResult sql.Result
	sqlResult, err = tx.ExecContext(ctx, sqlCode,
		toNullUUID(clientUUID),
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

func UpsertClientMemoryCapacityKB(ctx context.Context, memInfo types.MemoryDataUpdateDTO) (err error) {
	if err := types.IsTagnumberInt64Valid(&memInfo.Tagnumber); err != nil {
		return fmt.Errorf("%w: %s (%w)", types.InvalidFieldError, "tagnumber", err)
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

	clientUUID, err := lockClientRowByTagnumber(ctx, tx, memInfo.Tagnumber)
	if err != nil {
		return fmt.Errorf("%w: %w", types.DatabaseQueryError, err)
	}

	const sqlCode = `
		INSERT INTO 
			job_queue (
				client_uuid, 
				tagnumber,
				memory_capacity_kb
			) 
		VALUES 
			(
				$1,
				$2, 
				$3
			)
		ON CONFLICT (client_uuid) DO UPDATE SET 
			tagnumber = EXCLUDED.tagnumber,
			memory_capacity_kb = EXCLUDED.memory_capacity_kb
	;`
	var sqlResult sql.Result
	sqlResult, err = tx.ExecContext(ctx, sqlCode,
		clientUUID,
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

func UpsertClientCPUUsage(ctx context.Context, cpuData *types.CPUDataUpdateDTO) (err error) {
	if cpuData == nil {
		return fmt.Errorf("CPU data is required")
	}

	if err := types.IsTagnumberInt64Valid(&cpuData.Tagnumber); err != nil {
		return fmt.Errorf("%w: %s (%w)", types.InvalidFieldError, "tagnumber", err)
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

	clientUUID, err := lockClientRowByTagnumber(ctx, tx, cpuData.Tagnumber)
	if err != nil {
		return fmt.Errorf("%w: %w", types.DatabaseQueryError, err)
	}

	const sqlCode = `
		INSERT INTO 
			live_os_data (
				client_uuid,
				cpu_usage
			) 
		VALUES (
			$1, 
			$2
		)
		ON CONFLICT (client_uuid) DO UPDATE SET 
			cpu_usage = EXCLUDED.cpu_usage
	;`
	var sqlResult sql.Result
	sqlResult, err = tx.ExecContext(ctx, sqlCode,
		toNullUUID(clientUUID),
		cpuData.UsagePercent, // Usage can be at 0%, so don't set to null if it's 0
	)
	if err != nil {
		return fmt.Errorf("%w: %w", types.DatabaseUpdateError, err)
	}
	if err := VerifyRowsAffected(sqlResult, 1); err != nil {
		return err
	}
	return nil
}

func UpsertClientCPUMHz(ctx context.Context, cpuData *types.CPUDataUpdateDTO) (err error) {
	if cpuData == nil {
		return fmt.Errorf("CPU data is required")
	}

	if err := types.IsTagnumberInt64Valid(&cpuData.Tagnumber); err != nil {
		return fmt.Errorf("%w: %s (%w)", types.InvalidFieldError, "tagnumber", err)
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
		ON CONFLICT (client_uuid) DO UPDATE SET 
			tagnumber = EXCLUDED.tagnumber,
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
		return fmt.Errorf("%w: %s", types.InvalidStructureError, "NetworkData is nil")
	}
	if err := types.IsTagnumberInt64Valid(&networkData.Tagnumber); err != nil {
		return fmt.Errorf("%w: %s (%w)", types.InvalidFieldError, "tagnumber", err)
	}
	if networkData.NetworkUsage == nil {
		return fmt.Errorf("%w: %s", types.MissingFieldError, "network usage")
	}
	if networkData.LinkSpeed == nil {
		return fmt.Errorf("%w: %s", types.MissingFieldError, "link speed")
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

	clientUUID, err := lockClientRowByTagnumber(ctx, tx, networkData.Tagnumber)
	if err != nil {
		return fmt.Errorf("error locking client row: %w", err)
	}
	const sqlCode = `
		INSERT INTO live_os_data (
			client_uuid, 
			network_usage, 
			link_speed
		) VALUES (
			$1, 
			$2,
			$3
		)
		ON CONFLICT (client_uuid) DO UPDATE SET 
			network_usage = EXCLUDED.network_usage,
			link_speed = EXCLUDED.link_speed;`
	var sqlResult sql.Result
	sqlResult, err = tx.ExecContext(ctx, sqlCode,
		toNullUUID(clientUUID),
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

func UpsertClientCPUTemperature(ctx context.Context, cpuTempData *types.CPUDataUpdateDTO) (err error) {
	if cpuTempData == nil {
		return fmt.Errorf("%w: %s", types.InvalidStructureError, "CPUData is nil")
	}
	if err := types.IsTagnumberInt64Valid(&cpuTempData.Tagnumber); err != nil {
		return fmt.Errorf("%w: %s (%w)", types.InvalidFieldError, "tagnumber", err)
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
		ON CONFLICT (client_uuid) DO UPDATE SET 
			tagnumber = EXCLUDED.tagnumber,
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

	clientUUID, err := lockClientRowByTagnumber(ctx, tx, tag)
	if err != nil {
		return fmt.Errorf("error locking client row: %w", err)
	}

	const sqlCode = `
		INSERT INTO 
			live_os_data (
				client_uuid,
				system_uptime
			) VALUES (
				$1, 
			 	$2
			)
		ON CONFLICT (client_uuid) DO UPDATE SET 
		system_uptime = COALESCE(EXCLUDED.system_uptime, live_os_data.system_uptime)
	;`
	var sqlResult sql.Result
	sqlResult, err = tx.ExecContext(ctx, sqlCode,
		toNullUUID(clientUUID),
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

	clientUUID, err := lockClientRowByTagnumber(ctx, tx, tag)
	if err != nil {
		return fmt.Errorf("error locking client row: %w", err)
	}

	const sqlCode = `
		INSERT INTO 
			live_os_data (
				client_uuid,
				client_app_uptime
			) VALUES (
				$1, 
			 	$2
			)
		ON CONFLICT (client_uuid) DO UPDATE SET 
		client_app_uptime = COALESCE(EXCLUDED.client_app_uptime, live_os_data.client_app_uptime)
	;`
	var sqlResult sql.Result
	sqlResult, err = tx.ExecContext(ctx, sqlCode,
		toNullUUID(clientUUID),
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
		return fmt.Errorf("%w: %s", types.InvalidStructureError, "healthCheck is nil")
	}
	if err := types.IsTagnumberInt64Valid(&healthCheck.Tagnumber); err != nil {
		return fmt.Errorf("%w: %s (%w)", types.InvalidFieldError, "tagnumber", err)
	}
	if healthCheck.TransactionUUID == "" {
		return fmt.Errorf("%w: %s", types.MissingFieldError, "transaction UUID")
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

	clientUUID, err := lockClientRowByTagnumber(ctx, tx, healthCheck.Tagnumber)
	if err != nil {
		return err
	}
	const clientHealthCheckSQL = `
		INSERT INTO 
			client_health (
				time, 
				transaction_uuid, 
				client_uuid, 
				last_hardware_check 
			) 
		VALUES (
			CURRENT_TIMESTAMP, 
			$1,
			$2,
			COALESCE($3, CURRENT_TIMESTAMP)
		)
		ON CONFLICT (client_uuid) DO UPDATE SET 
			time = CURRENT_TIMESTAMP,
			transaction_uuid = EXCLUDED.transaction_uuid,
			last_hardware_check = EXCLUDED.last_hardware_check
		;`
	var sqlResult sql.Result
	sqlResult, err = tx.ExecContext(ctx, clientHealthCheckSQL,
		toNullString(healthCheck.TransactionUUID),
		clientUUID,
		ptrToNullTime(healthCheck.LastHardwareCheck),
	)
	if err != nil {
		return fmt.Errorf("%w: %w", types.DatabaseUpdateError, err)
	}
	if err := VerifyRowsAffected(sqlResult, 1); err != nil {
		return err
	}

	const clientHardwareSQL = `
		INSERT INTO hardware_data
		(
			time,
			transaction_uuid,
			client_uuid,
			tpm_version
		) VALUES (
			CURRENT_TIMESTAMP,
			$1,
			$2,
			$3
		) ON CONFLICT (client_uuid) DO UPDATE SET
			time = CURRENT_TIMESTAMP,
		 	transaction_uuid = COALESCE(EXCLUDED.transaction_uuid, hardware_data.transaction_uuid),
			tpm_version = COALESCE(EXCLUDED.tpm_version, hardware_data.tpm_version)
	;`
	sqlResult, err = tx.ExecContext(ctx, clientHardwareSQL,
		toNullString(healthCheck.TransactionUUID),
		clientUUID,
		ptrToNullString(healthCheck.TPMVersion),
	)
	if err != nil {
		return fmt.Errorf("%w: %w", types.DatabaseUpdateError, err)
	}
	if err := VerifyRowsAffected(sqlResult, 1); err != nil {
		return err
	}

	const clientFirmwareTableInsertSQL = `
		INSERT INTO 
			historical_firmware_data (
				time, 
				transaction_uuid,
				client_uuid, 
				bios_version,
				bios_release_date
			) 
		VALUES (
			CURRENT_TIMESTAMP,
			$1,
			$2,
			$3,
			$4
		) ON CONFLICT (transaction_uuid) DO UPDATE SET
		 	time = CURRENT_TIMESTAMP,
			client_uuid = COALESCE(EXCLUDED.client_uuid, historical_firmware_data.client_uuid),
			bios_version = COALESCE(EXCLUDED.bios_version, historical_firmware_data.bios_version),
			bios_release_date = COALESCE(EXCLUDED.bios_release_date, historical_firmware_data.bios_release_date)
	;`

	sqlResult, err = tx.ExecContext(ctx, clientFirmwareTableInsertSQL,
		toNullString(healthCheck.TransactionUUID),
		clientUUID,
		ptrToNullString(healthCheck.BIOSVersion),
		ptrToNullTime(healthCheck.BIOSReleaseDate),
	)
	if err != nil {
		return fmt.Errorf("%w: %w", types.DatabaseUpdateError, err)
	}
	if err := VerifyRowsAffected(sqlResult, 1); err != nil {
		return err
	}

	return nil
}

// Function to be used from Linux request
func UpdateClientHardwareData(ctx context.Context, hardwareData *types.ClientHardwareView) (err error) {
	if hardwareData == nil || hardwareData.SystemSerial == nil || strings.TrimSpace(hardwareData.TransactionUUID) == "" {
		return fmt.Errorf("hardwareData is invalid")
	}
	if ctx.Err() != nil {
		return fmt.Errorf("context error: %w", ctx.Err())
	}

	pgxPool, err := config.GetPGXPool()
	if err != nil {
		return fmt.Errorf("%w: %w", types.DatabaseConnError, err)
	}

	tx, err := pgxPool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return fmt.Errorf("%w: %w", types.DatabaseTransactionError, err)
	}

	defer func() {
		if err != nil {
			_ = tx.Rollback(ctx)
			return
		}
		if commitErr := tx.Commit(ctx); commitErr != nil {
			err = commitErr
		}
	}()

	clientUUID, err := lockClientRowBySystemSerialPGX(ctx, tx, *hardwareData.SystemSerial)
	if err != nil {
		return err
	}

	const hardwareDataTable = `
	INSERT INTO hardware_data
		(
			transaction_uuid,
			time,
			client_uuid,
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
			wifi_mac,
			tpm_version
		) VALUES (
			$1,
			CURRENT_TIMESTAMP,
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
			$19
		) ON CONFLICT (client_uuid)
		 DO UPDATE SET
		 	transaction_uuid = COALESCE(EXCLUDED.transaction_uuid, hardware_data.transaction_uuid),
			time = CURRENT_TIMESTAMP,
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
			wifi_mac = COALESCE(EXCLUDED.wifi_mac, hardware_data.wifi_mac),
			tpm_version = COALESCE(EXCLUDED.tpm_version, hardware_data.tpm_version)
	;`

	hardwareDataResult, err := tx.Exec(ctx, hardwareDataTable,
		ptrToNullString(&hardwareData.TransactionUUID),
		clientUUID,
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
		ptrToNullString(hardwareData.TPMVersion),
	)
	if err != nil {
		return fmt.Errorf("%w: %w", types.DatabaseUpdateError, err)
	}
	if hardwareDataResult.RowsAffected() != 1 {
		return fmt.Errorf("%w: expected exactly 1 row(s), got %d", types.DatabaseAffectedRowsError, hardwareDataResult.RowsAffected())
	}

	const historicalDiskDataInsertSQL = `
	INSERT INTO historical_disk_data (
		time, 
		transaction_uuid, 
		updated_from_windows, 
		client_uuid, 
		disk_model, 
		disk_type, 
		disk_size_kb, 
		disk_serial, 
		disk_firmware_version, 
		disk_reads_kb, 
		disk_writes_kb, 
		disk_power_cycles, 
		disk_power_on_hours, 
		disk_error_count
	) VALUES (
		CURRENT_TIMESTAMP, 
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
		$13
	) ON CONFLICT (transaction_uuid) DO UPDATE SET
	 	time = CURRENT_TIMESTAMP,
		updated_from_windows = COALESCE(EXCLUDED.updated_from_windows, historical_disk_data.updated_from_windows),
		client_uuid = COALESCE(EXCLUDED.client_uuid, historical_disk_data.client_uuid),
		disk_model = COALESCE(EXCLUDED.disk_model, historical_disk_data.disk_model),
		disk_type = COALESCE(EXCLUDED.disk_type, historical_disk_data.disk_type),
		disk_size_kb = COALESCE(EXCLUDED.disk_size_kb, historical_disk_data.disk_size_kb),
		disk_serial = COALESCE(EXCLUDED.disk_serial, historical_disk_data.disk_serial),
		disk_firmware_version = COALESCE(EXCLUDED.disk_firmware_version, historical_disk_data.disk_firmware_version),
		disk_reads_kb = COALESCE(EXCLUDED.disk_reads_kb, historical_disk_data.disk_reads_kb),
		disk_writes_kb = COALESCE(EXCLUDED.disk_writes_kb, historical_disk_data.disk_writes_kb),
		disk_power_cycles = COALESCE(EXCLUDED.disk_power_cycles, historical_disk_data.disk_power_cycles),
		disk_power_on_hours = COALESCE(EXCLUDED.disk_power_on_hours, historical_disk_data.disk_power_on_hours),
		disk_error_count = COALESCE(EXCLUDED.disk_error_count, historical_disk_data.disk_error_count)	
	;`

	historicalDiskDataResult, err := tx.Exec(ctx, historicalDiskDataInsertSQL,
		hardwareData.TransactionUUID,
		false,
		clientUUID,
		ptrToNullString(hardwareData.DiskModel),
		ptrToNullString(hardwareData.DiskType),
		ptrToNullInt64(hardwareData.DiskSize),
		ptrToNullString(hardwareData.DiskSerial),
		ptrToNullString(hardwareData.DiskFirmware),
		ptrToNullInt64(hardwareData.DiskReadsKB),
		ptrToNullInt64(hardwareData.DiskWritesKB),
		ptrToNullInt64(hardwareData.DiskPowerCycles),
		ptrToNullInt64(hardwareData.DiskPowerOnHours),
		ptrToNullInt64(hardwareData.DiskErrors),
	)
	if err != nil {
		return fmt.Errorf("%w: %w", types.DatabaseUpdateError, err)
	}
	if historicalDiskDataResult.RowsAffected() != 1 {
		return fmt.Errorf("%w: expected exactly 1 row(s), got %d", types.DatabaseAffectedRowsError, historicalDiskDataResult.RowsAffected())
	}

	if hardwareData.BatterySerial != nil ||
		hardwareData.BatteryManufacturer != nil ||
		hardwareData.BatteryModel != nil ||
		hardwareData.BatteryChargeCycles != nil ||
		hardwareData.BatteryDesignCapacity != nil ||
		hardwareData.BatteryManufactureDate != nil ||
		hardwareData.BatteryCurrentMaxCapacity != nil {
		const historicalBatteryDataTableInsertSQL = `
		INSERT INTO historical_battery_data (
			time,
			transaction_uuid,
			updated_from_windows,
			client_uuid,
			battery_serial,
			battery_manufacturer,
			battery_model,
			battery_charge_cycles,
			battery_design_capacity,
			battery_manufacture_date,
			battery_current_max_capacity
		) VALUES (
			CURRENT_TIMESTAMP,
			$1,
			$2,
			$3,
			$4,
			$5,
			$6,
			$7,
			$8,
			$9,
			$10
		) ON CONFLICT (transaction_uuid) DO UPDATE SET
			time = CURRENT_TIMESTAMP,
			updated_from_windows = COALESCE(EXCLUDED.updated_from_windows, historical_battery_data.updated_from_windows),
			client_uuid = COALESCE(EXCLUDED.client_uuid, historical_battery_data.client_uuid),
			battery_serial = COALESCE(EXCLUDED.battery_serial, historical_battery_data.battery_serial),
			battery_manufacturer = COALESCE(EXCLUDED.battery_manufacturer, historical_battery_data.battery_manufacturer),
			battery_model = COALESCE(EXCLUDED.battery_model, historical_battery_data.battery_model),
			battery_charge_cycles = COALESCE(EXCLUDED.battery_charge_cycles, historical_battery_data.battery_charge_cycles),
			battery_design_capacity = COALESCE(EXCLUDED.battery_design_capacity, historical_battery_data.battery_design_capacity),
			battery_manufacture_date = COALESCE(EXCLUDED.battery_manufacture_date, historical_battery_data.battery_manufacture_date),
			battery_current_max_capacity = COALESCE(EXCLUDED.battery_current_max_capacity, historical_battery_data.battery_current_max_capacity)
		;`

		batteryHardwareDataSQLResult, err := tx.Exec(ctx, historicalBatteryDataTableInsertSQL,
			hardwareData.TransactionUUID,
			false,
			clientUUID,
			ptrToNullString(hardwareData.BatterySerial),
			ptrToNullString(hardwareData.BatteryManufacturer),
			ptrToNullString(hardwareData.BatteryModel),
			ptrToNullInt64(hardwareData.BatteryChargeCycles),
			ptrToNullFloat64(hardwareData.BatteryDesignCapacity),
			ptrToNullDate(hardwareData.BatteryManufactureDate),
			ptrToNullFloat64(hardwareData.BatteryCurrentMaxCapacity),
		)
		if err != nil {
			return fmt.Errorf("%w: %w", types.DatabaseUpdateError, err)
		}
		if batteryHardwareDataSQLResult.RowsAffected() != 1 {
			return fmt.Errorf("%w: expected exactly 1 row(s), got %d", types.DatabaseAffectedRowsError, batteryHardwareDataSQLResult.RowsAffected())
		}
	}

	const historicalHardwareDataTable = `INSERT INTO historical_hardware_data 
		(
			time, 
			transaction_uuid, 
			client_uuid, 
			memory_serial, 
			memory_capacity_kb, 
			memory_speed_mhz 
		) VALUES (
			CURRENT_TIMESTAMP,
			$1,
			$2,
			$3::TEXT[],
			$4,
			$5
		) ON CONFLICT (transaction_uuid) DO UPDATE SET
			time = CURRENT_TIMESTAMP,
			client_uuid = COALESCE(EXCLUDED.client_uuid, historical_hardware_data.client_uuid),
			memory_serial = COALESCE(EXCLUDED.memory_serial, historical_hardware_data.memory_serial),
			memory_capacity_kb = COALESCE(EXCLUDED.memory_capacity_kb, historical_hardware_data.memory_capacity_kb),
			memory_speed_mhz = COALESCE(EXCLUDED.memory_speed_mhz, historical_hardware_data.memory_speed_mhz)
	;`

	var memorySerialArray []string
	if hardwareData.MemorySerial != nil {
		memorySerialArray = append(memorySerialArray, hardwareData.MemorySerial...)
	}
	if len(memorySerialArray) == 0 {
		memorySerialArray = nil
	}
	hardwareHistoryResult, err := tx.Exec(ctx, historicalHardwareDataTable,
		toNullString(hardwareData.TransactionUUID),
		toNullUUID(clientUUID),
		memorySerialArray,
		ptrToNullInt64(hardwareData.MemoryCapacityKB),
		ptrToNullInt64(hardwareData.MemorySpeedMHz),
	)
	if err != nil {
		return err
	}
	if hardwareHistoryResult.RowsAffected() != 1 {
		return fmt.Errorf("%w: expected exactly 1 row(s), got %d", types.DatabaseAffectedRowsError, hardwareHistoryResult.RowsAffected())
	}

	const clientFirmwareInsertSQL = `
		INSERT INTO 
			historical_firmware_data (
				time, 
				transaction_uuid,
				client_uuid, 
				bios_version,
				bios_firmware,
				bios_release_date
			) 
		VALUES (
			CURRENT_TIMESTAMP,
			$1,
			$2,
			$3,
			$4,
			$5
		) ON CONFLICT (transaction_uuid) DO UPDATE SET
		 	time = CURRENT_TIMESTAMP,
			client_uuid = COALESCE(EXCLUDED.client_uuid, historical_firmware_data.client_uuid),
			bios_version = COALESCE(EXCLUDED.bios_version, historical_firmware_data.bios_version),
			bios_firmware = COALESCE(EXCLUDED.bios_firmware, historical_firmware_data.bios_firmware),
			bios_release_date = COALESCE(EXCLUDED.bios_release_date, historical_firmware_data.bios_release_date)
	;`

	firmwareSQLResult, err := tx.Exec(ctx, clientFirmwareInsertSQL,
		toNullString(hardwareData.TransactionUUID),
		clientUUID,
		ptrToNullString(hardwareData.BiosVersion),
		ptrToNullString(hardwareData.BiosFirmware),
		ptrToNullTime(hardwareData.BiosReleaseDate),
	)
	if err != nil {
		return fmt.Errorf("%w: %w", types.DatabaseUpdateError, err)
	}
	if firmwareSQLResult.RowsAffected() != 1 {
		return fmt.Errorf("%w: expected exactly 1 row(s), got %d", types.DatabaseAffectedRowsError, firmwareSQLResult.RowsAffected())
	}
	return nil
}

func (updateRepo *UpdateRepo) UpdateJobQueuedAt(ctx context.Context, jobQueue *types.JobQueueTableRowView) (err error) {
	if jobQueue == nil {
		return fmt.Errorf("%w: %s", types.InvalidStructureError, "jobQueue is nil")
	}
	if err := types.IsTagnumberInt64Valid(jobQueue.Tagnumber); err != nil {
		return fmt.Errorf("%w: %s (%w)", types.InvalidFieldError, "tagnumber", err)
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
		return fmt.Errorf("%w: %w", types.DatabaseUpdateError, err)
	}
	if err := VerifyRowsAffected(res, 1); err != nil {
		return fmt.Errorf("%w: %w", types.DatabaseAffectedRowsError, err)
	}

	return nil
}

func UpdateClientLastHeard(ctx context.Context, tag int64, lastHeard *time.Time) (err error) {
	if err := types.IsTagnumberInt64Valid(&tag); err != nil {
		return fmt.Errorf("%w: %s (%w)", types.InvalidFieldError, "tagnumber", err)
	}
	if lastHeard == nil || lastHeard.IsZero() {
		return fmt.Errorf("%w: %s", types.InvalidFieldError, "lastHeard")
	}
	if ctx.Err() != nil {
		return fmt.Errorf("%w: %w", types.ContextError, ctx.Err())
	}

	pgxPool, err := config.GetPGXPool()
	if err != nil {
		return fmt.Errorf("%w: %w", types.DatabaseConnError, err)
	}
	tx, err := pgxPool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return fmt.Errorf("%w: %w", types.DatabaseTransactionError, err)
	}

	defer func() {
		if err != nil {
			_ = tx.Rollback(ctx)
			return
		}
		if commitErr := tx.Commit(ctx); commitErr != nil {
			err = commitErr
		}
	}()

	clientUUID, err := GetClientUUIDByTag(ctx, pgxPool, tag)
	if err != nil {
		return fmt.Errorf("%w: %w", types.DatabaseUpdateError, err)
	}

	const sqlCode = `
		UPDATE 
			live_os_data 
		SET 
			last_heard = COALESCE($2, last_heard) 
		WHERE client_uuid = $1;`

	sqlResult, err := tx.Exec(ctx, sqlCode,
		toNullUUID(clientUUID),
		ptrToNullTime(lastHeard),
	)
	if err != nil {
		return err
	}
	if sqlResult.RowsAffected() != 1 {
		return types.DatabaseAffectedRowsError
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

func UpdateFromWindowsJSON(ctx context.Context, windowsUpdateDTO *types.WindowsUpdateDTO, transactionUUID uuid.UUID) (err error) {
	if transactionUUID == uuid.Nil || strings.TrimSpace(transactionUUID.String()) == "" {
		return fmt.Errorf("%w: %s", types.MissingFieldError, "transaction UUID")
	}
	if windowsUpdateDTO == nil {
		return fmt.Errorf("%w: %s", types.InvalidFieldError, "WindowsUpdateDTO")
	}
	if windowsUpdateDTO.RequestMetadata == nil {
		return fmt.Errorf("%w: %s", types.InvalidFieldError, "RequestMetadata")
	}
	if err := types.IsTagnumberInt64Valid(windowsUpdateDTO.RequestMetadata.Tagnumber); err != nil {
		return fmt.Errorf("%w: %s (%w)", types.InvalidFieldError, "tagnumber", err)
	}
	if windowsUpdateDTO.RequestMetadata.SystemSerial == nil || strings.TrimSpace(*windowsUpdateDTO.RequestMetadata.SystemSerial) == "" {
		return fmt.Errorf("%w: %s", types.MissingFieldError, "SystemSerial")
	}

	if strings.TrimSpace(transactionUUID.String()) == "" {
		return fmt.Errorf("%w: %s", types.MissingFieldError, "TransactionUUID")
	}

	pgxPool, err := config.GetPGXPool()
	if err != nil {
		return fmt.Errorf("%w: %w", types.DatabaseConnError, err)
	}
	tx, err := pgxPool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return fmt.Errorf("%w: %w", types.DatabaseTransactionError, err)
	}

	defer func() {
		if err != nil {
			_ = tx.Rollback(ctx)
			return
		}
		if commitErr := tx.Commit(ctx); commitErr != nil {
			err = commitErr
		}
	}()

	clientUUID, err := lockClientRowBySystemSerialPGX(ctx, tx, *windowsUpdateDTO.RequestMetadata.SystemSerial)
	if err != nil {
		return fmt.Errorf("%s: %w", "error while locking client row by system serial", err)
	}

	// hardware_data upsert
	const hardwareDataSql = `
		INSERT INTO hardware_data (
			time,
			transaction_uuid,
			updated_from_windows,
			client_uuid,
			system_serial,
			system_uuid,
			ethernet_mac,
			wifi_mac,
			system_manufacturer,
			system_model,
			cpu_model,
			cpu_core_count,
			cpu_thread_count,
			tpm_version
		) VALUES (
			CURRENT_TIMESTAMP,
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
			$13
		) ON CONFLICT (client_uuid) DO UPDATE SET
			time = CURRENT_TIMESTAMP,
			transaction_uuid = EXCLUDED.transaction_uuid,
			updated_from_windows = EXCLUDED.updated_from_windows,
			system_serial = COALESCE(EXCLUDED.system_serial, hardware_data.system_serial),
			system_uuid = COALESCE(EXCLUDED.system_uuid, hardware_data.system_uuid),
			ethernet_mac = COALESCE(EXCLUDED.ethernet_mac, hardware_data.ethernet_mac),
			wifi_mac = COALESCE(EXCLUDED.wifi_mac, hardware_data.wifi_mac),
			system_manufacturer = COALESCE(EXCLUDED.system_manufacturer, hardware_data.system_manufacturer),
			system_model = COALESCE(EXCLUDED.system_model, hardware_data.system_model),
			cpu_model = COALESCE(EXCLUDED.cpu_model, hardware_data.cpu_model),
			cpu_core_count = COALESCE(EXCLUDED.cpu_core_count, hardware_data.cpu_core_count),
			cpu_thread_count = COALESCE(EXCLUDED.cpu_thread_count, hardware_data.cpu_thread_count),
			tpm_version = COALESCE(EXCLUDED.tpm_version, hardware_data.tpm_version)
	;`

	hardwareDataResult, err := tx.Exec(ctx, hardwareDataSql,
		toNullUUID(transactionUUID),
		true,
		toNullUUID(clientUUID),
		ptrToNullString(windowsUpdateDTO.RequestMetadata.SystemSerial),
		ptrToNullString(windowsUpdateDTO.SystemUUID),
		ptrToNullString(windowsUpdateDTO.EthernetMACAddr),
		ptrToNullString(windowsUpdateDTO.WifiMACAddr),
		ptrToNullString(windowsUpdateDTO.SystemManufacturer),
		ptrToNullString(windowsUpdateDTO.SystemModel),
		ptrToNullString(windowsUpdateDTO.CPUModel),
		ptrToNullInt64(windowsUpdateDTO.CPUCoreCount),
		ptrToNullInt64(windowsUpdateDTO.CPUThreadCount),
		ptrToNullString(windowsUpdateDTO.TPMVersion),
	)
	if err != nil {
		return fmt.Errorf("%w: %w", types.DatabaseUpdateError, err)
	}
	if hardwareDataResult.RowsAffected() != 1 {
		return fmt.Errorf("%w: expected exactly 1 row(s), got %d", types.DatabaseAffectedRowsError, hardwareDataResult.RowsAffected())
	}

	// client_health upsert
	const clientHealthSql = `
		INSERT INTO client_health (
			time, 
			transaction_uuid, 
			updated_from_windows, 
			client_uuid, 
			battery_health_pcnt, 
			disk_free_space_kb, 
			last_hardware_check 
		) VALUES (
			CURRENT_TIMESTAMP, 
			$1, 
			$2, 
			$3, 
			$4, 
			$5, 
			$6 
		) ON CONFLICT (client_uuid) DO UPDATE SET
				time = CURRENT_TIMESTAMP,
				transaction_uuid = EXCLUDED.transaction_uuid,
				updated_from_windows = EXCLUDED.updated_from_windows, 
				battery_health_pcnt = COALESCE(EXCLUDED.battery_health_pcnt, client_health.battery_health_pcnt),
				disk_free_space_kb = COALESCE(EXCLUDED.disk_free_space_kb, client_health.disk_free_space_kb),
				last_hardware_check = COALESCE(EXCLUDED.last_hardware_check, client_health.last_hardware_check)
	;`

	clientHealthResult, err := tx.Exec(ctx, clientHealthSql,
		toNullUUID(transactionUUID),
		true,
		clientUUID,
		ptrToNullFloat64(windowsUpdateDTO.BatteryHealthPcnt),
		ptrToNullInt64(windowsUpdateDTO.DiskFreeSpaceKB),
		ptrToNullTime(windowsUpdateDTO.RequestMetadata.TimeStamp),
	)
	if err != nil {
		return fmt.Errorf("%w: %w", types.DatabaseUpdateError, err)
	}
	if clientHealthResult.RowsAffected() != 1 {
		return fmt.Errorf("%w: expected exactly 1 row(s), got %d", types.DatabaseAffectedRowsError, clientHealthResult.RowsAffected())
	}

	if windowsUpdateDTO.MemoryCapacityKB != nil &&
		windowsUpdateDTO.MemorySpeedMHz != nil {
		const memoryDataInsertSQL = `
		INSERT INTO historical_hardware_data (
			time,
			transaction_uuid,
			updated_from_windows,
			client_uuid,
			memory_capacity_kb,
			memory_speed_mhz,
			memory_serial
		) VALUES (
			CURRENT_TIMESTAMP,
			$1,
			$2,
			$3,
			$4,
			$5,
			$6::TEXT[]
		)
		;`

		memoryDataResult, err := tx.Exec(ctx, memoryDataInsertSQL,
			toNullUUID(transactionUUID),
			true,
			clientUUID,
			ptrToNullInt64(windowsUpdateDTO.MemoryCapacityKB),
			ptrToNullInt64(windowsUpdateDTO.MemorySpeedMHz),
			windowsUpdateDTO.MemorySerial,
		)
		if err != nil {
			return fmt.Errorf("%w: %w", types.DatabaseUpdateError, err)
		}
		if memoryDataResult.RowsAffected() != 1 {
			return fmt.Errorf("%w: expected exactly 1 row(s), got %d", types.DatabaseAffectedRowsError, memoryDataResult.RowsAffected())
		}
	}

	if windowsUpdateDTO.DiskModel != nil &&
		windowsUpdateDTO.DiskType != nil &&
		windowsUpdateDTO.DiskSizeKB != nil {
		const diskDataInsertSQL = `
		INSERT INTO historical_disk_data (
			time,
			transaction_uuid,
			updated_from_windows,
			client_uuid,
			disk_model,
			disk_type,
			disk_size_kb
		) VALUES (
			CURRENT_TIMESTAMP,
			$1,
			$2,
			$3,
			$4,
			$5,
			$6
		)
		;`

		diskDataResult, err := tx.Exec(ctx, diskDataInsertSQL,
			toNullUUID(transactionUUID),
			true,
			clientUUID,
			ptrToNullString(windowsUpdateDTO.DiskModel),
			ptrToNullString(windowsUpdateDTO.DiskType),
			ptrToNullInt64(windowsUpdateDTO.DiskSizeKB),
		)
		if err != nil {
			return fmt.Errorf("%w: %w", types.DatabaseUpdateError, err)
		}
		if diskDataResult.RowsAffected() != 1 {
			return fmt.Errorf("%w: expected exactly 1 row(s), got %d", types.DatabaseAffectedRowsError, diskDataResult.RowsAffected())
		}
	}

	if windowsUpdateDTO.BatteryChargeCycleCount != nil &&
		windowsUpdateDTO.BatteryDesignCapacity != nil &&
		windowsUpdateDTO.BatteryCurrentMaxCapacity != nil {
		// Battery data insert
		const batteryDataInsertSQL = `
		INSERT INTO historical_battery_data (
			time,
			transaction_uuid,
			updated_from_windows,
			client_uuid,
			battery_serial,
			battery_manufacturer,
			battery_model,
			battery_charge_cycles,
			battery_design_capacity,
			battery_manufacture_date,
			battery_current_max_capacity
		) VALUES (
			CURRENT_TIMESTAMP,
			$1,
			$2,
			$3,
			$4,
			$5,
			$6,
			$7,
			$8,
			$9,
			$10
		)
		;`

		batteryDataInsertResult, batteryDataInsertErr := tx.Exec(ctx, batteryDataInsertSQL,
			toNullUUID(transactionUUID),
			true,
			toNullUUID(clientUUID),
			windowsUpdateDTO.BatterySerial,
			windowsUpdateDTO.BatteryManufacturer,
			windowsUpdateDTO.BatteryModel,
			windowsUpdateDTO.BatteryChargeCycleCount,
			windowsUpdateDTO.BatteryDesignCapacity,
			windowsUpdateDTO.BatteryManufactureDate,
			windowsUpdateDTO.BatteryCurrentMaxCapacity,
		)

		if batteryDataInsertErr != nil {
			return fmt.Errorf("%w: %w", types.DatabaseUpdateError, batteryDataInsertErr)
		}
		if batteryDataInsertResult.RowsAffected() != 1 {
			return fmt.Errorf("%w: expected exactly 1 row(s), got %d", types.DatabaseAffectedRowsError, batteryDataInsertResult.RowsAffected())
		}
	}

	// os_info upsert
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
			is_disk_encrypted,
			admin_users,
			computer_name,
			ad_domain,
			ad_computer_name,
			ad_distinguished_name,
			is_intune_joined,
			secure_boot_enabled,
			updated_from_windows,
			installed_apps
		) VALUES (
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
			$19,
			$20,
			$21,
			$22::TEXT[]
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
			is_disk_encrypted = EXCLUDED.is_disk_encrypted,
			admin_users = EXCLUDED.admin_users,
			computer_name = EXCLUDED.computer_name,
			ad_domain = EXCLUDED.ad_domain,
			ad_computer_name = EXCLUDED.ad_computer_name,
			ad_distinguished_name = EXCLUDED.ad_distinguished_name,
			is_intune_joined = EXCLUDED.is_intune_joined,
			secure_boot_enabled = EXCLUDED.secure_boot_enabled,
			updated_from_windows = EXCLUDED.updated_from_windows,
			installed_apps = EXCLUDED.installed_apps
			;`

	adminUsers := windowsUpdateDTO.AdminUsers
	if len(adminUsers) == 0 {
		adminUsers = nil
	}

	osInfoResult, err := tx.Exec(ctx, osInfoSQLCode,
		toNullUUID(transactionUUID),
		ptrToNullInt64(windowsUpdateDTO.RequestMetadata.Tagnumber),
		ptrToNullString(windowsUpdateDTO.RequestMetadata.SystemSerial),
		ptrToNullTime(windowsUpdateDTO.OSInstalledAt),
		ptrToNullString(windowsUpdateDTO.OSVendor),
		ptrToNullString(windowsUpdateDTO.OSPlatform),
		ptrToNullString(windowsUpdateDTO.OSArchitecture),
		ptrToNullString(windowsUpdateDTO.OSName),
		ptrToNullString(windowsUpdateDTO.OSVersion),
		ptrToNullString(windowsUpdateDTO.WindowsDisplayVersion),
		ptrToNullInt64(windowsUpdateDTO.WindowsBuildNumber),
		ptrToNullInt64(windowsUpdateDTO.WindowsUBR),
		ptrToNullBool(windowsUpdateDTO.IsDiskEncrypted),
		adminUsers,
		ptrToNullString(windowsUpdateDTO.ComputerName),
		ptrToNullString(windowsUpdateDTO.ADDomain),
		ptrToNullString(windowsUpdateDTO.ADComputerName),
		ptrToNullString(windowsUpdateDTO.ADDistinguishedName),
		ptrToNullBool(windowsUpdateDTO.IsIntuneJoined),
		ptrToNullBool(windowsUpdateDTO.SecureBootEnabled),
		ptrToNullBool(windowsUpdateDTO.RequestMetadata.UpdatedFromWindows),
		windowsUpdateDTO.InstalledApps,
	)
	if err != nil {
		return fmt.Errorf("%w: %w", types.DatabaseUpdateError, err)
	}
	if osInfoResult.RowsAffected() != 1 {
		return fmt.Errorf("%w: expected exactly 1 row(s), got %d", types.DatabaseAffectedRowsError, osInfoResult.RowsAffected())
	}

	const clientFirmwareInsertSQL = `
		INSERT INTO 
			historical_firmware_data (
				time, 
				transaction_uuid,
				updated_from_windows,
				client_uuid, 
				bios_version,
				bios_release_date,
				has_2023_ca
			) 
		VALUES (
			CURRENT_TIMESTAMP,
			$1,
			$3,
			(SELECT uuid FROM ids WHERE tagnumber = $2 ORDER BY time DESC LIMIT 1),
			$4,
			$5,
			$6
		) ON CONFLICT (transaction_uuid) DO UPDATE SET
		 	time = CURRENT_TIMESTAMP,
			client_uuid = COALESCE(EXCLUDED.client_uuid, historical_firmware_data.client_uuid),
			updated_from_windows = EXCLUDED.updated_from_windows,
			bios_version = COALESCE(EXCLUDED.bios_version, historical_firmware_data.bios_version),
			bios_release_date = COALESCE(EXCLUDED.bios_release_date, historical_firmware_data.bios_release_date),
			has_2023_ca = COALESCE(EXCLUDED.has_2023_ca, historical_firmware_data.has_2023_ca)
	;`

	firmwareSQLResult, err := tx.Exec(ctx, clientFirmwareInsertSQL,
		toNullString(transactionUUID.String()),
		ptrToNullInt64(windowsUpdateDTO.RequestMetadata.Tagnumber),
		ptrToNullBool(windowsUpdateDTO.RequestMetadata.UpdatedFromWindows),
		ptrToNullString(windowsUpdateDTO.BIOSVersion),
		ptrToNullTime(windowsUpdateDTO.BIOSReleaseDate),
		ptrToNullBool(windowsUpdateDTO.Has2023CA),
	)
	if err != nil {
		return fmt.Errorf("%w: %w", types.DatabaseUpdateError, err)
	}
	if firmwareSQLResult.RowsAffected() != 1 {
		return fmt.Errorf("%w: expected exactly 1 row(s), got %d", types.DatabaseUpdateError, firmwareSQLResult.RowsAffected())
	}
	return nil
}

func InitClient(ctx context.Context, dto *types.ClientInitDTO) (clientUUID *string, err error) {
	if dto == nil {
		return nil, fmt.Errorf("%w: %s", types.InvalidStructureError, "ClientInitDTO")
	}
	dbConn, err := config.GetDatabaseConn()
	if err != nil {
		return nil, fmt.Errorf("%w: %w", types.DatabaseConnError, err)
	}
	tx, err := dbConn.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", types.DatabaseTransactionError, err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		} else {
			err = tx.Commit()
		}
	}()
	const sqlCode = `
	INSERT INTO ids (
		uuid, 
		time, 
		tagnumber, 
		system_serial
	) VALUES (
		uuidv7(), 
		CURRENT_TIMESTAMP, 
		$1, 
		$2
	)
	ON CONFLICT (system_serial) DO NOTHING
	RETURNING uuid;
	`
	var idResult sql.NullString
	err = tx.QueryRowContext(ctx, sqlCode,
		toNullInt64(dto.Tagnumber),
		toNullString(dto.SystemSerial),
	).Scan(&idResult)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("%w: %w", types.DatabaseQueryError, err)
	}
	if !idResult.Valid {
		return nil, fmt.Errorf("%w: no client UUID returned for tag '%d' and serial '%s'", types.DatabaseQueryError, dto.Tagnumber, dto.SystemSerial)
	}
	return &idResult.String, nil
}

func UpsertJobStats(ctx context.Context, JobStatsDTO *types.JobStatsDTO) (err error) {
	if JobStatsDTO == nil {
		return fmt.Errorf("%w: %s", types.InvalidStructureError, "JobStatsDTO")
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

	clientUUID, err := lockClientRowByTagnumber(ctx, tx, JobStatsDTO.Tagnumber)
	if err != nil {
		return fmt.Errorf("%w: %w", types.DatabaseUpdateError, err)
	}
	const sqlCode = `
	INSERT INTO jobstats (
		uuid,
		client_uuid,
		tagnumber,
		system_serial,
		time,
		disk_name,
		job_cancelled,
		erase_completed,
		erase_mode,
		erase_diskpercent,
		erase_time,
		clone_completed,
		clone_master,
		clone_image,
		clone_time
	) VALUES (
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
		$15
	) ON CONFLICT (uuid) DO UPDATE SET
		client_uuid = EXCLUDED.client_uuid,
		tagnumber = COALESCE(EXCLUDED.tagnumber, jobstats.tagnumber),
		system_serial = COALESCE(EXCLUDED.system_serial, jobstats.system_serial),
		time = COALESCE(EXCLUDED.time, jobstats.time),
		disk_name = COALESCE(EXCLUDED.disk_name, jobstats.disk_name),
		job_cancelled = COALESCE(EXCLUDED.job_cancelled, jobstats.job_cancelled, FALSE),
		erase_completed = COALESCE(EXCLUDED.erase_completed, jobstats.erase_completed, FALSE),
		erase_mode = COALESCE(EXCLUDED.erase_mode, jobstats.erase_mode),
		erase_diskpercent = COALESCE(EXCLUDED.erase_diskpercent, jobstats.erase_diskpercent),
		erase_time = COALESCE(EXCLUDED.erase_time, jobstats.erase_time),
		clone_completed = COALESCE(EXCLUDED.clone_completed, jobstats.clone_completed, FALSE),
		clone_master = COALESCE(EXCLUDED.clone_master, jobstats.clone_master, FALSE),
		clone_image = COALESCE(EXCLUDED.clone_image, jobstats.clone_image),
		clone_time = COALESCE(EXCLUDED.clone_time, jobstats.clone_time)
	;`

	sqlResult, err := tx.ExecContext(ctx, sqlCode,
		toNullString(JobStatsDTO.TransactionUUID),
		toNullUUID(clientUUID),
		toNullInt64(JobStatsDTO.Tagnumber),
		toNullString(JobStatsDTO.SystemSerial),
		toNullTime(JobStatsDTO.JobStartTime),
		toNullString(JobStatsDTO.DiskName),
		ptrToNullBool(JobStatsDTO.JobCancelled),
		ptrToNullBool(JobStatsDTO.EraseCompleted),
		toNullString(JobStatsDTO.EraseMode),
		toNullInt64(JobStatsDTO.EraseDiskPcnt),
		toNullInt64(JobStatsDTO.EraseDuration),
		ptrToNullBool(JobStatsDTO.CloneCompleted),
		ptrToNullBool(&JobStatsDTO.CloneMaster),
		toNullString(JobStatsDTO.CloneImageName),
		toNullInt64(JobStatsDTO.CloneDuration),
	)
	if err != nil {
		return fmt.Errorf("%w: %w", types.DatabaseUpdateError, err)
	}
	if err := VerifyRowsAffected(sqlResult, 1); err != nil {
		return err
	}

	if JobStatsDTO.CloneMaster {
		const cloneMasterUpdateSQL = `
		WITH newest_image AS (
			SELECT clone_image FROM jobstats WHERE clone_master = TRUE AND client_uuid = $1 ORDER BY time DESC LIMIT 1
		)
		INSERT INTO static_image_names (
			image_name,
			last_updated
		)
		SELECT 
			newest_image.clone_image,
			CURRENT_TIMESTAMP
		FROM newest_image
		ON CONFLICT (image_name) DO UPDATE SET
			last_updated = CURRENT_TIMESTAMP
		;`

		_, err = tx.ExecContext(ctx, cloneMasterUpdateSQL, toNullUUID(clientUUID))
		if err != nil {
			return fmt.Errorf("%w: %w", types.DatabaseUpdateError, err)
		}

		if err != nil {
			return fmt.Errorf("%w: %w", types.DatabaseUpdateError, err)
		}
		if err := VerifyRowsAffected(sqlResult, 1); err != nil {
			return err
		}
	}

	return nil
}

func DeleteOSInfoByTagnumber(ctx context.Context, tagnumber int64, serial string) (err error) {
	if err := types.IsTagnumberInt64Valid(&tagnumber); err != nil {
		return fmt.Errorf("%w: %s (%w)", types.InvalidFieldError, "tagnumber", err)
	}

	pgxPool, err := config.GetPGXPool()
	if err != nil {
		return fmt.Errorf("%w: %w", types.DatabaseConnError, err)
	}

	var clientUUID uuid.UUID

	clientUUIDFromTag, err := GetClientUUIDByTag(ctx, pgxPool, tagnumber)
	if err != nil {
		return fmt.Errorf("%w: %w", types.DatabaseQueryError, err)
	}

	clientUUIDFromSerial, err := GetClientUUIDBySerial(ctx, pgxPool, serial)
	if err != nil {
		return fmt.Errorf("%w: %w", types.DatabaseQueryError, err)
	}

	if clientUUIDFromTag != uuid.Nil && clientUUIDFromSerial != uuid.Nil {
		if clientUUIDFromTag != clientUUIDFromSerial {
			return fmt.Errorf("%w: %s (%d) and %s (%s) do not match", types.InvalidFieldError, "tagnumber", tagnumber, "serial", serial)
		}
		clientUUID = clientUUIDFromTag
	}

	if clientUUID == uuid.Nil {
		return fmt.Errorf("%w: %s (%d) and %s (%s) do not match any client", types.InvalidFieldError, "tagnumber", tagnumber, "serial", serial)
	}

	tx, err := pgxPool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return fmt.Errorf("%w: %w", types.DatabaseTransactionError, err)
	}

	defer func() {
		if err != nil {
			_ = tx.Rollback(ctx)
			return
		}
		if commitErr := tx.Commit(ctx); commitErr != nil {
			err = commitErr
		}
	}()

	const sqlCode = `
	DELETE FROM os_info
	WHERE client_uuid = $1;
	`

	sqlResult, err := tx.Exec(ctx, sqlCode, clientUUID)
	if err != nil {
		return fmt.Errorf("%w: %w", types.DatabaseDeletionError, err)
	}

	if sqlResult.RowsAffected() != 1 {
		if sqlResult.RowsAffected() == 0 {
			// return fmt.Errorf("%w: no rows found for client_uuid '%s' in table os_info", types.DatabaseDeletionError, clientUUID)
			return nil
		}
		return fmt.Errorf("%w: expected exactly 1 row(s), got %d", types.DatabaseAffectedRowsError, sqlResult.RowsAffected())
	}
	return nil
}
