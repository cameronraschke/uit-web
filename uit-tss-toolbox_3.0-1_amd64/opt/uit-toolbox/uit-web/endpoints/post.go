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
	"strconv"
	"strings"
	"time"
	config "uit-toolbox/config"
	"uit-toolbox/database"
	middleware "uit-toolbox/middleware"
	"unicode"
	"unicode/utf8"

	"github.com/google/uuid"
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
	ctx := req.Context()
	log := config.GetLogger()
	requestIP, ok := middleware.GetRequestIPFromRequestContext(req)
	if !ok {
		fmt.Println("IP address not stored in context (WebAuthEndpoint): (" + requestIP.String() + " " + req.Method + " " + req.URL.Path + ")")
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}
	requestPath, ok := middleware.GetRequestPathFromRequestContext(req)
	if !ok {
		fmt.Println("Request URL not stored in context (WebAuthEndpoint): (" + requestIP.String() + " " + req.Method + " " + requestPath + ")")
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}

	// Sanitize login POST request
	if req.Method != http.MethodPost || !strings.HasSuffix(requestPath, "/login") {
		log.Warning("Invalid method or URL for auth form sanitization: " + requestIP.String() + " ( " + requestPath + ")")
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}

	body, err := io.ReadAll(req.Body)
	if err != nil {
		log.Warning("Cannot read request body: " + err.Error() + " (" + requestIP.String() + ")")
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}
	defer req.Body.Close()

	base64String := strings.TrimSpace(string(body))
	if base64String == "" {
		log.Warning("No base64 string provided in auth form: " + requestIP.String())
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}

	decodedBytes, err := base64.StdEncoding.DecodeString(base64String)
	if err != nil {
		log.Warning("Invalid base64 encoding: " + err.Error() + " (" + requestIP.String() + ")")
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}

	if !utf8.Valid(decodedBytes) {
		log.Warning("Invalid UTF-8 in decoded data: " + requestIP.String())
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}
	var clientFormAuthData struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := json.Unmarshal(decodedBytes, &clientFormAuthData); err != nil {
		log.Warning("Invalid JSON structure: " + err.Error() + " (" + requestIP.String() + ")")
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}

	// Validate input data
	if err := middleware.ValidateAuthFormInputSHA256(clientFormAuthData.Username, clientFormAuthData.Password); err != nil {
		log.Warning("Invalid auth input: " + err.Error() + " (" + requestIP.String() + ")")
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}

	// Authenticate with bcrypt
	authenticated, err := middleware.CheckAuthCredentials(ctx, clientFormAuthData.Username, clientFormAuthData.Password)
	if err != nil || !authenticated {
		log.Info("Authentication failed for " + requestIP.String() + ": " + err.Error())
		middleware.WriteJsonError(w, http.StatusUnauthorized)
		http.Redirect(w, req, "/login?error=1", http.StatusSeeOther)
		return
	}

	sessionID, basicToken, bearerToken, csrfToken, err := config.CreateAuthSession(requestIP)
	if err != nil {
		log.Error("Failed to generate tokens: " + err.Error())
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}

	sessionCount := config.GetAuthSessionCount()
	log.Info("New auth session created: " + requestIP.String() + " (Sessions: " + strconv.Itoa(int(sessionCount)) + ")")

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
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}
	ctx := requestInfo.Ctx
	log := requestInfo.Log
	requestIP := requestInfo.IP
	requestURL := requestInfo.URL
	requestMethod := req.Method

	// Sanitize POST request
	if requestMethod != http.MethodPost || !(strings.HasSuffix(requestURL, "/notes")) {
		log.Warning("Invalid method or URL for note insertion: " + requestIP.String() + " ( " + requestURL + ")")
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
		log.Warning("Cannot decode note JSON: " + err.Error() + " (" + requestIP.String() + ")")
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}
	defer req.Body.Close()

	newNote.NoteType = strings.TrimSpace(newNote.NoteType)
	if len(newNote.NoteType) > 64 {
		log.Warning("Note type too long: " + requestIP.String())
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(newNote.NoteType) == "" {
		log.Warning("Empty note type: " + requestIP.String())
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}
	newNote.Content = strings.TrimSpace(newNote.Content)
	if len(newNote.Content) > 2000 {
		log.Warning("Note content too long: " + requestIP.String())
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(newNote.Content) == "" {
		log.Warning("Empty note content: " + requestIP.String())
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}

	// Insert note into database
	dbConn := config.GetDatabaseConn()
	insertRepo := database.NewRepo(dbConn)
	err = insertRepo.InsertNewNote(ctx, time.Now(), newNote.NoteType, newNote.Content)
	if err != nil {
		log.Error("Failed to insert new note: " + err.Error() + " (" + requestIP.String() + ")")
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}
}

func UpdateInventory(w http.ResponseWriter, req *http.Request) {
	// Field tagnumber: required (int64), 6 digits, cannot be below 1 or above 999999, numeric ASCII only
	// Field system_serial: required (string), min 4 chars, max 64 chars, alphanumeric ASCII only
	// Field location: required (string), min 1 char, max 128 chars, printable ASCII only
	// Field system_manufacturer: optional (string), min 1 char, max 24 chars, alphanumeric ASCII only
	// Field system_model: optional (string), min 1 char, max 64 chars, alphanumeric ASCII only
	// Field department: optional (string), must match existing department in database
	// Field domain: optional (string), must match existing domain in database
	// Field broken: optional (bool)
	// Field status: mandatory (string), must match existing status in database table client_location_status, printable ASCII only
	// Field note: optional (string), max 2000 chars, printable ASCII only
	requestInfo, err := GetRequestInfo(req)
	if err != nil {
		fmt.Println("Cannot get request info error: " + err.Error())
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}
	ctx := requestInfo.Ctx
	log := requestInfo.Log
	requestIP := requestInfo.IP
	requestURL := requestInfo.URL
	requestMethod := req.Method

	// Check for POST method and correct URL
	if requestMethod != http.MethodPost || !(strings.HasSuffix(requestURL, "/update_inventory")) {
		log.Warning("Invalid method or URL for inventory update: " + requestIP.String() + " ( " + requestURL + ")")
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}

	// Parse inventory data
	if err := req.ParseMultipartForm(64 << 20); err != nil {
		log.Warning("Cannot parse multipart form: " + err.Error() + " (" + requestIP.String() + ")")
		middleware.WriteJsonError(w, http.StatusRequestEntityTooLarge)
		return
	}

	jsonFile, _, err := req.FormFile("json")
	if err != nil {
		log.Warning("No JSON data provided in form: " + requestIP.String())
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}
	defer jsonFile.Close()

	var inventoryUpdate database.InventoryUpdateFormInput
	if err := json.NewDecoder(jsonFile).Decode(&inventoryUpdate); err != nil {
		log.Warning("Cannot decode inventory JSON: " + err.Error() + " (" + requestIP.String() + ")")
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}

	// Validate and sanitize input data
	// Tag number (required, 6 digits)
	if inventoryUpdate.Tagnumber == nil || *inventoryUpdate.Tagnumber == 0 {
		log.Warning("No tag number provided for inventory update: " + requestIP.String())
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}
	if !middleware.IsNumericAscii([]byte(strconv.FormatInt(*inventoryUpdate.Tagnumber, 10))) {
		log.Warning("Non-digit characters in tag number field for inventory update: " + requestIP.String())
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}
	if utf8.RuneCountInString(strconv.FormatInt(*inventoryUpdate.Tagnumber, 10)) != 6 {
		log.Warning("Tag number not 6 digits for inventory update: " + requestIP.String())
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}
	tagnumber, err := strconv.ParseInt(strconv.FormatInt(*inventoryUpdate.Tagnumber, 10), 10, 64)
	if err != nil {
		log.Warning("Cannot parse tag number for inventory update: " + requestIP.String())
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}
	if tagnumber < 1 || tagnumber > 999999 {
		log.Warning("Invalid tag number provided for inventory update: " + requestIP.String())
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}

	// System serial (required, min 4 chars, max 64 chars)
	if inventoryUpdate.SystemSerial == nil || strings.TrimSpace(*inventoryUpdate.SystemSerial) == "" {
		log.Warning("Invalid system serial provided for inventory update: " + requestIP.String())
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}
	if utf8.RuneCountInString(*inventoryUpdate.SystemSerial) < 4 || utf8.RuneCountInString(*inventoryUpdate.SystemSerial) > 64 {
		log.Warning("Invalid system serial length provided for inventory update: " + requestIP.String())
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}
	if !middleware.IsAlphanumericAscii([]byte(*inventoryUpdate.SystemSerial)) {
		log.Warning("Non-alphanumeric characters in system serial field for inventory update: " + requestIP.String())
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}
	*inventoryUpdate.SystemSerial = strings.TrimSpace(*inventoryUpdate.SystemSerial)

	// Location (required, min 1 char, max 128 chars)
	if inventoryUpdate.Location == nil || strings.TrimSpace(*inventoryUpdate.Location) == "" {
		log.Warning("No location provided for inventory update: " + requestIP.String())
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}
	if utf8.RuneCountInString(*inventoryUpdate.Location) < 1 || utf8.RuneCountInString(*inventoryUpdate.Location) > 128 {
		log.Warning("Invalid location length for inventory update: " + requestIP.String())
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}
	if !middleware.IsPrintableASCII([]byte(*inventoryUpdate.Location)) {
		log.Warning("Non-printable ASCII characters in location field for inventory update: " + requestIP.String())
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}
	*inventoryUpdate.Location = strings.TrimSpace(*inventoryUpdate.Location)

	// System manufacturer (optional, min 1 char, max 24 chars)
	if inventoryUpdate.SystemManufacturer != nil && strings.TrimSpace(*inventoryUpdate.SystemManufacturer) != "" {
		if utf8.RuneCountInString(*inventoryUpdate.SystemManufacturer) < 1 || utf8.RuneCountInString(*inventoryUpdate.SystemManufacturer) > 24 {
			log.Warning("Invalid system manufacturer length for inventory update: " + requestIP.String())
			middleware.WriteJsonError(w, http.StatusBadRequest)
			return
		}
		if !middleware.IsAlphanumericAscii([]byte(*inventoryUpdate.SystemManufacturer)) {
			log.Warning("Non-alphanumeric characters in system manufacturer field for inventory update: " + requestIP.String())
			middleware.WriteJsonError(w, http.StatusBadRequest)
			return
		}
		*inventoryUpdate.SystemManufacturer = strings.TrimSpace(*inventoryUpdate.SystemManufacturer)
	} else {
		log.Info("No system manufacturer provided for inventory update: " + requestIP.String())
	}

	// System model (optional, min 1 char, max 64 chars)
	if inventoryUpdate.SystemModel != nil && strings.TrimSpace(*inventoryUpdate.SystemModel) != "" {
		if utf8.RuneCountInString(*inventoryUpdate.SystemModel) < 1 || utf8.RuneCountInString(*inventoryUpdate.SystemModel) > 64 {
			log.Warning("Invalid system model length for inventory update: " + requestIP.String())
			middleware.WriteJsonError(w, http.StatusBadRequest)
			return
		}
		if !middleware.IsAlphanumericAscii([]byte(*inventoryUpdate.SystemModel)) {
			log.Warning("Non-alphanumeric characters in system model field for inventory update: " + requestIP.String())
			middleware.WriteJsonError(w, http.StatusBadRequest)
			return
		}
		*inventoryUpdate.SystemModel = strings.TrimSpace(*inventoryUpdate.SystemModel)
	} else {
		log.Info("No system model provided for inventory update: " + requestIP.String())
	}

	// Department (optional, max 24 chars)
	if inventoryUpdate.Department != nil && strings.TrimSpace(*inventoryUpdate.Department) != "" {
		if utf8.RuneCountInString(*inventoryUpdate.Department) < 1 || utf8.RuneCountInString(*inventoryUpdate.Department) > 24 {
			log.Warning("Invalid department length for inventory update: " + requestIP.String())
			middleware.WriteJsonError(w, http.StatusBadRequest)
			return
		}
		if !middleware.IsPrintableASCII([]byte(*inventoryUpdate.Department)) {
			log.Warning("Non-printable ASCII characters in department field for inventory update: " + requestIP.String())
			middleware.WriteJsonError(w, http.StatusBadRequest)
			return
		}
		*inventoryUpdate.Department = strings.TrimSpace(*inventoryUpdate.Department)
	} else {
		log.Info("No department provided for inventory update: " + requestIP.String())
	}

	// Domain (optional, min 1 char, max 24 chars)
	if inventoryUpdate.Domain != nil && strings.TrimSpace(*inventoryUpdate.Domain) != "" {
		if utf8.RuneCountInString(*inventoryUpdate.Domain) < 1 || utf8.RuneCountInString(*inventoryUpdate.Domain) > 24 {
			log.Warning("Invalid domain length for inventory update: " + requestIP.String())
			middleware.WriteJsonError(w, http.StatusBadRequest)
			return
		}
		if !middleware.IsPrintableASCII([]byte(*inventoryUpdate.Domain)) {
			log.Warning("Non-printable ASCII characters in domain field for inventory update: " + requestIP.String())
			middleware.WriteJsonError(w, http.StatusBadRequest)
			return
		}
		*inventoryUpdate.Domain = strings.TrimSpace(*inventoryUpdate.Domain)
	} else {
		log.Info("No domain provided for inventory update: " + requestIP.String())
	}

	// Broken (optional, bool)
	var brokenBool bool
	if inventoryUpdate.Broken != nil {
		brokenBool, err = strconv.ParseBool(strconv.FormatBool(*inventoryUpdate.Broken))
		if err != nil {
			log.Warning("Cannot parse broken bool value for inventory update: " + requestIP.String())
			middleware.WriteJsonError(w, http.StatusBadRequest)
			return
		}
		if !brokenBool && brokenBool {
			log.Warning("Invalid broken bool value for inventory update: " + requestIP.String())
			middleware.WriteJsonError(w, http.StatusBadRequest)
			return
		}
	} else {
		log.Info("No broken bool value provided for inventory update: " + requestIP.String())
	}
	*inventoryUpdate.Broken = brokenBool

	// Status (mandatory, max 64 chars)
	if inventoryUpdate.Status == nil || strings.TrimSpace(*inventoryUpdate.Status) == "" {
		log.Warning("No status provided for inventory update: " + requestIP.String())
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}
	if utf8.RuneCountInString(*inventoryUpdate.Status) < 1 || utf8.RuneCountInString(*inventoryUpdate.Status) > 64 {
		log.Warning("Invalid status length for inventory update: " + requestIP.String())
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}
	if !middleware.IsPrintableASCII([]byte(*inventoryUpdate.Status)) {
		log.Warning("Non-printable ASCII characters in status field for inventory update: " + requestIP.String())
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}
	*inventoryUpdate.Status = strings.TrimSpace(*inventoryUpdate.Status)

	// Note (optional, max 2000 chars)
	if inventoryUpdate.Note != nil && strings.TrimSpace(*inventoryUpdate.Note) != "" {
		if utf8.RuneCountInString(*inventoryUpdate.Note) < 1 || utf8.RuneCountInString(*inventoryUpdate.Note) > 2000 {
			log.Warning("Invalid note length for inventory update: " + requestIP.String())
			middleware.WriteJsonError(w, http.StatusBadRequest)
			return
		}
		for _, rune := range *inventoryUpdate.Note {
			if !unicode.IsPrint(rune) && !unicode.IsSpace(rune) {
				log.Warning("Non-printable characters in note field for inventory update: " + requestIP.String())
				middleware.WriteJsonError(w, http.StatusBadRequest)
				return
			}
		}
		*inventoryUpdate.Note = strings.TrimSpace(*inventoryUpdate.Note)
	} else {
		log.Warning("No note provided for inventory update: " + requestIP.String())
	}

	// Image (base64, optional, max 64MB, multiple file uploads supported)
	var files []*multipart.FileHeader
	if req.MultipartForm != nil && req.MultipartForm.File != nil {
		if f := req.MultipartForm.File["inventory-file-input"]; len(f) > 0 {
			files = f
		}
	}

	// Establish DB connection before processing files
	dbConn := config.GetDatabaseConn()
	if dbConn == nil {
		log.Error("No database connection available for inventory update: " + requestIP.String())
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}
	updateRepo := database.NewRepo(dbConn)

	const maxFileSize = 64 << 20 // 64 MB
	for _, fileHeader := range files {
		for _, char := range fileHeader.Filename {
			if !(unicode.IsLetter(char) || unicode.IsDigit(char) ||
				char == '.' || char == '-' || char == '_' || char == ' ' || char == '(' || char == ')') {
				log.Warning("Invalid characters in uploaded file name for inventory update: " + requestIP.String())
				middleware.WriteJsonError(w, http.StatusBadRequest)
				return
			}
		}
		if fileHeader.Size > maxFileSize {
			log.Warning("Uploaded file too large for inventory update: " + requestIP.String() + " (" + strconv.FormatInt(fileHeader.Size, 10) + " bytes)")
			middleware.WriteJsonError(w, http.StatusBadRequest)
			return
		}
		file, err := fileHeader.Open()
		if err != nil {
			log.Warning("Failed to open uploaded file for inventory update: " + err.Error() + " (" + requestIP.String() + ")")
			middleware.WriteJsonError(w, http.StatusBadRequest)
			return
		}
		defer file.Close()

		lr := &io.LimitedReader{R: file, N: maxFileSize + 1}
		fileData, err := io.ReadAll(lr)
		if err != nil {
			log.Warning("Failed to read uploaded file for inventory update: " + err.Error() + " (" + requestIP.String() + ")")
			middleware.WriteJsonError(w, http.StatusBadRequest)
			return
		}

		fileSize := int64(len(fileData))
		if fileSize > maxFileSize {
			log.Warning("Uploaded file too large for inventory update: " + requestIP.String() + " (" + strconv.FormatInt(fileSize, 10) + " bytes)")
			middleware.WriteJsonError(w, http.StatusBadRequest)
			return
		}
		if fileSize == 0 {
			log.Warning("Empty file uploaded for inventory update: " + requestIP.String())
			middleware.WriteJsonError(w, http.StatusBadRequest)
			return
		}
		if fileSize < 512 {
			log.Warning("Uploaded file too small for inventory update: " + requestIP.String() + " (" + strconv.FormatInt(fileSize, 10) + " bytes)")
			middleware.WriteJsonError(w, http.StatusBadRequest)
			return
		}
		allowedRegex := regexp.MustCompile(`^[a-zA-Z0-9.\-_ ()]+\.[a-zA-Z]+$`)
		if !allowedRegex.MatchString(fileHeader.Filename) {
			log.Warning("Invalid characters in uploaded file name for inventory update: " + requestIP.String())
			middleware.WriteJsonError(w, http.StatusBadRequest)
			return
		}
		disallowedExtensions := []string{".exe", ".bat", ".sh", ".js", ".html", ".zip", ".rar", ".7z", ".tar", ".gz", ".dll", ".sys", ".ps1", ".cmd"}
		lowerFileName := strings.ToLower(fileHeader.Filename)
		for _, ext := range disallowedExtensions {
			if strings.HasSuffix(lowerFileName, ext) {
				log.Warning("Uploaded file has disallowed extension for inventory update: " + requestIP.String() + " (" + fileHeader.Filename + ")")
				middleware.WriteJsonError(w, http.StatusBadRequest)
				return
			}
		}
		mimeType := http.DetectContentType(fileData[:fileSize])
		if !strings.HasPrefix(mimeType, "image/") {
			log.Warning("Uploaded file is not an image for inventory update: " + requestIP.String() + " (Content-Type: " + mimeType + ")")
			middleware.WriteJsonError(w, http.StatusUnsupportedMediaType)
			return
		}
		imageReader := bytes.NewReader(fileData)
		_, err = imageReader.Seek(0, io.SeekStart)
		if err != nil {
			log.Error("Failed to seek to start of uploaded image for inventory update: " + err.Error() + " (" + requestIP.String() + ")")
			middleware.WriteJsonError(w, http.StatusInternalServerError)
			return
		}
		decodedImage, imageFormat, err := image.Decode(imageReader)
		if err != nil {
			log.Error("Failed to decode uploaded image for thumbnail creation for inventory update: " + err.Error() + " (" + requestIP.String() + ")")
		}
		_, err = imageReader.Seek(0, io.SeekStart)
		if err != nil {
			log.Error("Failed to seek to start of uploaded image for inventory update: " + err.Error() + " (" + requestIP.String() + ")")
			middleware.WriteJsonError(w, http.StatusInternalServerError)
			return
		}
		decodedImageConfig, _, err := image.DecodeConfig(imageReader)
		if err != nil {
			log.Error("Failed to decode uploaded image config for inventory update: " + err.Error() + ": " + fileHeader.Filename + " (" + requestIP.String() + ")")
		}
		resolutionX := decodedImageConfig.Width
		resolutionY := decodedImageConfig.Height

		fileTimeStamp := time.Now().Format("2006-01-02-150405")
		fileUUID := uuid.New()
		var fileName string
		baseFileName := fileTimeStamp + "-" + fileUUID.String()
		switch mimeType {
		case "image/jpeg", "image/jpg":
			if imageFormat != "jpeg" {
				log.Warning("MIME type and image format mismatch for uploaded file for inventory update: " + requestIP.String() + " (MIME: " + mimeType + ", Format: " + imageFormat + ")")
				middleware.WriteJsonError(w, http.StatusBadRequest)
				return
			}
			fileName = baseFileName + ".jpeg"
		case "image/png":
			if imageFormat != "png" {
				log.Warning("MIME type and image format mismatch for uploaded file for inventory update: " + requestIP.String() + " (MIME: " + mimeType + ", Format: " + imageFormat + ")")
				middleware.WriteJsonError(w, http.StatusBadRequest)
				return
			}
			fileName = baseFileName + ".png"
		case "video/mp4":
			fileName = baseFileName + ".mp4"
		case "video/quicktime":
			fileName = baseFileName + ".mov"
		default:
			log.Warning("Unsupported image MIME type for inventory update: " + requestIP.String() + " (Content-Type: " + mimeType + ")")
			middleware.WriteJsonError(w, http.StatusUnsupportedMediaType)
			return
		}
		fileHash := crypto.SHA256.New()
		if _, err := fileHash.Write(fileData); err != nil {
			log.Error("Failed to compute hash of uploaded file for inventory update: " + err.Error() + " (" + requestIP.String() + ")")
			middleware.WriteJsonError(w, http.StatusInternalServerError)
			return
		}
		fileHashBytes := fileHash.Sum(nil)

		imageDirectoryPath := filepath.Join("./inventory-images", fmt.Sprintf("%06d", tagnumber))
		err = os.MkdirAll(imageDirectoryPath, 0755)
		if err != nil {
			log.Error("Failed to create directories for uploaded file for inventory update: " + err.Error() + " (" + requestIP.String() + ")")
		}

		fullFilePath := filepath.Join(imageDirectoryPath, fileName)
		if err := os.WriteFile(fullFilePath, fileData, 0644); err != nil {
			log.Error("Failed to save uploaded file for inventory update: " + err.Error() + " (" + requestIP.String() + ")")
			middleware.WriteJsonError(w, http.StatusInternalServerError)
			return
		}

		var fullThumbnailPath string
		if mimeType != "image/jpeg" && mimeType != "image/jpg" && mimeType != "image/png" {
			fullThumbnailPath := filepath.Join("./inventory-images", fmt.Sprintf("%06d", tagnumber), "thumbnail-"+baseFileName+".jpeg")
			thumbnailFile, err := os.Create(fullThumbnailPath)
			if err != nil {
				log.Error("Failed to create thumbnail file for inventory update: " + err.Error() + " (" + requestIP.String() + ")")
				middleware.WriteJsonError(w, http.StatusInternalServerError)
				return
			}
			defer thumbnailFile.Close()
			err = jpeg.Encode(thumbnailFile, decodedImage, &jpeg.Options{Quality: 50})
			if err != nil {
				log.Error("Failed to encode thumbnail image for inventory update: " + err.Error() + " (" + requestIP.String() + ")")
				middleware.WriteJsonError(w, http.StatusInternalServerError)
				return
			}
			thumbnailFile.Close()
		}

		// Insert image metadata into database
		fileSizeMB := float64(fileSize) / (2 << 20)
		hidden := false
		primaryImage := false
		err = updateRepo.UpdateClientImages(ctx, tagnumber, fileUUID.String(), &fileName, fullFilePath, &fullThumbnailPath, &fileSizeMB, &fileHashBytes, &mimeType, nil, &resolutionX, &resolutionY, nil, &hidden, &primaryImage)
		if err != nil {
			log.Error("Failed to update inventory image data: " + err.Error() + " (" + requestIP.String() + ")")
			middleware.WriteJsonError(w, http.StatusInternalServerError)
			return
		}
		log.Info(fmt.Sprintf("Uploaded file details - Name: %s, Size: %.2f MB, MIME Type: %s", fileName, fileSizeMB, mimeType) + " (" + requestIP.String() + ")")
		file.Close()
	}
	// Update db

	// No pointers here, pointers in repo
	// tagnumber and broken bool are converted above
	err = updateRepo.InsertInventory(ctx, tagnumber, inventoryUpdate.SystemSerial, inventoryUpdate.Location, inventoryUpdate.Department, inventoryUpdate.Domain, inventoryUpdate.Broken, inventoryUpdate.Status, inventoryUpdate.Note)
	if err != nil {
		log.Error("Failed to update inventory data: " + err.Error() + " (" + requestIP.String() + ")")
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}
	err = updateRepo.UpdateSystemData(ctx, tagnumber, inventoryUpdate.SystemManufacturer, inventoryUpdate.SystemModel)
	if err != nil {
		log.Error("Failed to update system data: " + err.Error() + " (" + requestIP.String() + ")")
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}

	middleware.WriteJson(w, http.StatusOK, "Update successful")
}

func TogglePinImage(w http.ResponseWriter, req *http.Request) {
	requestInfo, err := GetRequestInfo(req)
	if err != nil {
		fmt.Println("Cannot get request info error: " + err.Error())
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}
	ctx := requestInfo.Ctx
	log := requestInfo.Log
	requestIP := requestInfo.IP
	requestURL := requestInfo.URL
	requestMethod := req.Method
	if requestMethod != http.MethodPost || !(strings.HasPrefix(requestURL, "/api/images/toggle_pin/")) {
		log.Warning("Invalid method or URL for toggle pin image: " + requestIP.String() + " ( " + requestURL + ")")
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}
	tagnumber, ok := ConvertRequestTagnumber(req)
	if tagnumber == 0 || !ok {
		log.Warning("No or invalid tagnumber provided in request from: " + requestIP.String() + " (" + requestURL + ")")
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}

	// Decode JSON body
	var body struct {
		UUID      string `json:"uuid"`
		Tagnumber int64  `json:"tagnumber"`
	}
	if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
		log.Error("Failed to decode JSON body: " + err.Error() + " (" + requestIP.String() + ")")
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}

	body.UUID = strings.TrimPrefix(body.UUID, "/api/images/toggle_pin/")
	body.UUID = strings.TrimSuffix(body.UUID, ".jpeg")
	body.UUID = strings.TrimSuffix(body.UUID, ".png")
	body.UUID = strings.TrimSuffix(body.UUID, ".mp4")
	body.UUID = strings.TrimSuffix(body.UUID, ".mov")
	if body.UUID == "" {
		log.Warning("No image path provided in request from: " + requestIP.String() + " (" + requestURL + ")")
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}

	uuid := strings.TrimSpace(body.UUID)

	db := config.GetDatabaseConn()
	if db == nil {
		log.Warning("no database connection available")
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}
	repo := database.NewRepo(db)
	err = repo.TogglePinImage(ctx, uuid, tagnumber)
	if err != nil {
		log.Error("Failed to toggle pin image: " + err.Error() + " (" + requestIP.String() + ")")
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}
	middleware.WriteJson(w, http.StatusOK, "Image pin toggled successfully")
}
