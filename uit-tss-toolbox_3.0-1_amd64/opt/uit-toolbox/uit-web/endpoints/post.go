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
	if err := req.ParseMultipartForm(64 << 20); err != nil {
		log.Warning("Cannot parse multipart form: " + err.Error() + " (" + requestIP + ")")
		middleware.WriteJsonError(w, http.StatusRequestEntityTooLarge, "Request entity too large")
		return
	}

	jsonFile, _, err := req.FormFile("json")
	if err != nil {
		log.Warning("No JSON data provided in form: " + requestIP)
		middleware.WriteJsonError(w, http.StatusBadRequest, "Bad request")
		return
	}
	defer jsonFile.Close()

	var inventoryUpdate database.InventoryUpdateFormInput
	if err := json.NewDecoder(jsonFile).Decode(&inventoryUpdate); err != nil {
		log.Warning("Cannot decode inventory JSON: " + err.Error() + " (" + requestIP + ")")
		middleware.WriteJsonError(w, http.StatusBadRequest, "Bad request")
		return
	}

	// Validate and sanitize input data
	// Tag number (required, 6 digits)
	if !middleware.IsNumericAscii([]byte(strconv.Itoa(inventoryUpdate.Tagnumber))) {
		log.Warning("Non-digit characters in tag number field for inventory update: " + requestIP)
		middleware.WriteJsonError(w, http.StatusBadRequest, "Bad request")
		return
	}
	if utf8.RuneCountInString(strconv.Itoa(inventoryUpdate.Tagnumber)) != 6 {
		log.Warning("Tag number not 6 digits for inventory update: " + requestIP)
		middleware.WriteJsonError(w, http.StatusBadRequest, "Bad request")
		return
	}
	tagnumber, err := strconv.ParseInt(strconv.Itoa(inventoryUpdate.Tagnumber), 10, 64)
	if err != nil {
		log.Warning("Cannot parse tag number for inventory update: " + requestIP)
		middleware.WriteJsonError(w, http.StatusBadRequest, "Bad request")
		return
	}
	if tagnumber < 1 || tagnumber > 999999 {
		log.Warning("Invalid tag number provided for inventory update: " + requestIP)
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

	// Department (optional, max 24 chars)
	if inventoryUpdate.Department != nil && strings.TrimSpace(*inventoryUpdate.Department) != "" {
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
	} else {
		log.Warning("No department provided for inventory update: " + requestIP)
	}

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
	if inventoryUpdate.Working == nil {
		log.Warning("No working bool value provided for inventory update: " + requestIP)
		middleware.WriteJsonError(w, http.StatusBadRequest, "Bad request")
		return
	}
	var workingBool bool
	workingBool, err = strconv.ParseBool(strconv.FormatBool(*inventoryUpdate.Working))
	if err != nil {
		log.Warning("Cannot parse working bool value for inventory update: " + requestIP)
		middleware.WriteJsonError(w, http.StatusBadRequest, "Bad request")
		return
	}
	if !workingBool && workingBool {
		log.Warning("Invalid working bool value for inventory update: " + requestIP)
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
	var files []*multipart.FileHeader
	if req.MultipartForm != nil && req.MultipartForm.File != nil {
		if f := req.MultipartForm.File["inventory-file-input"]; len(f) > 0 {
			files = f
		}
	}

	// Establish DB connection before processing files
	dbConn := config.GetDatabaseConn()
	if dbConn == nil {
		log.Error("No database connection available for inventory update: " + requestIP)
		middleware.WriteJsonError(w, http.StatusInternalServerError, "Internal server error")
		return
	}
	updateRepo := database.NewRepo(dbConn)

	const maxFileSize = 64 << 20 // 64 MB
	for _, fileHeader := range files {
		for _, char := range fileHeader.Filename {
			if !(unicode.IsLetter(char) || unicode.IsDigit(char) || char == '.' || char == '-' || char == '_') {
				log.Warning("Invalid characters in uploaded file name for inventory update: " + requestIP)
				middleware.WriteJsonError(w, http.StatusBadRequest, "Bad request")
				return
			}
		}
		if fileHeader.Size > maxFileSize {
			log.Warning("Uploaded file too large for inventory update: " + requestIP + " (" + strconv.FormatInt(fileHeader.Size, 10) + " bytes)")
			middleware.WriteJsonError(w, http.StatusBadRequest, "File too large")
			return
		}
		file, err := fileHeader.Open()
		if err != nil {
			log.Warning("Failed to open uploaded file for inventory update: " + err.Error() + " (" + requestIP + ")")
			middleware.WriteJsonError(w, http.StatusBadRequest, "Bad request")
			return
		}
		defer file.Close()

		lr := &io.LimitedReader{R: file, N: maxFileSize + 1}
		fileData, err := io.ReadAll(lr)
		if err != nil {
			log.Warning("Failed to read uploaded file for inventory update: " + err.Error() + " (" + requestIP + ")")
			middleware.WriteJsonError(w, http.StatusBadRequest, "Bad request")
			return
		}

		fileSize := int64(len(fileData))
		if fileSize > maxFileSize {
			log.Warning("Uploaded file too large for inventory update: " + requestIP + " (" + strconv.FormatInt(fileSize, 10) + " bytes)")
			middleware.WriteJsonError(w, http.StatusBadRequest, "File too large")
			return
		}
		if fileSize == 0 {
			log.Warning("Empty file uploaded for inventory update: " + requestIP)
			middleware.WriteJsonError(w, http.StatusBadRequest, "Bad request")
			return
		}
		if fileSize < 512 {
			log.Warning("Uploaded file too small for inventory update: " + requestIP + " (" + strconv.FormatInt(fileSize, 10) + " bytes)")
			middleware.WriteJsonError(w, http.StatusBadRequest, "File too small")
			return
		}
		mimeType := http.DetectContentType(fileData[:fileSize])
		if !strings.HasPrefix(mimeType, "image/") {
			log.Warning("Uploaded file is not an image for inventory update: " + requestIP + " (Content-Type: " + mimeType + ")")
			middleware.WriteJsonError(w, http.StatusUnsupportedMediaType, "Unsupported media type")
			return
		}
		imageReader := bytes.NewReader(fileData)
		_, err = imageReader.Seek(0, io.SeekStart)
		if err != nil {
			log.Error("Failed to seek to start of uploaded image for inventory update: " + err.Error() + " (" + requestIP + ")")
			middleware.WriteJsonError(w, http.StatusInternalServerError, "Internal server error")
			return
		}
		decodedImage, imageFormat, err := image.Decode(imageReader)
		if err != nil {
			log.Error("Failed to decode uploaded image for thumbnail creation for inventory update: " + err.Error() + " (" + requestIP + ")")
			middleware.WriteJsonError(w, http.StatusInternalServerError, "Internal server error")
			return
		}
		decodedImageConfig, _, err := image.DecodeConfig(imageReader)
		if err != nil {
			log.Error("Failed to decode uploaded image config for inventory update: " + err.Error() + " (" + requestIP + ")")
			middleware.WriteJsonError(w, http.StatusInternalServerError, "Internal server error")
			return
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
				log.Warning("MIME type and image format mismatch for uploaded file for inventory update: " + requestIP + " (MIME: " + mimeType + ", Format: " + imageFormat + ")")
				middleware.WriteJsonError(w, http.StatusBadRequest, "Bad request")
				return
			}
			fileName = baseFileName + ".jpeg"
		case "image/png":
			if imageFormat != "png" {
				log.Warning("MIME type and image format mismatch for uploaded file for inventory update: " + requestIP + " (MIME: " + mimeType + ", Format: " + imageFormat + ")")
				middleware.WriteJsonError(w, http.StatusBadRequest, "Bad request")
				return
			}
			fileName = baseFileName + ".png"
		default:
			log.Warning("Unsupported image MIME type for inventory update: " + requestIP + " (Content-Type: " + mimeType + ")")
			middleware.WriteJsonError(w, http.StatusUnsupportedMediaType, "Unsupported media type")
			return
		}
		fileHash := crypto.SHA256.New()
		if _, err := fileHash.Write(fileData); err != nil {
			log.Error("Failed to compute hash of uploaded file for inventory update: " + err.Error() + " (" + requestIP + ")")
			middleware.WriteJsonError(w, http.StatusInternalServerError, "Internal server error")
			return
		}
		fileHashBytes := fileHash.Sum(nil)

		fullFilePath := filepath.Join("./inventory-images", fmt.Sprintf("%06d", tagnumber), fileName)
		if err := os.WriteFile(fullFilePath, fileData, 0644); err != nil {
			log.Error("Failed to save uploaded file for inventory update: " + err.Error() + " (" + requestIP + ")")
			middleware.WriteJsonError(w, http.StatusInternalServerError, "Internal server error")
			return
		}
		fullThumbnailPath := filepath.Join("./inventory-images", fmt.Sprintf("%06d", tagnumber), "thumbnail-"+baseFileName+".jpeg")
		thumbnailFile, err := os.Create(fullThumbnailPath)
		if err != nil {
			log.Error("Failed to create thumbnail file for inventory update: " + err.Error() + " (" + requestIP + ")")
			middleware.WriteJsonError(w, http.StatusInternalServerError, "Internal server error")
			return
		}
		defer thumbnailFile.Close()
		err = jpeg.Encode(thumbnailFile, decodedImage, &jpeg.Options{Quality: 50})
		if err != nil {
			log.Error("Failed to encode thumbnail image for inventory update: " + err.Error() + " (" + requestIP + ")")
			middleware.WriteJsonError(w, http.StatusInternalServerError, "Internal server error")
			return
		}
		thumbnailFile.Close()

		// Insert image metadata into database
		fileSizeMB := float64(fileSize) / (2 << 20)
		err = updateRepo.UpdateClientImages(ctx, tagnumber, fileUUID.String(), &fileName, fullFilePath, &fullThumbnailPath, &fileSizeMB, &fileHashBytes, &mimeType, nil, &resolutionX, &resolutionY, nil, nil, nil)
		if err != nil {
			log.Error("Failed to update inventory image data: " + err.Error() + " (" + requestIP + ")")
			middleware.WriteJsonError(w, http.StatusInternalServerError, "Internal server error")
			return
		}
		log.Info(fmt.Sprintf("Uploaded file details - Name: %s, Size: %.2f MB, MIME Type: %s", fileName, fileSizeMB, mimeType) + " (" + requestIP + ")")
		file.Close()
	}
	// Update db

	// No pointers here, pointers in repo
	// tagnumber and working are converted above
	err = updateRepo.InsertInventory(ctx, tagnumber, inventoryUpdate.SystemSerial, inventoryUpdate.Location, inventoryUpdate.Department, inventoryUpdate.Domain, workingBool, inventoryUpdate.Status, inventoryUpdate.Note)
	if err != nil {
		log.Error("Failed to update inventory data: " + err.Error() + " (" + requestIP + ")")
		middleware.WriteJsonError(w, http.StatusInternalServerError, "Internal server error")
		return
	}
	err = updateRepo.UpdateSystemData(ctx, tagnumber, inventoryUpdate.SystemManufacturer, inventoryUpdate.SystemModel)
	if err != nil {
		log.Error("Failed to update system data: " + err.Error() + " (" + requestIP + ")")
		middleware.WriteJsonError(w, http.StatusInternalServerError, "Internal server error")
		return
	}

	retMap := database.InventoryUpdateFormInput{}
	retMap.Tagnumber = int(tagnumber)
	retMap.SystemSerial = inventoryUpdate.SystemSerial
	retMap.Location = inventoryUpdate.Location
	retMap.SystemManufacturer = inventoryUpdate.SystemManufacturer
	retMap.SystemModel = inventoryUpdate.SystemModel
	retMap.Department = inventoryUpdate.Department
	retMap.Domain = inventoryUpdate.Domain
	retMap.Working = &workingBool
	retMap.Status = inventoryUpdate.Status
	retMap.Note = inventoryUpdate.Note
	retMapJson, err := json.Marshal(retMap)
	if err != nil {
		log.Error("Failed to marshal inventory update response JSON: " + err.Error() + " (" + requestIP + ")")
		middleware.WriteJsonError(w, http.StatusInternalServerError, "Internal server error")
		return
	}
	middleware.WriteJson(w, http.StatusOK, retMapJson)
}
