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
	log := middleware.GetLoggerFromContext(ctx)
	log = log.With(slog.String("func", "WebAuthEndpoint"))
	reqIP, err := middleware.GetRequestIPFromContext(ctx)
	if err != nil {
		log.Warn("Cannot retrieve request IP from context: " + err.Error())
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}

	appState, err := config.GetAppState()
	if err != nil {
		log.Warn("Cannot retrieve app state: " + err.Error())
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}
	maxLoginSizeBytes, _, _, _, _, err := appState.GetLoginFormSizeConstraint()
	if err != nil {
		log.Warn("Cannot retrieve login form constraints: " + err.Error())
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}
	req.Body = http.MaxBytesReader(w, req.Body, maxLoginSizeBytes)
	defer req.Body.Close()

	body, err := io.ReadAll(req.Body)
	if err != nil {
		if maxBytesErr, ok := errors.AsType[*http.MaxBytesError](err); ok {
			log.Warn("Login form size exceeds maximum allowed bytes: " + maxBytesErr.Error())
			middleware.WriteJsonError(w, http.StatusRequestEntityTooLarge)
			return
		}
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
	if !middleware.IsPrintableUnicode(base64Decoded) {
		log.Warn("Invalid UTF-8 in base64 data")
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}

	// Unmarshal JSON from base64 bytes
	clientFormAuthData := new(types.AuthFormData)
	if err := json.Unmarshal(base64Decoded, clientFormAuthData); err != nil {
		log.Warn("Cannot unmarshal JSON: " + err.Error())
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}

	// Validate input data
	if err := ValidateAuthFormInputSHA256(clientFormAuthData.Username, clientFormAuthData.Password); err != nil {
		log.Warn("Invalid username/password input: " + err.Error())
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}

	// Authenticate with bcrypt
	authenticated, err := CheckAuthCredentials(ctx, clientFormAuthData.Username, clientFormAuthData.Password)
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
	responseJson.ExpiresAt = time.Now().Add(authSession.SessionTTL)
	responseJson.TTL = authSession.SessionTTL

	middleware.WriteJson(w, http.StatusOK, responseJson)
}

func SetClientMemoryInfo(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	log := middleware.GetLoggerFromContext(ctx)
	log = log.With(slog.String("func", "SetClientMemoryInfo"))

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
	if !middleware.IsPrintableUnicode(requestBody) {
		log.Warn("Invalid UTF-8 in request body")
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}

	var memoryData types.MemoryData
	if err := json.Unmarshal(requestBody, &memoryData); err != nil {
		log.Warn("Cannot unmarshal JSON: " + err.Error())
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}
	if memoryData.Tagnumber == nil {
		log.Warn("Missing tag number")
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}
	if memoryData.TotalUsage == nil || memoryData.TotalCapacity == nil {
		log.Warn("Both memory usage and capacity are required")
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}

	updateRepo, err := database.NewUpdateRepo()
	if err != nil {
		log.Error("No database connection available for updating client memory info")
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}

	if err := updateRepo.UpdateClientMemoryInfo(ctx, &memoryData); err != nil {
		log.Error("Failed to update client memory info: " + err.Error())
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}
	middleware.WriteJson(w, http.StatusOK, map[string]string{"status": "success"})
}

func SetClientCPUUsage(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	log := middleware.GetLoggerFromContext(ctx)
	log = log.With(slog.String("func", "SetClientCPUUsage"))
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
	if !middleware.IsPrintableUnicode(requestBody) {
		log.Warn("Invalid UTF-8 in request body")
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}
	var cpuData types.CPUData
	if err := json.Unmarshal(requestBody, &cpuData); err != nil {
		log.Warn("Cannot unmarshal JSON: " + err.Error())
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}
	if cpuData.Tagnumber == nil || cpuData.UsagePercent == nil {
		log.Warn("Missing tag number or usage percent")
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}
	updateRepo, err := database.NewUpdateRepo()
	if err != nil {
		log.Error("No database connection available for updating client CPU usage")
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}
	if err := updateRepo.UpdateClientCPUUsage(ctx, &cpuData); err != nil {
		log.Error("Failed to update client CPU usage: " + err.Error())
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}
	middleware.WriteJson(w, http.StatusOK, map[string]string{"status": "success"})
}

func SetClientCPUTemperature(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	log := middleware.GetLoggerFromContext(ctx)
	log = log.With(slog.String("func", "SetClientCPUTemperature"))
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
	if !middleware.IsPrintableUnicode(requestBody) {
		log.Warn("Invalid UTF-8 in request body")
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}
	var cpuData types.CPUData
	if err := json.Unmarshal(requestBody, &cpuData); err != nil {
		log.Warn("Cannot unmarshal JSON: " + err.Error())
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}
	if cpuData.Tagnumber == nil || cpuData.MillidegreesC == nil {
		log.Warn("Missing tag number or temperature")
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}
	updateRepo, err := database.NewUpdateRepo()
	if err != nil {
		log.Error("No database connection available for updating client CPU temperature")
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}
	if err := updateRepo.UpdateClientCPUTemperature(ctx, &cpuData); err != nil {
		log.Error("Failed to update client CPU temperature: " + err.Error())
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}
	middleware.WriteJson(w, http.StatusOK, map[string]string{"status": "success"})
}

