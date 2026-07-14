package types

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

type UpdateJobStatsRequest struct {
	TransactionUUID string     `json:"transaction_uuid"`
	Tagnumber       *int64     `json:"tagnumber"`
	SystemSerial    *string    `json:"system_serial"`
	JobStartTime    *time.Time `json:"job_start_time"`
	DiskName        *string    `json:"disk_name"`
	JobCancelled    *bool      `json:"job_cancelled"`
	EraseCompleted  *bool      `json:"erase_completed"`
	EraseMode       *string    `json:"erase_mode"`
	EraseDiskPcnt   *int64     `json:"erase_disk_pcnt"`
	EraseDuration   *int64     `json:"erase_job_duration"`
	CloneCompleted  *bool      `json:"clone_completed"`
	CloneMaster     *string    `json:"clone_master"`
	CloneImageName  *string    `json:"clone_image_name"`
	CloneDuration   *int64     `json:"clone_job_duration"`
}

type JobStatsDTO struct {
	TransactionUUID string
	ClientUUID      string
	Tagnumber       int64
	SystemSerial    string
	JobStartTime    time.Time
	DiskName        string
	JobCancelled    *bool
	EraseCompleted  *bool
	EraseMode       string
	EraseDiskPcnt   int64
	EraseDuration   int64
	CloneCompleted  *bool
	CloneMaster     bool
	CloneImageName  string
	CloneDuration   int64
}

func (req *UpdateJobStatsRequest) ToDTO() (*JobStatsDTO, error) {
	dto := new(JobStatsDTO)
	if req == nil {
		return nil, fmt.Errorf("job stats request is nil")
	}

	// transaction UUID
	if strings.TrimSpace(req.TransactionUUID) == "" {
		return nil, fmt.Errorf("transaction UUID is required")
	}
	dto.TransactionUUID = req.TransactionUUID

	// tag number
	if err := IsTagnumberInt64Valid(req.Tagnumber); err != nil {
		return nil, fmt.Errorf("invalid tagnumber: %w", err)
	}
	dto.Tagnumber = *req.Tagnumber

	// system serial
	if req.SystemSerial == nil || strings.TrimSpace(*req.SystemSerial) == "" {
		return nil, fmt.Errorf("system serial is required")
	}
	dto.SystemSerial = *req.SystemSerial

	// job start time
	if req.JobStartTime != nil {
		if req.JobStartTime.IsZero() {
			return nil, fmt.Errorf("job start time is required")
		}
		dto.JobStartTime = *req.JobStartTime
	} else {
		// If job start time is not provided, use current time as default
		parsedTime := time.Now().UTC()
		dto.JobStartTime = parsedTime
	}

	// disk name
	if req.DiskName != nil {
		if strings.TrimSpace(*req.DiskName) == "" {
			return nil, fmt.Errorf("disk name is required")
		}
		dto.DiskName = *req.DiskName
	}

	// job cancelled
	if req.JobCancelled != nil {
		dto.JobCancelled = copyBoolPtr(req.JobCancelled)
	}

	// erase completed
	if req.EraseCompleted != nil {
		dto.EraseCompleted = copyBoolPtr(req.EraseCompleted)
	}

	// erase mode
	if req.EraseMode != nil {
		if strings.TrimSpace(*req.EraseMode) == "" {
			return nil, fmt.Errorf("erase mode is required")
		}
		dto.EraseMode = *req.EraseMode
	}

	// erase disk percentage
	if req.EraseDiskPcnt != nil {
		dto.EraseDiskPcnt = *req.EraseDiskPcnt
	}

	// erase duration
	if req.EraseDuration != nil {
		dto.EraseDuration = *req.EraseDuration
	}

	// clone completed
	if req.CloneCompleted != nil {
		dto.CloneCompleted = copyBoolPtr(req.CloneCompleted)
	}

	// clone master
	if req.CloneMaster != nil {
		if strings.TrimSpace(*req.CloneMaster) == "" {
			return nil, fmt.Errorf("clone master is required")
		}
		isCloneMaster, err := strconv.ParseBool(*req.CloneMaster)
		if err != nil {
			return nil, fmt.Errorf("invalid clone master value: %w", err)
		}
		dto.CloneMaster = isCloneMaster
	}

	// clone image name
	if req.CloneImageName != nil {
		if strings.TrimSpace(*req.CloneImageName) == "" {
			return nil, fmt.Errorf("clone image name is required")
		}
		dto.CloneImageName = *req.CloneImageName
	}

	// clone duration
	if req.CloneDuration != nil {
		dto.CloneDuration = *req.CloneDuration
	}
	return dto, nil
}
