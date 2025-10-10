package endpoints

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	_ "image/png"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
	config "uit-toolbox/config"
	"uit-toolbox/database"
	middleware "uit-toolbox/middleware"
	"unicode"
	"unicode/utf8"
)

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
	requestInfo, err := GetRequestInfo(req)
	if err != nil {
		fmt.Println("Cannot get request info error: " + err.Error())
		http.Error(w, middleware.FormatHttpError("Internal server error"), http.StatusInternalServerError)
		return
	}
	ctx := requestInfo.Ctx
	log := requestInfo.Log
	requestIP := requestInfo.IP
	requestURL := requestInfo.URL

	// Sanitize login POST request
	if req.Method != http.MethodPost || !(strings.HasSuffix(requestURL, "/login.html") || strings.HasSuffix(requestURL, "/login")) {
		log.Warning("Invalid method or URL for auth form sanitization: " + requestIP + " ( " + requestURL + ")")
		http.Error(w, middleware.FormatHttpError("Bad request"), http.StatusBadRequest)
		return
	}

	body, err := io.ReadAll(req.Body)
	if err != nil {
		log.Warning("Cannot read request body: " + err.Error() + " (" + requestIP + ")")
		http.Error(w, middleware.FormatHttpError("Bad request"), http.StatusBadRequest)
		return
	}
	defer req.Body.Close()

	base64String := strings.TrimSpace(string(body))
	if base64String == "" {
		log.Warning("No base64 string provided in auth form: " + requestIP)
		http.Error(w, middleware.FormatHttpError("Bad request"), http.StatusBadRequest)
		return
	}

	decodedBytes, err := base64.StdEncoding.DecodeString(base64String)
	if err != nil {
		log.Warning("Invalid base64 encoding: " + err.Error() + " (" + requestIP + ")")
		http.Error(w, middleware.FormatHttpError("Bad request"), http.StatusBadRequest)
		return
	}

	if !utf8.Valid(decodedBytes) {
		log.Warning("Invalid UTF-8 in decoded data: " + requestIP)
		http.Error(w, middleware.FormatHttpError("Bad request"), http.StatusBadRequest)
		return
	}
	var clientFormAuthData struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := json.Unmarshal(decodedBytes, &clientFormAuthData); err != nil {
		log.Warning("Invalid JSON structure: " + err.Error() + " (" + requestIP + ")")
		http.Error(w, middleware.FormatHttpError("Bad request"), http.StatusBadRequest)
		return
	}

	// Validate input data
	if err := middleware.ValidateAuthFormInputSHA256(clientFormAuthData.Username, clientFormAuthData.Password); err != nil {
		log.Warning("Invalid auth input: " + err.Error() + " (" + requestIP + ")")
		http.Error(w, middleware.FormatHttpError("Bad request"), http.StatusBadRequest)
		return
	}

	// Authenticate with bcrypt
	authenticated, err := middleware.CheckAuthCredentials(ctx, clientFormAuthData.Username, clientFormAuthData.Password)
	if err != nil || !authenticated {
		log.Info("Authentication failed for " + requestIP + ": " + err.Error())
		http.Error(w, middleware.FormatHttpError("Unauthorized"), http.StatusUnauthorized)
		http.Redirect(w, req, "/login?error=1", http.StatusSeeOther)
		return
	}

	sessionID, basicToken, bearerToken, csrfToken, err := config.CreateAuthSession(requestIP)
	if err != nil {
		log.Error("Failed to generate tokens: " + err.Error())
		http.Error(w, middleware.FormatHttpError("Internal middleware error"), http.StatusInternalServerError)
		return
	}

	sessionCount := config.GetAuthSessionCount()
	log.Info("New auth session created: " + requestIP + " (Sessions: " + strconv.Itoa(int(sessionCount)) + ")")

	sessionIDCookie, basicTokenCookie, bearerTokenCookie, csrfTokenCookie := middleware.GetAuthCookiesForResponse(sessionID, basicToken, bearerToken, csrfToken, 20*time.Minute)

	http.SetCookie(w, sessionIDCookie)
	http.SetCookie(w, basicTokenCookie)
	http.SetCookie(w, bearerTokenCookie)
	http.SetCookie(w, csrfTokenCookie)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	w.Write([]byte(`{"token":"` + sessionID + `"}`))

	http.Redirect(w, req, "/dashboard", http.StatusSeeOther)
}