func InsertNewNote(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	log := middleware.GetLoggerFromContext(ctx)
	log = log.With(slog.String("func", "InsertNewNote"))
	appState, err := config.GetAppState()
	if err != nil {
		log.Warn("Cannot get app state in InsertNewNote: " + err.Error())
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}
	noteFormMaxBytes, noteTypeMinChars, noteTypeMaxChars, noteContentMinChars, noteContentMaxChars, err := appState.GetNoteConstraints()
	if err != nil {
		log.Warn("Error retrieving note constraints in InsertNewNote: " + err.Error())
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}

	lr := io.LimitReader(req.Body, noteFormMaxBytes)

	// Parse and validate note data
	var newNote types.Note
	err = json.NewDecoder(lr).Decode(&newNote)
	if err != nil {
		log.Warn("Cannot decode note JSON: " + err.Error())
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}
	defer req.Body.Close()

	if utf8.RuneCountInString(strings.TrimSpace(newNote.NoteType)) <= noteTypeMinChars || utf8.RuneCountInString(newNote.NoteType) > noteTypeMaxChars {
		log.Warn("Note type outside of valid length range, not inserting new note")
		middleware.WriteJsonError(w, http.StatusRequestEntityTooLarge)
		return
	}
	if utf8.RuneCountInString(strings.TrimSpace(newNote.Content)) < noteContentMinChars || utf8.RuneCountInString(newNote.Content) > noteContentMaxChars {
		log.Warn("Note content outside of valid length range, not inserting new note")
		middleware.WriteJsonError(w, http.StatusRequestEntityTooLarge)
		return
	}

	// Insert note into database
	insertRepo, err := database.NewUpdateRepo()
	if err != nil {
		log.Error("No database connection available for inserting new note")
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}
	curTime := time.Now()
	err = insertRepo.InsertNewNote(ctx, &curTime, &newNote.NoteType, &newNote.Content)
	if err != nil {
		log.Error("Failed to insert new note: " + err.Error())
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}
}

