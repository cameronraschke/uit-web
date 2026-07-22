package endpoints

// For form sanitization, only trim spaces on minimum length check, max length takes spaces into account.

import (
	"bytes"
	"crypto"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"image/jpeg"
	_ "image/png"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
	"uit-toolbox/config"
	"uit-toolbox/database"
	"uit-toolbox/middleware"
	"uit-toolbox/types"
	"unicode/utf8"

	"github.com/google/uuid"
)

func WebAuthEndpoint(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	log := middleware.GetLoggerFromContext(ctx).With(slog.String("func", "WebAuthEndpoint"))
	reqIP, err := middleware.GetRequestIPFromContext(ctx)
	if err != nil {
		log.Warn("Cannot retrieve request IP from context: " + err.Error())
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}

	req.Body = http.MaxBytesReader(w, req.Body, 300)
	defer req.Body.Close()

	body, err := io.ReadAll(req.Body)
	if err != nil {
		log.Warn("Cannot read request body: " + err.Error())
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}

	// Decode base64
	base64Decoded, err := base64.RawURLEncoding.DecodeString(strings.TrimSpace(string(body)))
	if err != nil {
		log.Warn("Invalid base64 encoding: " + err.Error())
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}
	if len(base64Decoded) == 0 {
		log.Warn("Empty base64 data")
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}

	// Check if decoded base64 is valid UTF-8
	if !types.IsPrintableUnicode(base64Decoded) {
		log.Warn("Invalid UTF-8 in base64 data")
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}

	// Unmarshal JSON from base64 bytes
	clientFormAuthData := new(types.LoginRequest)
	if err := json.Unmarshal(base64Decoded, clientFormAuthData); err != nil {
		log.Warn("Cannot unmarshal JSON: " + err.Error())
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}

	// Validate input data
	if utf8.RuneCountInString(clientFormAuthData.Username) != 0 || utf8.RuneCountInString(clientFormAuthData.Password) != 0 {
		if err := ValidateAuthFormInputSHA256(clientFormAuthData.Username, clientFormAuthData.Password); err != nil {
			log.Warn("Invalid username/password input: " + err.Error())
			middleware.WriteJsonError(w, http.StatusBadRequest)
			return
		}
	}

	// Authenticate with bcrypt
	authenticated, err := CheckAuthCredentials(ctx, clientFormAuthData.Username, clientFormAuthData.Password, clientFormAuthData.TwoFactorCode)
	if err != nil || !authenticated {
		log.Info("Authentication failed: " + err.Error())
		middleware.WriteJsonError(w, http.StatusUnauthorized)
		return
	}

	authSession, err := config.CreateAuthSession(reqIP)
	if err != nil {
		log.Error("Cannot create auth session: " + err.Error())
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}

	sessionCount := config.GetAuthSessionCount()
	log.Info("New auth session created. Total sessions: " + strconv.Itoa(int(sessionCount)))

	authSessionCookies, err := middleware.UpdateAndGetAuthSession(authSession, true)
	if err != nil {
		log.Error("Cannot get auth cookies for response: " + err.Error())
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}

	http.SetCookie(w, authSessionCookies.SessionCookie)
	http.SetCookie(w, authSessionCookies.BasicCookie)
	http.SetCookie(w, authSessionCookies.BearerCookie)
	// http.SetCookie(w, authSessionCookies.CSRFCookie)

	var responseJson = new(types.AuthStatusResponse)
	responseJson.Status = "authenticated"
	responseJson.ExpiresAt = time.Now().Add(types.AuthSessionTTL)
	responseJson.TTL = types.AuthSessionTTL

	middleware.WriteJson(w, http.StatusOK, responseJson)
}

func SetClientMemoryUsageKB(w http.ResponseWriter, req *http.Request) {
	log := middleware.GetLoggerFromContext(req.Context()).With(slog.String("func", "SetClientMemoryUsageKB"))

	requestBody, err := io.ReadAll(req.Body)
	if err != nil {
		log.Warn("Cannot read request body: " + err.Error())
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}
	if len(requestBody) == 0 {
		log.Warn("Empty request body")
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}

	var memInfoRequest types.MemoryDataUpdateRequest
	if err := json.Unmarshal(requestBody, &memInfoRequest); err != nil {
		log.Warn(types.JSONUnmarshalError.Error() + ": " + err.Error())
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}
	memoryData, err := memInfoRequest.ToDTO()
	if err != nil {
		log.Warn("Invalid memory data request: " + err.Error())
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}
	if memoryData == nil {
		log.Warn("Memory data request is nil after mapping to DTO")
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}
	if memoryData.TotalUsageKB <= 0 {
		log.Warn("Invalid memory usage value")
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}

	if err := database.UpsertClientMemoryUsageKB(req.Context(), *memoryData); err != nil {
		log.Error("Failed to update client memory usage: " + err.Error())
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}
	middleware.WriteJson(w, http.StatusOK, map[string]string{"status": "success"})
}

func SetClientMemoryCapacityKB(w http.ResponseWriter, req *http.Request) {
	log := middleware.GetLoggerFromContext(req.Context()).With(slog.String("func", "SetClientMemoryCapacityKB"))

	requestBody, err := io.ReadAll(req.Body)
	if err != nil {
		log.Warn("Cannot read request body: " + err.Error())
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}
	if len(requestBody) == 0 {
		log.Warn("Empty request body")
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}

	var memInfoRequest types.MemoryDataUpdateRequest
	if err := json.Unmarshal(requestBody, &memInfoRequest); err != nil {
		log.Warn(types.JSONUnmarshalError.Error() + ": " + err.Error())
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}
	memoryData, err := memInfoRequest.ToDTO()
	if err != nil {
		log.Warn("Invalid memory data request: " + err.Error())
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}
	if memoryData == nil {
		log.Warn("Memory data request is nil after mapping to DTO")
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}
	if memoryData.TotalCapacityKB <= 0 {
		log.Warn("Invalid memory capacity value")
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}

	if err := database.UpsertClientMemoryCapacityKB(req.Context(), *memoryData); err != nil {
		log.Error("Failed to update client memory capacity: " + err.Error())
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}
	middleware.WriteJson(w, http.StatusOK, map[string]string{"status": "success"})
}

func SetClientCPUUsage(w http.ResponseWriter, req *http.Request) {
	log := middleware.GetLoggerFromContext(req.Context()).With(slog.String("func", "SetClientCPUUsage"))
	requestBody, err := io.ReadAll(req.Body)
	if err != nil {
		log.Warn("Cannot read request body: " + err.Error())
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}
	if len(requestBody) == 0 {
		log.Warn("Empty request body")
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}

	var cpuDataRequest types.CPUDataUpdateRequest
	if err := json.Unmarshal(requestBody, &cpuDataRequest); err != nil {
		log.Warn("Cannot unmarshal JSON: " + err.Error())
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}

	cpuDTO, err := cpuDataRequest.ToDTO()
	if err != nil {
		log.Warn("Invalid CPU data request: " + err.Error())
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}

	if err := database.UpsertClientCPUUsage(req.Context(), cpuDTO); err != nil {
		log.Error("Failed to update client CPU usage: " + err.Error())
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}

	middleware.WriteJson(w, http.StatusOK, map[string]string{"status": "success"})
}

func SetClientCPUMHz(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	log := middleware.GetLoggerFromContext(ctx).With(slog.String("func", "SetClientCPUMHz"))
	requestBody, err := io.ReadAll(req.Body)
	if err != nil {
		log.Warn("Cannot read request body: " + err.Error())
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}
	if len(requestBody) == 0 {
		log.Warn("Empty request body")
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}

	var cpuUpdateRequest types.CPUDataUpdateRequest
	if err := json.Unmarshal(requestBody, &cpuUpdateRequest); err != nil {
		log.Warn(types.JSONUnmarshalError.Error() + ": " + err.Error())
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}

	cpuData, err := cpuUpdateRequest.ToDTO()
	if err != nil {
		log.Warn("Invalid CPU data request: " + err.Error())
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}

	if err := database.UpsertClientCPUMHz(ctx, cpuData); err != nil {
		log.Error("Failed to update client CPU MHz: " + err.Error())
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}
	middleware.WriteJson(w, http.StatusOK, map[string]string{"status": "success"})
}

