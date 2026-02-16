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
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"
	config "uit-toolbox/config"
	"uit-toolbox/database"
	middleware "uit-toolbox/middleware"
	"unicode/utf8"

	"github.com/google/uuid"
)

type AuthFormData struct {
	Username      string `json:"username"`
	Password      string `json:"password"`
	ReturnedToken string `json:"token,omitempty"`
}

type RemoteTable struct {
	Tagnumber         *int       `sql:"tagnumber"`
	JobQueued         *string    `sql:"job_queued"`
	JobQueuedPosition *int       `sql:"job_queued_position"`
	JobActive         *bool      `sql:"job_queued_position"`
	CloneMode         *string    `sql:"clone_mode"`
	EraseMode         *string    `sql:"erase_mode"`
	LastJobTime       *time.Time `sql:"last_job_time"`
	Present           *time.Time `sql:"present"`
	PresentBool       *bool      `sql:"present_bool"`
	Status            *string    `sql:"status"`
	KernelUpdated     *bool      `sql:"kernel_updated"`
	BatteryCharge     *int       `sql:"battery_charge"`
	BatteryStatus     *string    `sql:"battery_status"`
	Uptime            *int       `sql:"uptime"`
	CpuTemp           *int       `sql:"cpu_temp"`
	DiskTemp          *int       `sql:"disk_temp"`
	MaxDiskTemp       *int       `sql:"max_disk_temp"`
	WattsNow          *int       `sql:"watts_now"`
	NetworkSpeed      *int       `sql:"network_speed"`
}

func WebAuthEndpoint(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	log := middleware.GetLoggerFromContext(ctx)
	requestIP, err := middleware.GetRequestIPFromContext(ctx)
	if err != nil {
		log.HTTPWarning(req, "Error retrieving request IP from context (WebAuthEndpoint): "+err.Error())
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}

	appState, err := config.GetAppState()
	if err != nil {
		log.HTTPWarning(req, "Cannot get app state in WebAuthEndpoint: "+err.Error())
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}
	maxLoginSizeBytes, _, _, _, _, err := appState.GetLoginFormSizeConstraint()
	if err != nil {
		log.HTTPWarning(req, "Error retrieving login form constraints: "+err.Error())
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}
	req.Body = http.MaxBytesReader(w, req.Body, maxLoginSizeBytes)
	defer req.Body.Close()

	body, err := io.ReadAll(req.Body)
	if err != nil {
		var maxBytesErr *http.MaxBytesError
		if errors.As(err, &maxBytesErr) {
			log.HTTPWarning(req, "Login form size exceeds maximum allowed: "+err.Error())
			middleware.WriteJsonError(w, http.StatusRequestEntityTooLarge)
			return
		}
		log.HTTPWarning(req, "Cannot read request body: "+err.Error())
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}

	decoded, err := base64.RawURLEncoding.DecodeString(strings.TrimSpace(string(body)))
	if err != nil {
		log.HTTPWarning(req, "Invalid base64 encoding: "+err.Error())
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}

	if !utf8.Valid(decoded) {
		log.HTTPWarning(req, "Invalid UTF-8 in decoded data")
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}

	var clientFormAuthData AuthFormData
	if err := json.Unmarshal(decoded, &clientFormAuthData); err != nil {
		log.HTTPWarning(req, "Invalid JSON structure in WebAuthEndpoint: "+err.Error())
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}

	// Validate input data
	if err := ValidateAuthFormInputSHA256(clientFormAuthData.Username, clientFormAuthData.Password); err != nil {
		log.HTTPWarning(req, "Invalid auth input: "+err.Error())
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}

	// Authenticate with bcrypt
	authenticated, err := CheckAuthCredentials(ctx, clientFormAuthData.Username, clientFormAuthData.Password)
	if err != nil || !authenticated {
		log.HTTPInfo(req, "Authentication failed: "+err.Error())
		middleware.WriteJsonError(w, http.StatusUnauthorized)
		return
	}

	sessionID, basicToken, bearerToken, csrfToken, err := config.CreateAuthSession(requestIP)
	if err != nil {
		log.HTTPError(req, "Failed to generate tokens: "+err.Error())
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}

	sessionCount := config.GetAuthSessionCount()
	log.HTTPInfo(req, "New auth session created. Total sessions: "+strconv.Itoa(int(sessionCount)))

	sessionIDCookie, basicTokenCookie, bearerTokenCookie, csrfTokenCookie := middleware.GetAuthCookiesForResponse(sessionID, basicToken, bearerToken, csrfToken, 20*time.Minute)

	http.SetCookie(w, sessionIDCookie)
	http.SetCookie(w, basicTokenCookie)
	http.SetCookie(w, bearerTokenCookie)
	http.SetCookie(w, csrfTokenCookie)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	returnedJson, err := json.Marshal(&AuthFormData{ReturnedToken: bearerToken})
	if err != nil {
		log.HTTPError(req, "Failed to marshal JSON response: "+err.Error())
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}
	w.Write(returnedJson)
}