func InsertNewNote(w http.ResponseWriter, req *http.Request) {
	requestInfo, err := GetRequestInfo(req)
	if err != nil {
		fmt.Println("Cannot get request info error: " + err.Error())
		http.Error(w, middleware.FormatHttpError("Internal server error"), http.StatusInternalServerError)
		return
	}
	ctx := requestInfo.Ctx
	log := requestInfo.Log
	requestIP := requestInfo.IP
	requestURL := requestInfo.URL
	requestMethod := req.Method

	// Sanitize POST request
	if requestMethod != http.MethodPost || !(strings.HasSuffix(requestURL, "/notes")) {
		log.Warning("Invalid method or URL for note insertion: " + requestIP + " ( " + requestURL + ")")
		http.Error(w, middleware.FormatHttpError("Bad request"), http.StatusBadRequest)
		return
	}

	// Parse and validate note data
	var newNote struct {
		NoteType string `json:"note_type"`
		Content  string `json:"note"`
	}
	err = json.NewDecoder(req.Body).Decode(&newNote)
	if err != nil {
		log.Warning("Cannot decode note JSON: " + err.Error() + " (" + requestIP + ")")
		http.Error(w, middleware.FormatHttpError("Bad request"), http.StatusBadRequest)
		return
	}
	defer req.Body.Close()

	newNote.NoteType = strings.TrimSpace(newNote.NoteType)
	if len(newNote.NoteType) > 64 {
		log.Warning("Note type too long: " + requestIP)
		http.Error(w, middleware.FormatHttpError("Note type too long"), http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(newNote.NoteType) == "" {
		log.Warning("Empty note type: " + requestIP)
		http.Error(w, middleware.FormatHttpError("Note type cannot be empty"), http.StatusBadRequest)
		return
	}
	newNote.Content = strings.TrimSpace(newNote.Content)
	if len(newNote.Content) > 2000 {
		log.Warning("Note content too long: " + requestIP)
		http.Error(w, middleware.FormatHttpError("Note content too long"), http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(newNote.Content) == "" {
		log.Warning("Empty note content: " + requestIP)
		http.Error(w, middleware.FormatHttpError("Note content cannot be empty"), http.StatusBadRequest)
		return
	}

	// Insert note into database
	dbConn := config.GetDatabaseConn()
	insertRepo := database.NewRepo(dbConn)
	err = insertRepo.InsertNewNote(ctx, time.Now(), newNote.NoteType, newNote.Content)
	if err != nil {
		log.Error("Failed to insert new note: " + err.Error() + " (" + requestIP + ")")
		http.Error(w, middleware.FormatHttpError("Internal server error"), http.StatusInternalServerError)
		return
	}
}

type InventoryUpdate struct {
	Tagnumber          int     `json:"tagnumber"`
	SystemSerial       string  `json:"system_serial"`
	Location           string  `json:"location"`
	SystemManufacturer *string `json:"system_manufacturer"`
	SystemModel        *string `json:"system_model"`
	Department         *string `json:"department"`
	Domain             *string `json:"domain"`
	Working            *bool   `json:"working"`
	Status             *string `json:"status"`
	Note               *string `json:"note"`
	Image              *string `json:"image"`
}

func UpdateInventory(w http.ResponseWriter, req *http.Request) {
	requestInfo, err := GetRequestInfo(req)
	if err != nil {
		fmt.Println("Cannot get request info error: " + err.Error())
		http.Error(w, middleware.FormatHttpError("Internal server error"), http.StatusInternalServerError)
		return
	}
	ctx := requestInfo.Ctx
	log := requestInfo.Log
	requestIP := requestInfo.IP
	requestURL := requestInfo.URL
	requestMethod := req.Method

	// Check for POST method and correct URL
	if requestMethod != http.MethodPost || !(strings.HasSuffix(requestURL, "/update_inventory")) {
		log.Warning("Invalid method or URL for inventory update: " + requestIP + " ( " + requestURL + ")")
		middleware.WriteJsonError(w, http.StatusBadRequest, "Bad request")
		return
	}
	// Parse inventory data
	var inventoryUpdate InventoryUpdate
	err = json.NewDecoder(req.Body).Decode(&inventoryUpdate)
	if err != nil {
		log.Warning("Cannot decode inventory JSON: " + err.Error() + " (" + requestIP + ")")
		middleware.WriteJsonError(w, http.StatusBadRequest, "Bad request")
		return
	}
	defer req.Body.Close()

	// Validate and sanitize input data
	// Tag number (required, 6 digits)
	if inventoryUpdate.Tagnumber == 0 {
		log.Warning("No tag number provided for inventory update: " + requestIP)
		middleware.WriteJsonError(w, http.StatusBadRequest, "Bad request")
		return
	}
	if inventoryUpdate.Tagnumber < 1 || inventoryUpdate.Tagnumber > 999999 {
		log.Warning("Invalid tag number provided for inventory update: " + requestIP)
		middleware.WriteJsonError(w, http.StatusBadRequest, "Bad request")
		return
	}
	if middleware.CountDigits(inventoryUpdate.Tagnumber) != 6 {
		log.Warning("Tag number not 6 digits for inventory update: " + requestIP)
		middleware.WriteJsonError(w, http.StatusBadRequest, "Bad request")
		return
	}

	// System serial (required, min 4 chars, max 64 chars)
	if inventoryUpdate.SystemSerial == "" {
		log.Warning("Invalid system serial provided for inventory update: " + requestIP)
		middleware.WriteJsonError(w, http.StatusBadRequest, "Bad request")
		return
	}
	if utf8.RuneCountInString(inventoryUpdate.SystemSerial) < 4 || utf8.RuneCountInString(inventoryUpdate.SystemSerial) > 64 {
		log.Warning("Invalid system serial length provided for inventory update: " + requestIP)
		middleware.WriteJsonError(w, http.StatusBadRequest, "Bad request")
		return
	}
	if !middleware.IsAlphanumericAscii([]byte(inventoryUpdate.SystemSerial)) {
		log.Warning("Non-alphanumeric characters in system serial field for inventory update: " + requestIP)
		middleware.WriteJsonError(w, http.StatusBadRequest, "Bad request")
		return
	}
	inventoryUpdate.SystemSerial = strings.TrimSpace(inventoryUpdate.SystemSerial)

	// Location (required, min 1 char, max 128 chars)
	if inventoryUpdate.Location == "" {
		log.Warning("No location provided for inventory update: " + requestIP)
		middleware.WriteJsonError(w, http.StatusBadRequest, "Bad request")
		return
	}
	if utf8.RuneCountInString(inventoryUpdate.Location) < 1 || utf8.RuneCountInString(inventoryUpdate.Location) > 128 {
		log.Warning("Invalid location length for inventory update: " + requestIP)
		middleware.WriteJsonError(w, http.StatusBadRequest, "Bad request")
		return
	}
	if !middleware.IsPrintableASCII([]byte(inventoryUpdate.Location)) {
		log.Warning("Non-printable ASCII characters in location field for inventory update: " + requestIP)
		middleware.WriteJsonError(w, http.StatusBadRequest, "Bad request")
		return
	}
	inventoryUpdate.Location = strings.TrimSpace(inventoryUpdate.Location)

	// System manufacturer (optional, min 1 char, max 24 chars)
	if inventoryUpdate.SystemManufacturer != nil && strings.TrimSpace(*inventoryUpdate.SystemManufacturer) != "" {
		if utf8.RuneCountInString(*inventoryUpdate.SystemManufacturer) < 1 || utf8.RuneCountInString(*inventoryUpdate.SystemManufacturer) > 24 {
			log.Warning("Invalid system manufacturer length for inventory update: " + requestIP)
			middleware.WriteJsonError(w, http.StatusBadRequest, "Bad request")
			return
		}
		if !middleware.IsAlphanumericAscii([]byte(*inventoryUpdate.SystemManufacturer)) {
			log.Warning("Non-alphanumeric characters in system manufacturer field for inventory update: " + requestIP)
			middleware.WriteJsonError(w, http.StatusBadRequest, "Bad request")
			return
		}
		*inventoryUpdate.SystemManufacturer = strings.TrimSpace(*inventoryUpdate.SystemManufacturer)
	} else {
		log.Warning("No system manufacturer provided for inventory update: " + requestIP)
	}

	// System model (optional, min 1 char, max 64 chars)
	if inventoryUpdate.SystemModel != nil && strings.TrimSpace(*inventoryUpdate.SystemModel) != "" {
		if utf8.RuneCountInString(*inventoryUpdate.SystemModel) < 1 || utf8.RuneCountInString(*inventoryUpdate.SystemModel) > 64 {
			log.Warning("Invalid system model length for inventory update: " + requestIP)
			middleware.WriteJsonError(w, http.StatusBadRequest, "Bad request")
			return
		}
		if !middleware.IsAlphanumericAscii([]byte(*inventoryUpdate.SystemModel)) {
			log.Warning("Non-alphanumeric characters in system model field for inventory update: " + requestIP)
			middleware.WriteJsonError(w, http.StatusBadRequest, "Bad request")
			return
		}
		*inventoryUpdate.SystemModel = strings.TrimSpace(*inventoryUpdate.SystemModel)
	} else {
		log.Warning("No system model provided for inventory update: " + requestIP)
	}

	// Department (required, max 24 chars)
	if inventoryUpdate.Department == nil || strings.TrimSpace(*inventoryUpdate.Department) == "" {
		log.Warning("No department provided for inventory update: " + requestIP)
		middleware.WriteJsonError(w, http.StatusBadRequest, "Bad request")
		return
	}
	if utf8.RuneCountInString(*inventoryUpdate.Department) < 1 || utf8.RuneCountInString(*inventoryUpdate.Department) > 24 {
		log.Warning("Invalid department length for inventory update: " + requestIP)
		middleware.WriteJsonError(w, http.StatusBadRequest, "Bad request")
		return
	}
	if !middleware.IsPrintableASCII([]byte(*inventoryUpdate.Department)) {
		log.Warning("Non-printable ASCII characters in department field for inventory update: " + requestIP)
		middleware.WriteJsonError(w, http.StatusBadRequest, "Bad request")
		return
	}
	*inventoryUpdate.Department = strings.TrimSpace(*inventoryUpdate.Department)

	// Domain (optional, min 1 char, max 24 chars)
	if inventoryUpdate.Domain != nil && strings.TrimSpace(*inventoryUpdate.Domain) != "" {
		if utf8.RuneCountInString(*inventoryUpdate.Domain) < 1 || utf8.RuneCountInString(*inventoryUpdate.Domain) > 24 {
			log.Warning("Invalid domain length for inventory update: " + requestIP)
			middleware.WriteJsonError(w, http.StatusBadRequest, "Bad request")
			return
		}
		if !middleware.IsPrintableASCII([]byte(*inventoryUpdate.Domain)) {
			log.Warning("Non-printable ASCII characters in domain field for inventory update: " + requestIP)
			middleware.WriteJsonError(w, http.StatusBadRequest, "Bad request")
			return
		}
		*inventoryUpdate.Domain = strings.TrimSpace(*inventoryUpdate.Domain)
	} else {
		log.Warning("No domain provided for inventory update: " + requestIP)
	}

	// Working (mandatory, bool)
	if inventoryUpdate.Working == nil || !*inventoryUpdate.Working {
		log.Warning("No working status provided for inventory update: " + requestIP)
		middleware.WriteJsonError(w, http.StatusBadRequest, "Bad request")
		return
	}

	// Status (mandatory, max 64 chars)
	if inventoryUpdate.Status == nil || strings.TrimSpace(*inventoryUpdate.Status) == "" {
		log.Warning("No status provided for inventory update: " + requestIP)
		middleware.WriteJsonError(w, http.StatusBadRequest, "Bad request")
		return
	}
	if utf8.RuneCountInString(*inventoryUpdate.Status) < 1 || utf8.RuneCountInString(*inventoryUpdate.Status) > 64 {
		log.Warning("Invalid status length for inventory update: " + requestIP)
		middleware.WriteJsonError(w, http.StatusBadRequest, "Bad request")
		return
	}
	if !middleware.IsPrintableASCII([]byte(*inventoryUpdate.Status)) {
		log.Warning("Non-printable ASCII characters in status field for inventory update: " + requestIP)
		middleware.WriteJsonError(w, http.StatusBadRequest, "Bad request")
		return
	}
	*inventoryUpdate.Status = strings.TrimSpace(*inventoryUpdate.Status)

	// Note (optional, max 2000 chars)
	if inventoryUpdate.Note != nil && strings.TrimSpace(*inventoryUpdate.Note) != "" {
		if utf8.RuneCountInString(*inventoryUpdate.Note) < 1 || utf8.RuneCountInString(*inventoryUpdate.Note) > 2000 {
			log.Warning("Invalid note length for inventory update: " + requestIP)
			middleware.WriteJsonError(w, http.StatusBadRequest, "Bad request")
			return
		}
		for _, rune := range *inventoryUpdate.Note {
			if !unicode.IsPrint(rune) && !unicode.IsSpace(rune) {
				log.Warning("Non-printable characters in note field for inventory update: " + requestIP)
				middleware.WriteJsonError(w, http.StatusBadRequest, "Bad request")
				return
			}
		}
		*inventoryUpdate.Note = strings.TrimSpace(*inventoryUpdate.Note)
	} else {
		log.Warning("No note provided for inventory update: " + requestIP)
	}

	// Image (base64, optional, max 64MB, multiple file uploads supported)
	file, handler, err := req.FormFile("inventory-file-input")
	if err != nil {
		log.Warning("Failed to retrieve file from form: " + requestIP)
		middleware.WriteJsonError(w, http.StatusBadRequest, "Bad request")
		return
	}
	defer file.Close()

	if handler.Size > 64<<20 {
		log.Warning("Uploaded file too large for inventory update: " + requestIP)
		middleware.WriteJsonError(w, http.StatusBadRequest, "File too large")
		return
	}

	for _, rune := range handler.Filename {
		if !(unicode.IsLetter(rune) || unicode.IsDigit(rune) || rune == '.' || rune == '-' || rune == '_') {
			log.Warning("Invalid characters in uploaded file name for inventory update: " + requestIP)
			middleware.WriteJsonError(w, http.StatusBadRequest, "Bad request")
			return
		}
	}

	fileBytes, err := io.ReadAll(file)
	if err != nil {
		log.Warning("Failed to read uploaded file for inventory update: " + err.Error() + " (" + requestIP + ")")
		middleware.WriteJsonError(w, http.StatusBadRequest, "Bad request")
		return
	}
	if len(fileBytes) == 0 {
		log.Warning("Empty file uploaded for inventory update: " + requestIP)
		middleware.WriteJsonError(w, http.StatusBadRequest, "Bad request")
		return
	}

	// Update db
	dbConn := config.GetDatabaseConn()
	updateRepo := database.NewRepo(dbConn)
	// No pointers here, pointers in repo
	err = updateRepo.UpdateInventory(ctx, inventoryUpdate.Tagnumber, inventoryUpdate.SystemSerial, inventoryUpdate.Location, inventoryUpdate.SystemManufacturer, inventoryUpdate.SystemModel, inventoryUpdate.Department, inventoryUpdate.Domain, inventoryUpdate.Working, inventoryUpdate.Status, inventoryUpdate.Note, inventoryUpdate.Image)
	if err != nil {
		log.Error("Failed to update inventory: " + err.Error() + " (" + requestIP + ")")
		middleware.WriteJsonError(w, http.StatusInternalServerError, "Internal server error")
		return
	}
	w.WriteHeader(http.StatusOK)
}