func SetClientHealth(w http.ResponseWriter, req *http.Request) {
	log := middleware.GetLoggerFromContext(req.Context()).With(slog.String("func", "SetClientHealth"))
	requestBody, err := io.ReadAll(req.Body)
	if err != nil {
		log.Warn("Cannot read request body: " + err.Error())
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}
	if len(requestBody) == 0 {
		log.Warn("Empty request body")
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}
	if !types.IsPrintableUnicode(requestBody) {
		log.Warn("Invalid UTF-8 in request body")
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}
	var clientHealth types.ClientHealthUpdateRequest
	if err := json.Unmarshal(requestBody, &clientHealth); err != nil {
		log.Warn("Cannot unmarshal JSON: " + err.Error())
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}
	partialDTO, err := clientHealth.ToDTO()
	if err != nil {
		log.Warn("Unable to map client health update: %w" + err.Error())
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}

	transactionUUID, err := uuid.NewV7()
	if err != nil {
		log.Error("Failed to generate transaction UUID: " + err.Error())
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}
	if transactionUUID == uuid.Nil || transactionUUID.String() == "" {
		log.Error("Generated transaction UUID is nil")
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}

	if err := database.UpdateClientHealthUpdate(req.Context(), transactionUUID, partialDTO); err != nil {
		log.Error("database error: " + err.Error())
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}
	middleware.WriteJson(w, http.StatusOK, map[string]string{"status": "success"})
}

func SetClientCPUTemperature(w http.ResponseWriter, req *http.Request) {
	log := middleware.GetLoggerFromContext(req.Context()).With(slog.String("func", "SetClientCPUTemperature"))
	requestBody, err := io.ReadAll(req.Body)
	if err != nil {
		log.Warn("Cannot read request body: " + err.Error())
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}
	if len(requestBody) == 0 {
		log.Warn("Empty request body")
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}

	var cpuDataRequest types.CPUDataUpdateRequest
	if err := json.Unmarshal(requestBody, &cpuDataRequest); err != nil {
		log.Warn("Cannot unmarshal JSON: " + err.Error())
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}

	cpuData, err := cpuDataRequest.ToDTO()
	if err != nil {
		log.Warn("Invalid CPU data request: " + err.Error())
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}

	if err := database.UpsertClientCPUTemperature(req.Context(), cpuData); err != nil {
		log.Error("Failed to update client CPU temperature: " + err.Error())
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}
	middleware.WriteJson(w, http.StatusOK, map[string]string{"status": "success"})
}

func SetClientNetworkUsage(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	log := middleware.GetLoggerFromContext(ctx)
	log = log.With(slog.String("func", "SetClientNetworkUsage"))
	requestBody, err := io.ReadAll(req.Body)
	if err != nil {
		log.Warn("Cannot read request body: " + err.Error())
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}
	if len(requestBody) == 0 {
		log.Warn("Empty request body")
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}
	if !types.IsPrintableUnicode(requestBody) {
		log.Warn("Invalid UTF-8 in request body")
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}
	var networkData types.NetworkData
	if err := json.Unmarshal(requestBody, &networkData); err != nil {
		log.Warn("Cannot unmarshal JSON: " + err.Error())
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}
	if err := types.IsTagnumberInt64Valid(&networkData.Tagnumber); err != nil {
		log.Warn("Invalid tagnumber: " + err.Error())
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}
	if networkData.NetworkUsage == nil {
		log.Warn("Request is missing network usage")
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}
	if networkData.LinkSpeed == nil {
		log.Warn("Request is missing link speed")
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}

	updateRepo, err := database.NewUpdateRepo()
	if err != nil {
		log.Error("No database connection available for updating client network usage")
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}
	if err := updateRepo.UpdateClientNetworkUsage(ctx, &networkData); err != nil {
		log.Error("Failed to update client network usage: " + err.Error())
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}
	middleware.WriteJson(w, http.StatusOK, map[string]string{"status": "success"})
}

func SetClientUptime(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	log := middleware.GetLoggerFromContext(ctx)
	log = log.With(slog.String("func", "SetClientUptime"))
	requestBody, err := io.ReadAll(req.Body)
	if err != nil {
		log.Warn("Cannot read request body: " + err.Error())
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}
	if len(requestBody) == 0 {
		log.Warn("Empty request body")
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}
	if !types.IsPrintableUnicode(requestBody) {
		log.Warn("Invalid UTF-8 in request body")
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}
	var uptimeData types.ClientUptime
	if err := json.Unmarshal(requestBody, &uptimeData); err != nil {
		log.Warn("Cannot unmarshal JSON: " + err.Error())
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}
	if err := types.IsTagnumberInt64Valid(&uptimeData.Tagnumber); err != nil {
		log.Warn(fmt.Sprintf("%v: %s (%v)", types.InvalidRequestFieldError, "tagnumber", err))
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}
	if uptimeData.ClientAppUptime == 0 && uptimeData.SystemUptime == 0 {
		log.Warn(fmt.Sprintf("%v: %s (%v)", types.InvalidRequestFieldError, "uptime data", "both clientAppUptime and systemUptime have zero values"))
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}

	if uptimeData.ClientAppUptime != 0 {
		clientAppUptime := uptimeData.ClientAppUptime.Duration()
		if err := config.UpdateClientAppUptime(uptimeData.Tagnumber, clientAppUptime); err != nil {
			log.Error(fmt.Sprintf("%v '%s': %v", types.ErrFailedToUpdateRealtimeData, "clientAppUptime", err))
			middleware.WriteJsonError(w, http.StatusInternalServerError)
			return
		}
	}

	if uptimeData.SystemUptime != 0 {
		systemUptime := uptimeData.SystemUptime.Duration()
		if err := config.UpdateClientSystemUptime(uptimeData.Tagnumber, systemUptime); err != nil {
			log.Error(fmt.Sprintf("%v '%s': %v", types.ErrFailedToUpdateRealtimeData, "systemUptime", err))
			middleware.WriteJsonError(w, http.StatusInternalServerError)
			return
		}
	}

	middleware.WriteJson(w, http.StatusOK, map[string]string{"status": "success"})
}

func SetClientLastHeard(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	log := middleware.GetLoggerFromContext(ctx).With(slog.String("func", "SetClientLastHeard"))
	requestBody, err := io.ReadAll(req.Body)
	if err != nil {
		log.Warn("Cannot read request body: " + err.Error())
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}
	if len(requestBody) == 0 {
		log.Warn("Empty request body")
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}
	if !types.IsPrintableUnicode(requestBody) {
		log.Warn("Invalid UTF-8 in request body")
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}
	var lastHeardData struct {
		Tagnumber int64     `json:"tagnumber"`
		LastHeard time.Time `json:"last_heard"`
	}
	if err := json.Unmarshal(requestBody, &lastHeardData); err != nil {
		log.Warn("Cannot unmarshal JSON: " + err.Error())
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}

	// Check if tagnumber is valid
	if err := types.IsTagnumberInt64Valid(&lastHeardData.Tagnumber); err != nil {
		log.Warn(fmt.Sprintf("%v: %s (%v)", types.InvalidRequestFieldError, "tagnumber", err))
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}
	// Check if lastHeard is valid
	if lastHeardData.LastHeard.IsZero() || lastHeardData.LastHeard.Unix() <= 0 {
		log.Warn(fmt.Sprintf("%v '%s': %v", types.InvalidRequestFieldError, "lastHeard", "value is zero"))
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}
	if lastHeardData.LastHeard.UTC().After(time.Now().UTC().Add(1 * time.Minute)) {
		log.Warn(fmt.Sprintf("%v '%s': %v", types.InvalidRequestFieldError, "lastHeard", "value is in the future"))
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}

	clientUUID, err := config.GetRealtimeClientUUID(lastHeardData.Tagnumber)
	if err != nil {
		log.Info(fmt.Sprintf("%v '%d': %v", types.ErrClientUUIDMissingError, lastHeardData.Tagnumber, err))
		if !errors.Is(err, types.ErrClientNotFound) {
			middleware.WriteJsonError(w, http.StatusInternalServerError)
			return
		}
	}

	if clientUUID == uuid.Nil {
		pgxPool, err := config.GetPGXPool()
		if err != nil {
			log.Error("No database connection available for updating client last heard")
			middleware.WriteJsonError(w, http.StatusInternalServerError)
			return
		}
		clientUUID, err = database.GetClientUUIDByTag(ctx, pgxPool, lastHeardData.Tagnumber)
		if err != nil {
			log.Error(fmt.Sprintf("%v '%d': %v", types.ErrClientUUIDNotFoundInDB, lastHeardData.Tagnumber, err))
			middleware.WriteJsonError(w, http.StatusNotFound)
			return
		}
		if err := config.SetRealtimeClientUUID(lastHeardData.Tagnumber, clientUUID); err != nil {
			log.Error(fmt.Sprintf("%v '%s': %v", types.ErrFailedToUpdateRealtimeData, "clientUUID", err))
			middleware.WriteJsonError(w, http.StatusInternalServerError)
			return
		}
	}

	lastHeard := lastHeardData.LastHeard.UTC()
	if err := config.UpdateClientLastHeard(lastHeardData.Tagnumber, &lastHeard); err != nil {
		log.Error(fmt.Sprintf("%v '%s': %v", types.ErrFailedToUpdateRealtimeData, "lastHeard", err))
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}

	middleware.WriteJson(w, http.StatusOK, map[string]string{"status": "success"})
}

