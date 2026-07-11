package endpoints

import (
	"bytes"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"uit-toolbox/config"
	"uit-toolbox/database"
	"uit-toolbox/middleware"
	"uit-toolbox/types"

	"github.com/google/uuid"
)

func decodeMaybeBase64URLJSON(raw string) ([]byte, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return nil, fmt.Errorf("empty query value")
	}

	// Backward compatible: allow plain JSON in URL parameters.
	if strings.HasPrefix(trimmed, "{") || strings.HasPrefix(trimmed, "[") {
		return []byte(trimmed), nil
	}

	decoded, err := base64.RawURLEncoding.DecodeString(trimmed)
	if err != nil {
		return nil, fmt.Errorf("invalid base64url json: %w", err)
	}
	return decoded, nil
}

// Per-client functions
func GetServerTime(w http.ResponseWriter, req *http.Request) {
	format := middleware.GetStrQuery(req.URL.Query(), "format")
	curTime := time.Now().Format(time.RFC3339)
	if format != nil && *format == "unix" {
		curTime = time.Now().Format(time.UnixDate)
	}
	middleware.WriteJson(w, http.StatusOK, ServerTime{Time: curTime})
}

func GetClientIDs(w http.ResponseWriter, req *http.Request) {
	log := middleware.GetLoggerFromContext(req.Context()).With(slog.String("func", "GetClientIDs"))

	// Need either tag or serial, tag is preferred if both are provided
	var tagnumber, tagErr = types.ConvertAndVerifyTagnumber(req.URL.Query().Get("tagnumber"))
	var systemSerial = middleware.GetStrQuery(req.URL.Query(), "system_serial")
	serialErr := types.IsSystemSerialValid(systemSerial)

	if tagErr != nil && serialErr != nil {
		log.Warn("No tagnumber or system_serial provided")
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}

	var clientLookup = new(types.ClientLookupRow)
	var clientLookupErr error
	clientLookup, clientLookupErr = database.ClientIDLookup(req.Context(), tagnumber, systemSerial)
	if clientLookupErr != nil {
		log.Warn("error during client lookup: " + clientLookupErr.Error())
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}
	middleware.WriteJson(w, http.StatusOK, clientLookup)
}

func GetAllClientIDs(w http.ResponseWriter, req *http.Request) {
	log := middleware.GetLoggerFromContext(req.Context()).With(slog.String("func", "GetAllClientIDs"))

	clientIDs, err := database.SelectAllIDs(req.Context())
	if err != nil {
		log.Warn("error during client IDs lookup: " + err.Error())
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}
	middleware.WriteJson(w, http.StatusOK, clientIDs)
}

// func GetBiosData(w http.ResponseWriter, req *http.Request) {
// 	ctx := req.Context()
// 	log := middleware.GetLoggerFromContext(ctx)
// 	tagnumber, err := types.ConvertAndVerifyTagnumber(req.URL.Query().Get("tagnumber"))
// 	if err != nil || tagnumber == nil {
// 		log.Warn("Invalid tagnumber provided in GetBiosData: " + err.Error())
// 		middleware.WriteJsonError(w, http.StatusBadRequest)
// 		return
// 	}

// 	db, err := database.NewSelectRepo()
// 	if err != nil {
// 		log.Warn("Error creating select repository in GetBiosData: " + err.Error())
// 		middleware.WriteJsonError(w, http.StatusInternalServerError)
// 		return
// 	}

// 	biosData, err := db.GetBiosData(ctx, tagnumber)
// 	if err != nil {
// 		log.Warn("Query error in GetBiosData: " + err.Error())
// 		middleware.WriteJsonError(w, http.StatusInternalServerError)
// 		return
// 	}
// 	middleware.WriteJson(w, http.StatusOK, biosData)
// }

func IsClientJobAvailable(w http.ResponseWriter, req *http.Request) {
	log := middleware.GetLoggerFromContext(req.Context()).With(slog.String("func", "IsClientJobAvailable"))
	tagnumber, err := types.ConvertAndVerifyTagnumber(req.URL.Query().Get("tagnumber"))
	if err != nil {
		log.Warn("Invalid tagnumber provided in IsClientJobAvailable: " + err.Error())
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}

	availableJobs, err := database.SelectIsClientJobAvailable(req.Context(), tagnumber)
	if err != nil {
		log.Warn("Query error in IsClientJobAvailable: " + err.Error())
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}
	middleware.WriteJson(w, http.StatusOK, availableJobs)
}

func GetNotes(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	log := middleware.GetLoggerFromContext(ctx).With(slog.String("func", "GetNotes"))
	noteType := middleware.GetStrQuery(req.URL.Query(), "note_type")
	if noteType == nil || strings.TrimSpace(*noteType) == "" {
		log.Info("No note_type provided, defaulting to 'general'")
		defaultNoteType := "general"
		noteType = &defaultNoteType
	}

	notesData, err := database.GetNotes(ctx, noteType)
	if err != nil {
		log.Warn(fmt.Sprintf("%v: %v", types.DatabaseQueryError, err))
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}
	middleware.WriteJson(w, http.StatusOK, notesData)
}

