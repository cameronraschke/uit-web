package endpoints

import (
	"bytes"
	"crypto"
	"encoding/base64"
	"encoding/json"
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

const maxInventoryFileSizeBytes = 64 << 20  // 64 MB
const minInventoryFileSizeBytes = 512       // 512 bytes
const maxInventoryFormSizeBytes = 128 << 20 // 128 MB

const allowedFileNameRegex = `^[a-zA-Z0-9.\-_ ()]+\.[a-zA-Z]+$` // file name + extension
var allowedFileExtensions = []string{".jpg", ".jpeg", ".jfif", ".png"}

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
	requestPath, err := middleware.GetRequestPathFromContext(ctx)
	if err != nil {
		log.HTTPWarning(req, "Error retrieving request path from context (WebAuthEndpoint): "+err.Error())
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}

	// Sanitize login POST request
	if req.Method != http.MethodPost || !strings.HasSuffix(requestPath, "/login") {
		log.HTTPWarning(req, "Invalid method or URL for auth form sanitization")
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}

	body, err := io.ReadAll(req.Body)
	if err != nil {
		log.HTTPWarning(req, "Cannot read request body: "+err.Error())
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}
	defer req.Body.Close()

	base64String := strings.TrimSpace(string(body))
	if base64String == "" {
		log.HTTPWarning(req, "No base64 string provided in auth form")
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}

	decodedBytes, err := base64.RawURLEncoding.DecodeString(base64String)
	if err != nil {
		log.HTTPWarning(req, "Invalid base64 encoding: "+err.Error())
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}

	if !utf8.Valid(decodedBytes) {
		log.HTTPWarning(req, "Invalid UTF-8 in decoded data")
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}
	var clientFormAuthData struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := json.Unmarshal(decodedBytes, &clientFormAuthData); err != nil {
		log.HTTPWarning(req, "Invalid JSON structure: "+err.Error())
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}

	// Validate input data
	if err := middleware.ValidateAuthFormInputSHA256(clientFormAuthData.Username, clientFormAuthData.Password); err != nil {
		log.HTTPWarning(req, "Invalid auth input: "+err.Error())
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}

	// Authenticate with bcrypt
	authenticated, err := middleware.CheckAuthCredentials(ctx, clientFormAuthData.Username, clientFormAuthData.Password)
	if err != nil || !authenticated {
		log.HTTPInfo(req, "Authentication failed for "+requestIP.String()+": "+err.Error())
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
	log.HTTPInfo(req, "New auth session created: "+requestIP.String()+" (Sessions: "+strconv.Itoa(int(sessionCount))+")")

	sessionIDCookie, basicTokenCookie, bearerTokenCookie, csrfTokenCookie := middleware.GetAuthCookiesForResponse(sessionID, basicToken, bearerToken, csrfToken, 20*time.Minute)

	http.SetCookie(w, sessionIDCookie)
	http.SetCookie(w, basicTokenCookie)
	http.SetCookie(w, bearerTokenCookie)
	http.SetCookie(w, csrfTokenCookie)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	w.Write([]byte(`{"token":"` + bearerToken + `"}`))
}

func InsertNewNote(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	log := middleware.GetLoggerFromContext(ctx)
	requestPath, err := middleware.GetRequestPathFromContext(ctx)
	if err != nil {
		log.HTTPWarning(req, "Error retrieving request path from context for InsertNewNote")
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}
	requestMethod := req.Method

	// Sanitize POST request
	if requestMethod != http.MethodPost || !(strings.HasSuffix(requestPath, "/notes")) {
		log.HTTPWarning(req, "Invalid method or URL for InsertNewNote")
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}

	// Parse and validate note data
	var newNote struct {
		NoteType string `json:"note_type"`
		Content  string `json:"note"`
	}
	err = json.NewDecoder(req.Body).Decode(&newNote)
	if err != nil {
		log.HTTPWarning(req, "Cannot decode note JSON: "+err.Error())
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}
	defer req.Body.Close()

	newNote.NoteType = strings.TrimSpace(newNote.NoteType)
	if len(newNote.NoteType) > 64 {
		log.HTTPWarning(req, "Note type too long, not inserting new note")
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(newNote.NoteType) == "" {
		log.HTTPWarning(req, "Note type unspecified, not inserting new note")
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}
	newNote.Content = strings.TrimSpace(newNote.Content)
	if len(newNote.Content) > 32768 { // 8192 runes * 4 bytes per rune (4 bytes includes emojis)
		log.HTTPWarning(req, "Note content too long, not inserting new note")
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}
	if utf8.RuneCountInString(newNote.Content) > 8192 {
		log.HTTPWarning(req, "Note content exceeds rune count limit, not inserting new note")
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}

	// Insert note into database
	dbConn := config.GetDatabaseConn()
	insertRepo := database.NewRepo(dbConn)
	err = insertRepo.InsertNewNote(ctx, time.Now(), newNote.NoteType, newNote.Content)
	if err != nil {
		log.HTTPError(req, "Failed to insert new note: "+err.Error())
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}
}

func UpdateInventory(w http.ResponseWriter, req *http.Request) {
	// Field time: auto-set to current timestamp at DB insertion
	// Field tagnumber: required (int64), 6 digits, cannot be below 100000 or above 999999, numeric ASCII only
	// Field system_serial: required (string), min 4 chars, max 128 chars, alphanumeric ASCII only
	// Field location: required (string), min 1 char, max 128 chars, printable ASCII only
	// Field is_broken: optional (bool)
	// Field disk_removed: optional (bool)
	// Field department: required (string), must match foreign key in database
	// Field domain: required (string), must match foreign key in database
	// Field note: optional (string), max 512 chars, printable ASCII only
	// Field status: mandatory (string), must match existing foreign key in DB
	// Field system_manufacturer: optional (string), min 1 char, max 24 chars, alphanumeric ASCII only
	// Field system_model: optional (string), min 1 char, max 64 chars, alphanumeric ASCII only
	ctx := req.Context()
	log := middleware.GetLoggerFromContext(ctx)
	requestPath, err := middleware.GetRequestPathFromContext(ctx)
	if err != nil {
		log.HTTPWarning(req, "Error retrieving URL path from context for UpdateInventory")
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}
	requestMethod := req.Method

	// Check for POST method and correct URL
	if requestMethod != http.MethodPost || !(strings.HasSuffix(requestPath, "/update_inventory")) {
		log.HTTPWarning(req, "Invalid method or URL for inventory update")
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}

	// Parse inventory data
	if err := req.ParseMultipartForm(maxInventoryFormSizeBytes); err != nil {
		log.HTTPWarning(req, "Cannot parse multipart form: "+err.Error())
		middleware.WriteJsonError(w, http.StatusRequestEntityTooLarge)
		return
	}

	jsonFile, _, err := req.FormFile("json")
	if err != nil {
		log.HTTPWarning(req, "Error retrieving JSON data provided in form: "+err.Error())
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}
	defer jsonFile.Close()

	var inventoryUpdate database.InventoryUpdateFormInput
	if err := json.NewDecoder(jsonFile).Decode(&inventoryUpdate); err != nil {
		log.HTTPWarning(req, "Cannot decode JSON for UpdateInventory: "+err.Error())
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}

	// Validate and sanitize input data
	// Tag number (required, 6 digits)
	if inventoryUpdate.Tagnumber == nil || *inventoryUpdate.Tagnumber == 0 {
		log.HTTPWarning(req, "No tag number provided for UpdateInventory")
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}
	if !middleware.IsNumericAscii([]byte(strconv.FormatInt(*inventoryUpdate.Tagnumber, 10))) {
		log.HTTPWarning(req, "Non-digit characters in tag number field for UpdateInventory")
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}
	if utf8.RuneCountInString(strconv.FormatInt(*inventoryUpdate.Tagnumber, 10)) != 6 {
		log.HTTPWarning(req, "Tag number is not 6 digits for UpdateInventory")
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}
	tagnumber, err := strconv.ParseInt(strconv.FormatInt(*inventoryUpdate.Tagnumber, 10), 10, 64)
	if err != nil {
		log.HTTPWarning(req, "Cannot parse tag number in UpdateInventory")
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}
	if tagnumber < 100000 || tagnumber > 999999 {
		log.HTTPWarning(req, "Invalid range for tag number provided in UpdateInventory")
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}

	// System serial (required, min 4 chars, max 128 chars)
	if inventoryUpdate.SystemSerial == nil || strings.TrimSpace(*inventoryUpdate.SystemSerial) == "" {
		log.HTTPWarning(req, "Invalid system serial provided for UpdateInventory")
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}
	if utf8.RuneCountInString(*inventoryUpdate.SystemSerial) < 4 || utf8.RuneCountInString(*inventoryUpdate.SystemSerial) > 128 {
		log.HTTPWarning(req, "Invalid system serial length provided for UpdateInventory")
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}
	if !middleware.IsASCIIStringPrintable(*inventoryUpdate.SystemSerial) {
		log.HTTPWarning(req, "Non-alphanumeric characters in system serial field for UpdateInventory")
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}
	*inventoryUpdate.SystemSerial = strings.TrimSpace(*inventoryUpdate.SystemSerial)

	// Location (required, min 1 char, max 128 chars)
	if inventoryUpdate.Location == nil || strings.TrimSpace(*inventoryUpdate.Location) == "" {
		log.HTTPWarning(req, "No location provided for UpdateInventory")
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}
	if utf8.RuneCountInString(*inventoryUpdate.Location) < 1 || utf8.RuneCountInString(*inventoryUpdate.Location) > 128 {
		log.HTTPWarning(req, "Invalid location length for UpdateInventory")
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}
	if !utf8.ValidString(*inventoryUpdate.Location) {
		log.HTTPWarning(req, "Invalid UTF-8 in location field for inventory update")
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}
	*inventoryUpdate.Location = strings.TrimSpace(*inventoryUpdate.Location)

	// Broken (optional, bool)
	if inventoryUpdate.Broken == nil {
		log.HTTPInfo(req, "No broken bool value provided for inventory update")
	}

	// Disk removed (optional, bool)
	if inventoryUpdate.DiskRemoved == nil {
		log.HTTPInfo(req, "No disk removed bool value provided for inventory update")
	}

	// Department (required, max 24 chars)
	if inventoryUpdate.Department == nil || strings.TrimSpace(*inventoryUpdate.Department) == "" {
		log.HTTPWarning(req, "No department provided for inventory update")
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}
	if utf8.RuneCountInString(*inventoryUpdate.Department) < 1 || utf8.RuneCountInString(*inventoryUpdate.Department) > 24 {
		log.HTTPWarning(req, "Invalid department length for inventory update")
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}
	if !middleware.IsASCIIStringPrintable(*inventoryUpdate.Department) {
		log.HTTPWarning(req, "Non-printable ASCII characters in department field for inventory update")
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}
	*inventoryUpdate.Department = strings.TrimSpace(*inventoryUpdate.Department)

	// Domain (required, min 1 char, max 24 chars)
	if inventoryUpdate.Domain == nil || strings.TrimSpace(*inventoryUpdate.Domain) == "" {
		log.HTTPWarning(req, "No domain provided for inventory update")
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}
	if utf8.RuneCountInString(*inventoryUpdate.Domain) < 1 || utf8.RuneCountInString(*inventoryUpdate.Domain) > 24 {
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

	// Note (optional, max 512 chars)
	if inventoryUpdate.Note != nil && strings.TrimSpace(*inventoryUpdate.Note) != "" {
		if utf8.RuneCountInString(*inventoryUpdate.Note) < 1 || utf8.RuneCountInString(*inventoryUpdate.Note) > 512 {
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

	// Status (required, max 64 chars)
	if inventoryUpdate.Status == nil || strings.TrimSpace(*inventoryUpdate.Status) == "" {
		log.HTTPWarning(req, "No status provided for inventory update")
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}
	if utf8.RuneCountInString(*inventoryUpdate.Status) < 1 || utf8.RuneCountInString(*inventoryUpdate.Status) > 64 {
		log.HTTPWarning(req, "Invalid status length for inventory update")
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}
	if !middleware.IsASCIIStringPrintable(*inventoryUpdate.Status) {
		log.HTTPWarning(req, "Non-printable ASCII characters in status field for inventory update")
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}
	*inventoryUpdate.Status = strings.TrimSpace(*inventoryUpdate.Status)

	// System manufacturer (optional, min 1 char, max 24 chars)
	if inventoryUpdate.SystemManufacturer != nil && strings.TrimSpace(*inventoryUpdate.SystemManufacturer) != "" {
		if utf8.RuneCountInString(*inventoryUpdate.SystemManufacturer) < 1 || utf8.RuneCountInString(*inventoryUpdate.SystemManufacturer) > 24 {
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

	// System model (optional, min 1 char, max 64 chars)
	if inventoryUpdate.SystemModel != nil && strings.TrimSpace(*inventoryUpdate.SystemModel) != "" {
		if utf8.RuneCountInString(*inventoryUpdate.SystemModel) < 1 || utf8.RuneCountInString(*inventoryUpdate.SystemModel) > 64 {
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

	// acquired date, optional, process as UTC
	if inventoryUpdate.AcquiredDate != nil {
		acquiredDateUTC := inventoryUpdate.AcquiredDate.UTC()
		inventoryUpdate.AcquiredDate = &acquiredDateUTC
	} else {
		log.HTTPInfo(req, "No acquired date provided for inventory update")
	}

	// Other part of form:
	// Image/File uploads (base64, optional, max 64MB, multiple file uploads supported)
	var files []*multipart.FileHeader
	if req.MultipartForm != nil && req.MultipartForm.File != nil {
		if f := req.MultipartForm.File["inventory-file-input"]; len(f) > 0 {
			files = f
		}
	}

	// Establish DB connection before processing files
	dbConn := config.GetDatabaseConn()
	if dbConn == nil {
		log.HTTPError(req, "No database connection available for UpdateInventory")
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}
	updateRepo := database.NewRepo(dbConn)

	// Process uploaded files
	for _, fileHeader := range files {
		var manifest database.ImageManifest
		// Check multipart headers

		// File name/extension checks
		if matched, _ := regexp.MatchString(allowedFileNameRegex, fileHeader.Filename); !matched {
			log.HTTPWarning(req, "Invalid characters in uploaded file name for UpdateInventory")
			continue
		}
		if !slices.Contains(allowedFileExtensions, strings.ToLower(filepath.Ext(fileHeader.Filename))) {
			log.HTTPWarning(req, "Uploaded file has disallowed extension for UpdateInventory: ("+fileHeader.Filename+")")
			continue
		}

		// File size from multipart header checks
		if fileHeader.Size > maxInventoryFileSizeBytes {
			log.HTTPWarning(req, "Multipart header size value is too large for UpdateInventory ("+strconv.FormatInt(fileHeader.Size, 10)+" bytes)")
			continue
		}
		if fileHeader.Size == 0 {
			log.HTTPWarning(req, "Multipart header size value is empty for UpdateInventory: "+fileHeader.Filename)
			continue
		}
		if fileHeader.Size < minInventoryFileSizeBytes {
			log.HTTPWarning(req, "Multipart header size value too small for UpdateInventory: "+fileHeader.Filename+" ("+strconv.FormatInt(fileHeader.Size, 10)+" bytes)")
			continue
		}

		// Open uploaded file
		file, err := fileHeader.Open()
		if err != nil {
			log.HTTPWarning(req, "Failed to open uploaded file for UpdateInventory: "+err.Error())
			continue
		}
		defer file.Close()

		lr := &io.LimitedReader{R: file, N: maxInventoryFileSizeBytes + 1}
		fileBytes, err := io.ReadAll(lr)
		if err != nil {
			_ = file.Close()
			log.HTTPWarning(req, "Failed to read uploaded file for UpdateInventory: "+err.Error())
			continue
		}

		// File size checks (in addition to header checks - not necessarily same value)
		fileSize := len(fileBytes)
		if fileSize > maxInventoryFileSizeBytes {
			_ = file.Close()
			log.HTTPWarning(req, "Uploaded file too large for UpdateInventory ("+strconv.Itoa(fileSize)+" bytes)")
			continue
		}
		if fileSize == 0 {
			_ = file.Close()
			log.HTTPWarning(req, "Empty file uploaded for UpdateInventory: "+fileHeader.Filename)
			continue
		}
		if fileSize < minInventoryFileSizeBytes {
			_ = file.Close()
			log.HTTPWarning(req, "Uploaded file too small for UpdateInventory: "+fileHeader.Filename+" ("+strconv.Itoa(fileSize)+" bytes)")
			continue
		}
		*manifest.FileSize = int64(fileSize)

		// MIME type detection
		mimeType := http.DetectContentType(fileBytes[:fileSize])
		if !strings.HasPrefix(mimeType, "image/") { // temporary while implementing video support
			_ = file.Close()
			log.HTTPWarning(req, "Uploaded file has a non-accepted MIME type for UpdateInventory: (Content-Type: "+mimeType+")")
			continue
		}
		*manifest.MimeType = mimeType

		// Create reader (stream) for image decoding
		imageReader := bytes.NewReader(fileBytes)

		// Rewind and decode image to get image.Image
		_, err = imageReader.Seek(0, io.SeekStart)
		if err != nil {
			_ = file.Close()
			log.HTTPError(req, "Failed to seek to start of uploaded image for UpdateInventory: "+err.Error())
			continue
		}
		decodedImage, imageFormat, err := image.Decode(imageReader)
		if err != nil {
			_ = file.Close()
			log.HTTPError(req, "Failed to decode thumbnail in UpdateInventory: "+err.Error()+" ("+fileHeader.Filename+")")
			continue
		}

		// Rewind and decode image to get image config
		_, err = imageReader.Seek(0, io.SeekStart)
		if err != nil {
			_ = file.Close()
			log.HTTPError(req, "Failed to seek to start of uploaded image for UpdateInventory: "+err.Error()+" ("+fileHeader.Filename+")")
			continue
		}
		decodedImageConfig, _, err := image.DecodeConfig(imageReader)
		if err != nil {
			_ = file.Close()
			log.HTTPError(req, "Failed to decode uploaded image config for UpdateInventory: "+err.Error()+": "+fileHeader.Filename+" ("+fileHeader.Filename+")")
			continue
		}
		*manifest.ResolutionX = int64(decodedImageConfig.Width)
		*manifest.ResolutionY = int64(decodedImageConfig.Height)

		// Get upload timestamp
		fileTimeStamp := time.Now()
		*manifest.Time = fileTimeStamp.UTC()

		// Generate unique file name
		fileTimeStampFormatted := fileTimeStamp.Format("2006-01-02-150405")
		fileUUID := uuid.New()
		var fileName string
		baseFileName := fileTimeStampFormatted + "-" + fileUUID.String()
		switch mimeType {
		case "image/jpeg", "image/jpg":
			if imageFormat != "jpeg" {
				log.HTTPWarning(req, "MIME type and image format mismatch for uploaded file for UpdateInventory: (MIME: "+mimeType+", Format: "+imageFormat+")")
				continue
			}
			fileName = baseFileName + ".jpeg"
		case "image/png":
			if imageFormat != "png" {
				log.HTTPWarning(req, "MIME type and image format mismatch for uploaded file for UpdateInventory: (MIME: "+mimeType+", Format: "+imageFormat+")")
				continue
			}
			fileName = baseFileName + ".png"
		case "video/mp4":
			fileName = baseFileName + ".mp4"
		case "video/quicktime":
			fileName = baseFileName + ".mov"
		default:
			log.HTTPWarning(req, "Unsupported image MIME type for UpdateInventory: (Content-Type: "+mimeType+")")
			continue
		}
		*manifest.FileName = fileName
		*manifest.UUID = fileUUID.String()

		// Compute SHA256 hash of file
		fileHash := crypto.SHA256.New()
		if _, err := fileHash.Write(fileBytes); err != nil {
			_ = file.Close()
			log.HTTPError(req, "Failed to compute hash of uploaded file for UpdateInventory: "+err.Error()+" ("+fileHeader.Filename+")")
			middleware.WriteJsonError(w, http.StatusInternalServerError)
			return
		}
		fileHashBytes := fileHash.Sum(nil)
		*manifest.SHA256Hash = fmt.Sprintf("%x", fileHashBytes)

		// Create directories if not existing
		imageDirectoryPath := filepath.Join("./inventory-images", fmt.Sprintf("%06d", tagnumber))
		err = os.MkdirAll(imageDirectoryPath, 0755)
		if err != nil {
			_ = file.Close()
			log.HTTPError(req, "Failed to create directories for uploaded file for UpdateInventory: "+err.Error())
			middleware.WriteJsonError(w, http.StatusInternalServerError)
			return
		}

		// Set file/directory permissions
		if err := os.Chmod(imageDirectoryPath, 0755); err != nil {
			_ = file.Close()
			log.HTTPError(req, "Failed to set directory permissions: "+err.Error()+" ("+fileHeader.Filename+")")
			middleware.WriteJsonError(w, http.StatusInternalServerError)
			return
		}

		fullFilePath := filepath.Join(imageDirectoryPath, fileName)
		if err := os.WriteFile(fullFilePath, fileBytes, 0644); err != nil {
			_ = file.Close()
			log.HTTPError(req, "Failed to save uploaded file for UpdateInventory: "+err.Error()+" ("+fileHeader.Filename+")")
			middleware.WriteJsonError(w, http.StatusInternalServerError)
			return
		}
		*manifest.FilePath = fullFilePath

		// Close the uploaded file, not needed anymore
		_ = file.Close()

		var fullThumbnailPath string
		if strings.HasPrefix(mimeType, "image/") {
			fullThumbnailPath = filepath.Join("./inventory-images", fmt.Sprintf("%06d", tagnumber), "thumbnail-"+baseFileName+".jpeg")
			thumbnailFile, err := os.Create(fullThumbnailPath)
			if err != nil {
				log.HTTPError(req, "Failed to create thumbnail file for UpdateInventory: "+err.Error()+" ("+fileHeader.Filename+")")
				middleware.WriteJsonError(w, http.StatusInternalServerError)
				return
			}
			defer thumbnailFile.Close()

			err = jpeg.Encode(thumbnailFile, decodedImage, &jpeg.Options{Quality: 50})
			if err != nil {
				_ = thumbnailFile.Close()
				log.HTTPError(req, "Failed to encode thumbnail image for UpdateInventory: "+err.Error()+" ("+fileHeader.Filename+")")
				middleware.WriteJsonError(w, http.StatusInternalServerError)
				return
			}
			thumbnailFile.Close()
		}
		*manifest.ThumbnailFilePath = fullThumbnailPath

		// Insert image metadata into database
		*manifest.Tagnumber = tagnumber
		*manifest.Hidden = false
		*manifest.PrimaryImage = false

		err = updateRepo.UpdateClientImages(ctx, manifest)
		if err != nil {
			log.HTTPError(req, "Failed to update inventory image data: "+err.Error()+" ("+fileHeader.Filename+")")
			middleware.WriteJsonError(w, http.StatusInternalServerError)
			return
		}
		log.HTTPInfo(req, fmt.Sprintf("Uploaded file details - Name: %s, Size: %.2f MB, MIME Type: %s", fileName, float64(*manifest.FileSize)/1024/1024, mimeType)+" ("+fileHeader.Filename+")")
		file.Close()
	}
	// Update db

	// No pointers here, pointers in repo
	// tagnumber and broken bool are converted above
	err = updateRepo.InsertInventory(ctx, &inventoryUpdate)
	if err != nil {
		log.HTTPError(req, "Failed to update inventory data: "+err.Error())
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}
	err = updateRepo.UpdateSystemData(ctx, tagnumber, inventoryUpdate.SystemManufacturer, inventoryUpdate.SystemModel)
	if err != nil {
		log.HTTPError(req, "Failed to update system data: "+err.Error())
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}

	var jsonResponse = struct {
		Tagnumber int64  `json:"tagnumber"`
		Message   string `json:"message"`
	}{
		Tagnumber: tagnumber,
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

	db := config.GetDatabaseConn()
	if db == nil {
		log.HTTPWarning(req, "no database connection available")
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}
	repo := database.NewRepo(db)
	err := repo.TogglePinImage(ctx, uuid, tagnumber)
	if err != nil {
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
	db := config.GetDatabaseConn()
	if db == nil {
		log.HTTPWarning(req, "no database connection available for SetClientBatteryHealth")
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}
	repo := database.NewRepo(db)
	err = repo.SetClientBatteryHealth(ctx, uuid, &body.BatteryHealth)
	if err != nil {
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

	db := config.GetDatabaseConn()
	if db == nil {
		log.HTTPWarning(req, "no database connection available for SetAllJobs")
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}
	repo := database.NewRepo(db)
	err = repo.SetAllJobs(ctx, clientJson)
	if err != nil {
		log.HTTPError(req, "Failed to set all jobs: "+err.Error())
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}
	middleware.WriteJson(w, http.StatusOK, "All jobs set successfully")
}