func UpdateClientBatteryChargePcnt(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	log := middleware.GetLoggerFromContext(ctx).With(slog.String("func", "UpdateClientBatteryChargePcnt"))
	requestBody, err := io.ReadAll(req.Body)
	if err != nil {
		log.Warn("Cannot read request body: " + err.Error())
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}
	if len(requestBody) == 0 {
		log.Warn("Empty request body")
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}
	if !types.IsPrintableUnicode(requestBody) {
		log.Warn("Invalid UTF-8 in request body")
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}
	batteryData := new(types.BatteryDataRequest)
	if err := json.Unmarshal(requestBody, batteryData); err != nil {
		log.Warn("Cannot unmarshal JSON: " + err.Error())
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}
	if err := types.IsTagnumberInt64Valid(batteryData.Tagnumber); err != nil {
		log.Warn(fmt.Sprintf("%v for '%s': %v", types.InvalidRequestFieldError, "tagnumber", err))
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}
	if batteryData.BatteryChargePcnt == nil {
		log.Warn(fmt.Sprintf("%v for '%s': %v", types.InvalidRequestFieldError, "batteryChargePcnt", "value is nil or zero"))
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}
	if *batteryData.BatteryChargePcnt < 0 || *batteryData.BatteryChargePcnt > 100 {
		log.Warn("Battery percentage out of valid range (0-100)")
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}
	updateRepo, err := database.NewUpdateRepo()
	if err != nil {
		log.Error(fmt.Sprintf("%v while updating '%s': %v", types.DatabaseConnError, "clientBatteryChargePcnt", err))
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}
	if err := updateRepo.UpdateClientBatteryChargePcnt(ctx, batteryData.Tagnumber, batteryData.BatteryChargePcnt); err != nil {
		log.Error(fmt.Sprintf("%v '%s': %v", types.FailedToUpdateDatabaseValueError, "clientBatteryChargePcnt", err))
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}
	middleware.WriteJson(w, http.StatusOK, map[string]string{"status": "success"})
}

func InsertNewNote(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	log := middleware.GetLoggerFromContext(ctx).With(slog.String("func", "InsertNewNote"))
	appState, err := config.GetAppState()
	if err != nil {
		log.Warn("Cannot get app state in InsertNewNote: " + err.Error())
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}
	htmlFormConstraints, err := appState.GetFormConstraints()
	if err != nil {
		log.Error("Cannot retrieve HTMLFormConstraints: " + err.Error())
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}
	maxNoteJSONBytes := int64((htmlFormConstraints.GeneralNote.MaxFormBytes)*4 + 512)
	req.Body = http.MaxBytesReader(w, req.Body, maxNoteJSONBytes)
	defer req.Body.Close()

	// Parse and validate note data
	var newNote types.GeneralNoteResponse
	err = json.NewDecoder(req.Body).Decode(&newNote)
	if err != nil {
		log.Warn("Cannot decode note JSON: " + err.Error())
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}

	if newNote.NoteType == nil || newNote.NoteContent == nil {
		log.Warn("Note type or content is nil, not inserting new note")
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}
	if err := types.ValidatePrintableStrLen(newNote.NoteType, 1, 64); err != nil {
		log.Warn(types.CreateInvalidFieldErrorStr("note_type", err))
		middleware.WriteJsonError(w, http.StatusRequestEntityTooLarge)
		return
	}
	if err := types.ValidatePrintableStrLen(newNote.NoteContent, 0, 32768); err != nil {
		log.Warn(types.CreateInvalidFieldErrorStr("note_content", err))
		middleware.WriteJsonError(w, http.StatusRequestEntityTooLarge)
		return
	}

	// Insert note into database
	curTime := time.Now().UTC()
	err = database.InsertNewNote(ctx, &curTime, newNote.NoteType, newNote.NoteContent)
	if err != nil {
		log.Error(fmt.Sprintf("%v '%s': %v", types.FailedToUpdateDatabaseValueError, "newNote", err))
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}
	middleware.WriteJson(w, http.StatusOK, map[string]string{"status": "success"})
}

func InsertInventoryUpdate(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	log := middleware.GetLoggerFromContext(ctx).With(slog.String("func", "InsertInventoryUpdate"))

	// Parse inventory data
	appState, err := config.GetAppState()
	if err != nil {
		log.Warn("Cannot get app state: " + err.Error())
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}

	htmlFormConstraints, err := appState.GetFormConstraints()
	if err != nil {
		log.Error("Cannot retrieve HTMLFormConstraints: " + err.Error())
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}

	// JSON part
	jsonFile, _, err := req.FormFile("json")
	if err != nil {
		log.Warn("Error retrieving JSON data provided in form: " + err.Error())
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}
	defer jsonFile.Close()

	jsonReader := &io.LimitedReader{R: jsonFile, N: htmlFormConstraints.InventoryForm.MaxJSONBytes + 1}
	jsonBytes, err := io.ReadAll(jsonReader)
	if err != nil {
		log.Warn("Error reading JSON data from form: " + err.Error())
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}
	if int64(len(jsonBytes)) > htmlFormConstraints.InventoryForm.MaxJSONBytes {
		log.Warn("JSON data in form exceeds maximum allowed size after reading")
		middleware.WriteJsonError(w, http.StatusRequestEntityTooLarge)
		return
	}

	var inventoryUpdateReq types.InventoryUpdateRequest
	if err := json.Unmarshal(jsonBytes, &inventoryUpdateReq); err != nil {
		log.Warn("Cannot decode JSON: " + err.Error())
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}
	if !utf8.Valid(jsonBytes) {
		log.Warn("Invalid UTF-8 in JSON data")
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}

	inventoryDomain, err := inventoryUpdateReq.ToDTO(htmlFormConstraints)
	if err != nil {
		log.Warn("Invalid inventory request payload: " + err.Error())
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}

	// Generate transaction UUID to share between multiple DB tables
	transactionUUID, err := uuid.NewV7()
	if err != nil {
		log.Error("error generation a transaction UUID")
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}
	if transactionUUID == uuid.Nil {
		log.Error("transaction UUID is nil")
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}

	// Update db
	inventoryData := inventoryDomain.ToLocationWriteModel(transactionUUID)
	if err := database.InsertInventoryUpdate(ctx, transactionUUID, inventoryData); err != nil {
		log.Error(fmt.Sprintf("%v '%s': %v", types.FailedToUpdateDatabaseValueError, "inventoryData", err))
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}
	clientHardwareData := inventoryDomain.ToHardwareWriteModel(transactionUUID)
	if err := database.UpdateInventoryHardwareData(ctx, transactionUUID, clientHardwareData); err != nil {
		log.Error("Failed to update inventory hardware info: " + err.Error())
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}
	clientHealthData := inventoryDomain.ToClientHealthWriteModel(transactionUUID)
	if err := database.UpdateClientHealthUpdate(ctx, transactionUUID, clientHealthData); err != nil {
		log.Error("Failed to update inventory health info: " + err.Error())
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}
	checkoutData := inventoryDomain.ToCheckoutWriteModel(transactionUUID)
	if checkoutData != nil {
		if err := database.InsertClientCheckoutsUpdate(ctx, transactionUUID, checkoutData); err != nil {
			log.Error("Failed to update inventory checkout info: " + err.Error())
			middleware.WriteJsonError(w, http.StatusInternalServerError)
			return
		}
	}

	var jsonResponse = struct {
		Tagnumber int64  `json:"tagnumber"`
		Message   string `json:"message"`
	}{
		Tagnumber: inventoryDomain.Tagnumber,
		Message:   "update successful",
	}

	middleware.WriteJson(w, http.StatusOK, jsonResponse)
}