func GetLocationFormData(w http.ResponseWriter, req *http.Request) {
	log := middleware.GetLoggerFromContext(req.Context()).With(slog.String("func", "GetLocationFormData"))

	serial := middleware.GetStrQuery(req.URL.Query(), "system_serial")
	tagnumber := middleware.GetInt64Query(req.URL.Query(), "tagnumber")
	tagErr := types.IsTagnumberInt64Valid(tagnumber)
	if tagErr != nil && (serial == nil || strings.TrimSpace(*serial) == "") {
		log.Warn("Missing/invalid tagnumber and system_serial provided")
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}

	locationData, err := database.GetLocationFormData(req.Context(), tagnumber, serial)
	if err != nil {
		log.Warn("Query error: " + err.Error())
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}
	middleware.WriteJson(w, http.StatusOK, locationData)
}

func GetClientImagesManifest(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	log := middleware.GetLoggerFromContext(ctx).With(slog.String("func", "GetClientImagesManifest"))
	requestQueries := req.URL.Query()
	tagnumber, err := types.ConvertAndVerifyTagnumber(requestQueries.Get("tagnumber"))
	if err != nil {
		log.Warn("Invalid tagnumber provided in request to GetClientImagesManifest: " + err.Error())
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}

	appState, err := config.GetAppState()
	if err != nil {
		log.Warn("Error getting app state in GetClientImagesManifest: " + err.Error())
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}
	fileConstraints, err := appState.GetFileUploadConstraints()
	if err != nil {
		log.Warn("Error creating select repository in GetClientImagesManifest: " + err.Error())
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}

	imageManifests, err := database.GetClientImageManifestByTag(ctx, tagnumber)
	if err != nil && err != sql.ErrNoRows {
		log.Warn("Query error in GetClientImagesManifest: " + err.Error())
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}
	if len(imageManifests) == 0 {
		log.Warn("No image manifest data for client: " + fmt.Sprintf("%d", *tagnumber))
		middleware.WriteJsonError(w, http.StatusNotFound)
		return
	}

	var filteredImageManifests []types.ImageManifestResponse
	for _, imageManifest := range imageManifests {
		var responseManifest = new(types.ImageManifestResponse)
		if imageManifest.Time.IsZero() {
			log.Warn("Image manifest has zero time for file with tagnumber: " + fmt.Sprintf("%d", *tagnumber))
			continue
		}
		manifestTime := imageManifest.Time.UTC()
		responseManifest.Time = &manifestTime

		// UUID of file
		if imageManifest.FileUUID == nil || strings.TrimSpace(*imageManifest.FileUUID) == "" {
			log.Warn("Image manifest FileUUID is nil or empty in GetClientImagesManifest")
			continue
		}
		fileUUID := strings.TrimSpace(*imageManifest.FileUUID)
		responseManifest.FileUUID = &fileUUID

		// Check if marked as hidden
		if imageManifest.Hidden != nil && *imageManifest.Hidden {
			// log.Debug("Hidden file not sent in response to client: " + fileUUID)
			continue
		}

		// File Name
		if imageManifest.FileName == nil || strings.TrimSpace(*imageManifest.FileName) == "" {
			log.Warn("Image manifest FileName is nil or empty in GetClientImagesManifest")
			continue
		}
		fileName := strings.TrimSpace(*imageManifest.FileName)

		// Client UUID
		if imageManifest.ClientUUID == nil || strings.TrimSpace(*imageManifest.ClientUUID) == "" {
			log.Warn("Image manifest ClientUUID is nil or empty in GetClientImagesManifest")
			continue
		}
		clientUUID := strings.TrimSpace(*imageManifest.ClientUUID)
		responseManifest.ClientUUID = &clientUUID

		// SHA-256 hash
		// if imageManifest.SHA256Hash == nil || len(*imageManifest.SHA256Hash) == 0 {
		// 	log.Warn("Image manifest SHA256Hash is nil or empty in GetClientImagesManifest for file: " + fileUUID)
		// 	continue
		// }
		// responseManifest.SHA256Hash = imageManifest.SHA256Hash

		// File path
		filePath := filepath.Join("/opt/inventory_images", clientUUID, fileName)
		filePath = filepath.Clean(filePath)

		// URL to send to client
		urlStr := url.URL{
			Scheme: "https",
			Host:   req.Host,
			Path:   filepath.Join("/api/client/files"),
		}
		q := urlStr.Query()
		q.Set("file_uuid", fileUUID)
		q.Set("client_uuid", clientUUID)
		urlStr.RawQuery = q.Encode()

		if err != nil {
			log.Warn("Error joining URL paths in GetClientImagesManifest: " + err.Error())
			continue
		}
		responseManifest.URL = &urlStr.RawQuery

		// Check if pinned
		if imageManifest.Pinned != nil && *imageManifest.Pinned {
			responseManifest.Pinned = imageManifest.Pinned
		}

		// Copy caption
		if imageManifest.Caption != nil && strings.TrimSpace(*imageManifest.Caption) != "" {
			caption := strings.TrimSpace(*imageManifest.Caption)
			responseManifest.Caption = &caption
		}

		// Check MIME type from DB
		if imageManifest.MimeType == nil {
			log.Warn("File '" + fileUUID + "' has a nil MIME type in DB")
			continue
		}
		mimeType := strings.TrimSpace(*imageManifest.MimeType)

		// File extension
		fileExtension := strings.ToLower(filepath.Ext(filePath))

		// Open file and read metadata
		file, err := os.Open(filePath)
		if err != nil {
			log.Warn("Cannot open file '" + fileUUID + "': " + err.Error())
			continue
		}
		isValidFile := func() bool {
			defer func() {
				_ = file.Close()
			}()

			imageStat, err := file.Stat()
			if err != nil {
				log.Warn("Cannot stat file '" + fileUUID + "': " + err.Error())
				return false
			}

			// Get file size
			metadataFileSize := imageStat.Size()
			responseManifest.FileSize = &metadataFileSize

			// If an image
			if _, ok := fileConstraints.ImageConstraints.AcceptedImageExtensionsAndMimeTypes[fileExtension]; ok {
				// Check file size from metadata
				if metadataFileSize < fileConstraints.ImageConstraints.MinFileSize || metadataFileSize > fileConstraints.ImageConstraints.MaxFileSize {
					log.Warn("Image file size is out of bounds for file '" + fileUUID + "' in GetClientImagesManifest: File size: " + fmt.Sprintf("%d", metadataFileSize))
					return false
				}

				imageBytes, err := io.ReadAll(io.LimitReader(file, fileConstraints.ImageConstraints.MaxFileSize+1))
				if err != nil {
					log.Warn("Error reading image file in GetClientImagesManifest: " + filePath + " " + err.Error())
					return false
				}
				if len(imageBytes) > int(fileConstraints.ImageConstraints.MaxFileSize) {
					log.Warn("Image file size exceeds maximum after reading in GetClientImagesManifest: " + filePath + " -> File size: " + fmt.Sprintf("%d", len(imageBytes)))
					return false
				}

				// File size
				responseManifest.FileSize = &metadataFileSize

				// Get image metadata
				imageConfig, imageType, err := image.DecodeConfig(bytes.NewReader(imageBytes))
				if err != nil {
					log.Warn("Cannot decode image in GetClientImagesManifest: " + filePath + " " + err.Error())
					return false
				}

				// Check if MIME type matches file content
				mt := http.DetectContentType(imageBytes)
				if mt != mimeType {
					log.Warn("MIME in type in DB does not match file content for file '" + fileUUID + "': Detected MIME type: " + mt + ", MIME type in DB: " + mimeType)
					return false
				}
				// Check http's library MIME type against image library's detected type
				if "image/"+imageType != fileConstraints.ImageConstraints.AcceptedImageExtensionsAndMimeTypes[fileExtension] {
					log.Warn("Image '" + fileUUID + "' has invalid file type in GetClientImagesManifest: Image type: " + "image/" + imageType + ", Accepted matched type: " + fileConstraints.ImageConstraints.AcceptedImageExtensionsAndMimeTypes[fileExtension] + ", File extension: " + fileExtension)
					return false
				}
				responseManifest.MimeType = &mimeType

				// If image has zero width or height, continue
				if imageConfig.Width == 0 || imageConfig.Height == 0 {
					log.Warn("Image '" + fileUUID + "' has invalid dimensions in GetClientImagesManifest: " + filePath)
					return false
				}
				resX := int64(imageConfig.Width)
				resY := int64(imageConfig.Height)
				responseManifest.ResolutionX = &resX
				responseManifest.ResolutionY = &resY
				return true
			}

			if _, ok := fileConstraints.VideoConstraints.AcceptedVideoExtensionsAndMimeTypes[fileExtension]; ok { // If a video file
				// Check file size from metadata
				if metadataFileSize < fileConstraints.VideoConstraints.MinFileSize || metadataFileSize > fileConstraints.VideoConstraints.MaxFileSize {
					log.Warn("Video file size is out of bounds in GetClientImagesManifest: " + filePath + " -> File size: " + fmt.Sprintf("%d", metadataFileSize))
					return false
				}

				videoBytes, err := io.ReadAll(io.LimitReader(file, fileConstraints.VideoConstraints.MaxFileSize+1))
				if err != nil {
					log.Warn("Error reading video file in GetClientImagesManifest: " + filePath + " " + err.Error())
					return false
				}
				if len(videoBytes) > int(fileConstraints.VideoConstraints.MaxFileSize) {
					log.Warn("Video file size exceeds maximum after reading in GetClientImagesManifest: " + filePath + " -> File size: " + fmt.Sprintf("%d", len(videoBytes)))
					return false
				}

				// Check if MIME type matches file content
				mt := http.DetectContentType(videoBytes)
				if mt != mimeType {
					log.Warn("MIME in type in DB does not match file content for file '" + fileUUID + "': Detected MIME type: " + mt + ", MIME type in DB: " + mimeType)
					return false
				}
				// Check MIME type matches expected MIME type for video extension
				if fileConstraints.VideoConstraints.AcceptedVideoExtensionsAndMimeTypes[fileExtension] != mimeType {
					log.Warn("Video manifest has mismatched MIME type in GetClientImagesManifest: " + filePath + " -> Detected MIME type: " + mimeType + ", Expected MIME type: " + fileConstraints.VideoConstraints.AcceptedVideoExtensionsAndMimeTypes[fileExtension])
					return false
				}
				responseManifest.MimeType = &mimeType
				return true
			}

			log.Warn("File has unsupported extension in GetClientImagesManifest: " + filePath)
			return false
		}()

		if !isValidFile {
			continue
		}

		filteredImageManifests = append(filteredImageManifests, *responseManifest)
	}

	// If all manifests were filtered out
	if len(filteredImageManifests) == 0 {
		log.Warn("No valid image manifests to return in GetClientImagesManifest for client: " + fmt.Sprintf("%d", *tagnumber))
		middleware.WriteJsonError(w, http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	middleware.WriteJson(w, http.StatusOK, filteredImageManifests)
}

func GetImage(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	log := middleware.GetLoggerFromContext(ctx).With(slog.String("func", "GetImage"))
	appState, err := config.GetAppState()
	if err != nil {
		log.Warn("Error getting app state: " + err.Error())
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}
	fileConstraints, err := appState.GetFileUploadConstraints()
	if err != nil {
		log.Error("Cannot retrieve FileConstraints: " + err.Error())
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}

	// local filepath example: inventory-images/{tag}/{date --iso}-{uuid}.{file extension}
	// incoming request url: /api/client/files/{tag}/{uuid}.{file extension}
	fileUUID := middleware.GetStrQuery(req.URL.Query(), "file_uuid")
	if fileUUID == nil || strings.TrimSpace(*fileUUID) == "" {
		log.Warn("No image UUID provided in request to GetImage")
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}
	imageUUID := strings.TrimSpace(strings.ToLower(*fileUUID))
	for ext := range fileConstraints.ImageConstraints.AcceptedImageExtensionsAndMimeTypes {
		if filepath.Ext(imageUUID) == ext {
			imageUUID = strings.TrimSuffix(imageUUID, ext)
		}
	}
	for ext := range fileConstraints.VideoConstraints.AcceptedVideoExtensionsAndMimeTypes {
		if filepath.Ext(imageUUID) == ext {
			imageUUID = strings.TrimSuffix(imageUUID, ext)
		}
	}
	if imageUUID == "" {
		log.Warn("No uuid provided in request")
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}

	log.Debug("Serving image request for: " + imageUUID)
	imageManifest, err := database.GetClientImageManifestByFileUUID(ctx, imageUUID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			log.Info("Image not found from UUID lookup: " + imageUUID + " " + err.Error())
			middleware.WriteJsonError(w, http.StatusNotFound)
			return
		}
		log.Info("Client image query error: " + imageUUID + " " + err.Error())
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}

	if imageManifest == nil {
		log.Info("No image manifest data found for UUID: " + imageUUID)
		middleware.WriteJsonError(w, http.StatusNotFound)
		return
	}

	if imageManifest.Hidden != nil && *imageManifest.Hidden {
		log.Warn("Attempt to access hidden image: " + imageUUID)
		middleware.WriteJsonError(w, http.StatusNotFound)
		return
	}

	if imageManifest.ClientUUID == nil || strings.TrimSpace(*imageManifest.ClientUUID) == "" {
		log.Warn("Client UUID for image is nil or empty: " + imageUUID)
		middleware.WriteJsonError(w, http.StatusNotFound)
		return
	}
	if imageManifest.FileName == nil || strings.TrimSpace(*imageManifest.FileName) == "" {
		log.Warn("File name for image is nil or empty: " + imageUUID)
		middleware.WriteJsonError(w, http.StatusNotFound)
		return
	}
	filePath := path.Join("/opt/inventory_images", *imageManifest.ClientUUID, *imageManifest.FileName)

	imageFile, err := os.Open(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			log.Warn("Image not found on disk: " + imageUUID + " " + err.Error())
			middleware.WriteJsonError(w, http.StatusNotFound)
			return
		}
		log.Warn("Image cannot be opened: " + imageUUID + " " + err.Error())
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}
	defer imageFile.Close()

	http.ServeContent(w, req, imageFile.Name(), time.Time{}, imageFile)
}

