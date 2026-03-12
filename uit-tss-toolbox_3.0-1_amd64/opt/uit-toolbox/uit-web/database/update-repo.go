package database

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
)

type Update interface {
	InsertNewNote(ctx context.Context, time *time.Time, noteType *string, note *string) (err error)
	InsertInventoryUpdate(ctx context.Context, transactionUUID uuid.UUID, inventoryUpdate *types.InventoryLocationWriteModel) (err error)
	UpdateClientHealthUpdate(ctx context.Context, transactionUUID uuid.UUID, clientHealthData *types.InventoryClientHealthWriteModel) (err error)
	InsertClientCheckoutsUpdate(ctx context.Context, transactionUUID uuid.UUID, checkoutData *types.InventoryCheckoutWriteModel) (err error)
	UpdateInventoryHardwareData(ctx context.Context, transactionUUID uuid.UUID, hardwareData *types.InventoryHardwareWriteModel) (err error)
	UpdateClientImages(ctx context.Context, transactionUUID uuid.UUID, manifest *types.ImageManifest) (err error)
	HideClientImageByUUID(ctx context.Context, tagnumber *int64, uuid *string) (err error)
	DeleteClientImageByUUID(ctx context.Context, tagnumber *int64, uuid *string) (err error)
	TogglePinImage(ctx context.Context, tagnumber *int64, uuid *string) (err error)
	SetAllOnlineClientJobs(ctx context.Context, allJobs *types.AllJobs) (err error)
	SetClientJob(ctx context.Context, tag *int64, clientJob *string) (err error)
	UpdateClientMemoryInfo(ctx context.Context, memInfo *types.MemoryDataRequest) (err error)
	UpdateClientCPUUsage(ctx context.Context, cpuData *types.CPUData) (err error)
	UpdateClientCPUTemperature(ctx context.Context, cpuTempData *types.CPUData) (err error)
	UpdateClientNetworkUsage(ctx context.Context, networkData *types.NetworkData) (err error)
	UpdateClientUptime(ctx context.Context, uptimeData *types.ClientUptime) (err error)
	UpdateClientLastHardwareCheck(ctx context.Context, tagnumber int64, lastCheck time.Time) (err error)
	UpdateClientHardwareData(ctx context.Context, hardwareData *types.ClientHardwareView) (err error)
	UpdateClientCPUMHz(ctx context.Context, cpuData *types.CPUData) (err error)
	UpdateClientHealth(ctx context.Context, clientHealth *types.ClientHealthDTO) (err error)
	UpdateJobQueuedAt(ctx context.Context, jobQueue *types.JobQueueTableRowView) (err error)
	UpdateClientLastHeard(ctx context.Context, tag *int64, lastHeard *time.Time) (err error)
	UpdateClientBatteryChargePcnt(ctx context.Context, tag *int64, percent *float64) (err error)
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
		ptrToNullTime(time),
		ptrToNullString(noteType),
		ptrToNullString(note),
	)
	if err != nil {
		return fmt.Errorf("error inserting new note: %w", err)
	}
	if err := VerifyRowsAffected(sqlResult, 1); err != nil {
		return fmt.Errorf("error while checking rows affected when inserting new note: %w", err)
	}
	return err
}