func InsertInventoryUpdateForm(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	log := middleware.GetLoggerFromContext(ctx)
	log = log.With(slog.String("func", "InsertInventoryUpdateForm"))

	// Parse inventory data
	appState, err := config.GetAppState()
	if err != nil {
		log.Warn("Cannot get app state in InsertInventoryUpdateForm: " + err.Error())
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}

	maxInventoryFormJsonBytes, err := appState.GetInventoryUpdateJsonConstraints()
	if err != nil {
		log.Warn("Error retrieving inventory update form size constraint: " + err.Error())
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}
	_, _, defaultFileUploadMaxTotalSize, err := appState.GetFileUploadDefaultConstraints()
	if err != nil {
		log.Warn("Error retrieving file upload constraints: " + err.Error())
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}

	totalAllowedBytes := maxInventoryFormJsonBytes + defaultFileUploadMaxTotalSize
	req.Body = http.MaxBytesReader(w, req.Body, totalAllowedBytes)
	defer req.Body.Close()

	if err := req.ParseMultipartForm(totalAllowedBytes); err != nil {
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
		if maxBytesErr, ok := errors.AsType[*http.MaxBytesError](err); ok {
			log.Warn("Request body too large: " + maxBytesErr.Error())
			middleware.WriteJsonError(w, http.StatusRequestEntityTooLarge)
			return
		}
		log.Warn("Cannot parse multipart form: " + err.Error())
		middleware.WriteJsonError(w, http.StatusRequestEntityTooLarge)
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

	jsonReader := &io.LimitedReader{R: jsonFile, N: maxInventoryFormJsonBytes + 1}
	jsonBytes, err := io.ReadAll(jsonReader)
	if err != nil {
		log.Warn("Error reading JSON data from form: " + err.Error())
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}
	if int64(len(jsonBytes)) > maxInventoryFormJsonBytes {
		log.Warn("JSON data in form exceeds maximum allowed size after reading")
		middleware.WriteJsonError(w, http.StatusRequestEntityTooLarge)
		return
	}

	var inventoryUpdate types.InventoryUpdateForm
	if err := json.Unmarshal(jsonBytes, &inventoryUpdate); err != nil {
		log.Warn("Cannot decode JSON (InsertInventoryUpdateForm): " + err.Error())
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}
	if !utf8.Valid(jsonBytes) {
		log.Warn("Invalid UTF-8 in JSON data")
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}

	// Validate and sanitize input data
	// Tag number (required, 6 numeric digits, 100000-999999)
	if err := IsTagnumberInt64Valid(inventoryUpdate.Tagnumber); err != nil {
		log.Warn("Invalid tag number provided: " + err.Error())
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}

	// System serial
	if inventoryUpdate.SystemSerial == nil {
		log.Warn("Invalid system serial provided")
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}
	serialMinChars, serialMaxChars, err := appState.GetSystemSerialConstraints()
	if err != nil {
		log.Warn("Error retrieving system serial constraints: " + err.Error())
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}
	if utf8.RuneCountInString(strings.TrimSpace(*inventoryUpdate.SystemSerial)) < serialMinChars || utf8.RuneCountInString(*inventoryUpdate.SystemSerial) > serialMaxChars {
		log.Warn("Invalid system serial length provided")
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}
	if !middleware.IsASCIIStringPrintable(*inventoryUpdate.SystemSerial) {
		log.Warn("Non-printable ASCII characters in system serial field")
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}
	*inventoryUpdate.SystemSerial = strings.TrimSpace(*inventoryUpdate.SystemSerial)

	// Location
	if inventoryUpdate.Location == nil {
		log.Warn("No location provided")
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}
	locationMinChars, locationMaxChars, err := appState.GetLocationConstraints()
	if err != nil {
		log.Warn("Error retrieving location constraints: " + err.Error())
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}
	if utf8.RuneCountInString(strings.TrimSpace(*inventoryUpdate.Location)) < locationMinChars || utf8.RuneCountInString(*inventoryUpdate.Location) > locationMaxChars {
		log.Warn("Invalid location length")
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}
	if !middleware.IsPrintableUnicodeString(*inventoryUpdate.Location) {
		log.Warn("Invalid UTF-8 in location field for inventory update")
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}
	*inventoryUpdate.Location = strings.TrimSpace(*inventoryUpdate.Location)

	// Building (optional)
	if inventoryUpdate.Building != nil {
		buildingMinChars, buildingMaxChars, err := appState.GetBuildingConstraints()
		if err != nil {
			log.Warn("Error retrieving building constraints: " + err.Error())
			middleware.WriteJsonError(w, http.StatusInternalServerError)
			return
		}
		if utf8.RuneCountInString(strings.TrimSpace(*inventoryUpdate.Building)) < buildingMinChars || utf8.RuneCountInString(*inventoryUpdate.Building) > buildingMaxChars {
			log.Warn("Invalid building length")
			middleware.WriteJsonError(w, http.StatusBadRequest)
			return
		}
		if !middleware.IsPrintableUnicodeString(*inventoryUpdate.Building) {
			log.Warn("Invalid UTF-8 in building field")
			middleware.WriteJsonError(w, http.StatusBadRequest)
			return
		}
		*inventoryUpdate.Building = strings.TrimSpace(*inventoryUpdate.Building)
	} else {
		log.Info("No building provided")
	}

	// Room (optional)
	if inventoryUpdate.Room != nil {
		roomMinChars, roomMaxChars, err := appState.GetRoomConstraints()
		if err != nil {
			log.Warn("Error retrieving room constraints: " + err.Error())
			middleware.WriteJsonError(w, http.StatusInternalServerError)
			return
		}
		if utf8.RuneCountInString(strings.TrimSpace(*inventoryUpdate.Room)) < roomMinChars || utf8.RuneCountInString(*inventoryUpdate.Room) > roomMaxChars {
			log.Warn("Invalid room length")
			middleware.WriteJsonError(w, http.StatusBadRequest)
			return
		}
		if !middleware.IsPrintableUnicodeString(*inventoryUpdate.Room) {
			log.Warn("Invalid UTF-8 in room field")
			middleware.WriteJsonError(w, http.StatusBadRequest)
			return
		}
		*inventoryUpdate.Room = strings.TrimSpace(*inventoryUpdate.Room)
	} else {
		log.Info("No room provided")
	}

	// System manufacturer (optional, min 1 char, max 24, Unicode chars)
	if inventoryUpdate.SystemManufacturer != nil {
		systemManufacturerMinChars, systemManufacturerMaxChars, err := appState.GetManufacturerConstraints()
		if err != nil {
			log.Warn("Error retrieving system manufacturer constraints: " + err.Error())
			middleware.WriteJsonError(w, http.StatusInternalServerError)
			return
		}
		if utf8.RuneCountInString(strings.TrimSpace(*inventoryUpdate.SystemManufacturer)) < systemManufacturerMinChars || utf8.RuneCountInString(*inventoryUpdate.SystemManufacturer) > systemManufacturerMaxChars {
			log.Warn("Invalid system manufacturer length")
			middleware.WriteJsonError(w, http.StatusBadRequest)
			return
		}
		if !middleware.IsPrintableUnicodeString(*inventoryUpdate.SystemManufacturer) {
			log.Warn("Non-printable Unicode characters in system manufacturer field")
			middleware.WriteJsonError(w, http.StatusBadRequest)
			return
		}
		*inventoryUpdate.SystemManufacturer = strings.TrimSpace(*inventoryUpdate.SystemManufacturer)
	} else {
		log.Info("No system manufacturer provided")
	}

	// System model (optional, min 1 char, max 64 Unicode chars)
	if inventoryUpdate.SystemModel != nil {
		systemModelMinChars, systemModelMaxChars, err := appState.GetSystemModelConstraints()
		if err != nil {
			log.Warn("Error retrieving system model constraints: " + err.Error())
			middleware.WriteJsonError(w, http.StatusInternalServerError)
			return
		}
		if utf8.RuneCountInString(strings.TrimSpace(*inventoryUpdate.SystemModel)) < systemModelMinChars || utf8.RuneCountInString(*inventoryUpdate.SystemModel) > systemModelMaxChars {
			log.Warn("Invalid system model length")
			middleware.WriteJsonError(w, http.StatusBadRequest)
			return
		}
		if !middleware.IsPrintableUnicodeString(*inventoryUpdate.SystemModel) {
			log.Warn("Non-printable Unicode characters in system model field")
			middleware.WriteJsonError(w, http.StatusBadRequest)
			return
		}
		*inventoryUpdate.SystemModel = strings.TrimSpace(*inventoryUpdate.SystemModel)
	} else {
		log.Info("No system model provided")
	}

	// Department (required, min 1 char, max 64 chars, printable ASCII only)
	if inventoryUpdate.Department == nil {
		log.Warn("No department_name provided")
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}
	departmentMinChars, departmentMaxChars, err := appState.GetDepartmentConstraints()
	if err != nil {
		log.Warn("Error retrieving department constraints: " + err.Error())
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}
	if utf8.RuneCountInString(strings.TrimSpace(*inventoryUpdate.Department)) < departmentMinChars || utf8.RuneCountInString(*inventoryUpdate.Department) > departmentMaxChars {
		log.Warn("Invalid department_name length")
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}
	if !middleware.IsASCIIStringPrintable(*inventoryUpdate.Department) {
		log.Warn("Non-printable ASCII characters in department_name field")
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}
	*inventoryUpdate.Department = strings.TrimSpace(*inventoryUpdate.Department)

	// Domain (required, min 1 char, max 64 chars)
	if inventoryUpdate.Domain == nil {
		log.Warn("No domain provided")
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}
	domainMinChars, domainMaxChars, err := appState.GetDomainConstraints()
	if err != nil {
		log.Warn("Error retrieving domain constraints: " + err.Error())
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}
	if utf8.RuneCountInString(strings.TrimSpace(*inventoryUpdate.Domain)) < domainMinChars || utf8.RuneCountInString(*inventoryUpdate.Domain) > domainMaxChars {
		log.Warn("Invalid domain length")
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}
	if !middleware.IsASCIIStringPrintable(*inventoryUpdate.Domain) {
		log.Warn("Non-printable ASCII characters in domain field")
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}
	*inventoryUpdate.Domain = strings.TrimSpace(*inventoryUpdate.Domain)

	// Property custodian (optional, min 1 char, max 64 Unicode chars)
	if inventoryUpdate.PropertyCustodian != nil {
		propertyCustodianMinChars, propertyCustodianMaxChars, err := appState.GetPropertyCustodianConstraints()
		if err != nil {
			log.Warn("Error retrieving property custodian constraints: " + err.Error())
			middleware.WriteJsonError(w, http.StatusInternalServerError)
			return
		}
		if utf8.RuneCountInString(strings.TrimSpace(*inventoryUpdate.PropertyCustodian)) < propertyCustodianMinChars || utf8.RuneCountInString(*inventoryUpdate.PropertyCustodian) > propertyCustodianMaxChars {
			log.Warn("Invalid property custodian length")
			middleware.WriteJsonError(w, http.StatusBadRequest)
			return
		}
		if !middleware.IsPrintableUnicodeString(*inventoryUpdate.PropertyCustodian) {
			log.Warn("Non-printable Unicode characters in property custodian field")
			middleware.WriteJsonError(w, http.StatusBadRequest)
			return
		}
		*inventoryUpdate.PropertyCustodian = strings.TrimSpace(*inventoryUpdate.PropertyCustodian)
	}

	// Acquired date, optional, process as UTC
	if inventoryUpdate.AcquiredDate != nil {
		acquiredDateUTC := inventoryUpdate.AcquiredDate.UTC()
		inventoryUpdate.AcquiredDate = &acquiredDateUTC
	} else {
		log.Info("No acquired date provided")
	}

	// Retired date, optional, process as UTC
	if inventoryUpdate.RetiredDate != nil {
		retiredDateUTC := inventoryUpdate.RetiredDate.UTC()
		inventoryUpdate.RetiredDate = &retiredDateUTC
	} else {
		log.Info("No retired date provided")
	}

	// Broken (optional, bool)
	if inventoryUpdate.Broken == nil {
		log.Info("No is_broken bool value provided")
	}

	// Disk removed (optional, bool)
	if inventoryUpdate.DiskRemoved == nil {
		log.Info("No disk_removed bool value provided")
	}

	// Last hardware check (optional, process as UTC)
	if inventoryUpdate.LastHardwareCheck != nil {
		lastHardwareCheckUTC := inventoryUpdate.LastHardwareCheck.UTC()
		inventoryUpdate.LastHardwareCheck = &lastHardwareCheckUTC
	} else {
		log.Info("No last_hardware_check date provided")
	}

	// Status (required, min 1, max 24, ASCII printable chars only)
	if inventoryUpdate.ClientStatus == nil {
		log.Warn("No status provided")
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}
	clientStatusMinChars, clientStatusMaxChars, err := appState.GetClientStatusConstraints()
	if err != nil {
		log.Warn("Error retrieving client status constraints: " + err.Error())
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}
	if utf8.RuneCountInString(strings.TrimSpace(*inventoryUpdate.ClientStatus)) < clientStatusMinChars || utf8.RuneCountInString(*inventoryUpdate.ClientStatus) > clientStatusMaxChars {
		log.Warn("Invalid status length")
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}
	if !middleware.IsASCIIStringPrintable(*inventoryUpdate.ClientStatus) {
		log.Warn("Non-printable ASCII characters in status field")
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}
	*inventoryUpdate.ClientStatus = strings.TrimSpace(*inventoryUpdate.ClientStatus)

	// Checkout bool (optional)
	checkoutDateMandatory, returnDateMandatory, checkoutBoolMandatory, err := appState.GetCheckoutConstraints()
	if err != nil {
		log.Warn("Error retrieving checkout constraints: " + err.Error())
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}

	// Checkout date (optional, process as UTC)
	if checkoutDateMandatory && inventoryUpdate.CheckoutDate == nil {
		log.Warn("No checkout_date provided, not updating inventory entry.")
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}
	if inventoryUpdate.CheckoutDate != nil {
		checkoutDateUTC := inventoryUpdate.CheckoutDate.UTC()
		inventoryUpdate.CheckoutDate = &checkoutDateUTC
	} else {
		log.Info("No checkout_date provided")
	}

	// Return date (optional, process as UTC)
	if returnDateMandatory && inventoryUpdate.ReturnDate == nil {
		log.Warn("No return_date provided, not updating inventory entry.")
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}
	if inventoryUpdate.ReturnDate != nil {
		returnDateUTC := inventoryUpdate.ReturnDate.UTC()
		inventoryUpdate.ReturnDate = &returnDateUTC
	} else {
		log.Info("No return_date provided")
	}

	if checkoutBoolMandatory && inventoryUpdate.CheckoutBool == nil {
		log.Warn("No is_checked_out bool value provided, not updating inventory entry.")
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}

	// Note (optional)
	if inventoryUpdate.Note != nil {
		noteMinChars, noteMaxChars, err := appState.GetClientNoteConstraints()
		if err != nil {
			log.Warn("Error retrieving client note constraints: " + err.Error())
			middleware.WriteJsonError(w, http.StatusInternalServerError)
			return
		}
		if utf8.RuneCountInString(strings.TrimSpace(*inventoryUpdate.Note)) < noteMinChars || utf8.RuneCountInString(*inventoryUpdate.Note) > noteMaxChars {
			log.Warn("Invalid note length")
			middleware.WriteJsonError(w, http.StatusBadRequest)
			return
		}
		if !middleware.IsPrintableUnicodeString(*inventoryUpdate.Note) {
			log.Warn("Non-printable characters in note field")
			middleware.WriteJsonError(w, http.StatusBadRequest)
			return
		}
		*inventoryUpdate.Note = strings.TrimSpace(*inventoryUpdate.Note)
	} else {
		log.Info("No note provided")
	}

	// File upload part of form:
	if req.MultipartForm == nil || req.MultipartForm.File == nil {
		log.Info("File upload part of inventory update is nil, continuing")
	}
	files := req.MultipartForm.File["inventory-file-input"]

	// Generate transaction UUID to share between multiple DB tables
	transactionUUID, err := uuid.NewUUID()
	if err != nil {
		log.Error("error generation a transaction UUID (InsertInventoryUpdateForm)")
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}
	if transactionUUID == uuid.Nil {
		log.Error("transaction UUID in InsertInventoryUpdateForm is nil")
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}

	// Establish DB connection before opening files
	updateRepo, err := database.NewUpdateRepo()
	if err != nil {
		log.Error("No database connection available")
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}

	minImgFileSize, maxImgFileSize, maxImgFileCount, acceptedImageExtensionsAndMimeTypes, err := appState.GetFileUploadImageConstraints()
	if err != nil {
		log.Warn("Error getting file upload image constraints: " + err.Error())
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}
	minVideoFileSize, maxVideoFileSize, maxVideoFileCount, acceptedVideoExtensionsAndMimeTypes, err := appState.GetFileUploadVideoConstraints()
	if err != nil {
		log.Warn("Error getting file upload video constraints: " + err.Error())
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}

	var totalImageFileCount int
	var totalImageUploadSize int64
	var totalVideoFileCount int
	var totalVideoUploadSize int64
	// var totalInvalidFileCount int = 2
	// var totalInvalidUploadSize int64 = 1 << 10
	for _, fileHeader := range files {
		var manifest types.ImageManifest

		if !middleware.IsPrintableUnicodeString(fileHeader.Filename) {
			log.Warn("Non-printable characters in uploaded file name: " + fileHeader.Filename)
			middleware.WriteJsonError(w, http.StatusBadRequest)
			return
		}

		fileHeader.Filename = filepath.Join(fileHeader.Filename)
		fileHeader.Filename = filepath.Clean(fileHeader.Filename)

		createNecessaryDirs := func(tagnumber int64) (string, error) {
			// Create directories if not existing
			imageDirectoryPath := filepath.Join("./inventory-images", fmt.Sprintf("%06d", tagnumber))
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
			continue
		}

		lr := &io.LimitedReader{R: file, N: maxImgFileSize + maxVideoFileSize + 1}
		fileBytes, err := io.ReadAll(lr)
		file.Close()
		if err != nil {
			if maxBytesErr, ok := errors.AsType[*http.MaxBytesError](err); ok {
				log.Warn("Uploaded file '" + fileHeader.Filename + "' size exceeds maximum allowed size: " + maxBytesErr.Error())
				middleware.WriteJsonError(w, http.StatusRequestEntityTooLarge)
				return
			}
			log.Warn("Failed to read uploaded file '" + fileHeader.Filename + "': " + err.Error())
			continue
		}

		// File size
		fileSize := int64(len(fileBytes))
		manifest.FileSize = &fileSize

		// MIME type detection
		mimeType := http.DetectContentType(fileBytes)
		if mimeType == "application/octet-stream" {
			log.Warn("Unknown MIME type for file '" + fileHeader.Filename + "'")
			middleware.WriteJsonError(w, http.StatusUnsupportedMediaType)
			return
		}
		manifest.MimeType = &mimeType
		ext := strings.ToLower(filepath.Ext(fileHeader.Filename))

		if acceptedImageExtensionsAndMimeTypes[ext] != mimeType && acceptedVideoExtensionsAndMimeTypes[ext] != mimeType {
			log.Warn("Unsupported file type for file '" + fileHeader.Filename + "': detected MIME type '" + mimeType + "' does not match expected MIME type for file extension '" + ext + "'")
			middleware.WriteJsonError(w, http.StatusUnsupportedMediaType)
			return
		}

		// Get upload timestamp
		fileTimeStamp := time.Now()
		timeUTC := fileTimeStamp.UTC()
		manifest.Time = &timeUTC

		// Generate unique file name
		fileTimeStampFormatted := fileTimeStamp.Format("2006-01-02-150405")
		fileUUID := uuid.New().String()
		fileName := fileTimeStampFormatted + "-" + fileUUID + ext
		manifest.FileName = &fileName
		manifest.UUID = &fileUUID

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
		// fileHashString := fmt.Sprintf("%x", fileHashBytes)
		manifest.SHA256Hash = &fileHashBytes

		selectRepo, err := database.NewSelectRepo()
		if err != nil {
			log.Error("No database connection available for retrieving existing file hashes")
			middleware.WriteJsonError(w, http.StatusInternalServerError)
			return
		}
		hashes, err := selectRepo.GetFileHashesFromTag(ctx, inventoryUpdate.Tagnumber)
		if err != nil {
			log.Error("Failed to get file hashes from tag '" + strconv.FormatInt(*inventoryUpdate.Tagnumber, 10) + "': " + err.Error())
			middleware.WriteJsonError(w, http.StatusInternalServerError)
			return
		}
		hashFound := false
		for _, hash := range hashes {
			if bytes.Equal(fileHashBytes, hash) {
				hashFound = true
				break
			}
		}
		if hashFound {
			log.Warn("Duplicate file upload detected for tag '" + strconv.FormatInt(*inventoryUpdate.Tagnumber, 10) + "': file '" + fileHeader.Filename + "' (" + fmt.Sprintf("%x", fileHashBytes) + ") has same hash as existing file, skipping")
			continue
		}

		if acceptedImageExtensionsAndMimeTypes[ext] == mimeType { // Image file processing
			if totalImageFileCount >= maxImgFileCount {
				log.Warn("Number of uploaded image files exceeds maximum allowed: " + strconv.Itoa(totalImageFileCount))
				middleware.WriteJsonError(w, http.StatusRequestEntityTooLarge)
				return
			}
			if fileSize > maxImgFileSize {
				log.Warn("Uploaded image file '" + fileHeader.Filename + "' too large (" + strconv.FormatInt(int64(fileSize), 10) + " bytes)")
				middleware.WriteJsonError(w, http.StatusRequestEntityTooLarge)
				return
			}
			if fileSize < minImgFileSize {
				log.Warn("Uploaded image file too small: " + fileHeader.Filename + " (" + strconv.FormatInt(int64(fileSize), 10) + " bytes)")
				middleware.WriteJsonError(w, http.StatusRequestEntityTooLarge)
				return
			}
			// Create reader (stream) for image decoding
			imageReader := bytes.NewReader(fileBytes)

			// Rewind and decode image to get image.Image
			_, err = imageReader.Seek(0, io.SeekStart)
			if err != nil {
				log.Error("Failed to seek to start of uploaded image '" + fileHeader.Filename + "': " + err.Error())
				continue
			}
			decodedImage, _, err := image.Decode(imageReader)
			if err != nil {
				log.Error("Failed to decode uploaded image '" + fileHeader.Filename + "': " + err.Error())
				continue
			}

			// Rewind and decode image to get image config
			_, err = imageReader.Seek(0, io.SeekStart)
			if err != nil {
				log.Error("Failed to seek to start of uploaded image '" + fileHeader.Filename + "': " + err.Error())
				continue
			}
			decodedImageConfig, _, err := image.DecodeConfig(imageReader)
			if err != nil {
				log.Error("Failed to decode uploaded image config '" + fileHeader.Filename + "': " + err.Error())
				continue
			}
			resX := int64(decodedImageConfig.Width)
			manifest.ResolutionX = &resX
			resY := int64(decodedImageConfig.Height)
			manifest.ResolutionY = &resY

			// Generate jpeg thumbnail
			imageDirectoryPath, err := createNecessaryDirs(*inventoryUpdate.Tagnumber)
			if err != nil {
				log.Error("Failed to create necessary directories for thumbnail of '" + fileHeader.Filename + "': " + err.Error())
				middleware.WriteJsonError(w, http.StatusInternalServerError)
				return
			}
			strippedFileName := strings.TrimSuffix(fileName, filepath.Ext(fileName))
			fullThumbnailPath := filepath.Join(imageDirectoryPath, strippedFileName+"-thumbnail.jpeg")
			thumbnailFile, err := os.Create(fullThumbnailPath)
			if err != nil {
				log.Error("Failed to create thumbnail file '" + fullThumbnailPath + "': " + err.Error())
				middleware.WriteJsonError(w, http.StatusInternalServerError)
				return
			}
			if err := os.Chmod(fullThumbnailPath, 0644); err != nil {
				log.Error("Failed to set permissions for thumbnail file '" + fullThumbnailPath + "': " + err.Error())
				middleware.WriteJsonError(w, http.StatusInternalServerError)
				return
			}
			if err := jpeg.Encode(thumbnailFile, decodedImage, &jpeg.Options{Quality: 50}); err != nil {
				log.Error("Failed to encode thumbnail image '" + fullThumbnailPath + "': " + err.Error())
				middleware.WriteJsonError(w, http.StatusInternalServerError)
				return
			}
			_ = thumbnailFile.Close()
			manifest.ThumbnailFilePath = &fullThumbnailPath
			totalImageUploadSize += fileSize
			totalImageFileCount++
		} else if acceptedVideoExtensionsAndMimeTypes[ext] == mimeType { // Video file processing
			if totalVideoFileCount >= maxVideoFileCount {
				log.Warn("Number of uploaded video files exceeds maximum allowed (InsertInventoryUpdateForm): " + strconv.Itoa(totalVideoFileCount))
				middleware.WriteJsonError(w, http.StatusRequestEntityTooLarge)
				return
			}
			if fileSize > maxVideoFileSize {
				log.Warn("Uploaded video file too large (InsertInventoryUpdateForm) (" + strconv.FormatInt(int64(fileSize), 10) + " bytes)")
				middleware.WriteJsonError(w, http.StatusRequestEntityTooLarge)
				return
			}
			if fileSize < minVideoFileSize {
				log.Warn("Uploaded video file too small (InsertInventoryUpdateForm): " + fileHeader.Filename + " (" + strconv.FormatInt(int64(fileSize), 10) + " bytes)")
				middleware.WriteJsonError(w, http.StatusRequestEntityTooLarge)
				return
			}
			totalVideoFileCount++
			totalVideoUploadSize += fileSize
		} else {
			log.Warn("Unsupported MIME type for '" + fileHeader.Filename + "' (InsertInventoryUpdateForm): MIME Type: " + mimeType)
			middleware.WriteJsonError(w, http.StatusUnsupportedMediaType)
			// totalInvalidFileCount++
			// totalInvalidUploadSize += fileSize
			return
		}

		imageDirectoryPath, err := createNecessaryDirs(*inventoryUpdate.Tagnumber)
		if err != nil {
			log.Error("Failed to create necessary directories for '" + fileHeader.Filename + "': " + err.Error())
			middleware.WriteJsonError(w, http.StatusInternalServerError)
			return
		}
		fullFilePath := filepath.Join(imageDirectoryPath, fileName)
		if err := os.WriteFile(fullFilePath, fileBytes, 0640); err != nil {
			log.Error("Failed to save uploaded file '" + fullFilePath + "': " + err.Error())
			middleware.WriteJsonError(w, http.StatusInternalServerError)
			return
		}
		manifest.FilePath = &fullFilePath

		// Close the uploaded file, not needed anymore
		_ = file.Close()

		// Insert image metadata into database
		manifest.Tagnumber = inventoryUpdate.Tagnumber
		manifest.Hidden = new(bool)
		*manifest.Hidden = false
		manifest.Pinned = new(bool)
		*manifest.Pinned = false

		if err := updateRepo.UpdateClientImages(ctx, transactionUUID, &manifest); err != nil {
			log.Error("Failed to update inventory image data for '" + fullFilePath + "': " + err.Error())
			middleware.WriteJsonError(w, http.StatusInternalServerError)
			return
		}
		log.Info(fmt.Sprintf("Uploaded file '%s', Size: %.2f MB, MIME Type: %s", fileName, float64(*manifest.FileSize)/1024/1024, mimeType))
		_ = file.Close()
	}
	fileUploadCount := totalImageFileCount + totalVideoFileCount
	totalActualFileBytes := totalImageUploadSize + totalVideoUploadSize
	if fileUploadCount > 0 && totalActualFileBytes > 0 {
		log.Info(fmt.Sprintf("Total uploaded files: %d, Total size of uploaded files: %.2f MB", fileUploadCount, float64(totalActualFileBytes)/1024/1024))
	}

	// Update db
	if err := updateRepo.InsertInventoryUpdateForm(ctx, transactionUUID, &inventoryUpdate); err != nil {
		log.Error("Failed to update inventory data: " + err.Error())
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}

	var jsonResponse = struct {
		Tagnumber int64  `json:"tagnumber"`
		Message   string `json:"message"`
	}{
		Tagnumber: *inventoryUpdate.Tagnumber,
		Message:   "update successful",
	}

	middleware.WriteJson(w, http.StatusOK, jsonResponse)
}

func TogglePinImage(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	log := middleware.GetLoggerFromContext(ctx)

	// Decode JSON body
	var body struct {
		UUID      string `json:"uuid"`
		Tagnumber int64  `json:"tagnumber"`
	}
	if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
		log.Error("Cannot decode TogglePinImage JSON: " + err.Error())
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}

	uuid := strings.TrimSpace(body.UUID)
	if uuid == "" {
		log.Warn("No image UUID provided in TogglePinImage body")
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}
	if uuid == "" {
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

	updateRepo, err := database.NewUpdateRepo()
	if err != nil {
		log.Error("No database connection available for TogglePinImage")
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}
	if err := updateRepo.TogglePinImage(ctx, &tagnumber, &uuid); err != nil {
		log.Error("Failed to toggle pin image: " + err.Error())
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}
	middleware.WriteJson(w, http.StatusOK, "Image pin toggled successfully")
}

func SetClientBatteryHealth(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	log := middleware.GetLoggerFromContext(ctx)
	requestQueries, err := middleware.GetRequestQueryFromContext(ctx)
	if err != nil {
		log.Warn("Error retrieving URL queries from context for SetClientBatteryHealth: " + err.Error())
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}
	if req.Method != http.MethodPost {
		log.Warn("Invalid HTTP method for SetClientBatteryHealth")
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}
	uuid := strings.TrimSpace(requestQueries.Get("uuid"))
	if uuid == "" {
		log.Warn("No UUID provided for SetClientBatteryHealth")
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}
	var body struct {
		BatteryHealth int64 `json:"battery_health"`
	}

	if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
		log.Warn("Cannot decode SetClientBatteryHealth JSON: " + err.Error())
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}
	// if body.BatteryHealth < 0 || body.BatteryHealth > 100 {
	// 	log.Warn("Invalid battery health percentage provided for SetClientBatteryHealth")
	// 	middleware.WriteJsonError(w, http.StatusBadRequest)
	// 	return
	// }
	updateRepo, err := database.NewUpdateRepo()
	if err != nil {
		log.Error("No database connection available for SetClientBatteryHealth")
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}
	if err = updateRepo.SetClientBatteryHealth(ctx, &uuid, &body.BatteryHealth); err != nil {
		log.Error("Failed to set client battery health: " + err.Error())
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}
	middleware.WriteJson(w, http.StatusOK, "Battery health updated successfully")
}