// Overview section
func GetJobQueueTable(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	log := middleware.GetLoggerFromContext(ctx).With(slog.String("func", "GetJobQueueTable"))

	jobQueueTable, err := database.GetJobQueueTable(ctx)
	if err != nil {
		log.Warn("Query error (GetJobQueueTable): " + err.Error())
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}

	middleware.WriteJson(w, http.StatusOK, jobQueueTable)
}

func GetInventoryTableData(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	log := middleware.GetLoggerFromContext(ctx).With(slog.String("func", "GetInventoryTableData"))
	requestQueries := req.URL.Query()

	filterOptions := new(types.InventoryAdvSearchOptions)

	// Location filter
	if ok := req.URL.Query().Has("filter_location") && req.URL.Query().Get("filter_location") != ""; ok {
		locationFilter := new(types.AdvSearchOptionString)
		jsonValue, err := decodeMaybeBase64URLJSON(req.URL.Query().Get("filter_location"))
		if err != nil {
			log.Warn("Error decoding filter_location: " + err.Error())
			middleware.WriteJsonError(w, http.StatusBadRequest)
			return
		}
		err = json.Unmarshal(jsonValue, locationFilter)
		if err != nil {
			log.Warn("Error parsing filter_location: " + err.Error())
			middleware.WriteJsonError(w, http.StatusBadRequest)
			return
		}
		filterOptions.Location = locationFilter
	}

	// Building/room filter
	if ok := req.URL.Query().Has("filter_building_room") && req.URL.Query().Get("filter_building_room") != ""; ok {
		buildingRoomFilter := new(types.AdvSearchOptionString)
		jsonValue, err := decodeMaybeBase64URLJSON(req.URL.Query().Get("filter_building_room"))
		if err != nil {
			log.Warn("Error decoding filter_building_room: " + err.Error())
			middleware.WriteJsonError(w, http.StatusBadRequest)
			return
		}
		err = json.Unmarshal(jsonValue, buildingRoomFilter)
		if err != nil {
			log.Warn("Error parsing filter_building_room: " + err.Error())
			middleware.WriteJsonError(w, http.StatusBadRequest)
			return
		}
		if buildingRoomFilter.ParamValue != nil {
			trimmed := strings.TrimSpace(*buildingRoomFilter.ParamValue)
			if trimmed == "" {
				log.Warn("filter_building_room parameter is empty after trimming whitespace")
			}
			buildingRoomArr := strings.Split(*buildingRoomFilter.ParamValue, "#")
			*buildingRoomFilter.ParamValue = trimmed
			filterOptions.Building = &buildingRoomArr[0]
			if len(buildingRoomArr) > 1 {
				filterOptions.Room = &buildingRoomArr[1]
			}
		}
		filterOptions.BuildingAndRoom = buildingRoomFilter
	}

	// System manufacturer filter
	if ok := req.URL.Query().Has("filter_system_manufacturer") && req.URL.Query().Get("filter_system_manufacturer") != ""; ok {
		manufacturerFilter := new(types.AdvSearchOptionString)
		jsonValue, err := decodeMaybeBase64URLJSON(req.URL.Query().Get("filter_system_manufacturer"))
		if err != nil {
			log.Warn("Error decoding filter_system_manufacturer: " + err.Error())
			middleware.WriteJsonError(w, http.StatusBadRequest)
			return
		}
		err = json.Unmarshal(jsonValue, manufacturerFilter)
		if err != nil {
			log.Warn("Error parsing filter_system_manufacturer: " + err.Error())
			middleware.WriteJsonError(w, http.StatusBadRequest)
			return
		}
		filterOptions.SystemManufacturer = manufacturerFilter
	}

	// System model filter
	if ok := req.URL.Query().Has("filter_system_model") && req.URL.Query().Get("filter_system_model") != ""; ok {
		modelFilter := new(types.AdvSearchOptionString)
		jsonValue, err := decodeMaybeBase64URLJSON(req.URL.Query().Get("filter_system_model"))
		if err != nil {
			log.Warn("Error decoding filter_system_model: " + err.Error())
			middleware.WriteJsonError(w, http.StatusBadRequest)
			return
		}
		err = json.Unmarshal(jsonValue, modelFilter)
		if err != nil {
			log.Warn("Error parsing filter_system_model: " + err.Error())
			middleware.WriteJsonError(w, http.StatusBadRequest)
			return
		}
		filterOptions.SystemModel = modelFilter
	}

	// Device type filter
	if ok := req.URL.Query().Has("filter_device_type") && req.URL.Query().Get("filter_device_type") != ""; ok {
		deviceTypeFilter := new(types.AdvSearchOptionString)
		jsonValue, err := decodeMaybeBase64URLJSON(req.URL.Query().Get("filter_device_type"))
		if err != nil {
			log.Warn("Error decoding filter_device_type: " + err.Error())
			middleware.WriteJsonError(w, http.StatusBadRequest)
			return
		}
		err = json.Unmarshal(jsonValue, deviceTypeFilter)
		if err != nil {
			log.Warn("Error parsing filter_device_type: " + err.Error())
			middleware.WriteJsonError(w, http.StatusBadRequest)
			return
		}
		filterOptions.DeviceType = deviceTypeFilter
	}

	// Department filter
	if ok := req.URL.Query().Has("filter_department_name") && req.URL.Query().Get("filter_department_name") != ""; ok {
		departmentFilter := new(types.AdvSearchOptionString)
		jsonValue, err := decodeMaybeBase64URLJSON(req.URL.Query().Get("filter_department_name"))
		if err != nil {
			log.Warn("Error decoding filter_department_name: " + err.Error())
			middleware.WriteJsonError(w, http.StatusBadRequest)
			return
		}
		err = json.Unmarshal(jsonValue, departmentFilter)
		if err != nil {
			log.Warn("Error parsing filter_department_name: " + err.Error())
			middleware.WriteJsonError(w, http.StatusBadRequest)
			return
		}
		filterOptions.Department = departmentFilter
	}

	// AD domain/OU filter
	if ok := req.URL.Query().Has("filter_ad_domain") && req.URL.Query().Get("filter_ad_domain") != ""; ok {
		adDomainFilter := new(types.AdvSearchOptionString)
		jsonValue, err := decodeMaybeBase64URLJSON(req.URL.Query().Get("filter_ad_domain"))
		if err != nil {
			log.Warn("Error decoding filter_ad_domain: " + err.Error())
			middleware.WriteJsonError(w, http.StatusBadRequest)
			return
		}
		err = json.Unmarshal(jsonValue, adDomainFilter)
		if err != nil {
			log.Warn("Error parsing filter_ad_domain: " + err.Error())
			middleware.WriteJsonError(w, http.StatusBadRequest)
			return
		}
		filterOptions.ADDomain = adDomainFilter
	}

	// Client status filter
	if ok := req.URL.Query().Has("filter_status") && req.URL.Query().Get("filter_status") != ""; ok {
		statusFilter := new(types.AdvSearchOptionString)
		jsonValue, err := decodeMaybeBase64URLJSON(req.URL.Query().Get("filter_status"))
		if err != nil {
			log.Warn("Error decoding filter_status: " + err.Error())
			middleware.WriteJsonError(w, http.StatusBadRequest)
			return
		}
		err = json.Unmarshal(jsonValue, statusFilter)
		if err != nil {
			log.Warn("Error parsing filter_status: " + err.Error())
			middleware.WriteJsonError(w, http.StatusBadRequest)
			return
		}
		filterOptions.Status = statusFilter
	}

	// Is broken filter
	if ok := req.URL.Query().Has("filter_is_broken") && req.URL.Query().Get("filter_is_broken") != ""; ok {
		isBrokenFilter := new(types.AdvSearchOptionBool)
		jsonValue, err := decodeMaybeBase64URLJSON(req.URL.Query().Get("filter_is_broken"))
		if err != nil {
			log.Warn("Error decoding filter_is_broken: " + err.Error())
			middleware.WriteJsonError(w, http.StatusBadRequest)
			return
		}
		err = json.Unmarshal(jsonValue, isBrokenFilter)
		if err != nil {
			log.Warn("Error parsing filter_is_broken: " + err.Error())
			middleware.WriteJsonError(w, http.StatusBadRequest)
			return
		}
		filterOptions.IsBroken = isBrokenFilter
	}

	// Has images filter
	if ok := req.URL.Query().Has("filter_has_images") && req.URL.Query().Get("filter_has_images") != ""; ok {
		hasImagesFilter := new(types.AdvSearchOptionBool)
		jsonValue, err := decodeMaybeBase64URLJSON(req.URL.Query().Get("filter_has_images"))
		if err != nil {
			log.Warn("Error decoding filter_has_images: " + err.Error())
			middleware.WriteJsonError(w, http.StatusBadRequest)
			return
		}
		err = json.Unmarshal(jsonValue, hasImagesFilter)
		if err != nil {
			log.Warn("Error parsing filter_has_images: " + err.Error())
			middleware.WriteJsonError(w, http.StatusBadRequest)
			return
		}
		filterOptions.HasImages = hasImagesFilter
	}

	// log.Debug(fmt.Sprintf("Filter options: %+v", filterOptions))

	inventoryTableData, err := database.GetInventoryTableData(ctx, filterOptions)
	if err != nil {
		log.Warn("Query error (GetInventoryTableData): " + err.Error())
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}
	if requestQueries.Get("csv") == "true" {
		log.Debug("CSV file requested in GetInventoryTableData")
		tagArr := make([]int64, 0, len(inventoryTableData))
		for _, row := range inventoryTableData {
			if row.Tagnumber != nil {
				tagArr = append(tagArr, *row.Tagnumber)
			}
		}
		csvBytes, err := database.ConvertClientInfoToCSV(ctx, tagArr)
		if err != nil {
			log.Warn("Error converting inventory table data to CSV in GetInventoryTableData: " + err.Error())
			middleware.WriteJsonError(w, http.StatusInternalServerError)
			return
		}
		if csvBytes == nil || csvBytes.Len() == 0 {
			log.Warn("No CSV data generated in GetInventoryTableData")
			middleware.WriteJsonError(w, http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "text/csv; charset=utf-8")
		w.Header().Set("Content-Disposition", "attachment; filename=\"inventory_table_data-"+time.Now().Format("01-02-2006-150405-MST")+".csv\"")
		if _, err = w.Write(csvBytes.Bytes()); err != nil {
			log.Warn("Error writing CSV data to response: " + err.Error())
		}
		return
	} else {
		middleware.WriteJson(w, http.StatusOK, inventoryTableData)
	}
}

func GetClientConfig(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	log := middleware.GetLoggerFromContext(ctx)

	clientConfig, err := config.GetClientConfig()
	if err != nil {
		log.Warn("Error getting client config in GetClientConfig: " + err.Error())
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}

	if req.URL.Query().Get("json") == "true" {
		middleware.WriteJson(w, http.StatusOK, clientConfig)
		return
	}

	if err := json.NewEncoder(w).Encode(clientConfig); err != nil {
		log.Warn("Error encoding client config to JSON in GetClientConfig: " + err.Error())
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}
	middleware.WriteJson(w, http.StatusOK, clientConfig)
}

func GetManufacturersAndModels(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	log := middleware.GetLoggerFromContext(ctx).With(slog.String("func", "GetManufacturersAndModels"))

	manufacturersAndModels, err := database.SelectAllManufacturersAndModels(ctx)
	if err != nil {
		log.Warn("Query error in SelectAllManufacturersAndModels: " + err.Error())
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}
	middleware.WriteJson(w, http.StatusOK, manufacturersAndModels)
}

func GetAllDomains(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	log := middleware.GetLoggerFromContext(ctx).With(slog.String("func", "GetAllDomains"))

	domains, err := database.GetAllDomains(ctx)
	if err != nil {
		log.Warn("Query error: " + err.Error())
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}
	middleware.WriteJson(w, http.StatusOK, domains)
}

func GetDepartments(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	log := middleware.GetLoggerFromContext(ctx).With(slog.String("func", "GetDepartments"))

	departments, err := database.GetAllDepartments(ctx)
	if err != nil {
		log.Warn("Query error: " + err.Error())
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}
	middleware.WriteJson(w, http.StatusOK, departments)
}

func GetAllJobs(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	log := middleware.GetLoggerFromContext(ctx).With(slog.String("func", "GetAllJobs"))

	allJobs, err := database.SelectAllJobs(ctx)
	if err != nil {
		log.Warn("Query error in SelectAllJobs: " + err.Error())
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}
	middleware.WriteJson(w, http.StatusOK, allJobs)
}

func GetAllLocations(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	log := middleware.GetLoggerFromContext(ctx).With(slog.String("func", "GetAllLocations"))

	allLocations, err := database.GetAllLocations(ctx)
	if err != nil {
		log.Warn(fmt.Sprintf("%v: %v", types.DatabaseQueryError, err))
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}
	middleware.WriteJson(w, http.StatusOK, allLocations)
}

func GetAllStatuses(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	log := middleware.GetLoggerFromContext(ctx).With(slog.String("func", "GetAllStatuses"))

	allStatuses, err := database.GetAllStatuses(ctx)
	if err != nil {
		log.Warn("Query error: " + err.Error())
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}
	middleware.WriteJson(w, http.StatusOK, allStatuses)
}

func GetAllDeviceTypes(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	log := middleware.GetLoggerFromContext(ctx).With(slog.String("func", "GetAllDeviceTypes"))

	allDeviceTypes, err := database.GetAllDeviceTypes(ctx)
	if err != nil {
		log.Warn(fmt.Sprintf("%v: %v", types.DatabaseQueryError, err))
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}
	middleware.WriteJson(w, http.StatusOK, allDeviceTypes)
}

func FetchClientHardwareData(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	log := middleware.GetLoggerFromContext(ctx).With(slog.String("func", "FetchClientHardwareData"))
	tagnumber, err := types.ConvertAndVerifyTagnumber(req.URL.Query().Get("tagnumber"))
	if err != nil {
		log.Warn(fmt.Sprintf("%v: %v", types.InvalidRequestFieldError, err))
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}

	clientOverview, err := database.GetClientHardwareOverview(ctx, *tagnumber)
	if err != nil {
		log.Warn("Query error: " + err.Error())
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}
	middleware.WriteJson(w, http.StatusOK, clientOverview)
}

func FetchClientJobQueuePosition(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	log := middleware.GetLoggerFromContext(ctx).With(slog.String("func", "FetchClientJobQueuePosition"))

	tagnumber, err := types.ConvertAndVerifyTagnumber(req.URL.Query().Get("tagnumber"))
	if err != nil {
		log.Warn("Invalid tagnumber provided: " + err.Error())
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}

	queuePosition, err := database.SelectJobQueuePosition(ctx, *tagnumber)
	if err != nil {
		log.Warn("DB error: " + err.Error())
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}

	returnedJson := struct {
		Position *int64 `json:"job_queue_position"`
	}{
		Position: &queuePosition,
	}
	middleware.WriteJson(w, http.StatusOK, returnedJson)
}

func FetchClientJobName(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	log := middleware.GetLoggerFromContext(ctx).With(slog.String("func", "FetchClientJobName"))

	tagnumber, err := types.ConvertAndVerifyTagnumber(req.URL.Query().Get("tagnumber"))
	if err != nil {
		log.Warn("Invalid tagnumber provided: " + err.Error())
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}

	jobName, err := database.GetJobName(ctx, *tagnumber)
	if err != nil {
		log.Warn("DB error: " + err.Error())
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}

	returnedJson := struct {
		JobName string `json:"job_name"`
	}{
		JobName: func() string {
			if jobName == nil {
				return "" // This is because bash will treat "nil" as a string, can't -z the value accurately
			}
			return *jobName
		}(),
	}
	middleware.WriteJson(w, http.StatusOK, returnedJson)
}

func FetchFormattedJobName(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	log := middleware.GetLoggerFromContext(ctx).With(slog.String("func", "FetchFormattedJobName"))

	jobName := strings.TrimSpace(req.URL.Query().Get("job_name"))
	if jobName == "" {
		log.Debug("Empty job_name provided")
		middleware.WriteJson(w, http.StatusOK,
			struct {
				JobNameFormatted string `json:"job_name_formatted"`
			}{
				JobNameFormatted: "",
			})
		return
	}

	jobNameFormatted, err := database.GetFormattedJobName(ctx, jobName)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			log.Warn("Job name not found")
			middleware.WriteJsonError(w, http.StatusNotFound)
			return
		} else {
			log.Warn("DB error: " + err.Error())
			middleware.WriteJsonError(w, http.StatusInternalServerError)
			return
		}
	}

	returnedJson := struct {
		JobNameFormatted string `json:"job_name_formatted"`
	}{
		JobNameFormatted: func() string {
			if jobNameFormatted == nil {
				return "" // This is because bash will treat "nil" as a string, can't -z the value accurately
			}
			return *jobNameFormatted
		}(),
	}
	middleware.WriteJson(w, http.StatusOK, returnedJson)
}