func (updateRepo *UpdateRepo) UpdateClientHealthUpdate(ctx context.Context, transactionUUID uuid.UUID, clientHealthData *types.InventoryClientHealthWriteModel) (err error) {
	if transactionUUID == uuid.Nil || strings.TrimSpace(transactionUUID.String()) == "" {
		return fmt.Errorf("generated transaction UUID is nil")
	}
	if clientHealthData == nil || clientHealthData.Tagnumber == 0 {
		return fmt.Errorf("clientHealthData is invalid")
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
	// Insert/update client_health table
	const clientHealthSql = `INSERT INTO client_health
		(time, tagnumber, last_hardware_check, transaction_uuid) VALUES
		(CURRENT_TIMESTAMP, $1, $2, $3)
		ON CONFLICT (tagnumber)
		DO UPDATE SET
			time = CURRENT_TIMESTAMP,
			last_hardware_check = EXCLUDED.last_hardware_check,
			transaction_uuid = EXCLUDED.transaction_uuid;`

	var clientHealthResult sql.Result
	clientHealthResult, err = tx.ExecContext(ctx, clientHealthSql,
		clientHealthData.Tagnumber,
		ptrToNullTime(clientHealthData.LastHardwareCheck),
		transactionUUID,
	)
	if err != nil {
		return fmt.Errorf("error inserting/updating client health data: %w", err)
	}
	if err := VerifyRowsAffected(clientHealthResult, 1); err != nil {
		return fmt.Errorf("error while checking rows affected on client_health table insert/update: %w", err)
	}

	return nil
}

func (updateRepo *UpdateRepo) InsertClientCheckoutsUpdate(ctx context.Context, transactionUUID uuid.UUID, checkoutData *types.InventoryCheckoutWriteModel) (err error) {
	if transactionUUID == uuid.Nil || strings.TrimSpace(transactionUUID.String()) == "" {
		return fmt.Errorf("generated transaction UUID is nil")
	}
	if checkoutData == nil || checkoutData.Tagnumber == 0 {
		return fmt.Errorf("checkoutData is invalid")
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
	// Insert into checkout_log table if necessary fields are present
	if checkoutData.CheckoutDate != nil || checkoutData.ReturnDate != nil || (checkoutData.CheckoutBool != nil && *checkoutData.CheckoutBool) {
		var checkoutLogResult sql.Result
		const checkoutSql = `INSERT INTO checkout_log
			(log_entry_time, transaction_uuid, tagnumber, checkout_date, return_date, checkout_bool)
			VALUES (CURRENT_TIMESTAMP, $1, $2, $3, $4, $5);`

		checkoutLogResult, err = tx.ExecContext(ctx, checkoutSql,
			transactionUUID,
			checkoutData.Tagnumber,
			ptrToNullTime(checkoutData.CheckoutDate),
			ptrToNullTime(checkoutData.ReturnDate),
			ptrToNullBool(checkoutData.CheckoutBool),
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

func (updateRepo *UpdateRepo) UpdateInventoryHardwareData(ctx context.Context, transactionUUID uuid.UUID, hardwareData *types.InventoryHardwareWriteModel) (err error) {
	if transactionUUID == uuid.Nil || strings.TrimSpace(transactionUUID.String()) == "" {
		return fmt.Errorf("generated transaction UUID is nil")
	}
	if hardwareData == nil || hardwareData.Tagnumber == 0 {
		return fmt.Errorf("hardwareData is invalid")
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
		hardwareData.Tagnumber,
		ptrToNullString(hardwareData.SystemManufacturer),
		ptrToNullString(hardwareData.SystemModel),
		ptrToNullString(hardwareData.DeviceType),
	)
	if err != nil {
		return fmt.Errorf("error inserting/updating hardware data: %w", err)
	}
	if err := VerifyRowsAffected(hardwareDataResult, 1); err != nil {
		return fmt.Errorf("error while checking rows affected on hardware_data table insert/update: %w", err)
	}

	return nil
}

func (updateRepo *UpdateRepo) InsertInventoryUpdate(ctx context.Context, transactionUUID uuid.UUID, inventoryUpdate *types.InventoryLocationWriteModel) (err error) {
	if transactionUUID == uuid.Nil || strings.TrimSpace(transactionUUID.String()) == "" {
		return fmt.Errorf("generated transaction UUID is nil")
	}
	if inventoryUpdate == nil || inventoryUpdate.Tagnumber == 0 {
		return fmt.Errorf("inventoryUpdate is invalid")
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
		inventoryUpdate.Tagnumber,
		inventoryUpdate.SystemSerial,
		inventoryUpdate.Location,
		ptrToNullString(inventoryUpdate.Building),
		ptrToNullString(inventoryUpdate.Room),
		inventoryUpdate.Department,
		inventoryUpdate.Domain,
		ptrToNullString(inventoryUpdate.PropertyCustodian),
		ptrToNullTime(inventoryUpdate.AcquiredDate),
		ptrToNullTime(inventoryUpdate.RetiredDate),
		ptrToNullBool(inventoryUpdate.Broken),
		ptrToNullBool(inventoryUpdate.DiskRemoved),
		inventoryUpdate.ClientStatus,
		ptrToNullString(inventoryUpdate.Note),
	)
	if err != nil {
		return fmt.Errorf("error inserting location data: %w", err)
	}
	if err := VerifyRowsAffected(locationsResult, 1); err != nil {
		return fmt.Errorf("error while checking rows affected on locations table insert: %w", err)
	}

	return nil
}

func (updateRepo *UpdateRepo) UpdateClientImages(ctx context.Context, transactionUUID uuid.UUID, manifest *types.ImageManifest) (err error) {
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
		pinned)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15);`
	var sqlResult sql.Result
	sqlResult, err = tx.ExecContext(ctx, sqlCode,
		ptrToNullString(manifest.UUID),
		ptrToNullTime(manifest.Time),
		ptrToNullInt64(manifest.Tagnumber),
		ptrToNullString(manifest.FileName),
		ptrToNullString(manifest.FilePath),
		ptrToNullString(manifest.ThumbnailFilePath),
		ptrToNullInt64(manifest.FileSize),
		manifest.SHA256Hash,
		ptrToNullString(manifest.MimeType),
		ptrToNullTime(manifest.ExifTimestamp),
		ptrToNullInt64(manifest.ResolutionX),
		ptrToNullInt64(manifest.ResolutionY),
		ptrToNullString(manifest.Note),
		ptrToNullBool(manifest.Hidden),
		ptrToNullBool(manifest.Pinned),
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
		ptrToNullInt64(tagnumber),
		ptrToNullString(uuid),
	)
	if err != nil {
		return fmt.Errorf("error hiding client image: %w", err)
	}
	if err := VerifyRowsAffected(sqlResult, 1); err != nil {
		return fmt.Errorf("error while checking rows affected on client_images table update: %w", err)
	}
	return nil
}

func (updateRepo *UpdateRepo) DeleteClientImageByUUID(ctx context.Context, tagnumber *int64, uuid *string) (err error) {
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

	const sqlQuery = `DELETE FROM client_images WHERE tagnumber = $1 AND uuid = $2;`
	var sqlResult sql.Result
	sqlResult, err = tx.ExecContext(ctx, sqlQuery,
		ptrToNullInt64(tagnumber),
		ptrToNullString(uuid),
	)
	if err != nil {
		return fmt.Errorf("error deleting client image: %w", err)
	}
	if err := VerifyRowsAffected(sqlResult, 1); err != nil {
		return fmt.Errorf("error while checking rows affected on client_images table delete: %w", err)
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

	const sqlQuery = `UPDATE client_images SET pinned = NOT COALESCE(pinned, FALSE) WHERE uuid = $1 AND tagnumber = $2;`
	var sqlResult sql.Result
	sqlResult, err = tx.ExecContext(ctx, sqlQuery,
		ptrToNullString(uuid),
		ptrToNullInt64(tagnumber),
	)
	if err != nil {
		return fmt.Errorf("error toggling pin on client image: %w", err)
	}
	if err := VerifyRowsAffected(sqlResult, 1); err != nil {
		return fmt.Errorf("error while checking rows affected on client_images table update: %w", err)
	}
	return nil
}

func (updateRepo *UpdateRepo) SetAllOnlineClientJobs(ctx context.Context, allJobs *types.AllJobs) (err error) {
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

	const sqlCode = `UPDATE job_queue SET job_name = $1 WHERE NOW() - last_heard < INTERVAL '30 SECONDS';`
	_, err = tx.ExecContext(ctx, sqlCode, ptrStringToString(allJobs.JobName))
	if err != nil {
		return fmt.Errorf("error while updating job queue: %w", err)
	}
	return nil
}

func (updateRepo *UpdateRepo) SetClientJob(ctx context.Context, tag *int64, clientJob *string) (err error) {
	if tag == nil || clientJob == nil {
		return fmt.Errorf("tag and clientJob are both required")
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

	const sqlCode = `UPDATE job_queue SET job_queued = TRUE, job_name = $1, job_active = FALSE WHERE tagnumber = $2;`
	var sqlResult sql.Result
	sqlResult, err = tx.ExecContext(ctx, sqlCode, ptrStringToString(clientJob), ptrToNullInt64(tag))
	if err != nil {
		return fmt.Errorf("error while updating client job: %w", err)
	}
	if err := VerifyRowsAffected(sqlResult, 1); err != nil {
		return fmt.Errorf("error while checking rows affected on job_queue table update: %w", err)
	}
	return nil
}

func (updateRepo *UpdateRepo) UpdateClientMemoryInfo(ctx context.Context, memInfo *types.MemoryDataRequest) (err error) {
	if memInfo == nil {
		return fmt.Errorf("memory data is required")
	}

	if memInfo.Tagnumber == 0 {
		return fmt.Errorf("tagnumber is required in memory data")
	}
	if memInfo.TotalUsageKB == nil || memInfo.TotalCapacityKB == nil {
		return fmt.Errorf("both total usage and total capacity are required")
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

	const sqlCode = `INSERT INTO job_queue (tagnumber, memory_capacity_kb, memory_usage_kb) VALUES ($1, $2, $3)
		ON CONFLICT (tagnumber) DO UPDATE SET memory_capacity_kb = EXCLUDED.memory_capacity_kb, memory_usage_kb = EXCLUDED.memory_usage_kb;`
	var sqlResult sql.Result
	sqlResult, err = tx.ExecContext(ctx, sqlCode,
		toNullInt64(memInfo.Tagnumber),
		toNullInt64(*memInfo.TotalCapacityKB),
		toNullInt64(*memInfo.TotalUsageKB),
	)
	if err != nil {
		return fmt.Errorf("error updating memory usage: %w", err)
	}
	if err := VerifyRowsAffected(sqlResult, 1); err != nil {
		return fmt.Errorf("error while checking rows affected on job_queue table update: %w", err)
	}
	return nil
}

func (updateRepo *UpdateRepo) UpdateClientCPUUsage(ctx context.Context, cpuData *types.CPUData) (err error) {
	if cpuData == nil {
		return fmt.Errorf("CPU data is required")
	}

	if cpuData.Tagnumber == 0 || cpuData.UsagePercent == nil {
		return fmt.Errorf("both tagnumber and usage percent are required")
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

	const sqlCode = `INSERT INTO job_queue (tagnumber, cpu_usage) VALUES ($1, $2)
		ON CONFLICT (tagnumber) DO UPDATE SET cpu_usage = EXCLUDED.cpu_usage;`
	var sqlResult sql.Result
	sqlResult, err = tx.ExecContext(ctx, sqlCode,
		toNullInt64(cpuData.Tagnumber),
		ptrToNullFloat64(cpuData.UsagePercent),
	)
	if err != nil {
		return fmt.Errorf("error updating CPU usage: %w", err)
	}
	if err := VerifyRowsAffected(sqlResult, 1); err != nil {
		return fmt.Errorf("error while checking rows affected on job_queue table update: %w", err)
	}
	return nil
}

func (updateRepo *UpdateRepo) UpdateClientCPUMHz(ctx context.Context, cpuData *types.CPUData) (err error) {
	if cpuData == nil {
		return fmt.Errorf("CPU data is required")
	}
	if cpuData.Tagnumber == 0 || cpuData.MHz == nil {
		return fmt.Errorf("both tagnumber and CPU MHz are required")
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

	const sqlCode = `INSERT INTO job_queue (tagnumber, cpu_mhz) VALUES ($1, $2)
		ON CONFLICT (tagnumber) DO UPDATE SET cpu_mhz = EXCLUDED.cpu_mhz;`
	var sqlResult sql.Result
	sqlResult, err = tx.ExecContext(ctx, sqlCode,
		toNullInt64(cpuData.Tagnumber),
		ptrToNullFloat64(cpuData.MHz),
	)
	if err != nil {
		return fmt.Errorf("error updating CPU MHz: %w", err)
	}
	if err := VerifyRowsAffected(sqlResult, 1); err != nil {
		return fmt.Errorf("error while checking rows affected on CPU MHz update: %w", err)
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
	const sqlCode = `INSERT INTO job_queue (tagnumber, network_usage, link_speed) VALUES ($1, $2, $3)
		ON CONFLICT (tagnumber) DO UPDATE SET network_usage = EXCLUDED.network_usage, link_speed = EXCLUDED.link_speed;`
	var sqlResult sql.Result
	sqlResult, err = tx.ExecContext(ctx, sqlCode,
		toNullInt64(networkData.Tagnumber),
		ptrToNullInt64(networkData.NetworkUsage),
		ptrToNullInt64(networkData.LinkSpeed),
	)
	if err != nil {
		return fmt.Errorf("error updating network usage: %w", err)
	}
	if err := VerifyRowsAffected(sqlResult, 1); err != nil {
		return fmt.Errorf("error while checking rows affected on job_queue table update: %w", err)
	}
	return nil
}

func (updateRepo *UpdateRepo) UpdateClientCPUTemperature(ctx context.Context, cpuTempData *types.CPUData) (err error) {
	if cpuTempData == nil {
		return fmt.Errorf("CPU data is required")
	}
	if cpuTempData.Tagnumber == 0 || cpuTempData.MillidegreesC == nil {
		return fmt.Errorf("both tagnumber and temperature are required")
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

	degreesC := float64(*cpuTempData.MillidegreesC) / 1000

	const sqlCode = `INSERT INTO job_queue (tagnumber, cpu_temp) VALUES ($1, $2)
		ON CONFLICT (tagnumber) DO UPDATE SET cpu_temp = EXCLUDED.cpu_temp;`
	var sqlResult sql.Result
	sqlResult, err = tx.ExecContext(ctx, sqlCode,
		toNullInt64(cpuTempData.Tagnumber),
		ptrToNullFloat64(&degreesC),
	)
	if err != nil {
		return fmt.Errorf("error updating CPU temperature: %w", err)
	}
	if err := VerifyRowsAffected(sqlResult, 1); err != nil {
		return fmt.Errorf("error while checking rows affected on job_queue table update: %w", err)
	}
	return nil
}

func (updateRepo *UpdateRepo) UpdateClientUptime(ctx context.Context, uptimeData *types.ClientUptime) (err error) {
	if uptimeData == nil {
		return fmt.Errorf("uptime data is required")
	}
	if uptimeData.Tagnumber == 0 || uptimeData.ClientAppUptime == 0 || uptimeData.SystemUptime == 0 {
		return fmt.Errorf("tagnumber, client app uptime, and system uptime are required")
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

	const sqlCode = `INSERT INTO job_queue (tagnumber, client_app_uptime, system_uptime) VALUES ($1, $2, $3)
		ON CONFLICT (tagnumber) DO UPDATE SET client_app_uptime = EXCLUDED.client_app_uptime, system_uptime = EXCLUDED.system_uptime;`
	var sqlResult sql.Result
	sqlResult, err = tx.ExecContext(ctx, sqlCode,
		toNullInt64(uptimeData.Tagnumber),
		toNullDuration(uptimeData.ClientAppUptime),
		toNullDuration(uptimeData.SystemUptime),
	)
	if err != nil {
		return fmt.Errorf("error updating client uptime: %w", err)
	}
	if err := VerifyRowsAffected(sqlResult, 1); err != nil {
		return fmt.Errorf("error while checking rows affected on job_queue table update: %w", err)
	}
	return nil
}

func (updateRepo *UpdateRepo) UpdateClientLastHardwareCheck(ctx context.Context, tagnumber int64, lastCheck time.Time) (err error) {
	if tagnumber == 0 {
		return fmt.Errorf("tagnumber is required")
	}
	if lastCheck.IsZero() {
		return fmt.Errorf("last hardware check time is required")
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
	const sqlCode = `INSERT INTO client_health (tagnumber, last_hardware_check) VALUES ($1, $2)
		ON CONFLICT (tagnumber) DO UPDATE SET last_hardware_check = EXCLUDED.last_hardware_check;`
	var sqlResult sql.Result
	sqlResult, err = tx.ExecContext(ctx, sqlCode,
		ptrToNullInt64(&tagnumber),
		ptrToNullTime(&lastCheck),
	)
	if err != nil {
		return fmt.Errorf("error updating last hardware check time: %w", err)
	}
	if err := VerifyRowsAffected(sqlResult, 1); err != nil {
		return fmt.Errorf("error while checking rows affected on client_health table update: %w", err)
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
		return fmt.Errorf("error inserting/updating hardware data: %w", err)
	}
	if err := VerifyRowsAffected(hardwareResult, 1); err != nil {
		return fmt.Errorf("error while checking rows affected on hardware_data table insert/update: %w", err)
	}

	const historicalHardwareDataTable = `INSERT INTO historical_hardware_data 
		(
			transaction_uuid,
			time,
			tagnumber, 
			system_serial, 
			ethernet_mac, 
			wifi_mac, 
			disk_model,
			disk_type,
			disk_size,
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
			tagnumber =  COALESCE(EXCLUDED.tagnumber, historical_hardware_data.tagnumber),
			system_serial = COALESCE(EXCLUDED.system_serial, historical_hardware_data.system_serial),
			ethernet_mac = COALESCE(EXCLUDED.ethernet_mac, historical_hardware_data.ethernet_mac),
			wifi_mac =  COALESCE(EXCLUDED.wifi_mac, historical_hardware_data.wifi_mac),
			disk_model = COALESCE(EXCLUDED.disk_model, historical_hardware_data.disk_model),
			disk_type = COALESCE(EXCLUDED.disk_type, historical_hardware_data.disk_type),
			disk_size = COALESCE(EXCLUDED.disk_size, historical_hardware_data.disk_size),
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
			;
		`

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
		return fmt.Errorf("error inserting/updating historical hardware data: %w", err)
	}
	if err := VerifyRowsAffected(hardwareHistoryResult, 1); err != nil {
		return fmt.Errorf("error while checking rows affected on historical hardware data table update: %w", err)
	}
	return nil
}

func (updateRepo *UpdateRepo) UpdateClientHealth(ctx context.Context, clientHealth *types.ClientHealthDTO) (err error) {
	if clientHealth == nil || clientHealth.Tagnumber == 0 || strings.TrimSpace(clientHealth.TransactionUUID) == "" {
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

	const sqlCode = `
	WITH 
	total_job_count AS (
		SELECT COUNT(*) AS "job_count" FROM jobstats WHERE (erase_completed = TRUE OR clone_completed = TRUE) AND tagnumber = $2
	),
	avg_erase_times AS (
		SELECT AVG(erase_time) AS "erase_time" FROM jobstats WHERE tagnumber = $2
	),
	avg_clone_times AS (
		SELECT AVG(clone_time) AS "clone_time" FROM jobstats WHERE tagnumber = $2
	),
	most_recent_job AS (
		SELECT tagnumber, erase_completed, clone_completed, clone_image FROM jobstats WHERE time IN (SELECT MAX(time) FROM jobstats WHERE (erase_completed = TRUE OR clone_completed = TRUE) AND tagnumber = $2)
	),
	most_recent_erase_job AS (
		SELECT tagnumber, time FROM jobstats WHERE erase_completed = TRUE AND tagnumber = $2 ORDER BY time DESC NULLS LAST LIMIT 1
	),
	most_recent_clone_job AS (
		SELECT tagnumber, time FROM jobstats WHERE clone_completed = TRUE AND tagnumber = $2 ORDER BY time DESC NULLS LAST LIMIT 1
	)
	INSERT INTO client_health (
		time,
		tagnumber,
		transaction_uuid,
		system_serial,
		tpm_version,
		bios_updated,
		os_installed,
		os_name,
		disk_health_pcnt,
		battery_health_pcnt,
		avg_erase_time,
		avg_clone_time,
		last_erase_job_time,
		last_clone_job_time,
		total_jobs_completed,
		last_hardware_check
	)
	SELECT
		$1 AS "time",
		$2 AS "tagnumber",
		$3 AS "transaction_uuid",
		$4 AS "system_serial",
		$5 AS "tpm_version",
		CASE 
			WHEN $6 = static_bios_stats.bios_version THEN TRUE
			ELSE FALSE
		END AS "bios_updated",
		CASE 
			WHEN $7 = TRUE THEN TRUE
			WHEN $7 = FALSE THEN FALSE
			WHEN $7 IS NULL AND most_recent_job.clone_completed = TRUE THEN TRUE
			ELSE FALSE
		END AS "os_installed",
		static_image_names.image_name_readable AS "os_name",
		(100 - ((historical_hardware_data.disk_power_on_hours::decimal / static_disk_stats.disk_mtbf::decimal) + (historical_hardware_data.disk_writes_kb::decimal / 100000000 / static_disk_stats.disk_tbw::decimal)) / 2) AS "disk_health_pcnt",
		ROUND((historical_hardware_data.battery_current_max_capacity::decimal / historical_hardware_data.battery_design_capacity::decimal * 100), 2) AS "battery_health_pcnt",
		avg_erase_times.erase_time AS "avg_erase_time",
		avg_clone_times.clone_time AS "avg_clone_time",
		COALESCE($8, most_recent_erase_job.time) AS "last_erase_job_time",
		COALESCE($9, most_recent_clone_job.time) AS "last_clone_job_time",
		total_job_count.job_count AS "total_jobs_completed",
		$10 AS "last_hardware_check"
	FROM 
		hardware_data
	LEFT JOIN 
		static_bios_stats ON hardware_data.system_model = static_bios_stats.system_model
	LEFT JOIN 
		historical_hardware_data ON hardware_data.tagnumber = historical_hardware_data.tagnumber AND historical_hardware_data.time IN (SELECT MAX(time) FROM historical_hardware_data WHERE tagnumber = $2)
	LEFT JOIN 
		static_disk_stats ON historical_hardware_data.disk_model = static_disk_stats.disk_model
	LEFT JOIN 
		most_recent_job ON hardware_data.tagnumber = most_recent_job.tagnumber
	LEFT JOIN
		static_image_names ON most_recent_job.clone_image = static_image_names.image_name
	CROSS JOIN
		avg_erase_times
	CROSS JOIN
		avg_clone_times
	CROSS JOIN
		most_recent_erase_job
	CROSS JOIN
		most_recent_clone_job
	CROSS JOIN
		total_job_count
	WHERE hardware_data.tagnumber = $2
	ON CONFLICT (tagnumber)
	 DO UPDATE SET
		time = COALESCE(EXCLUDED.time, client_health.time),
		transaction_uuid = COALESCE(EXCLUDED.transaction_uuid, client_health.transaction_uuid),
		system_serial = COALESCE(EXCLUDED.system_serial, client_health.system_serial),
		tpm_version = COALESCE(EXCLUDED.tpm_version, client_health.tpm_version),
		bios_updated = COALESCE(EXCLUDED.bios_updated, client_health.bios_updated),
		os_installed = COALESCE(EXCLUDED.os_installed, client_health.os_installed),
		os_name = COALESCE(EXCLUDED.os_name, client_health.os_name),
		disk_health_pcnt = COALESCE(EXCLUDED.disk_health_pcnt, client_health.disk_health_pcnt),
		battery_health_pcnt = COALESCE(EXCLUDED.battery_health_pcnt, client_health.battery_health_pcnt),
		avg_erase_time = COALESCE(EXCLUDED.avg_erase_time, client_health.avg_erase_time),
		avg_clone_time = COALESCE(EXCLUDED.avg_clone_time, client_health.avg_clone_time),
		last_erase_job_time = COALESCE(EXCLUDED.last_erase_job_time, client_health.last_erase_job_time),
		last_clone_job_time = COALESCE(EXCLUDED.last_clone_job_time, client_health.last_clone_job_time),
		total_jobs_completed = COALESCE(EXCLUDED.total_jobs_completed, client_health.total_jobs_completed),
		last_hardware_check = COALESCE(EXCLUDED.last_hardware_check, client_health.last_hardware_check)
		;
	`
	var clientHealthResult sql.Result
	clientHealthResult, err = tx.ExecContext(ctx, sqlCode,
		ptrToNullTime(clientHealth.Time),
		clientHealth.Tagnumber,
		clientHealth.TransactionUUID,
		ptrToNullString(clientHealth.SystemSerial),
		ptrToNullString(clientHealth.TPMVersion),
		ptrToNullBool(clientHealth.BIOSUpdated),
		ptrToNullBool(clientHealth.OSInstalled),
		ptrToNullTime(clientHealth.LastEraseJobTime),
		ptrToNullTime(clientHealth.LastCloneJobTime),
		ptrToNullTime(clientHealth.LastHardwareCheck),
	)
	if err != nil {
		return fmt.Errorf("error inserting/updating client health data: %w", err)
	}
	if err := VerifyRowsAffected(clientHealthResult, 1); err != nil {
		return fmt.Errorf("error while checking rows affected on client_health table insert/update: %w", err)
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
			tagnumber = $1
	;`

	var res sql.Result
	res, err = tx.ExecContext(ctx, sqlCode,
		ptrToNullInt64(jobQueue.Tagnumber),
		ptrToNullTime(jobQueue.JobQueuedAt),
	)
	if err != nil {
		return fmt.Errorf("error updating job_queued_at: %w", err)
	}
	if err := VerifyRowsAffected(res, 1); err != nil {
		return fmt.Errorf("rows affected are out of range: %w", err)
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

	const sqlCode = `UPDATE job_queue SET last_heard = COALESCE($2, CURRENT_TIMESTAMP) WHERE tagnumber = $1;`
	var sqlResult sql.Result
	sqlResult, err = tx.ExecContext(ctx, sqlCode,
		ptrToNullInt64(tag),
		ptrToNullTime(lastHeard),
	)
	if err != nil {
		return fmt.Errorf("error updating client's last heard time: %w", err)
	}
	if err := VerifyRowsAffected(sqlResult, 1); err != nil {
		return fmt.Errorf("error while checking rows affected on job_queue table update: %w", err)
	}
	return nil
}

func (updateRepo *UpdateRepo) UpdateClientBatteryChargePcnt(ctx context.Context, tag *int64, percent *float64) (err error) {
	if tag == nil || *tag == 0 {
		return fmt.Errorf("tagnumber is required")
	}
	if percent == nil || *percent < 0 || *percent > 100 {
		return fmt.Errorf("percent must be between 0 and 100")
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
	const sqlCode = `UPDATE job_queue SET battery_charge_pcnt = $2 WHERE tagnumber = $1;`
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
