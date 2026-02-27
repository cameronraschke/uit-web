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

	appState, err := config.GetAppState()
	if err != nil {
		log.Warn("Cannot retrieve app state: " + err.Error())
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}
	htmlFormConstraints, err := appState.GetFormConstraints()
	if err != nil {
		log.Error("Cannot retrieve HTMLFormConstraints: " + err.Error())
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}
	req.Body = http.MaxBytesReader(w, req.Body, htmlFormConstraints.LoginForm.MaxFormBytes)
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
	clientFormAuthData := new(types.AuthRequest)
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
	if !types.IsPrintableUnicode(requestBody) {
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
	if memoryData.Tagnumber == 0 {
		log.Warn("Missing tag number")
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}
	if memoryData.TotalUsage == nil || memoryData.TotalCapacity == 0 {
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
	if !types.IsPrintableUnicode(requestBody) {
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
	if cpuData.Tagnumber == 0 || cpuData.UsagePercent == nil {
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
	if !types.IsPrintableUnicode(requestBody) {
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
	if cpuData.Tagnumber == 0 || cpuData.MillidegreesC == nil {
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
	if networkData.Tagnumber == 0 || networkData.NetworkUsage == nil || networkData.LinkSpeed == nil {
		log.Warn("Missing tag number, network usage, or link speed")
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
	if uptimeData.Tagnumber == 0 || uptimeData.ClientAppUptime == 0 || uptimeData.SystemUptime == 0 {
		log.Warn("Missing tag number, client app uptime, or system uptime")
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}
	updateRepo, err := database.NewUpdateRepo()
	if err != nil {
		log.Error("No database connection available for updating client uptime")
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}
	if err := updateRepo.UpdateClientUptime(ctx, &uptimeData); err != nil {
		log.Error("Failed to update client uptime: " + err.Error())
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

	lr := io.LimitReader(req.Body, int64(htmlFormConstraints.GeneralNote.NoteTypeMaxChars*4))

	// Parse and validate note data
	var newNote types.Note
	err = json.NewDecoder(lr).Decode(&newNote)
	if err != nil {
		log.Warn("Cannot decode note JSON: " + err.Error())
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}
	defer req.Body.Close()

	if newNote.NoteType == nil || newNote.Content == nil {
		log.Warn("Note type or content is nil, not inserting new note")
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}
	if utf8.RuneCountInString(strings.TrimSpace(*newNote.NoteType)) <= htmlFormConstraints.GeneralNote.NoteTypeMinChars || utf8.RuneCountInString(*newNote.NoteType) > htmlFormConstraints.GeneralNote.NoteTypeMaxChars {
		log.Warn("Note type outside of valid length range, not inserting new note")
		middleware.WriteJsonError(w, http.StatusRequestEntityTooLarge)
		return
	}
	if utf8.RuneCountInString(strings.TrimSpace(*newNote.Content)) < htmlFormConstraints.GeneralNote.NoteContentMinChars || utf8.RuneCountInString(*newNote.Content) > htmlFormConstraints.GeneralNote.NoteContentMaxChars {
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
	curTime := time.Now().UTC()
	err = insertRepo.InsertNewNote(ctx, &curTime, newNote.NoteType, newNote.Content)
	if err != nil {
		log.Error("Failed to insert new note: " + err.Error())
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}
	middleware.WriteJson(w, http.StatusOK, map[string]string{"status": "success"})
}

func InsertInventoryUpdate(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	log := middleware.GetLoggerFromContext(ctx).With(slog.String("func", "InsertInventoryUpdate"))
	endpointConfig, err := config.GetWebEndpointConfig(req.URL.Path)
	if err != nil {
		log.Warn("Cannot get endpoint config in InsertInventoryUpdate: " + err.Error())
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}
	// Parse inventory data
	appState, err := config.GetAppState()
	if err != nil {
		log.Warn("Cannot get app state in InsertInventoryUpdate: " + err.Error())
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}
	htmlFormConstraints, err := appState.GetFormConstraints()
	if err != nil {
		log.Error("Cannot retrieve HTMLFormConstraints: " + err.Error())
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}
	fileUploadConstraints, err := appState.GetFileUploadConstraints()
	if err != nil {
		log.Error("Cannot retrieve FileConstraints: " + err.Error())
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}

	maxUpload := endpointConfig.MaxUploadSize
	if maxUpload == nil {
		log.Error("Max upload size is not defined for this endpoint")
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}
	totalAllowedBytes := *maxUpload
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
		log.Warn("Cannot decode JSON (InsertInventoryUpdate): " + err.Error())
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}
	if !utf8.Valid(jsonBytes) {
		log.Warn("Invalid UTF-8 in JSON data")
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}

	inventoryDomain, err := types.CreateInventoryUpdateDTO(&inventoryUpdateReq, htmlFormConstraints)
	if err != nil {
		log.Warn("Invalid inventory request payload: " + err.Error())
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}

	// File upload part of form:
	if req.MultipartForm == nil || req.MultipartForm.File == nil {
		log.Info("File upload part of inventory update is nil, continuing")
	}
	files := req.MultipartForm.File["inventory-file-input"]

	// Generate transaction UUID to share between multiple DB tables
	transactionUUID, err := uuid.NewUUID()
	if err != nil {
		log.Error("error generation a transaction UUID (InsertInventoryUpdate)")
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}
	if transactionUUID == uuid.Nil {
		log.Error("transaction UUID in InsertInventoryUpdate is nil")
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

	var totalImageFileCount int
	var totalImageUploadSize int64
	var totalVideoFileCount int
	var totalVideoUploadSize int64
	// var totalInvalidFileCount int = 2
	// var totalInvalidUploadSize int64 = 1 << 10
	for _, fileHeader := range files {
		var manifest types.ImageManifest

		if !types.IsPrintableUnicodeString(fileHeader.Filename) {
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

		lr := &io.LimitedReader{R: file, N: fileUploadConstraints.ImageConstraints.MaxFileSize + fileUploadConstraints.VideoConstraints.MaxFileSize + 1}
		fileBytes, err := io.ReadAll(lr)
		file.Close()
		if err != nil {
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

		if fileUploadConstraints.ImageConstraints.AcceptedImageExtensionsAndMimeTypes[ext] != mimeType && fileUploadConstraints.VideoConstraints.AcceptedVideoExtensionsAndMimeTypes[ext] != mimeType {
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
		hashes, err := selectRepo.GetFileHashesFromTag(ctx, &inventoryDomain.Tagnumber)
		if err != nil {
			log.Error("Failed to get file hashes from tag '" + strconv.FormatInt(inventoryDomain.Tagnumber, 10) + "': " + err.Error())
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
			log.Warn("Duplicate file upload detected for tag '" + strconv.FormatInt(inventoryDomain.Tagnumber, 10) + "': file '" + fileHeader.Filename + "' (" + fmt.Sprintf("%x", fileHashBytes) + ") has same hash as existing file, skipping")
			continue
		}

		if fileUploadConstraints.ImageConstraints.AcceptedImageExtensionsAndMimeTypes[ext] == mimeType { // Image file processing
			if totalImageFileCount >= fileUploadConstraints.ImageConstraints.MaxFileCount {
				log.Warn("Number of uploaded image files exceeds maximum allowed: " + strconv.Itoa(totalImageFileCount))
				middleware.WriteJsonError(w, http.StatusRequestEntityTooLarge)
				return
			}
			if fileSize > fileUploadConstraints.ImageConstraints.MaxFileSize {
				log.Warn("Uploaded image file '" + fileHeader.Filename + "' too large (" + strconv.FormatInt(int64(fileSize), 10) + " bytes)")
				middleware.WriteJsonError(w, http.StatusRequestEntityTooLarge)
				return
			}
			if fileSize < fileUploadConstraints.ImageConstraints.MinFileSize {
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
			imageDirectoryPath, err := createNecessaryDirs(inventoryDomain.Tagnumber)
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
		} else if fileUploadConstraints.VideoConstraints.AcceptedVideoExtensionsAndMimeTypes[ext] == mimeType { // Video file processing
			if totalVideoFileCount >= fileUploadConstraints.VideoConstraints.MaxFileCount {
				log.Warn("Number of uploaded video files exceeds maximum allowed (InsertInventoryUpdate): " + strconv.Itoa(totalVideoFileCount))
				middleware.WriteJsonError(w, http.StatusRequestEntityTooLarge)
				return
			}
			if fileSize > fileUploadConstraints.VideoConstraints.MaxFileSize {
				log.Warn("Uploaded video file too large (InsertInventoryUpdate) (" + strconv.FormatInt(int64(fileSize), 10) + " bytes)")
				middleware.WriteJsonError(w, http.StatusRequestEntityTooLarge)
				return
			}
			if fileSize < fileUploadConstraints.VideoConstraints.MinFileSize {
				log.Warn("Uploaded video file too small (InsertInventoryUpdate): " + fileHeader.Filename + " (" + strconv.FormatInt(int64(fileSize), 10) + " bytes)")
				middleware.WriteJsonError(w, http.StatusRequestEntityTooLarge)
				return
			}
			totalVideoFileCount++
			totalVideoUploadSize += fileSize
		} else {
			log.Warn("Unsupported MIME type for '" + fileHeader.Filename + "' (InsertInventoryUpdate): MIME Type: " + mimeType)
			middleware.WriteJsonError(w, http.StatusUnsupportedMediaType)
			// totalInvalidFileCount++
			// totalInvalidUploadSize += fileSize
			return
		}

		imageDirectoryPath, err := createNecessaryDirs(inventoryDomain.Tagnumber)
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
		manifest.Tagnumber = &inventoryDomain.Tagnumber
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
	inventoryData := types.MapInventoryUpdateDomainToLocationWriteModel(transactionUUID, inventoryDomain)
	if err := updateRepo.InsertInventoryUpdate(ctx, transactionUUID, inventoryData); err != nil {
		log.Error("Failed to update inventory data: " + err.Error())
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}
	clientHardwareData := types.MapInventoryUpdateDomainToHardwareWriteModel(transactionUUID, inventoryDomain)
	if err := updateRepo.UpdateClientHardwareData(ctx, transactionUUID, clientHardwareData); err != nil {
		log.Error("Failed to update inventory hardware info: " + err.Error())
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}
	clientHealthData := types.MapInventoryUpdateDomainToClientHealthWriteModel(transactionUUID, inventoryDomain)
	if err := updateRepo.UpdateClientHealthUpdate(ctx, transactionUUID, clientHealthData); err != nil {
		log.Error("Failed to update inventory health info: " + err.Error())
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}
	checkoutData := types.MapInventoryUpdateDomainToCheckoutWriteModel(transactionUUID, inventoryDomain)
	if err := updateRepo.InsertClientCheckoutsUpdate(ctx, transactionUUID, checkoutData); err != nil {
		log.Error("Failed to update inventory checkout info: " + err.Error())
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
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

	if err := types.IsTagnumberInt64Valid(clientJson.Tagnumber); err != nil {
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

	if !types.IsASCIIStringPrintable(*clientJson.JobName) {
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

func SetClientLastHardwareCheck(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	log := middleware.GetLoggerFromContext(ctx).With(slog.String("func", "SetClientLastHardwareCheck"))

	clientBody, err := io.ReadAll(req.Body)
	if err != nil {
		log.Warn("Cannot read request body for SetClientLastHardwareCheck: " + err.Error())
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}

	var hardwareCheckData types.ClientHardwareCheck
	if err := json.Unmarshal(clientBody, &hardwareCheckData); err != nil {
		log.Warn("Cannot decode SetClientLastHardwareCheck JSON: " + err.Error())
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}
	if err := types.IsTagnumberInt64Valid(&hardwareCheckData.Tagnumber); err != nil {
		log.Warn("Invalid tagnumber: " + err.Error())
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}
	if hardwareCheckData.LastHardwareCheck == nil || hardwareCheckData.LastHardwareCheck.IsZero() {
		log.Warn("Last hardware check time is missing or zero")
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}
	updateRepo, err := database.NewUpdateRepo()
	if err != nil {
		log.Error("No database connection available for SetClientLastHardwareCheck")
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}
	if err = updateRepo.UpdateClientLastHardwareCheck(ctx, hardwareCheckData.Tagnumber, (*hardwareCheckData.LastHardwareCheck).UTC()); err != nil {
		log.Error("Failed to update client last hardware check: " + err.Error())
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}
	middleware.WriteJson(w, http.StatusOK, "Client last hardware check updated successfully")
}