func UploadClientImage(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	log := middleware.GetLoggerFromContext(req.Context()).With(slog.String("func", "UploadClientImage"))

	if req.Method != http.MethodPost {
		log.Warn("Invalid method for image upload request: " + req.Method)
		middleware.WriteJsonError(w, http.StatusMethodNotAllowed)
		return
	}

	contentType := strings.TrimSpace(req.Header.Get("Content-Type"))
	if !strings.HasPrefix(strings.ToLower(contentType), "multipart/form-data") {
		log.Warn("Invalid content type for image upload request: " + contentType)
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}

	queryValues := req.URL.Query()
	if len(queryValues) != 1 || len(queryValues["tagnumber"]) != 1 || strings.TrimSpace(queryValues.Get("tagnumber")) == "" {
		log.Warn("Image upload request requires exactly one query key to be populated: tagnumber")
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}

	tag := middleware.GetInt64Query(req.URL.Query(), "tagnumber")
	if err := types.IsTagnumberInt64Valid(tag); err != nil {
		log.Warn("Invalid tagnumber query parameter: " + err.Error())
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}

	appState, err := config.GetAppState()
	if err != nil {
		log.Warn("Cannot get app state: " + err.Error())
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}

	endpointConfig, err := config.GetWebEndpointConfig(req.URL.Path)
	if err != nil {
		log.Warn("Cannot get endpoint config from AppState for path: " + req.URL.Path + " - " + err.Error())
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}

	maxAllowedUploadBytes := endpointConfig.MaxUploadSize
	if maxAllowedUploadBytes == nil {
		log.Error("Max upload size is not defined for endpoint path: " + req.URL.Path)
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}
	req.Body = http.MaxBytesReader(w, req.Body, *maxAllowedUploadBytes)
	defer req.Body.Close()

	if err := req.ParseMultipartForm(*maxAllowedUploadBytes); err != nil {
		if errors.Is(err, http.ErrNotMultipart) {
			log.Warn("Request body is not multipart form data: " + err.Error())
			middleware.WriteJsonError(w, http.StatusBadRequest)
			return
		}
		if errors.Is(err, http.ErrMissingBoundary) {
			log.Warn("Multipart form data missing boundary: " + err.Error())
			middleware.WriteJsonError(w, http.StatusBadRequest)
			return
		}
		if os.IsTimeout(err) {
			log.Warn("Request timed out while reading multipart form: " + err.Error())
			middleware.WriteJsonError(w, http.StatusRequestTimeout)
			return
		}
		if maxBytesErr, ok := errors.AsType[*http.MaxBytesError](err); ok {
			log.Warn("Request body too large: " + maxBytesErr.Error())
			middleware.WriteJsonError(w, http.StatusRequestEntityTooLarge)
			return
		}
		log.Warn("Cannot parse multipart form: " + err.Error())
		middleware.WriteJsonError(w, http.StatusRequestEntityTooLarge)
		return
	}
	if req.MultipartForm != nil {
		defer req.MultipartForm.RemoveAll()
	}

	fileUploadConstraints, err := appState.GetFileUploadConstraints()
	if err != nil {
		log.Error("Cannot retrieve FileConstraints: " + err.Error())
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}

	clientLookupResult, err := database.ClientIDLookup(ctx, tag, nil)
	if err != nil {
		log.Warn("Error looking up client ID for tag '" + strconv.FormatInt(*tag, 10) + "': " + err.Error())
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}

	// File upload part of form:
	if req.MultipartForm == nil || req.MultipartForm.File == nil {
		log.Info("File upload part of image upload is nil")
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}
	files := req.MultipartForm.File["files"]
	if len(files) == 0 {
		if req.MultipartForm.File["files"] == nil {
			log.Info("No client images provided in request, exiting early")
			middleware.WriteJsonError(w, http.StatusBadRequest)
			return
		} else {
			files = req.MultipartForm.File["files"]
		}
	}

	type uploadFileResult struct {
		OriginalName string `json:"original_name"`
		StoredName   string `json:"stored_name,omitempty"`
		FileUUID     string `json:"file_uuid,omitempty"`
		MimeType     string `json:"mime_type,omitempty"`
		Bytes        int64  `json:"bytes,omitempty"`
		Category     string `json:"category,omitempty"`
		Status       string `json:"status"`
		Reason       string `json:"reason,omitempty"`
	}

	type uploadSummary struct {
		Status             string             `json:"status"`
		Tagnumber          int64              `json:"tagnumber"`
		TotalReceived      int                `json:"total_received"`
		UploadedCount      int                `json:"uploaded_count"`
		SkippedCount       int                `json:"skipped_count"`
		FailedCount        int                `json:"failed_count"`
		ImageCount         int                `json:"image_count"`
		VideoCount         int                `json:"video_count"`
		TotalUploadedBytes int64              `json:"total_uploaded_bytes"`
		Results            []uploadFileResult `json:"results"`
	}

	dbManifest, err := database.GetClientImageManifestByTag(ctx, tag)
	if err != nil {
		log.Error("Failed to get file hashes from tag '" + strconv.FormatInt(*tag, 10) + "': " + err.Error())
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}

	var totalImageFileCount int
	var totalImageUploadSize int64
	var totalVideoFileCount int
	var totalVideoUploadSize int64
	results := make([]uploadFileResult, 0, len(files))
	var skippedCount int
	var failedCount int
	var uploadRequestHashes [][]byte
	for _, fileHeader := range files {
		var manifest = new(types.ImageManifestDTO)
		result := uploadFileResult{
			OriginalName: fileHeader.Filename,
		}

		if !types.IsPrintableUnicodeString(fileHeader.Filename) {
			log.Warn("Non-printable characters in uploaded file name: " + fileHeader.Filename)
			result.Status = "failed"
			result.Reason = "invalid_filename"
			results = append(results, result)
			failedCount++
			continue
		}

		cleanName := filepath.Clean(fileHeader.Filename)
		if cleanName == "." || cleanName == string(filepath.Separator) {
			result.Status = "failed"
			result.Reason = "invalid_filename"
			results = append(results, result)
			failedCount++
			continue
		}
		fileHeader.Filename = filepath.Base(cleanName)

		createNecessaryDirs := func(clientUUID *string) (string, error) {
			// Check client UUID
			if clientUUID == nil || strings.TrimSpace(*clientUUID) == "" {
				return "", fmt.Errorf("client UUID is nil for tagnumber: %d", *tag)
			}
			// Create directories if not existing
			imageDirectoryPath := filepath.Join("/opt/inventory_images", *clientUUID)
			if err := os.MkdirAll(imageDirectoryPath, 0755); err != nil {
				return "", fmt.Errorf("cannot create parent directories for '"+imageDirectoryPath+"': %w", err)
			}

			// Set file/directory permissions
			if err := os.Chmod(imageDirectoryPath, 0755); err != nil {
				return "", fmt.Errorf("cannot set directory permissions for '"+imageDirectoryPath+"': %w", err)
			}
			if err := os.Chown(imageDirectoryPath, os.Getuid(), os.Getgid()); err != nil {
				return "", fmt.Errorf("cannot set directory ownership for '"+imageDirectoryPath+"': %w", err)
			}
			return imageDirectoryPath, nil
		}

		// Open uploaded file
		file, err := fileHeader.Open()
		if err != nil {
			log.Warn("Failed to open uploaded file '" + fileHeader.Filename + "': " + err.Error())
			result.Status = "failed"
			result.Reason = "open_failed"
			results = append(results, result)
			failedCount++
			continue
		}

		lr := &io.LimitedReader{R: file, N: fileUploadConstraints.ImageConstraints.MaxFileSize + fileUploadConstraints.VideoConstraints.MaxFileSize + 1}
		fileBytes, err := io.ReadAll(lr)
		_ = file.Close()
		if err != nil {
			log.Warn("Failed to read uploaded file '" + fileHeader.Filename + "': " + err.Error())
			result.Status = "failed"
			result.Reason = "read_failed"
			results = append(results, result)
			failedCount++
			continue
		}

		// File size
		manifest.FileSize = int64(len(fileBytes))

		// MIME type detection
		mimeType := http.DetectContentType(fileBytes)
		if mimeType == "application/octet-stream" {
			log.Warn("Unknown MIME type for file '" + fileHeader.Filename + "'")
			result.Status = "failed"
			result.Reason = "unknown_mime_type"
			results = append(results, result)
			failedCount++
			continue
		}
		manifest.MimeType = mimeType
		ext := strings.ToLower(filepath.Ext(fileHeader.Filename))

		if fileUploadConstraints.ImageConstraints.AcceptedImageExtensionsAndMimeTypes[ext] != mimeType && fileUploadConstraints.VideoConstraints.AcceptedVideoExtensionsAndMimeTypes[ext] != mimeType {
			log.Warn("Unsupported file type for file '" + fileHeader.Filename + "': detected MIME type '" + mimeType + "' does not match expected MIME type for file extension '" + ext + "'")
			result.Status = "failed"
			result.Reason = "unsupported_file_type"
			results = append(results, result)
			failedCount++
			continue
		}

		// Get upload timestamp
		manifest.Time = time.Now().UTC()

		// Generate unique file name
		fileTimeStampFormatted := manifest.Time.Format("2006-01-02-150405")
		fileUUID, err := uuid.NewV7()
		if err != nil {
			log.Error("Failed to generate file UUID for uploaded file '" + fileHeader.Filename + "': " + err.Error())
			middleware.WriteJsonError(w, http.StatusInternalServerError)
			return
		}
		fileUUIDStr := fileUUID.String()

		fileName := fileTimeStampFormatted + "-" + fileUUIDStr + ext
		manifest.FileName = fileName
		manifest.FileUUID = fileUUIDStr

		// Compute SHA256 hash of file
		fileHash := crypto.SHA256.New()
		if _, err := fileHash.Write(fileBytes); err != nil {
			log.Error("Failed to compute hash of uploaded file '" + fileHeader.Filename + "': " + err.Error())
			middleware.WriteJsonError(w, http.StatusInternalServerError)
			return
		}
		shaSum := fileHash.Sum(nil)
		fileHashBytes := make([]uint8, 32)
		copy(fileHashBytes, shaSum[:32])
		manifest.SHA256Hash = fileHashBytes
		uploadRequestHashes = append(uploadRequestHashes, fileHashBytes)
		if len(uploadRequestHashes) > 1 {
			for i := 0; i < len(uploadRequestHashes)-1; i++ {
				if bytes.Equal(uploadRequestHashes[i], fileHashBytes) {
					log.Warn("Duplicate file upload detected within same request for tag '" + strconv.FormatInt(*tag, 10) + "': file '" + fileHeader.Filename + "' has same hash as another uploaded file, skipping")
					result.Status = "skipped"
					result.Reason = "duplicate_hash_in_request"
					results = append(results, result)
					skippedCount++
					continue
				}
			}
		}

		for _, m := range dbManifest {
			if m.SHA256Hash != nil && bytes.Equal(*m.SHA256Hash, manifest.SHA256Hash) {
				// Duplicate files don't matter if original copy is marked as hidden because original hidden file should be deleted from filesystem
				if !manifest.Hidden {
					log.Warn("Duplicate file upload detected for tag '" + strconv.FormatInt(*tag, 10) + "': file '" + fileHeader.Filename + "' (" + fmt.Sprintf("%x", fileHashBytes) + ") has same hash as existing file, skipping")
					result.Status = "skipped"
					result.Reason = "duplicate_hash"
					results = append(results, result)
					skippedCount++
					continue
				}
			}
			if m.FileName != nil {
				if m.FileName == &manifest.FileName {
					log.Warn("Duplicate file name detected for tag '" + strconv.FormatInt(*tag, 10) + "': file '" + fileHeader.Filename + "' has same generated file name as existing file, skipping")
					result.Status = "skipped"
					result.Reason = "duplicate_filename"
					results = append(results, result)
					skippedCount++
					continue
				}
			}
		}

		thumbnailPath := ""
		fileCategory := ""

		if fileUploadConstraints.ImageConstraints.AcceptedImageExtensionsAndMimeTypes[ext] == mimeType { // Image file processing
			if totalImageFileCount >= fileUploadConstraints.ImageConstraints.MaxFileCount {
				log.Warn("Number of uploaded image files exceeds maximum allowed: " + strconv.Itoa(totalImageFileCount))
				result.Status = "failed"
				result.Reason = "max_image_file_count_exceeded"
				results = append(results, result)
				failedCount++
				continue
			}
			if manifest.FileSize > fileUploadConstraints.ImageConstraints.MaxFileSize {
				log.Warn("Uploaded image file '" + fileHeader.Filename + "' too large (" + strconv.FormatInt(manifest.FileSize, 10) + " bytes)")
				result.Status = "failed"
				result.Reason = "image_file_too_large"
				results = append(results, result)
				failedCount++
				continue
			}
			if manifest.FileSize < fileUploadConstraints.ImageConstraints.MinFileSize {
				log.Warn("Uploaded image file too small: " + fileHeader.Filename + " (" + strconv.FormatInt(manifest.FileSize, 10) + " bytes)")
				result.Status = "failed"
				result.Reason = "image_file_too_small"
				results = append(results, result)
				failedCount++
				continue
			}
			// Create reader (stream) for image decoding
			imageReader := bytes.NewReader(fileBytes)

			// Rewind and decode image to get image.Image
			_, err = imageReader.Seek(0, io.SeekStart)
			if err != nil {
				log.Error("Failed to seek to start of uploaded image '" + fileHeader.Filename + "': " + err.Error())
				result.Status = "failed"
				result.Reason = "image_seek_failed"
				results = append(results, result)
				failedCount++
				continue
			}
			decodedImage, _, err := image.Decode(imageReader)
			if err != nil {
				log.Error("Failed to decode uploaded image '" + fileHeader.Filename + "': " + err.Error())
				result.Status = "failed"
				result.Reason = "image_decode_failed"
				results = append(results, result)
				failedCount++
				continue
			}

			// Rewind and decode image to get image config
			_, err = imageReader.Seek(0, io.SeekStart)
			if err != nil {
				log.Error("Failed to seek to start of uploaded image '" + fileHeader.Filename + "': " + err.Error())
				result.Status = "failed"
				result.Reason = "image_seek_failed"
				results = append(results, result)
				failedCount++
				continue
			}
			decodedImageConfig, _, err := image.DecodeConfig(imageReader)
			if err != nil {
				log.Error("Failed to decode uploaded image config '" + fileHeader.Filename + "': " + err.Error())
				result.Status = "failed"
				result.Reason = "image_decode_config_failed"
				results = append(results, result)
				failedCount++
				continue
			}
			resX := int64(decodedImageConfig.Width)
			manifest.ResolutionX = &resX
			resY := int64(decodedImageConfig.Height)
			manifest.ResolutionY = &resY

			// Generate jpeg thumbnail
			imageDirectoryPath, err := createNecessaryDirs(clientLookupResult.ClientUUID)
			if err != nil {
				log.Error("Failed to create necessary directories for thumbnail of '" + fileHeader.Filename + "': " + err.Error())
				result.Status = "failed"
				result.Reason = "thumbnail_directory_failed"
				results = append(results, result)
				failedCount++
				continue
			}
			strippedFileName := strings.TrimSuffix(fileName, filepath.Ext(fileName))
			fullThumbnailPath := filepath.Join(imageDirectoryPath, strippedFileName+"-thumbnail.jpeg")
			thumbnailPath = fullThumbnailPath
			thumbnailFile, err := os.Create(fullThumbnailPath)
			if err != nil {
				log.Error("Failed to create thumbnail file '" + fullThumbnailPath + "': " + err.Error())
				result.Status = "failed"
				result.Reason = "thumbnail_create_failed"
				results = append(results, result)
				failedCount++
				continue
			}
			if err := os.Chmod(fullThumbnailPath, 0644); err != nil {
				_ = thumbnailFile.Close()
				log.Error("Failed to set permissions for thumbnail file '" + fullThumbnailPath + "': " + err.Error())
				result.Status = "failed"
				result.Reason = "thumbnail_permission_failed"
				results = append(results, result)
				failedCount++
				continue
			}
			if err := jpeg.Encode(thumbnailFile, decodedImage, &jpeg.Options{Quality: 50}); err != nil {
				_ = thumbnailFile.Close()
				log.Error("Failed to encode thumbnail image '" + fullThumbnailPath + "': " + err.Error())
				result.Status = "failed"
				result.Reason = "thumbnail_encode_failed"
				results = append(results, result)
				failedCount++
				continue
			}
			_ = thumbnailFile.Close()
			thumbnailFileName := strippedFileName + "-thumbnail.jpeg"
			manifest.ThumbnailFileName = &thumbnailFileName
			totalImageUploadSize += manifest.FileSize
			totalImageFileCount++
			fileCategory = "image"
		} else if fileUploadConstraints.VideoConstraints.AcceptedVideoExtensionsAndMimeTypes[ext] == mimeType { // Video file processing
			if totalVideoFileCount >= fileUploadConstraints.VideoConstraints.MaxFileCount {
				log.Warn("Number of uploaded video files exceeds maximum allowed: " + strconv.Itoa(totalVideoFileCount))
				result.Status = "failed"
				result.Reason = "max_video_file_count_exceeded"
				results = append(results, result)
				failedCount++
				continue
			}
			if manifest.FileSize > fileUploadConstraints.VideoConstraints.MaxFileSize {
				log.Warn("Uploaded video file too large (" + strconv.FormatInt(manifest.FileSize, 10) + " bytes)")
				result.Status = "failed"
				result.Reason = "video_file_too_large"
				results = append(results, result)
				failedCount++
				continue
			}
			if manifest.FileSize < fileUploadConstraints.VideoConstraints.MinFileSize {
				log.Warn("Uploaded video file too small: " + fileHeader.Filename + " (" + strconv.FormatInt(manifest.FileSize, 10) + " bytes)")
				result.Status = "failed"
				result.Reason = "video_file_too_small"
				results = append(results, result)
				failedCount++
				continue
			}
			totalVideoFileCount++
			totalVideoUploadSize += manifest.FileSize
			fileCategory = "video"
		} else {
			log.Warn("Unsupported MIME type for '" + fileHeader.Filename + "': MIME Type: " + mimeType)
			result.Status = "failed"
			result.Reason = "unsupported_mime_type"
			results = append(results, result)
			failedCount++
			continue
		}

		imageDirectoryPath, err := createNecessaryDirs(clientLookupResult.ClientUUID)
		if err != nil {
			log.Error("Failed to create necessary directories for '" + fileHeader.Filename + "': " + err.Error())
			result.Status = "failed"
			result.Reason = "storage_directory_failed"
			results = append(results, result)
			failedCount++
			continue
		}
		fullFilePath := filepath.Join(imageDirectoryPath, fileName)
		if err := os.WriteFile(fullFilePath, fileBytes, 0644); err != nil {
			log.Error("Failed to save uploaded file '" + fullFilePath + "': " + err.Error())
			result.Status = "failed"
			result.Reason = "file_save_failed"
			results = append(results, result)
			failedCount++
			continue
		}

		// Insert image metadata into database
		manifest.Tagnumber = *tag
		manifest.Hidden = false
		manifest.Pinned = false

		transactionUUID, err := uuid.NewV7()
		if err != nil {
			log.Error("error generation a transaction UUID (BulkUpdateInventoryLocation)")
			_ = os.Remove(fullFilePath)
			if thumbnailPath != "" {
				_ = os.Remove(thumbnailPath)
			}
			result.Status = "failed"
			result.Reason = "transaction_uuid_generation_failed"
			results = append(results, result)
			failedCount++
			continue
		}
		if transactionUUID == uuid.Nil {
			log.Error("transaction UUID in BulkUpdateInventoryLocation is nil")
			_ = os.Remove(fullFilePath)
			if thumbnailPath != "" {
				_ = os.Remove(thumbnailPath)
			}
			result.Status = "failed"
			result.Reason = "transaction_uuid_nil"
			results = append(results, result)
			failedCount++
			continue
		}

		if err := database.UpdateClientImages(ctx, transactionUUID, manifest); err != nil {
			log.Error("Failed to update inventory image data for '" + fullFilePath + "': " + err.Error())
			if removeErr := os.Remove(fullFilePath); removeErr != nil && !errors.Is(removeErr, os.ErrNotExist) {
				log.Warn("Failed to rollback uploaded file after DB error for '" + fullFilePath + "': " + removeErr.Error())
			}
			if thumbnailPath != "" {
				if removeThumbErr := os.Remove(thumbnailPath); removeThumbErr != nil && !errors.Is(removeThumbErr, os.ErrNotExist) {
					log.Warn("Failed to rollback thumbnail after DB error for '" + thumbnailPath + "': " + removeThumbErr.Error())
				}
			}
			result.Status = "failed"
			result.Reason = "database_write_failed"
			results = append(results, result)
			failedCount++
			continue
		}

		result.StoredName = manifest.FileName
		result.FileUUID = manifest.FileUUID
		result.MimeType = mimeType
		result.Bytes = manifest.FileSize
		result.Category = fileCategory
		result.Status = "uploaded"
		results = append(results, result)
		log.Info(fmt.Sprintf("Uploaded file '%s', Size: %.2f MB, MIME Type: %s", fileName, float64(manifest.FileSize)/1024/1024, mimeType))
	}
	fileUploadCount := totalImageFileCount + totalVideoFileCount
	totalActualFileBytes := totalImageUploadSize + totalVideoUploadSize
	summary := uploadSummary{
		Status:             "success",
		Tagnumber:          *tag,
		TotalReceived:      len(files),
		UploadedCount:      fileUploadCount,
		SkippedCount:       skippedCount,
		FailedCount:        failedCount,
		ImageCount:         totalImageFileCount,
		VideoCount:         totalVideoFileCount,
		TotalUploadedBytes: totalActualFileBytes,
		Results:            results,
	}

	statusCode := http.StatusAccepted
	if summary.UploadedCount == 0 {
		summary.Status = "failed"
		statusCode = http.StatusBadRequest
	} else if summary.SkippedCount > 0 || summary.FailedCount > 0 {
		summary.Status = "partial_success"
	}

	if fileUploadCount > 0 && totalActualFileBytes > 0 {
		log.Info(fmt.Sprintf("Total uploaded files: %d, Total size of uploaded files: %.2f MB", fileUploadCount, float64(totalActualFileBytes)/1024/1024))
	}

	middleware.WriteJson(w, statusCode, summary)
}