func InsertNewNote(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	log := middleware.GetLoggerFromContext(ctx)

	appState, err := config.GetAppState()
	if err != nil {
		log.HTTPWarning(req, "Cannot get app state in InsertNewNote: "+err.Error())
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}
	noteFormMaxBytes, noteTypeMinChars, noteTypeMaxChars, noteContentMinChars, noteContentMaxChars, err := appState.GetNoteConstraints()
	if err != nil {
		log.HTTPWarning(req, "Error retrieving note constraints in InsertNewNote: "+err.Error())
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}

	lr := io.LimitReader(req.Body, noteFormMaxBytes)

	// Parse and validate note data
	var newNote database.Note
	err = json.NewDecoder(lr).Decode(&newNote)
	if err != nil {
		log.HTTPWarning(req, "Cannot decode note JSON: "+err.Error())
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}
	defer req.Body.Close()

	if utf8.RuneCountInString(strings.TrimSpace(newNote.NoteType)) <= noteTypeMinChars || utf8.RuneCountInString(newNote.NoteType) > noteTypeMaxChars {
		log.HTTPWarning(req, "Note type outside of valid length range, not inserting new note")
		middleware.WriteJsonError(w, http.StatusRequestEntityTooLarge)
		return
	}
	if utf8.RuneCountInString(strings.TrimSpace(newNote.Content)) < noteContentMinChars || utf8.RuneCountInString(newNote.Content) > noteContentMaxChars {
		log.HTTPWarning(req, "Note content outside of valid length range, not inserting new note")
		middleware.WriteJsonError(w, http.StatusRequestEntityTooLarge)
		return
	}

	// Insert note into database
	insertRepo, err := database.NewUpdateRepo()
	if err != nil {
		log.HTTPError(req, "No database connection available for inserting new note")
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}
	curTime := time.Now()
	err = insertRepo.InsertNewNote(ctx, &curTime, &newNote.NoteType, &newNote.Content)
	if err != nil {
		log.HTTPError(req, "Failed to insert new note: "+err.Error())
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}
}

