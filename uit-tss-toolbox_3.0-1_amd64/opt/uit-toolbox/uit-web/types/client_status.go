package types

import (
	"fmt"
	"strings"
	"time"
)

type ClientStatusView struct {
	Tagnumber         *int64     `json:"tagnumber"`
	ClientStatus      *string    `json:"client_status"`
	BatteryHealthPcnt *string    `json:"battery_health_pcnt"`
	LastHardwareCheck *time.Time `json:"last_hardware_check"`
}

type ClientUptime struct {
	Tagnumber       int64         `json:"tagnumber"`
	ClientAppUptime time.Duration `json:"client_app_uptime"`
	SystemUptime    time.Duration `json:"system_uptime"`
}

type ClientStatus struct {
	Status          string `json:"status"`
	StatusFormatted string `json:"status_formatted"`
	SortOrder       int64  `json:"status_sort_order"`
}

type ActiveJobs struct {
	Tagnumber     int64  `json:"tagnumber"`
	JobQueued     *bool  `json:"job_queued"`
	JobName       string `json:"job_name"`
	JobActive     *bool  `json:"job_active"`
	QueuePosition int64  `json:"job_queue_position"`
}

type AvailableJobs struct {
	Tagnumber    int64 `json:"tagnumber"`
	JobAvailable *bool `json:"job_available"`
}

type ClientBatteryHealth struct {
	Time                time.Time `json:"time"`
	Tagnumber           int64     `json:"tagnumber"`
	JobstatsBattery     string    `json:"jobstatsHealthPcnt"`
	ClientHealthBattery string    `json:"clientHealthPcnt"`
	BatteryChargeCycles int64     `json:"chargeCycles"`
}

type ClientHealthUpdateRequest struct {
	Tagnumber         int64      `json:"tagnumber"`
	TransactionUUID   string     `json:"transaction_uuid"`
	SystemSerial      *string    `json:"system_serial"`
	TPMVersion        *string    `json:"tpm_version"`
	BIOSVersion       *string    `json:"bios_version"`
	EraseCompleted    *bool      `json:"erase_completed"`
	EraseJobDuration  *float64   `json:"erase_job_duration"`
	CloneCompleted    *bool      `json:"clone_completed"`
	CloneJobDuration  *float64   `json:"clone_job_duration"`
	LastHardwareCheck *time.Time `json:"last_hardware_check"`
}

type ClientHealthDTO struct {
	Time               *time.Time
	Tagnumber          int64
	TransactionUUID    string
	SystemSerial       *string
	TPMVersion         *string
	BIOSVersion        *string
	BIOSUpdated        *bool
	OSInstalled        *bool
	OSName             *string
	DiskHealthPcnt     *float64
	BatteryHealthPcnt  *float64
	AvgEraseTime       *float64
	AvgCloneTime       *float64
	LastCloneJobTime   *time.Time
	LastEraseJobTime   *time.Time
	TotalJobsCompleted *int64
	LastHardwareCheck  *time.Time
}

func CreatePartialClientHealthUpdateRequestDTO(request *ClientHealthUpdateRequest) (*ClientHealthDTO, error) {
	// Some of this mapping will have to be done in the database itself (aggregated, specific data)
	if request == nil {
		return nil, fmt.Errorf("nil input")
	}
	if request.Tagnumber == 0 || strings.TrimSpace(request.TransactionUUID) == "" {
		return nil, fmt.Errorf("missing tagnumber or transaction UUID")
	}
	dto := new(ClientHealthDTO)
	utcTime := time.Now().UTC()
	dto.Time = &utcTime
	dto.Tagnumber = request.Tagnumber
	dto.TransactionUUID = request.TransactionUUID
	if request.SystemSerial != nil && strings.TrimSpace(*request.SystemSerial) != "" {
		dto.SystemSerial = request.SystemSerial
	}
	if request.TPMVersion != nil && strings.TrimSpace(*request.TPMVersion) != "" {
		dto.TPMVersion = request.TPMVersion
	}
	if request.BIOSVersion != nil && strings.TrimSpace(*request.BIOSVersion) != "" {
		dto.BIOSVersion = request.BIOSVersion
	}
	if request.EraseCompleted != nil && *request.EraseCompleted {
		if request.EraseJobDuration != nil && *request.EraseJobDuration > 1 {
			dto.LastEraseJobTime = &utcTime
			*dto.OSInstalled = false // order matters
			dto.AvgEraseTime = request.EraseJobDuration
		}
	}
	if request.CloneCompleted != nil && *request.CloneCompleted {
		if request.CloneJobDuration != nil && *request.CloneJobDuration > 1 {
			dto.LastCloneJobTime = &utcTime
			*dto.OSInstalled = true // order matters
			dto.AvgCloneTime = request.CloneJobDuration
		}
	}
	dto.LastHardwareCheck = request.LastHardwareCheck

	return dto, nil
}