func DownloadLiveImage(w http.ResponseWriter, req *http.Request) {
	log := middleware.GetLoggerFromContext(req.Context()).With(slog.String("func", "DownloadLiveImage"))
	tag := middleware.GetInt64Query(req.URL.Query(), "tagnumber")
	if tag == nil || *tag == 0 {
		log.Info("Missing tagnumber in request")
	}
	imageBytes, err := config.GetLiveImage(*tag)
	if err != nil {
		log.Warn("Error getting live image: " + err.Error())
		if errors.Is(err, types.LiveImageMissingError) {
			middleware.WriteJsonError(w, http.StatusNotFound)
		} else {
			middleware.WriteJsonError(w, http.StatusInternalServerError)
		}
		return
	}
	if len(imageBytes) == 0 {
		log.Warn("Requested live image is empty")
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}
	if len(imageBytes) == 0 {
		log.Warn("Requested live image is too large")
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}
	reader := bytes.NewReader(imageBytes)
	var readSeeker io.ReadSeeker = reader
	w.Header().Set("Content-Type", "image/jpeg")
	http.ServeContent(w, req, strconv.Itoa(int(*tag))+".jpeg", time.Now().UTC(), readSeeker)
	// log.Info("Served live image '" + strconv.Itoa(int(*tag)) + "' (" + fmt.Sprintf("%.2f", float64(len(imageBytes))/1024/1024) + " MB)")
}