func InsertInventoryUpdateForm(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	log := middleware.GetLoggerFromContext(ctx)

	// Parse inventory data
	appState, err := config.GetAppState()
	if err != nil {
		log.HTTPWarning(req, "Cannot get app state in InsertInventoryUpdateForm: "+err.Error())
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}

	maxInventoryFormJsonBytes, err := appState.GetInventoryUpdateFormConstraints()
	if err != nil {
		log.HTTPWarning(req, "Error retrieving inventory update form size constraint: "+err.Error())
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}
	maxFileUploadTotalBytes, minFileUploadBytes, maxFileUploadBytes, maxFileUploadCount, err := appState.GetFileUploadSizeConstraints()
	if err != nil {
		log.HTTPWarning(req, "Error retrieving file upload size constraints: "+err.Error())
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}

	totalAllowedBytes := maxInventoryFormJsonBytes + maxFileUploadTotalBytes
	req.Body = http.MaxBytesReader(w, req.Body, totalAllowedBytes)
	defer req.Body.Close()

	if err := req.ParseMultipartForm(totalAllowedBytes); err != nil {
		var maxBytesErr *http.MaxBytesError
		if errors.Is(err, http.ErrNotMultipart) {
			log.HTTPWarning(req, "Request body is not multipart form data: "+err.Error())
			middleware.WriteJsonError(w, http.StatusBadRequest)
			return
		}
		if errors.Is(err, http.ErrMissingBoundary) {
			log.HTTPWarning(req, "Multipart form data missing boundary: "+err.Error())
			middleware.WriteJsonError(w, http.StatusBadRequest)
			return
		}
		if errors.As(err, &maxBytesErr) {
			log.HTTPWarning(req, "Request body too large: "+err.Error())
			middleware.WriteJsonError(w, http.StatusRequestEntityTooLarge)
			return
		}
		log.HTTPWarning(req, "Cannot parse multipart form: "+err.Error())
		middleware.WriteJsonError(w, http.StatusRequestEntityTooLarge)
		return
	}

	// json part
	jsonFile, _, err := req.FormFile("json")
	if err != nil {
		log.HTTPWarning(req, "Error retrieving JSON data provided in form: "+err.Error())
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}
	defer jsonFile.Close()

	jsonReader := &io.LimitedReader{R: jsonFile, N: maxInventoryFormJsonBytes}
	jsonBytes, err := io.ReadAll(jsonReader)
	if err != nil {
		var maxBytesErr *http.MaxBytesError
		if errors.As(err, &maxBytesErr) {
			log.HTTPWarning(req, "JSON data in form exceeds maximum allowed size: "+err.Error())
			middleware.WriteJsonError(w, http.StatusRequestEntityTooLarge)
			return
		}
		log.HTTPWarning(req, "Error reading JSON data from form: "+err.Error())
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}

	if int64(len(jsonBytes)) > maxInventoryFormJsonBytes {
		log.HTTPWarning(req, "JSON data in form exceeds maximum allowed size after reading")
		middleware.WriteJsonError(w, http.StatusRequestEntityTooLarge)
		return
	}

	var inventoryUpdate database.InventoryUpdateForm
	if err := json.NewDecoder(jsonFile).Decode(&inventoryUpdate); err != nil {
		log.HTTPWarning(req, "Cannot decode JSON for InsertInventoryUpdateForm: "+err.Error())
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}
	if !utf8.Valid(jsonBytes) {
		log.HTTPWarning(req, "Invalid UTF-8 in JSON data for InsertInventoryUpdateForm")
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}

	// Validate and sanitize input data

	// Tag number (required, 6 numeric digits, 100000-999999)
	if err := IsTagnumberInt64Valid(inventoryUpdate.Tagnumber); err != nil {
		log.HTTPWarning(req, "Invalid tag number provided for InsertInventoryUpdateForm: "+err.Error())
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}

	// System serial
	if inventoryUpdate.SystemSerial == nil {
		log.HTTPWarning(req, "Invalid system serial provided for InsertInventoryUpdateForm")
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}
	serialMinChars, serialMaxChars, err := appState.GetSystemSerialConstraints()
	if err != nil {
		log.HTTPWarning(req, "Error retrieving system serial constraints: "+err.Error())
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}
	if utf8.RuneCountInString(strings.TrimSpace(*inventoryUpdate.SystemSerial)) < serialMinChars || utf8.RuneCountInString(*inventoryUpdate.SystemSerial) > serialMaxChars {
		log.HTTPWarning(req, "Invalid system serial length provided for InsertInventoryUpdateForm")
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}
	if !middleware.IsASCIIStringPrintable(*inventoryUpdate.SystemSerial) {
		log.HTTPWarning(req, "Non-printable ASCII characters in system serial field for InsertInventoryUpdateForm")
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}
	*inventoryUpdate.SystemSerial = strings.TrimSpace(*inventoryUpdate.SystemSerial)

	// Location
	if inventoryUpdate.Location == nil {
		log.HTTPWarning(req, "No location provided for InsertInventoryUpdateForm")
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}
	locationMinChars, locationMaxChars, err := appState.GetLocationConstraints()
	if err != nil {
		log.HTTPWarning(req, "Error retrieving location constraints: "+err.Error())
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}
	if utf8.RuneCountInString(strings.TrimSpace(*inventoryUpdate.Location)) < locationMinChars || utf8.RuneCountInString(*inventoryUpdate.Location) > locationMaxChars {
		log.HTTPWarning(req, "Invalid location length for InsertInventoryUpdateForm")
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}
	if !middleware.IsPrintableUnicodeString(*inventoryUpdate.Location) {
		log.HTTPWarning(req, "Invalid UTF-8 in location field for inventory update")
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}
	*inventoryUpdate.Location = strings.TrimSpace(*inventoryUpdate.Location)

	// Building (optional)
	if inventoryUpdate.Building != nil {
		buildingMinChars, buildingMaxChars, err := appState.GetBuildingConstraints()
		if err != nil {
			log.HTTPWarning(req, "Error retrieving building constraints: "+err.Error())
			middleware.WriteJsonError(w, http.StatusInternalServerError)
			return
		}
		if utf8.RuneCountInString(strings.TrimSpace(*inventoryUpdate.Building)) < buildingMinChars || utf8.RuneCountInString(*inventoryUpdate.Building) > buildingMaxChars {
			log.HTTPWarning(req, "Invalid building length for inventory update")
			middleware.WriteJsonError(w, http.StatusBadRequest)
			return
		}
		if !middleware.IsPrintableUnicodeString(*inventoryUpdate.Building) {
			log.HTTPWarning(req, "Invalid UTF-8 in building field for inventory update")
			middleware.WriteJsonError(w, http.StatusBadRequest)
			return
		}
		*inventoryUpdate.Building = strings.TrimSpace(*inventoryUpdate.Building)
	} else {
		log.HTTPInfo(req, "No building provided for inventory update")
	}

	// Room (optional)
	if inventoryUpdate.Room != nil {
		roomMinChars, roomMaxChars, err := appState.GetRoomConstraints()
		if err != nil {
			log.HTTPWarning(req, "Error retrieving room constraints: "+err.Error())
			middleware.WriteJsonError(w, http.StatusInternalServerError)
			return
		}
		if utf8.RuneCountInString(strings.TrimSpace(*inventoryUpdate.Room)) < roomMinChars || utf8.RuneCountInString(*inventoryUpdate.Room) > roomMaxChars {
			log.HTTPWarning(req, "Invalid room length for inventory update")
			middleware.WriteJsonError(w, http.StatusBadRequest)
			return
		}
		if !middleware.IsPrintableUnicodeString(*inventoryUpdate.Room) {
			log.HTTPWarning(req, "Invalid UTF-8 in room field for inventory update")
			middleware.WriteJsonError(w, http.StatusBadRequest)
			return
		}
		*inventoryUpdate.Room = strings.TrimSpace(*inventoryUpdate.Room)
	} else {
		log.HTTPInfo(req, "No room provided for inventory update")
	}

	// System manufacturer (optional, min 1 char, max 24, Unicode chars)
	if inventoryUpdate.SystemManufacturer != nil {
		systemManufacturerMinChars, systemManufacturerMaxChars, err := appState.GetManufacturerConstraints()
		if err != nil {
			log.HTTPWarning(req, "Error retrieving system manufacturer constraints: "+err.Error())
			middleware.WriteJsonError(w, http.StatusInternalServerError)
			return
		}
		if utf8.RuneCountInString(strings.TrimSpace(*inventoryUpdate.SystemManufacturer)) < systemManufacturerMinChars || utf8.RuneCountInString(*inventoryUpdate.SystemManufacturer) > systemManufacturerMaxChars {
			log.HTTPWarning(req, "Invalid system manufacturer length for inventory update")
			middleware.WriteJsonError(w, http.StatusBadRequest)
			return
		}
		if !middleware.IsPrintableUnicodeString(*inventoryUpdate.SystemManufacturer) {
			log.HTTPWarning(req, "Non-printable Unicode characters in system manufacturer field for inventory update")
			middleware.WriteJsonError(w, http.StatusBadRequest)
			return
		}
		*inventoryUpdate.SystemManufacturer = strings.TrimSpace(*inventoryUpdate.SystemManufacturer)
	} else {
		log.HTTPInfo(req, "No system manufacturer provided for inventory update")
	}

	// System model (optional, min 1 char, max 64 Unicode chars)
	if inventoryUpdate.SystemModel != nil {
		systemModelMinChars, systemModelMaxChars, err := appState.GetSystemModelConstraints()
		if err != nil {
			log.HTTPWarning(req, "Error retrieving system model constraints: "+err.Error())
			middleware.WriteJsonError(w, http.StatusInternalServerError)
			return
		}
		if utf8.RuneCountInString(strings.TrimSpace(*inventoryUpdate.SystemModel)) < systemModelMinChars || utf8.RuneCountInString(*inventoryUpdate.SystemModel) > systemModelMaxChars {
			log.HTTPWarning(req, "Invalid system model length for inventory update")
			middleware.WriteJsonError(w, http.StatusBadRequest)
			return
		}
		if !middleware.IsPrintableUnicodeString(*inventoryUpdate.SystemModel) {
			log.HTTPWarning(req, "Non-printable Unicode characters in system model field for inventory update")
			middleware.WriteJsonError(w, http.StatusBadRequest)
			return
		}
		*inventoryUpdate.SystemModel = strings.TrimSpace(*inventoryUpdate.SystemModel)
	} else {
		log.HTTPInfo(req, "No system model provided for inventory update")
	}

	// Department (required, min 1 char, max 64 chars, printable ASCII only)
	if inventoryUpdate.Department == nil {
		log.HTTPWarning(req, "No department_name provided for inventory update")
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}
	departmentMinChars, departmentMaxChars, err := appState.GetDepartmentConstraints()
	if err != nil {
		log.HTTPWarning(req, "Error retrieving department constraints: "+err.Error())
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}
	if utf8.RuneCountInString(strings.TrimSpace(*inventoryUpdate.Department)) < departmentMinChars || utf8.RuneCountInString(*inventoryUpdate.Department) > departmentMaxChars {
		log.HTTPWarning(req, "Invalid department_name length for inventory update")
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}
	if !middleware.IsASCIIStringPrintable(*inventoryUpdate.Department) {
		log.HTTPWarning(req, "Non-printable ASCII characters in department_name field for inventory update")
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}
	*inventoryUpdate.Department = strings.TrimSpace(*inventoryUpdate.Department)

	// Domain (required, min 1 char, max 64 chars)
	if inventoryUpdate.Domain == nil {
		log.HTTPWarning(req, "No domain provided for inventory update")
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}
	domainMinChars, domainMaxChars, err := appState.GetDomainConstraints()
	if err != nil {
		log.HTTPWarning(req, "Error retrieving domain constraints: "+err.Error())
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}
	if utf8.RuneCountInString(strings.TrimSpace(*inventoryUpdate.Domain)) < domainMinChars || utf8.RuneCountInString(*inventoryUpdate.Domain) > domainMaxChars {
		log.HTTPWarning(req, "Invalid domain length for inventory update")
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}
	if !middleware.IsASCIIStringPrintable(*inventoryUpdate.Domain) {
		log.HTTPWarning(req, "Non-printable ASCII characters in domain field for inventory update")
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}
	*inventoryUpdate.Domain = strings.TrimSpace(*inventoryUpdate.Domain)

	// Property custodian (optional, min 1 char, max 64 Unicode chars)
	if inventoryUpdate.PropertyCustodian != nil {
		propertyCustodianMinChars, propertyCustodianMaxChars, err := appState.GetPropertyCustodianConstraints()
		if err != nil {
			log.HTTPWarning(req, "Error retrieving property custodian constraints: "+err.Error())
			middleware.WriteJsonError(w, http.StatusInternalServerError)
			return
		}
		if utf8.RuneCountInString(strings.TrimSpace(*inventoryUpdate.PropertyCustodian)) < propertyCustodianMinChars || utf8.RuneCountInString(*inventoryUpdate.PropertyCustodian) > propertyCustodianMaxChars {
			log.HTTPWarning(req, "Invalid property custodian length for inventory update")
			middleware.WriteJsonError(w, http.StatusBadRequest)
			return
		}
		if !middleware.IsPrintableUnicodeString(*inventoryUpdate.PropertyCustodian) {
			log.HTTPWarning(req, "Non-printable Unicode characters in property custodian field for inventory update")
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
		log.HTTPInfo(req, "No acquired date provided for inventory update")
	}

	// Retired date, optional, process as UTC
	if inventoryUpdate.RetiredDate != nil {
		retiredDateUTC := inventoryUpdate.RetiredDate.UTC()
		inventoryUpdate.RetiredDate = &retiredDateUTC
	} else {
		log.HTTPInfo(req, "No retired date provided for inventory update")
	}

	// Broken (optional, bool)
	if inventoryUpdate.Broken == nil {
		log.HTTPInfo(req, "No is_broken bool value provided for inventory update")
	}

	// Disk removed (optional, bool)
	if inventoryUpdate.DiskRemoved == nil {
		log.HTTPInfo(req, "No disk_removed bool value provided for inventory update")
	}

	// Last hardware check (optional, process as UTC)
	if inventoryUpdate.LastHardwareCheck != nil {
		lastHardwareCheckUTC := inventoryUpdate.LastHardwareCheck.UTC()
		inventoryUpdate.LastHardwareCheck = &lastHardwareCheckUTC
	} else {
		log.HTTPInfo(req, "No last_hardware_check date provided for inventory update")
	}

	// Status (required, min 1, max 24, ASCII printable chars only)
	if inventoryUpdate.ClientStatus == nil {
		log.HTTPWarning(req, "No status provided for inventory update")
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}
	clientStatusMinChars, clientStatusMaxChars, err := appState.GetClientStatusConstraints()
	if err != nil {
		log.HTTPWarning(req, "Error retrieving client status constraints: "+err.Error())
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}
	if utf8.RuneCountInString(strings.TrimSpace(*inventoryUpdate.ClientStatus)) < clientStatusMinChars || utf8.RuneCountInString(*inventoryUpdate.ClientStatus) > clientStatusMaxChars {
		log.HTTPWarning(req, "Invalid status length for inventory update")
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}
	if !middleware.IsASCIIStringPrintable(*inventoryUpdate.ClientStatus) {
		log.HTTPWarning(req, "Non-printable ASCII characters in status field for inventory update")
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}
	*inventoryUpdate.ClientStatus = strings.TrimSpace(*inventoryUpdate.ClientStatus)

	// Checkout bool (optional)
	checkoutDateMandatory, returnDateMandatory, checkoutBoolMandatory, err := appState.GetCheckoutConstraints()
	if err != nil {
		log.HTTPWarning(req, "Error retrieving checkout constraints: "+err.Error())
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}

	// Checkout date (optional, process as UTC)
	if checkoutDateMandatory && inventoryUpdate.CheckoutDate == nil {
		log.HTTPWarning(req, "No checkout_date provided for inventory update, not updating inventory entry.")
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}
	if inventoryUpdate.CheckoutDate != nil {
		checkoutDateUTC := inventoryUpdate.CheckoutDate.UTC()
		inventoryUpdate.CheckoutDate = &checkoutDateUTC
	} else {
		log.HTTPInfo(req, "No checkout_date provided for inventory update")
	}

	// Return date (optional, process as UTC)
	if returnDateMandatory && inventoryUpdate.ReturnDate == nil {
		log.HTTPWarning(req, "No return_date provided for inventory update, not updating inventory entry.")
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}
	if inventoryUpdate.ReturnDate != nil {
		returnDateUTC := inventoryUpdate.ReturnDate.UTC()
		inventoryUpdate.ReturnDate = &returnDateUTC
	} else {
		log.HTTPInfo(req, "No return_date provided for inventory update")
	}

	if checkoutBoolMandatory && inventoryUpdate.CheckoutBool == nil {
		log.HTTPWarning(req, "No is_checked_out bool value provided for inventory update, not updating inventory entry.")
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}

	// Note (optional)
	if inventoryUpdate.Note != nil {
		noteMinChars, noteMaxChars, err := appState.GetClientNoteConstraints()
		if err != nil {
			log.HTTPWarning(req, "Error retrieving client note constraints: "+err.Error())
			middleware.WriteJsonError(w, http.StatusInternalServerError)
			return
		}
		if utf8.RuneCountInString(strings.TrimSpace(*inventoryUpdate.Note)) < noteMinChars || utf8.RuneCountInString(*inventoryUpdate.Note) > noteMaxChars {
			log.HTTPWarning(req, "Invalid note length for inventory update")
			middleware.WriteJsonError(w, http.StatusBadRequest)
			return
		}
		if !middleware.IsPrintableUnicodeString(*inventoryUpdate.Note) {
			log.HTTPWarning(req, "Non-printable characters in note field for inventory update")
			middleware.WriteJsonError(w, http.StatusBadRequest)
			return
		}
		*inventoryUpdate.Note = strings.TrimSpace(*inventoryUpdate.Note)
	} else {
		log.HTTPInfo(req, "No note provided for inventory update")
	}

	// Other part of form:
	var files []*multipart.FileHeader
	if req.MultipartForm != nil && req.MultipartForm.File != nil {
		if f := req.MultipartForm.File["inventory-file-input"]; len(f) > 0 {
			files = f
		} else {
			log.HTTPInfo(req, "No files uploaded for inventory update")
		}
	}

	if len(files) > maxFileUploadCount {
		log.HTTPWarning(req, "Number of uploaded files exceeds maximum allowed for InsertInventoryUpdateForm: "+strconv.Itoa(len(files)))
		middleware.WriteJsonError(w, http.StatusRequestEntityTooLarge)
		return
	}

	var totalHeaderFileSize int64
	for _, fileHeader := range files {
		totalHeaderFileSize += fileHeader.Size
	}
	if totalHeaderFileSize > maxFileUploadTotalBytes {
		log.HTTPWarning(req, "Total size of uploaded files exceeds maximum allowed for InsertInventoryUpdateForm: "+strconv.FormatInt(totalHeaderFileSize, 10)+" bytes")
		middleware.WriteJsonError(w, http.StatusRequestEntityTooLarge)
		return
	}

	// Establish DB connection before processing files
	updateRepo, err := database.NewUpdateRepo()
	if err != nil {
		log.HTTPError(req, "No database connection available for inventory update")
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}

	// Generate transaction UUID for inventory update and associated file uploads
	transactionUUID, err := uuid.NewUUID()
	if err != nil {
		log.HTTPError(req, "error generation a transaction UUID for InsertInventoryUpdateForm")
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}
	if transactionUUID == uuid.Nil {
		log.HTTPError(req, "transaction UUID in InsertInventoryUpdateForm is nil")
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}

	allowedExtensions, allowedRegex, err := appState.GetFileUploadAllowedExtensionsAndRegex()
	if err != nil {
		log.HTTPWarning(req, "Error retrieving file upload constraints: "+err.Error())
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}
	// Process uploaded files
	var totalActualFileBytes int64
	for _, fileHeader := range files {
		var manifest database.ImageManifest
		// Check multipart headers

		// File name/extension checks
		if matched, _ := regexp.MatchString(allowedRegex, fileHeader.Filename); !matched {
			log.HTTPWarning(req, "Invalid characters in uploaded file name for InsertInventoryUpdateForm")
			continue
		}
		if !slices.Contains(allowedExtensions, strings.ToLower(filepath.Ext(fileHeader.Filename))) {
			log.HTTPWarning(req, "Uploaded file has disallowed extension for InsertInventoryUpdateForm: ("+fileHeader.Filename+")")
			continue
		}

		// File size from multipart header checks
		if fileHeader.Size > maxFileUploadBytes {
			log.HTTPWarning(req, "Multipart header size value is too large for InsertInventoryUpdateForm ("+strconv.FormatInt(fileHeader.Size, 10)+" bytes)")
			continue
		}
		if fileHeader.Size <= 0 {
			log.HTTPWarning(req, "Multipart header size value is empty for InsertInventoryUpdateForm: "+fileHeader.Filename)
			continue
		}
		if fileHeader.Size <= minFileUploadBytes {
			log.HTTPWarning(req, "Multipart header size value too small for InsertInventoryUpdateForm: "+fileHeader.Filename+" ("+strconv.FormatInt(fileHeader.Size, 10)+" bytes)")
			continue
		}

		// Open uploaded file
		file, err := fileHeader.Open()
		if err != nil {
			log.HTTPWarning(req, "Failed to open uploaded file for InsertInventoryUpdateForm: "+err.Error())
			continue
		}

		lr := &io.LimitedReader{R: file, N: maxFileUploadBytes + 1}
		fileBytes, err := io.ReadAll(lr)
		file.Close()
		if err != nil {
			if maxBytesErr, ok := errors.AsType[*http.MaxBytesError](err); ok {
				log.HTTPWarning(req, "Uploaded file size exceeds maximum allowed for InsertInventoryUpdateForm: "+maxBytesErr.Error())
				continue
			}
			log.HTTPWarning(req, "Failed to read uploaded file for InsertInventoryUpdateForm: "+err.Error())
			continue
		}

		// File size checks (in addition to header checks - not necessarily same value)
		fileSize := int64(len(fileBytes))
		if fileSize > maxFileUploadBytes {
			log.HTTPWarning(req, "Uploaded file too large for InsertInventoryUpdateForm ("+strconv.FormatInt(fileSize, 10)+" bytes)")
			continue
		}
		if fileSize == 0 {
			log.HTTPWarning(req, "Empty file uploaded for InsertInventoryUpdateForm: "+fileHeader.Filename)
			continue
		}
		if fileSize < minFileUploadBytes {
			log.HTTPWarning(req, "Uploaded file too small for InsertInventoryUpdateForm: "+fileHeader.Filename+" ("+strconv.FormatInt(fileSize, 10)+" bytes)")
			continue
		}
		manifest.FileSize = &fileSize
		totalActualFileBytes += fileSize

		// MIME type detection
		mimeType := http.DetectContentType(fileBytes[:fileSize])
		if !strings.HasPrefix(mimeType, "image/") { // temporary while implementing video support
			log.HTTPWarning(req, "Uploaded file has a non-accepted MIME type for InsertInventoryUpdateForm: (Content-Type: "+mimeType+")")
			continue
		}
		manifest.MimeType = &mimeType

		// Create reader (stream) for image decoding
		imageReader := bytes.NewReader(fileBytes)

		// Rewind and decode image to get image.Image
		_, err = imageReader.Seek(0, io.SeekStart)
		if err != nil {
			log.HTTPError(req, "Failed to seek to start of uploaded image for InsertInventoryUpdateForm: "+err.Error())
			continue
		}
		decodedImage, imageFormat, err := image.Decode(imageReader)
		if err != nil {
			log.HTTPError(req, "Failed to decode thumbnail in InsertInventoryUpdateForm: "+err.Error()+" ("+fileHeader.Filename+")")
			continue
		}

		// Rewind and decode image to get image config
		_, err = imageReader.Seek(0, io.SeekStart)
		if err != nil {
			log.HTTPError(req, "Failed to seek to start of uploaded image for InsertInventoryUpdateForm: "+err.Error()+" ("+fileHeader.Filename+")")
			continue
		}
		decodedImageConfig, _, err := image.DecodeConfig(imageReader)
		if err != nil {
			log.HTTPError(req, "Failed to decode uploaded image config for InsertInventoryUpdateForm: "+err.Error()+": "+fileHeader.Filename+" ("+fileHeader.Filename+")")
			continue
		}
		resX := int64(decodedImageConfig.Width)
		manifest.ResolutionX = &resX
		resY := int64(decodedImageConfig.Height)
		manifest.ResolutionY = &resY

		// Get upload timestamp
		fileTimeStamp := time.Now()
		timeUTC := fileTimeStamp.UTC()
		manifest.Time = &timeUTC

		// Generate unique file name
		fileTimeStampFormatted := fileTimeStamp.Format("2006-01-02-150405")
		fileUUID := uuid.New()
		fileUUIDString := fileUUID.String()
		var fileName string
		baseFileName := fileTimeStampFormatted + "-" + fileUUIDString
		switch mimeType {
		case "image/jpeg", "image/jpg":
			if imageFormat != "jpeg" {
				log.HTTPWarning(req, "MIME type and image format mismatch for uploaded file for InsertInventoryUpdateForm: (MIME: "+mimeType+", Format: "+imageFormat+")")
				continue
			}
			fileName = baseFileName + ".jpeg"
		case "image/png":
			if imageFormat != "png" {
				log.HTTPWarning(req, "MIME type and image format mismatch for uploaded file for InsertInventoryUpdateForm: (MIME: "+mimeType+", Format: "+imageFormat+")")
				continue
			}
			fileName = baseFileName + ".png"
		case "video/mp4":
			fileName = baseFileName + ".mp4"
		case "video/quicktime":
			fileName = baseFileName + ".mov"
		default:
			log.HTTPWarning(req, "Unsupported image MIME type for InsertInventoryUpdateForm: (Content-Type: "+mimeType+")")
			continue
		}
		manifest.FileName = &fileName
		manifest.UUID = &fileUUIDString

		// Compute SHA256 hash of file
		fileHash := crypto.SHA256.New()
		if _, err := fileHash.Write(fileBytes); err != nil {
			log.HTTPError(req, "Failed to compute hash of uploaded file for InsertInventoryUpdateForm: "+err.Error()+" ("+fileHeader.Filename+")")
			middleware.WriteJsonError(w, http.StatusInternalServerError)
			return
		}
		fileHashBytes := fileHash.Sum(nil)
		fileHashString := fmt.Sprintf("%x", fileHashBytes)
		manifest.SHA256Hash = &fileHashString

		// Create directories if not existing
		imageDirectoryPath := filepath.Join("./inventory-images", fmt.Sprintf("%06d", *inventoryUpdate.Tagnumber))
		err = os.MkdirAll(imageDirectoryPath, 0755)
		if err != nil {
			log.HTTPError(req, "Failed to create directories for uploaded file for InsertInventoryUpdateForm: "+err.Error())
			middleware.WriteJsonError(w, http.StatusInternalServerError)
			return
		}

		// Set file/directory permissions
		if err := os.Chmod(imageDirectoryPath, 0755); err != nil {
			log.HTTPError(req, "Failed to set directory permissions: "+err.Error()+" ("+fileHeader.Filename+")")
			middleware.WriteJsonError(w, http.StatusInternalServerError)
			return
		}

		fullFilePath := filepath.Join(imageDirectoryPath, fileName)
		if err := os.WriteFile(fullFilePath, fileBytes, 0644); err != nil {
			log.HTTPError(req, "Failed to save uploaded file for InsertInventoryUpdateForm: "+err.Error()+" ("+fileHeader.Filename+")")
			middleware.WriteJsonError(w, http.StatusInternalServerError)
			return
		}
		manifest.FilePath = &fullFilePath

		// Close the uploaded file, not needed anymore
		_ = file.Close()

		var fullThumbnailPath string
		if strings.HasPrefix(mimeType, "image/") {
			fullThumbnailPath = filepath.Join("./inventory-images", fmt.Sprintf("%06d", *inventoryUpdate.Tagnumber), "thumbnail-"+baseFileName+".jpeg")
			thumbnailFile, err := os.Create(fullThumbnailPath)
			if err != nil {
				log.HTTPError(req, "Failed to create thumbnail file for InsertInventoryUpdateForm: "+err.Error()+" ("+fileHeader.Filename+")")
				middleware.WriteJsonError(w, http.StatusInternalServerError)
				return
			}

			if err = jpeg.Encode(thumbnailFile, decodedImage, &jpeg.Options{Quality: 50}); err != nil {
				log.HTTPError(req, "Failed to encode thumbnail image for InsertInventoryUpdateForm: "+err.Error()+" ("+fileHeader.Filename+")")
				middleware.WriteJsonError(w, http.StatusInternalServerError)
				return
			}
			thumbnailFile.Close()
		}
		manifest.ThumbnailFilePath = &fullThumbnailPath

		// Insert image metadata into database
		manifest.Tagnumber = inventoryUpdate.Tagnumber
		manifest.Hidden = new(bool)
		*manifest.Hidden = false
		manifest.PrimaryImage = new(bool)
		*manifest.PrimaryImage = false

		err = updateRepo.UpdateClientImages(ctx, transactionUUID, &manifest)
		if err != nil {
			log.HTTPError(req, "Failed to update inventory image data: "+err.Error()+" ("+fileHeader.Filename+")")
			middleware.WriteJsonError(w, http.StatusInternalServerError)
			return
		}
		log.HTTPInfo(req, fmt.Sprintf("Uploaded file details - Name: %s, Size: %.2f MB, MIME Type: %s", fileName, float64(*manifest.FileSize)/1024/1024, mimeType)+" ("+fileHeader.Filename+")")
		_ = file.Close()
	}
	// Update db
	// tagnumber and broken bool are converted above
	err = updateRepo.InsertInventoryUpdateForm(ctx, transactionUUID, &inventoryUpdate)
	if err != nil {
		log.HTTPError(req, "Failed to update inventory data: "+err.Error())
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
		log.HTTPError(req, "Cannot decode TogglePinImage JSON: "+err.Error())
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}

	uuid := strings.TrimSpace(body.UUID)
	if uuid == "" {
		log.HTTPWarning(req, "No image UUID provided in TogglePinImage body")
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}
	uuid = strings.TrimSuffix(uuid, ".jpeg")
	uuid = strings.TrimSuffix(uuid, ".png")
	uuid = strings.TrimSuffix(uuid, ".mp4")
	uuid = strings.TrimSuffix(uuid, ".mov")
	if uuid == "" {
		log.HTTPWarning(req, "No image path provided for TogglePinImage body")
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}

	tagnumber := body.Tagnumber
	if tagnumber < 1 || tagnumber > 999999 {
		log.HTTPWarning(req, "Invalid tag number provided in TogglePinImage body")
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}

	updateRepo, err := database.NewUpdateRepo()
	if err != nil {
		log.HTTPError(req, "No database connection available for TogglePinImage")
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}
	if err = updateRepo.TogglePinImage(ctx, &tagnumber, &uuid); err != nil {
		log.HTTPError(req, "Failed to toggle pin image: "+err.Error())
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
		log.HTTPWarning(req, "Error retrieving URL queries from context for SetClientBatteryHealth: "+err.Error())
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}
	if req.Method != http.MethodPost {
		log.HTTPWarning(req, "Invalid HTTP method for SetClientBatteryHealth")
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}
	uuid := strings.TrimSpace(requestQueries.Get("uuid"))
	if uuid == "" {
		log.HTTPWarning(req, "No UUID provided for SetClientBatteryHealth")
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}
	var body struct {
		BatteryHealth int64 `json:"battery_health"`
	}

	if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
		log.HTTPWarning(req, "Cannot decode SetClientBatteryHealth JSON: "+err.Error())
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}
	// if body.BatteryHealth < 0 || body.BatteryHealth > 100 {
	// 	log.HTTPWarning(req, "Invalid battery health percentage provided for SetClientBatteryHealth")
	// 	middleware.WriteJsonError(w, http.StatusBadRequest)
	// 	return
	// }
	updateRepo, err := database.NewUpdateRepo()
	if err != nil {
		log.HTTPError(req, "No database connection available for SetClientBatteryHealth")
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}
	if err = updateRepo.SetClientBatteryHealth(ctx, &uuid, &body.BatteryHealth); err != nil {
		log.HTTPError(req, "Failed to set client battery health: "+err.Error())
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}
	middleware.WriteJson(w, http.StatusOK, "Battery health updated successfully")
}

func SetAllJobs(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	log := middleware.GetLoggerFromContext(ctx)

	if req.Method != http.MethodPost {
		log.HTTPWarning(req, "Invalid HTTP method for SetAllJobs")
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}

	clientBody, err := io.ReadAll(req.Body)
	if err != nil {
		log.HTTPWarning(req, "Cannot read request body for SetAllJobs: "+err.Error())
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}

	var clientJson database.AllJobs
	if err := json.Unmarshal(clientBody, &clientJson); err != nil {
		log.HTTPWarning(req, "Cannot decode SetAllJobs JSON: "+err.Error())
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}

	updateRepo, err := database.NewUpdateRepo()
	if err != nil {
		log.HTTPError(req, "No database connection available for SetAllJobs")
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}
	if err = updateRepo.SetAllOnlineClientJobs(ctx, &clientJson); err != nil {
		log.HTTPError(req, "Failed to set all jobs: "+err.Error())
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}
	middleware.WriteJson(w, http.StatusOK, "All jobs set successfully")
}