func TogglePinImage(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	log := middleware.GetLoggerFromContext(ctx)

	// Decode JSON body
	var body struct {
		FileUUID  string `json:"file_uuid"`
		Tagnumber int64  `json:"tagnumber"`
	}
	if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
		log.Error("Cannot decode TogglePinImage JSON: " + err.Error())
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}

	fileUUID := strings.TrimSpace(body.FileUUID)
	if fileUUID == "" {
		log.Warn("No image UUID provided in TogglePinImage body")
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}
	if fileUUID == "" {
		log.Warn("No image path provided for TogglePinImage body")
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}

	tagnumber := body.Tagnumber
	if tagnumber < 1 || tagnumber > 999999 {
		log.Warn("Invalid tag number provided in TogglePinImage body")
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}

	if err := database.TogglePinImage(ctx, &tagnumber, &fileUUID); err != nil {
		log.Error("Failed to toggle pin image: " + err.Error())
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}
	middleware.WriteJson(w, http.StatusOK, "Image pin toggled successfully")
}

func SetAllJobs(w http.ResponseWriter, req *http.Request) {
	log := middleware.GetLoggerFromContext(req.Context()).With(slog.String("func", "SetAllJobs"))

	clientBody, err := io.ReadAll(req.Body)
	if err != nil {
		log.Warn("Cannot read request body: " + err.Error())
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}

	var clientJson types.JobQueueTableRowView
	if err := json.Unmarshal(clientBody, &clientJson); err != nil {
		log.Warn("Cannot decode JSON: " + err.Error())
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}

	if clientJson.JobName == nil || strings.TrimSpace(*clientJson.JobName) == "" {
		log.Warn("Job name is missing")
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}
	if utf8.RuneCountInString(strings.TrimSpace(*clientJson.JobName)) < 1 ||
		utf8.RuneCountInString(strings.TrimSpace(*clientJson.JobName)) > 64 {
		log.Warn("Invalid job name length")
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}
	if !types.IsPrintableASCII([]byte(*clientJson.JobName)) {
		log.Warn("Non-printable ASCII characters in job name field")
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}

	if err = database.SetAllOnlineClientJobs(req.Context(), *clientJson.JobName); err != nil {
		log.Error("Failed to set all jobs: " + err.Error())
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}
	middleware.WriteJson(w, http.StatusOK, struct {
		Message string `json:"response_status"`
	}{Message: "All client jobs set successfully"})
}