func FetchAllBuildingsAndRooms(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	log := middleware.GetLoggerFromContext(ctx).With(slog.String("func", "FetchAllBuildingsAndRooms"))

	result, err := database.GetAllBuildingsAndRooms(ctx)
	if err != nil {
		log.Warn("Error fetching buildings and rooms: " + err.Error())
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}
	middleware.WriteJson(w, http.StatusOK, result)
}

func InitClient(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	log := middleware.GetLoggerFromContext(ctx).With(slog.String("func", "InitClient"))

	body, err := io.ReadAll(req.Body)
	if err != nil {
		log.Warn("Error reading request body: " + err.Error())
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}
	var requestData types.ClientInitRequest
	if err := json.Unmarshal(body, &requestData); err != nil {
		log.Warn("Error unmarshalling request body: " + err.Error())
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}

	dto, err := requestData.ToDTO()
	if err != nil {
		log.Warn("Error converting request data to DTO: " + err.Error())
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}

	var clientUUIDStr *string
	var clientUUID uuid.UUID
	clientUUIDStr, err = database.InitClient(ctx, dto)
	if err != nil {
		log.Warn("Error initializing client: " + err.Error())
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}
	if clientUUIDStr != nil && strings.TrimSpace(*clientUUIDStr) != "" {
		log.Warn("New client inserted with serial number: " + *requestData.SystemSerial)
	} else {
		pgxPool, err := config.GetPGXPool()
		if err != nil {
			log.Warn("Error getting pgx pool: " + err.Error())
			middleware.WriteJsonError(w, http.StatusInternalServerError)
			return
		}
		clientUUID, err = database.GetClientUUIDBySerial(ctx, pgxPool, *requestData.SystemSerial)
		if err != nil {
			log.Warn("Error fetching client UUID by serial number: " + err.Error())
			middleware.WriteJsonError(w, http.StatusInternalServerError)
			return
		}
	}

	returnedJson := struct {
		ClientUUID string `json:"client_uuid"`
	}{}

	if clientUUIDStr != nil && strings.TrimSpace(*clientUUIDStr) != "" {
		returnedJson.ClientUUID = *clientUUIDStr
	}
	if clientUUID != uuid.Nil {
		returnedJson.ClientUUID = clientUUID.String()
	}
	middleware.WriteJson(w, http.StatusOK, returnedJson)
}