func SetAllJobs(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	log := middleware.GetLoggerFromContext(ctx)

	if req.Method != http.MethodPost {
		log.Warn("Invalid HTTP method for SetAllJobs")
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}

	clientBody, err := io.ReadAll(req.Body)
	if err != nil {
		log.Warn("Cannot read request body for SetAllJobs: " + err.Error())
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}

	var clientJson types.AllJobs
	if err := json.Unmarshal(clientBody, &clientJson); err != nil {
		log.Warn("Cannot decode SetAllJobs JSON: " + err.Error())
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}

	updateRepo, err := database.NewUpdateRepo()
	if err != nil {
		log.Error("No database connection available for SetAllJobs")
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}
	if err = updateRepo.SetAllOnlineClientJobs(ctx, &clientJson); err != nil {
		log.Error("Failed to set all jobs: " + err.Error())
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}
	middleware.WriteJson(w, http.StatusOK, "All jobs set successfully")
}

func SetClientJob(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	log := middleware.GetLoggerFromContext(ctx)
	log = log.With(slog.String("func", "SetClientJob"))

	appState, err := config.GetAppState()
	if err != nil {
		log.Warn("Error retrieving application state: " + err.Error())
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}
	_, _, maxReqSize, err := appState.GetFileUploadDefaultConstraints()
	if err != nil {
		log.Warn("Error retrieving file upload default constraints: " + err.Error())
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}

	req.Body = http.MaxBytesReader(w, req.Body, maxReqSize)
	clientBody, err := io.ReadAll(req.Body)
	if err != nil {
		log.Warn("Cannot read request body: " + err.Error())
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}

	var clientJson types.JobQueueTableRow
	if err := json.Unmarshal(clientBody, &clientJson); err != nil {
		log.Warn("Cannot decode request JSON: " + err.Error())
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}

	if err := IsTagnumberInt64Valid(clientJson.Tagnumber); err != nil {
		log.Warn("Invalid tagnumber: " + err.Error())
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}
	if clientJson.JobName == nil {
		log.Warn("Job name is missing")
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}
	if utf8.RuneCountInString(strings.TrimSpace(*clientJson.JobName)) < 1 || utf8.RuneCountInString(strings.TrimSpace(*clientJson.JobName)) > 64 {
		log.Warn("Invalid job name length")
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}

	if !middleware.IsASCIIStringPrintable(*clientJson.JobName) {
		log.Warn("Non-printable ASCII characters in job name field")
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}

	updateRepo, err := database.NewUpdateRepo()
	if err != nil {
		log.Error("No database connection available for SetClientJob")
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}
	if err = updateRepo.SetClientJob(ctx, clientJson.Tagnumber, clientJson.JobName); err != nil {
		log.Error("Failed to set client job: " + err.Error())
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}
	middleware.WriteJson(w, http.StatusOK, "Client job set successfully")
}