func SetClientJob(w http.ResponseWriter, req *http.Request) {
	log := middleware.GetLoggerFromContext(req.Context()).With(slog.String("func", "SetClientJob"))

	clientBody, err := io.ReadAll(req.Body)
	if err != nil {
		log.Warn("Cannot read request body: " + err.Error())
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}

	var clientJson types.JobQueueTableRowView
	if err := json.Unmarshal(clientBody, &clientJson); err != nil {
		log.Warn("Cannot decode JSON: " + err.Error())
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}

	if err := types.IsTagnumberInt64Valid(clientJson.Tagnumber); err != nil {
		log.Warn("Invalid tagnumber: " + err.Error())
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}

	// Job name checks
	if clientJson.JobName == nil || strings.TrimSpace(*clientJson.JobName) == "" {
		log.Warn("Job name is missing")
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}
	if utf8.RuneCountInString(strings.TrimSpace(*clientJson.JobName)) < 1 ||
		utf8.RuneCountInString(strings.TrimSpace(*clientJson.JobName)) > 64 {
		log.Warn("Invalid job name length")
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}
	if !types.IsPrintableASCII([]byte(*clientJson.JobName)) {
		log.Warn("Non-printable ASCII characters in job name field")
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}

	if err = database.SetClientJob(req.Context(), *clientJson.Tagnumber, *clientJson.JobName); err != nil {
		log.Error("Failed to set client job: " + err.Error())
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}
	middleware.WriteJson(w, http.StatusOK, struct {
		Message string `json:"response_status"`
	}{Message: "Client job set successfully"})
}