func FetchCheckoutData(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	log := middleware.GetLoggerFromContext(ctx).With(slog.String("func", "FetchCheckoutData"))
	tagnumber, err := types.ConvertAndVerifyTagnumber(req.URL.Query().Get("tagnumber"))
	if err != nil {
		log.Warn("Invalid tagnumber provided: " + err.Error())
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}
	checkoutData, err := database.SelectCheckoutData(ctx, tagnumber)
	if err != nil {
		log.Warn("Error fetching checkout data: " + err.Error())
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}
	middleware.WriteJson(w, http.StatusOK, checkoutData)
}

func GetNewTransactionUUID(w http.ResponseWriter, req *http.Request) {
	log := middleware.GetLoggerFromContext(req.Context()).With(slog.String("func", "GetNewTransactionUUID"))
	uuid, err := uuid.NewV7()
	if err != nil {
		log.Warn("Error generating new transaction UUID: " + err.Error())
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}
	returnedJson := struct {
		TransactionUUID string `json:"transaction_uuid"`
	}{
		TransactionUUID: uuid.String(),
	}
	middleware.WriteJson(w, http.StatusOK, returnedJson)
}

func GetClientInfo(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	log := middleware.GetLoggerFromContext(ctx).With(slog.String("func", "GetClientInfo"))
	tagnumber, err := types.ConvertAndVerifyTagnumber(req.URL.Query().Get("tagnumber"))
	if err != nil {
		log.Warn("Invalid tagnumber provided: " + err.Error())
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}
	clientInfo, err := database.SelectClientInfo(ctx, *tagnumber)
	if err != nil || clientInfo == nil {
		log.Warn("Error fetching client info: " + err.Error())
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}
	middleware.WriteJson(w, http.StatusOK, clientInfo)
}