func UpdateClientHealthCheck(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	log := middleware.GetLoggerFromContext(ctx).With(slog.String("func", "UpdateClientHealthCheck"))

	clientBody, err := io.ReadAll(req.Body)
	if err != nil {
		log.Warn("Cannot read request body: " + err.Error())
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}

	var hardwareCheckData types.ClientHealthCheck
	if err := json.Unmarshal(clientBody, &hardwareCheckData); err != nil {
		log.Warn("Cannot decode JSON: " + err.Error())
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}
	if err := types.IsTagnumberInt64Valid(&hardwareCheckData.Tagnumber); err != nil {
		log.Warn("Invalid tagnumber: " + err.Error())
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}
	// if hardwareCheckData.LastHardwareCheck == nil || hardwareCheckData.LastHardwareCheck.IsZero() {
	// 	log.Warn("Last hardware check time is missing or zero")
	// 	middleware.WriteJsonError(w, http.StatusBadRequest)
	// 	return
	// }
	if hardwareCheckData.LastHardwareCheck != nil {
		ptrTime := hardwareCheckData.LastHardwareCheck.UTC()
		hardwareCheckData.LastHardwareCheck = &ptrTime
	}

	if err = database.UpsertClientHealthCheck(ctx, &hardwareCheckData); err != nil {
		log.Error("Failed to update client last hardware check: " + err.Error())
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}
	middleware.WriteJson(w, http.StatusOK, "Client last hardware check updated successfully")
}

func SetClientHardwareData(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	log := middleware.GetLoggerFromContext(ctx).With(slog.String("func", "SetClientHardwareData"))
	clientBody, err := io.ReadAll(req.Body)
	if err != nil {
		log.Warn("Cannot read request body for SetClientHardwareData: " + err.Error())
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}
	var hardwareData *types.ClientHardwareView
	if err := json.Unmarshal(clientBody, &hardwareData); err != nil {
		log.Warn("Cannot decode SetClientHardwareData JSON: " + err.Error())
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}
	if hardwareData.Tagnumber != nil && *hardwareData.Tagnumber != 0 {
		if err := types.IsTagnumberInt64Valid(hardwareData.Tagnumber); err != nil {
			log.Warn("Invalid tagnumber: " + err.Error())
			middleware.WriteJsonError(w, http.StatusBadRequest)
			return
		}
	}
	if hardwareData.SystemSerial == nil || strings.TrimSpace(*hardwareData.SystemSerial) == "" {
		log.Warn("System serial number is missing or empty in SetClientHardwareData")
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}

	if hardwareData == nil {
		log.Warn("Empty hardware data provided in SetClientHardwareData")
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}

	if strings.TrimSpace(hardwareData.TransactionUUID) == "" {
		log.Warn("Transaction UUID is missing or nil in SetClientHardwareData")
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}
	if hardwareData.BatteryManufactureDate != nil {
		USAMatched := types.USADateRegex.MatchString(*hardwareData.BatteryManufactureDate)
		ISOMatched := types.ISODateRegex.MatchString(*hardwareData.BatteryManufactureDate)
		if !USAMatched && !ISOMatched {
			*hardwareData.BatteryManufactureDate = ""
		}
		if USAMatched {
			parsedTime, err := time.Parse("01/02/2006", *hardwareData.BatteryManufactureDate)
			if err != nil {
				log.Warn("Failed to parse battery manufacture date in MM/DD/YYYY format: " + err.Error())
				*hardwareData.BatteryManufactureDate = ""
			} else {
				*hardwareData.BatteryManufactureDate = parsedTime.Format("2006-01-02")
			}
		}
	}
	if err := database.UpdateClientHardwareData(ctx, hardwareData); err != nil {
		log.Error("Failed to update client hardware data: " + err.Error())
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}
	middleware.WriteJson(w, http.StatusOK, "Client hardware data updated successfully")
}

func SetJobQueuedAt(w http.ResponseWriter, req *http.Request) {
	log := middleware.GetLoggerFromContext(req.Context()).With(slog.String("func", "SetJobQueuedAt"))
	body, err := io.ReadAll(req.Body)
	if err != nil {
		log.Warn("Error reading request body: " + err.Error())
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}

	var reqBody types.JobQueueTableRowView
	if err := json.Unmarshal(body, &reqBody); err != nil {
		log.Warn("Error unmarshaling request body" + err.Error())
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}
	if err := types.IsTagnumberInt64Valid(reqBody.Tagnumber); err != nil {
		log.Warn(fmt.Sprintf("%v for '%s': %v", types.InvalidRequestFieldError, "tagnumber", err))
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}

	db, err := database.NewUpdateRepo()
	if err != nil {
		log.Warn("Error creating DB connection" + err.Error())
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}

	if err := db.UpdateJobQueuedAt(req.Context(), &reqBody); err != nil {
		log.Warn("DB error: " + err.Error())
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}

	middleware.WritePlainTextResponse(w, "")
}

func UploadLiveImage(w http.ResponseWriter, req *http.Request) {
	log := middleware.GetLoggerFromContext(req.Context()).With(slog.String("func", "UploadLiveImage"))
	tag := middleware.GetInt64Query(req.URL.Query(), "tagnumber")
	if tag == nil || *tag == 0 {
		log.Info("Missing tagnumber in request")
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}
	lr := &io.LimitedReader{R: req.Body, N: types.MaxLiveImageBytes + 1}
	body, err := io.ReadAll(lr)
	if err != nil {
		log.Warn("Error reading request body: " + err.Error())
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}
	if len(body) == 0 {
		log.Warn("Request body is empty")
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}
	if int64(len(body)) > types.MaxLiveImageBytes {
		log.Warn("Request body is too large")
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}
	if err := config.UpdateLiveImageBytes(*tag, body); err != nil {
		log.Warn("Error updating live image for " + strconv.Itoa(int(*tag)) + ": " + err.Error())
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}

	middleware.WriteJson(w, http.StatusOK, struct {
		Status string `json:"status"`
	}{
		Status: "success",
	})
}

func BulkUpdateInventoryLocation(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	log := middleware.GetLoggerFromContext(ctx).With(slog.String("func", "BulkUpdateInventoryLocation"))

	requestBody, err := io.ReadAll(req.Body)
	if err != nil {
		log.Warn("Cannot read request body: " + err.Error())
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}
	var bulkUpdateReq types.BulkUpdateRequest
	if err := json.Unmarshal(requestBody, &bulkUpdateReq); err != nil {
		log.Warn("Cannot decode BulkUpdateInventoryLocation JSON: " + err.Error())
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}
	if bulkUpdateReq.Location == nil || strings.TrimSpace(*bulkUpdateReq.Location) == "" || len(bulkUpdateReq.Tagnumbers) == 0 {
		log.Warn("Bulk update request is invalid")
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}
	transactionUUID, err := uuid.NewV7()
	if err != nil {
		log.Error("error generation a transaction UUID (BulkUpdateInventoryLocation)")
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}
	if transactionUUID == uuid.Nil {
		log.Error("transaction UUID in BulkUpdateInventoryLocation is nil")
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}
	transactionUUIDStr := transactionUUID.String()
	updateRepo, err := database.NewUpdateRepo()
	if err != nil {
		log.Error("No database connection available for BulkUpdateInventoryLocation")
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}
	for _, tagnumber := range bulkUpdateReq.Tagnumbers {
		if err := types.IsTagnumberInt64Valid(&tagnumber); err != nil {
			log.Warn("Invalid tagnumber in bulk update request: " + strconv.FormatInt(tagnumber, 10))
			continue
		}
		if err := updateRepo.BulkUpdateClientLocation(ctx, &transactionUUIDStr, &tagnumber, bulkUpdateReq.Location); err != nil {
			log.Error("Failed to bulk update inventory location: " + err.Error())
			middleware.WriteJsonError(w, http.StatusInternalServerError)
			return
		}
	}
	middleware.WriteJson(w, http.StatusOK, struct {
		Status       string `json:"status"`
		UpdatedCount int    `json:"updated_count"`
	}{
		Status:       "success",
		UpdatedCount: len(bulkUpdateReq.Tagnumbers),
	})
}

func ReceiveWindowsClientInfo(w http.ResponseWriter, req *http.Request) {
	log := middleware.GetLoggerFromContext(req.Context()).With(slog.String("func", "ReceiveWindowsClientInfo"))

	if err := req.ParseMultipartForm(32 << 20); err != nil {
		if os.IsTimeout(err) {
			log.Warn("Request timed out while reading multipart form: " + err.Error())
			middleware.WriteJsonError(w, http.StatusRequestTimeout)
			return
		}
		log.Warn("Error parsing multipart form: " + err.Error())
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}

	jsonFile, _, err := req.FormFile("json_file")
	if err != nil {
		log.Warn("Error retrieving JSON data provided in form: " + err.Error())
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}
	defer jsonFile.Close()

	bodyBytes, err := io.ReadAll(jsonFile)
	if err != nil {
		log.Warn("Error reading JSON file: " + err.Error())
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}
	bodyBytes = bytes.TrimSpace(bodyBytes)
	bodyBytes = bytes.TrimPrefix(bodyBytes, []byte{0xEF, 0xBB, 0xBF}) // Trim UTF-8 BOM if present
	if len(bodyBytes) == 0 {
		log.Warn("JSON file is empty")
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}

	var requestData types.WindowsUpdateRequest
	if err := json.Unmarshal(bodyBytes, &requestData); err != nil {
		log.Warn("Error unmarshaling JSON file: " + err.Error())
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}

	dto, err := requestData.ToDTO()
	if err != nil {
		log.Warn("Error creating Windows update DTO: " + err.Error())
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}

	transactionUUID, err := uuid.NewV7()
	if err != nil {
		log.Error("error generation a transaction UUID: " + err.Error())
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}
	if transactionUUID == uuid.Nil || transactionUUID.String() == "" {
		log.Error("transaction UUID is nil")
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}

	if err := database.UpdateFromWindowsJSON(req.Context(), dto, transactionUUID); err != nil {
		log.Error("Failed to update Windows client info: " + err.Error())
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}

	middleware.WriteJson(w, http.StatusOK, struct {
		Status int64  `json:"tagnumber"`
		Serial string `json:"system_serial"`
	}{
		Status: *requestData.RequestMetadata.Tagnumber,
		Serial: *requestData.RequestMetadata.SystemSerial,
	})
}

func UpdateJobStats(w http.ResponseWriter, req *http.Request) {
	log := middleware.GetLoggerFromContext(req.Context()).With(slog.String("func", "UpdateJobStats"))
	body, err := io.ReadAll(req.Body)
	if err != nil {
		log.Warn("Error reading request body: " + err.Error())
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}
	var reqBody types.UpdateJobStatsRequest
	if err := json.Unmarshal(body, &reqBody); err != nil {
		log.Warn("Error unmarshaling request body: " + err.Error())
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}

	dto, err := reqBody.ToDTO()
	if err != nil {
		log.Warn("Error converting request body to DTO: " + err.Error())
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}

	if err := database.UpsertJobStats(req.Context(), dto); err != nil {
		log.Error("Failed to update job stats: " + err.Error())
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}
	middleware.WriteJson(w, http.StatusOK, struct {
		Status string `json:"status"`
	}{
		Status: "success",
	})
}

func StoreBulkUpdateData(w http.ResponseWriter, req *http.Request) {
	log := middleware.GetLoggerFromContext(req.Context()).With(slog.String("func", "StoreBulkUpdateData"))

	body, err := io.ReadAll(req.Body)
	if err != nil {
		log.Warn("Error reading request body: " + err.Error())
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}
	var reqBody types.BulkUpdateRequest
	if err := json.Unmarshal(body, &reqBody); err != nil {
		log.Warn("Error unmarshaling request body: " + err.Error())
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}

	if reqBody.SessionID == nil || strings.TrimSpace(*reqBody.SessionID) == "" {
		log.Warn("Missing session ID in request body")
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}

	authSession, err := config.GetAuthSessionByID(*reqBody.SessionID)
	if err != nil {
		log.Error("Failed to get auth session: " + err.Error())
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}

	type BulkUpdateAttributes struct {
		Tagnumbers []int64 `json:"tagnumbers"`
		Location   *string `json:"location"`
	}

	newAuthSession := authSession
	newAuthSession.Attributes.SetAuthAttributes("bulk_update", BulkUpdateAttributes{
		Tagnumbers: reqBody.Tagnumbers,
		Location:   reqBody.Location,
	})
	if err := config.UpdateAuthSession(*reqBody.SessionID, newAuthSession); err != nil {
		log.Error("Failed to update auth session: " + err.Error())
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}

	// dto, err := reqBody.ToDTO()
	// if err != nil {
	// 	log.Warn("Error converting request body to DTO: " + err.Error())
	// 	middleware.WriteJsonError(w, http.StatusBadRequest)
	// 	return
	// }

	// if err := database.StoreBulkUpdateData(req.Context(), dto); err != nil {
	// 	log.Error("Failed to store bulk update data: " + err.Error())
	// 	middleware.WriteJsonError(w, http.StatusInternalServerError)
	// 	return
	// }
	middleware.WriteJson(w, http.StatusOK, struct {
		Status BulkUpdateAttributes `json:"bulk_update_session_data"`
	}{
		Status: BulkUpdateAttributes{
			Tagnumbers: reqBody.Tagnumbers,
			Location:   reqBody.Location,
		},
	})
}