func GetDiskImageNameByModel(w http.ResponseWriter, req *http.Request) {
	log := middleware.GetLoggerFromContext(req.Context()).With(slog.String("func", "GetDiskImageNameByModel"))
	model := middleware.GetStrQuery(req.URL.Query(), "system_model")

	if model == nil || strings.TrimSpace(*model) == "" {
		log.Warn("No system model provided in request to GetDiskImageNameByModel")
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}

	diskImageRequest := new(types.DiskImageNameRequest)
	diskImageRequest.SystemModel = model

	diskImageResponse, err := database.SelectDiskImageByModel(req.Context(), diskImageRequest)
	if err != nil {
		log.Warn("Error fetching disk image name by model: " + err.Error())
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}

	if diskImageResponse == nil {
		log.Warn("No disk image name found for model: " + *model)
		middleware.WriteJsonError(w, http.StatusNotFound)
		return
	}

	if diskImageResponse.SystemModel == nil || strings.TrimSpace(*diskImageResponse.SystemModel) == "" {
		log.Warn("Disk image system model is nil or empty for model: " + *model)
		middleware.WriteJsonError(w, http.StatusNotFound)
		return
	}

	if diskImageResponse.ImageName == nil || strings.TrimSpace(*diskImageResponse.ImageName) == "" {
		log.Warn("Disk image name is nil or empty for model: " + *model)
		middleware.WriteJsonError(w, http.StatusNotFound)
		return
	}

	returnedJson := new(types.DiskImageNameResponse)
	returnedJson.SystemModel = diskImageResponse.SystemModel
	returnedJson.ImageName = diskImageResponse.ImageName
	middleware.WriteJson(w, http.StatusOK, returnedJson)
}
