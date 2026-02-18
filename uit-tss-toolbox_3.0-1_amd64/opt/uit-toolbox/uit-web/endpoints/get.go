package endpoints

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	config "uit-toolbox/config"
	"uit-toolbox/database"
	middleware "uit-toolbox/middleware"
)

// Per-client functions
func GetServerTime(w http.ResponseWriter, req *http.Request) {
	curTime := time.Now().Format(time.RFC3339)
	middleware.WriteJson(w, http.StatusOK, ServerTime{Time: curTime})
}

func GetClientLookup(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	log := middleware.GetLoggerFromContext(ctx)
	urlQueries, err := middleware.GetRequestQueryFromContext(ctx)
	if err != nil {
		log.HTTPWarning(req, "Error retrieving request query parameters from context for GetClientLookup: "+err.Error())
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}

	// No consequence for missing tag, acceptable if lookup by serial
	var tagnumber, tagErr = ConvertAndVerifyTagnumber(urlQueries.Get("tagnumber"))
	var systemSerial = middleware.GetStrQuery(urlQueries, "system_serial")

	if tagErr != nil && systemSerial == nil {
		log.HTTPWarning(req, "No tagnumber or system_serial provided in GetClientLookup")
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}

	if tagErr != nil {
		log.HTTPWarning(req, "Cannot convert tagnumber to int64 in GetClientLookup: "+tagErr.Error())
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}

	db, err := database.NewSelectRepo()
	if err != nil {
		log.HTTPWarning(req, "Error creating select repository in GetClientLookup: "+err.Error())
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}

	var hardwareData *database.ClientLookup
	var lookupSQLErr error
	if tagnumber != nil {
		hardwareData, lookupSQLErr = db.ClientLookupByTag(ctx, tagnumber)
	} else if systemSerial != nil {
		hardwareData, lookupSQLErr = db.ClientLookupBySerial(ctx, systemSerial)
	}
	if lookupSQLErr != nil {
		log.HTTPWarning(req, "error querying client in GetClientLookup: "+lookupSQLErr.Error())
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}
	middleware.WriteJson(w, http.StatusOK, hardwareData)
}

func GetAllTags(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	log := middleware.GetLoggerFromContext(ctx)

	db, err := database.NewSelectRepo()
	if err != nil {
		log.HTTPWarning(req, "Error creating select repository in GetAllTags: "+err.Error())
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}
	allTags, err := db.AllTags(ctx)
	if err != nil {
		log.HTTPWarning(req, "Query error in GetAllTags: "+err.Error())
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}
	middleware.WriteJson(w, http.StatusOK, allTags)
}

func GetHardwareIdentifiers(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	log := middleware.GetLoggerFromContext(ctx)

	urlQueries, err := middleware.GetRequestQueryFromContext(ctx)
	if err != nil {
		log.HTTPWarning(req, "Error retrieving query parameters from context in GetHardwareIdentifiers: "+err.Error())
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}
	tagnumber, err := ConvertAndVerifyTagnumber(urlQueries.Get("tagnumber"))
	if err != nil || tagnumber == nil {
		log.HTTPWarning(req, "Invalid tagnumber provided in GetHardwareIdentifiers: "+err.Error())
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}

	db, err := database.NewSelectRepo()
	if err != nil {
		log.HTTPWarning(req, "Error creating select repository in GetHardwareIdentifiers: "+err.Error())
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}
	hardwareData, err := db.GetHardwareIdentifiers(ctx, tagnumber)
	if err != nil {
		log.HTTPWarning(req, "Query error in GetHardwareIdentifiers: "+err.Error())
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}
	middleware.WriteJson(w, http.StatusOK, hardwareData)
}

func GetBiosData(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	log := middleware.GetLoggerFromContext(ctx)
	urlQueries, err := middleware.GetRequestQueryFromContext(ctx)
	if err != nil {
		log.HTTPWarning(req, "Error retrieving query parameters from context for GetBiosData: "+err.Error())
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}
	tagnumber, err := ConvertAndVerifyTagnumber(urlQueries.Get("tagnumber"))
	if err != nil || tagnumber == nil {
		log.HTTPWarning(req, "Invalid tagnumber provided in GetBiosData: "+err.Error())
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}

	db, err := database.NewSelectRepo()
	if err != nil {
		log.HTTPWarning(req, "Error creating select repository in GetBiosData: "+err.Error())
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}

	biosData, err := db.GetBiosData(ctx, tagnumber)
	if err != nil {
		log.HTTPWarning(req, "Query error in GetBiosData: "+err.Error())
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}
	middleware.WriteJson(w, http.StatusOK, biosData)
}

func GetOSData(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	log := middleware.GetLoggerFromContext(ctx)
	urlQueries, err := middleware.GetRequestQueryFromContext(ctx)
	if err != nil {
		log.HTTPWarning(req, "Error retrieving query parameters from context for GetOSData: "+err.Error())
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}
	tagnumber, err := ConvertAndVerifyTagnumber(urlQueries.Get("tagnumber"))
	if err != nil {
		log.HTTPWarning(req, "Invalid tagnumber provided in GetOSData: "+err.Error())
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}

	db, err := database.NewSelectRepo()
	if err != nil {
		log.HTTPWarning(req, "Error creating select repository in GetOSData: "+err.Error())
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}

	osData, err := db.GetOsData(ctx, tagnumber)
	if err != nil {
		log.HTTPWarning(req, "Query error in GetOSData: "+err.Error())
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}
	middleware.WriteJson(w, http.StatusOK, osData)
}

func GetClientQueuedJobs(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	log := middleware.GetLoggerFromContext(ctx)
	urlQueries, err := middleware.GetRequestQueryFromContext(ctx)
	if err != nil {
		log.HTTPWarning(req, "Error retrieving query parameters from context for GetClientQueuedJobs: "+err.Error())
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}
	tagnumber, err := ConvertAndVerifyTagnumber(urlQueries.Get("tagnumber"))
	if err != nil {
		log.HTTPWarning(req, "Invalid tagnumber provided in GetClientQueuedJobs: "+err.Error())
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}

	db, err := database.NewSelectRepo()
	if err != nil {
		log.HTTPWarning(req, "Error creating select repository in GetClientQueuedJobs: "+err.Error())
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}

	activeJobs, err := db.GetActiveJobs(ctx, tagnumber)
	if err != nil {
		log.HTTPWarning(req, "Query error in GetClientQueuedJobs: "+err.Error())
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}

	middleware.WriteJson(w, http.StatusOK, activeJobs)
}

func GetClientAvailableJobs(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	log := middleware.GetLoggerFromContext(ctx)
	urlQueries, err := middleware.GetRequestQueryFromContext(ctx)
	if err != nil {
		log.HTTPWarning(req, "Error retrieving query parameters from context for GetClientAvailableJobs: "+err.Error())
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}
	tagnumber, err := ConvertAndVerifyTagnumber(urlQueries.Get("tagnumber"))
	if err != nil {
		log.HTTPWarning(req, "Invalid tagnumber provided in GetClientAvailableJobs: "+err.Error())
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}

	db, err := database.NewSelectRepo()
	if err != nil {
		log.HTTPWarning(req, "Error creating select repository in GetClientAvailableJobs: "+err.Error())
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}

	availableJobs, err := db.GetAvailableJobs(ctx, tagnumber)
	if err != nil {
		log.HTTPWarning(req, "Query error in GetClientAvailableJobs: "+err.Error())
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}
	middleware.WriteJson(w, http.StatusOK, availableJobs)
}

func GetNotes(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	log := middleware.GetLoggerFromContext(ctx)
	urlQueries, err := middleware.GetRequestQueryFromContext(ctx)
	if err != nil {
		log.HTTPWarning(req, "Error retrieving query parameters from context for GetNotes: "+err.Error())
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}
	noteType := strings.TrimSpace(urlQueries.Get("note_type"))
	if noteType == "" {
		log.HTTPInfo(req, "No note_type provided, defaulting to 'general'")
		noteType = "general"
	}

	db, err := database.NewSelectRepo()
	if err != nil {
		log.HTTPWarning(req, "Error creating select repository in GetNotes: "+err.Error())
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}

	notesData, err := db.GetNotes(ctx, &noteType)
	if err != nil {
		log.HTTPWarning(req, "Query error in GetNotes: "+err.Error())
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}
	middleware.WriteJson(w, http.StatusOK, notesData)
}

func GetLocationFormData(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	log := middleware.GetLoggerFromContext(ctx)
	requestQueries, err := middleware.GetRequestQueryFromContext(ctx)
	if err != nil {
		log.HTTPWarning(req, "Error retrieving query parameters from context for GetLocationFormData: "+err.Error())
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}

	serial := strings.TrimSpace(requestQueries.Get("system_serial"))
	tagnumber, tagErr := ConvertAndVerifyTagnumber(requestQueries.Get("tagnumber"))
	if tagErr != nil {
		if serial == "" {
			log.HTTPWarning(req, "No or invalid tagnumber and no system_serial provided in GetLocationFormData")
			middleware.WriteJsonError(w, http.StatusBadRequest)
			return
		}
	}

	db, err := database.NewSelectRepo()
	if err != nil {
		log.HTTPWarning(req, "Error creating select repository in GetLocationFormData: "+err.Error())
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}
	locationData, err := db.GetLocationFormData(ctx, tagnumber, &serial)
	if err != nil {
		log.HTTPWarning(req, "Query error in GetLocationFormData: "+err.Error())
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}
	middleware.WriteJson(w, http.StatusOK, locationData)
}

func GetClientImagesManifest(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	log := middleware.GetLoggerFromContext(ctx)
	requestQueries, err := middleware.GetRequestQueryFromContext(ctx)
	if err != nil {
		log.HTTPWarning(req, "Error retrieving query parameters from context for GetClientImagesManifest: "+err.Error())
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}
	tagnumber, err := ConvertAndVerifyTagnumber(requestQueries.Get("tagnumber"))
	if err != nil {
		log.HTTPWarning(req, "Invalid tagnumber provided in request to GetClientImagesManifest: "+err.Error())
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}

	appState, err := config.GetAppState()
	if err != nil {
		log.HTTPWarning(req, "Error getting app state in GetClientImagesManifest: "+err.Error())
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}
	minImageSize, maxImageSize, _, acceptedImageExtensionsAndMimeTypes, err := appState.GetFileUploadImageConstraints()
	if err != nil {
		log.HTTPWarning(req, "Error getting accepted image extensions and mime types in GetClientImagesManifest: "+err.Error())
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}
	minVideoSize, maxVideoSize, _, acceptedVideoExtensionsAndMimeTypes, err := appState.GetFileUploadVideoConstraints()
	if err != nil {
		log.HTTPWarning(req, "Error getting accepted video extensions and mime types in GetClientImagesManifest: "+err.Error())
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}

	db, err := database.NewSelectRepo()
	if err != nil {
		log.HTTPWarning(req, "Error creating select repository in GetClientImagesManifest: "+err.Error())
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}

	imageManifests, err := db.GetClientImageManifestByTag(ctx, tagnumber)
	if err != nil && err != sql.ErrNoRows {
		log.HTTPWarning(req, "Query error in GetClientImagesManifest: "+err.Error())
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}
	if len(imageManifests) == 0 {
		log.HTTPWarning(req, "No image manifest data for client: "+fmt.Sprintf("%d", *tagnumber))
		middleware.WriteJsonError(w, http.StatusNotFound)
		return
	}

	var filteredImageManifests []database.ImageManifest
	for _, imageManifest := range imageManifests {
		// UUID of file
		if imageManifest.UUID == nil || strings.TrimSpace(*imageManifest.UUID) == "" {
			log.HTTPWarning(req, "Image manifest UUID is nil or empty in GetClientImagesManifest")
			continue
		}
		fileUUID := *imageManifest.UUID

		// File path
		if imageManifest.FilePath == nil || strings.TrimSpace(*imageManifest.FilePath) == "" {
			log.HTTPInfo(req, "File path provided to GetClientImagesManifest is nil")
			continue
		}
		filePath := strings.ToLower(strings.TrimSpace(*imageManifest.FilePath))

		// URL to send to client
		imageManifest.FilePath = nil // Hide actual file path from client
		urlStr, err := url.JoinPath("/api/images/", fmt.Sprintf("%d", *imageManifest.Tagnumber), fileUUID)
		if err != nil {
			log.HTTPWarning(req, "Error joining URL paths in GetClientImagesManifest: "+err.Error())
			continue
		}
		imageManifest.URL = &urlStr

		// Check if marked as hidden
		if imageManifest.Hidden != nil && *imageManifest.Hidden {
			log.HTTPInfo(req, "Hidden file requested, but not sent: "+filePath)
			continue
		}

		// Check MIME type in DB
		if imageManifest.MimeType == nil {
			log.HTTPWarning(req, "File '"+fileUUID+"' has a nil MIME type in DB (GetClientImagesManifest)")
			continue
		}
		mimeType := strings.TrimSpace(*imageManifest.MimeType)

		// File extension
		fileExtension := strings.ToLower(filepath.Ext(filePath))
		if fileExtension == "" || acceptedImageExtensionsAndMimeTypes[fileExtension] == "" {
			log.HTTPWarning(req, "File '"+fileUUID+"' has an unsupported file extension (GetClientImagesManifest): "+fileExtension)
			continue
		}

		// Open file and read metadata
		file, err := os.Open(filePath)
		if err != nil {
			log.HTTPWarning(req, "Cannot open file '"+fileUUID+"' (GetClientImagesManifest): "+err.Error())
			continue
		}
		isValidFile := func() bool {
			defer func() {
				_ = file.Close()
			}()

			imageStat, err := file.Stat()
			if err != nil {
				log.HTTPWarning(req, "Cannot stat file '"+fileUUID+"' (GetClientImagesManifest): "+err.Error())
				return false
			}

			// Check if MIME type matches file content
			mt := http.DetectContentType([]byte(filePath))
			if mt != mimeType {
				log.HTTPWarning(req, "MIME in type in DB does not match file content for file '"+fileUUID+"' (GetClientImagesManifest): Detected MIME type: "+mt+", MIME type in DB: "+mimeType)
				return false
			}
			imageManifest.MimeType = &mimeType

			// Get file size
			metadataFileSize := imageStat.Size()
			imageManifest.FileSize = &metadataFileSize

			// If an image
			if _, ok := acceptedImageExtensionsAndMimeTypes[fileExtension]; ok {
				// Check file size from metadata
				if metadataFileSize < minImageSize || metadataFileSize > maxImageSize {
					log.HTTPWarning(req, "Image file size is out of bounds for file '"+fileUUID+"' in GetClientImagesManifest: File size: "+fmt.Sprintf("%d", metadataFileSize))
					return false
				}

				imageBytes, err := io.ReadAll(io.LimitReader(file, maxImageSize+1))
				if err != nil {
					log.HTTPWarning(req, "Error reading image file in GetClientImagesManifest: "+filePath+" "+err.Error())
					return false
				}
				if len(imageBytes) > int(maxImageSize) {
					log.HTTPWarning(req, "Image file size exceeds maximum after reading in GetClientImagesManifest: "+filePath+" -> File size: "+fmt.Sprintf("%d", len(imageBytes)))
					return false
				}

				// Get image metadata
				imageConfig, imageType, err := image.DecodeConfig(bytes.NewReader(imageBytes))
				if err != nil {
					log.HTTPWarning(req, "Cannot decode image in GetClientImagesManifest: "+filePath+" "+err.Error())
					return false
				}

				// Check http's library MIME type against image library's detected type
				if imageType != "image/"+acceptedImageExtensionsAndMimeTypes[fileExtension] {
					log.HTTPWarning(req, "Image has invalid file type in GetClientImagesManifest: "+filePath+" -> Image type: "+imageType+", File extension: "+fileExtension)
					return false
				}

				// If image has zero width or height, continue
				if imageConfig.Width == 0 || imageConfig.Height == 0 {
					log.HTTPWarning(req, "Image has invalid dimensions in GetClientImagesManifest: "+filePath)
					return false
				}
				resX := int64(imageConfig.Width)
				resY := int64(imageConfig.Height)
				imageManifest.ResolutionX = &resX
				imageManifest.ResolutionY = &resY
				return true
			}

			if _, ok := acceptedVideoExtensionsAndMimeTypes[fileExtension]; ok { // If a video file
				// Check file size from metadata
				if metadataFileSize < minVideoSize || metadataFileSize > maxVideoSize {
					log.HTTPWarning(req, "Video file size is out of bounds in GetClientImagesManifest: "+filePath+" -> File size: "+fmt.Sprintf("%d", metadataFileSize))
					return false
				}

				videoBytes, err := io.ReadAll(io.LimitReader(file, maxVideoSize+1))
				if err != nil {
					log.HTTPWarning(req, "Error reading video file in GetClientImagesManifest: "+filePath+" "+err.Error())
					return false
				}
				if len(videoBytes) > int(maxVideoSize) {
					log.HTTPWarning(req, "Video file size exceeds maximum after reading in GetClientImagesManifest: "+filePath+" -> File size: "+fmt.Sprintf("%d", len(videoBytes)))
					return false
				}

				// Check MIME type matches expected MIME type for video extension
				if acceptedVideoExtensionsAndMimeTypes[fileExtension] != mimeType {
					log.HTTPWarning(req, "Video manifest has mismatched MIME type in GetClientImagesManifest: "+filePath+" -> Detected MIME type: "+mimeType+", Expected MIME type: "+acceptedVideoExtensionsAndMimeTypes[fileExtension])
					return false
				}
				return true
			}

			log.HTTPWarning(req, "File has unsupported extension in GetClientImagesManifest: "+filePath)
			return false
		}()

		if !isValidFile {
			continue
		}

		filteredImageManifests = append(filteredImageManifests, imageManifest)
	}

	// If all manifests were filtered out
	if len(filteredImageManifests) == 0 {
		log.HTTPWarning(req, "No valid image manifests to return in GetClientImagesManifest for client: "+fmt.Sprintf("%d", *tagnumber))
		middleware.WriteJsonError(w, http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	middleware.WriteJson(w, http.StatusOK, filteredImageManifests)
}

func GetImage(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	log := middleware.GetLoggerFromContext(ctx)
	requestedQueries, err := middleware.GetRequestQueryFromContext(ctx)
	if err != nil {
		log.HTTPWarning(req, "Error retrieving query parameters from context for GetImage: "+err.Error())
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}
	appState, err := config.GetAppState()
	if err != nil {
		log.HTTPWarning(req, "Error getting app state (GetImage): "+err.Error())
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}
	_, _, _, acceptedImageExtensionsAndMimeTypes, err := appState.GetFileUploadImageConstraints()
	if err != nil {
		log.HTTPWarning(req, "Error getting file upload image constraints (GetImage): "+err.Error())
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}
	_, _, _, acceptedVideoExtensionsAndMimeTypes, err := appState.GetFileUploadVideoConstraints()
	if err != nil {
		log.HTTPWarning(req, "Error getting file upload video constraints (GetImage): "+err.Error())
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}

	// local filepath example: inventory-images/{tag}/{date --iso}-{uuid}.{file extension}
	// incoming request url: /api/images/{tag}/{uuid}.{file extension}
	uuidInURLQuery := middleware.GetStrQuery(requestedQueries, "uuid")
	if uuidInURLQuery == nil || strings.TrimSpace(*uuidInURLQuery) == "" {
		log.HTTPWarning(req, "No image UUID provided in request to GetImage")
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}
	imageUUID := strings.TrimSpace(strings.ToLower(*uuidInURLQuery))
	for ext := range acceptedImageExtensionsAndMimeTypes {
		if filepath.Ext(imageUUID) == ext {
			imageUUID = strings.TrimSuffix(imageUUID, ext)
		}
	}
	for ext := range acceptedVideoExtensionsAndMimeTypes {
		if filepath.Ext(imageUUID) == ext {
			imageUUID = strings.TrimSuffix(imageUUID, ext)
		}
	}
	if imageUUID == "" {
		log.HTTPWarning(req, "No uuid provided in request (GetImage)")
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}

	db, err := database.NewSelectRepo()
	if err != nil {
		log.HTTPWarning(req, "Error creating select repository (GetImage): "+err.Error())
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}

	log.HTTPDebug(req, "Serving image request for: "+imageUUID)
	imageManifest, err := db.GetClientImageFilePathFromUUID(ctx, &imageUUID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			log.HTTPInfo(req, "Image not found from UUID lookup (GetImage): "+imageUUID+" "+err.Error())
			middleware.WriteJsonError(w, http.StatusNotFound)
			return
		}
		log.HTTPInfo(req, "Client image query error (GetImage): "+imageUUID+" "+err.Error())
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}

	if imageManifest == nil {
		log.HTTPInfo(req, "No image manifest data found for UUID (GetImage): "+imageUUID)
		middleware.WriteJsonError(w, http.StatusNotFound)
		return
	}

	if imageManifest.Hidden != nil && *imageManifest.Hidden {
		log.HTTPWarning(req, "Attempt to access hidden image (GetImage): "+imageUUID)
		middleware.WriteJsonError(w, http.StatusNotFound)
		return
	}

	if imageManifest.FilePath == nil || strings.TrimSpace(*imageManifest.FilePath) == "" {
		log.HTTPWarning(req, "File path for image is nil or empty (GetImage): "+imageUUID)
		middleware.WriteJsonError(w, http.StatusNotFound)
		return
	}

	imageFile, err := os.Open(*imageManifest.FilePath)
	if err != nil {
		if os.IsNotExist(err) {
			log.HTTPWarning(req, "Image not found on disk (GetImage): "+imageUUID+" "+err.Error())
			middleware.WriteJsonError(w, http.StatusNotFound)
			return
		}
		log.HTTPWarning(req, "Image cannot be opened (GetImage): "+imageUUID+" "+err.Error())
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}
	defer imageFile.Close()

	http.ServeContent(w, req, imageFile.Name(), time.Time{}, imageFile)
}

// Overview section
func GetJobQueueTable(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	log := middleware.GetLoggerFromContext(ctx)

	db, err := database.NewSelectRepo()
	if err != nil {
		log.HTTPWarning(req, "Error creating select repository (GetJobQueueTable): "+err.Error())
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}

	jobQueueTable, err := db.GetJobQueueTable(ctx)
	if err != nil {
		log.HTTPWarning(req, "Query error (GetJobQueueTable): "+err.Error())
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}

	middleware.WriteJson(w, http.StatusOK, jobQueueTable)
}

func GetJobQueueOverview(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	log := middleware.GetLoggerFromContext(ctx)

	db, err := database.NewSelectRepo()
	if err != nil {
		log.HTTPWarning(req, "Error creating select repository (GetJobQueueOverview): "+err.Error())
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}

	jobQueueOverview, err := db.GetJobQueueOverview(ctx)
	if err != nil {
		log.HTTPWarning(req, "Query error (GetJobQueueOverview): "+err.Error())
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}

	middleware.WriteJson(w, http.StatusOK, jobQueueOverview)
}

func GetDashboardInventorySummary(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	log := middleware.GetLoggerFromContext(ctx)

	db, err := database.NewSelectRepo()
	if err != nil {
		log.HTTPWarning(req, "Error creating select repository (GetDashboardInventorySummary): "+err.Error())
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}

	inventorySummary, err := db.GetDashboardInventorySummary(ctx)
	if err != nil {
		log.HTTPWarning(req, "Query error (GetDashboardInventorySummary): "+err.Error())
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}
	middleware.WriteJson(w, http.StatusOK, inventorySummary)
}

func GetInventoryTableData(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	log := middleware.GetLoggerFromContext(ctx)
	requestQueries, err := middleware.GetRequestQueryFromContext(ctx)
	if err != nil {
		log.HTTPWarning(req, "Error retrieving query parameters from context for GetInventoryTableData: "+err.Error())
		if requestQueries != nil {
			middleware.WriteJsonError(w, http.StatusBadRequest)
			return
		}
	}

	if requestQueries == nil || len(*requestQueries) == 0 {
		requestQueries = new(url.Values)
	}

	filterOptions := &database.InventoryAdvSearchOptions{
		Tagnumber:          middleware.GetInt64Query(requestQueries, "tagnumber"),
		SystemSerial:       middleware.GetStrQuery(requestQueries, "system_serial"),
		Location:           middleware.GetStrQuery(requestQueries, "location"),
		SystemManufacturer: middleware.GetStrQuery(requestQueries, "system_manufacturer"),
		SystemModel:        middleware.GetStrQuery(requestQueries, "system_model"),
		DeviceType:         middleware.GetStrQuery(requestQueries, "device_type"),
		Department:         middleware.GetStrQuery(requestQueries, "department_name"),
		Domain:             middleware.GetStrQuery(requestQueries, "ad_domain"),
		Status:             middleware.GetStrQuery(requestQueries, "status"),
		Broken:             middleware.GetBoolQuery(requestQueries, "is_broken"),
		HasImages:          middleware.GetBoolQuery(requestQueries, "has_images"),
	}

	// log.HTTPDebug(req, fmt.Sprintf("Filter options: %+v", filterOptions))

	db, err := database.NewSelectRepo()
	if err != nil {
		log.HTTPWarning(req, "Error creating select repository (GetInventoryTableData): "+err.Error())
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}

	inventoryTableData, err := db.GetInventoryTableData(ctx, filterOptions)
	if err != nil {
		log.HTTPWarning(req, "Query error (GetInventoryTableData): "+err.Error())
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}
	if requestQueries.Get("csv") == "true" {
		log.HTTPDebug(req, "CSV file requested in GetInventoryTableData")
		csvBytes, err := database.ConvertInventoryTableDataToCSV(ctx, inventoryTableData)
		if err != nil {
			log.HTTPWarning(req, "Error converting inventory table data to CSV in GetInventoryTableData: "+err.Error())
			middleware.WriteJsonError(w, http.StatusInternalServerError)
			return
		}
		if csvBytes == nil || csvBytes.Len() == 0 {
			log.HTTPWarning(req, "No CSV data generated in GetInventoryTableData")
			middleware.WriteJsonError(w, http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "text/csv; charset=utf-8")
		w.Header().Set("Content-Disposition", "attachment; filename=\"inventory_table_data-"+time.Now().Format("01-02-2006-150405-MST")+".csv\"")
		if _, err = w.Write(csvBytes.Bytes()); err != nil {
			log.HTTPWarning(req, "Error writing CSV data to response: "+err.Error())
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
		log.HTTPWarning(req, "Error getting client config in GetClientConfig: "+err.Error())
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}

	if req.URL.Query().Get("json") == "true" {
		middleware.WriteJson(w, http.StatusOK, clientConfig)
		return
	}

	if err := json.NewEncoder(w).Encode(clientConfig); err != nil {
		log.HTTPWarning(req, "Error encoding client config to JSON in GetClientConfig: "+err.Error())
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}
	middleware.WriteJson(w, http.StatusOK, clientConfig)
}

func GetManufacturersAndModels(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	log := middleware.GetLoggerFromContext(ctx)

	db, err := database.NewSelectRepo()
	if err != nil {
		log.HTTPWarning(req, "Error creating select repository in GetManufacturersAndModels: "+err.Error())
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}

	manufacturersAndModels, err := db.GetManufacturersAndModels(ctx)
	if err != nil {
		log.HTTPWarning(req, "Query error in GetManufacturersAndModels: "+err.Error())
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}
	middleware.WriteJson(w, http.StatusOK, manufacturersAndModels)
}

func GetClientBatteryHealth(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	log := middleware.GetLoggerFromContext(ctx)
	urlQueries, err := middleware.GetRequestQueryFromContext(ctx)
	if err != nil {
		log.HTTPWarning(req, "Error retrieving query parameters from context for GetClientBatteryHealth: "+err.Error())
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}
	tagnumber, err := ConvertAndVerifyTagnumber(urlQueries.Get("tagnumber"))
	if err != nil {
		log.HTTPWarning(req, "Invalid tagnumber provided in GetClientBatteryHealth: "+err.Error())
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}

	db, err := database.NewSelectRepo()
	if err != nil {
		log.HTTPWarning(req, "Error creating select repository in GetClientBatteryHealth: "+err.Error())
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}

	batteryHealthData, err := db.GetClientBatteryHealth(ctx, tagnumber)
	if err != nil {
		log.HTTPWarning(req, "Query error in GetClientBatteryHealth: "+err.Error())
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}

	middleware.WriteJson(w, http.StatusOK, batteryHealthData)
}

func GetDomains(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	log := middleware.GetLoggerFromContext(ctx)

	db, err := database.NewSelectRepo()
	if err != nil {
		log.HTTPWarning(req, "Error creating select repository in GetDomains: "+err.Error())
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}

	domains, err := db.GetDomains(ctx)
	if err != nil {
		log.HTTPWarning(req, "Query error in GetDomains: "+err.Error())
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}
	middleware.WriteJson(w, http.StatusOK, domains)
}

func GetDepartments(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	log := middleware.GetLoggerFromContext(ctx)

	db, err := database.NewSelectRepo()
	if err != nil {
		log.HTTPWarning(req, "Error creating select repository in GetDepartments: "+err.Error())
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}

	departments, err := db.GetDepartments(ctx)
	if err != nil {
		log.HTTPWarning(req, "Query error in GetDepartments: "+err.Error())
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}
	middleware.WriteJson(w, http.StatusOK, departments)
}

func CheckAuth(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	log := middleware.GetLoggerFromContext(ctx)
	if ctx.Err() != nil {
		log.HTTPInfo(req, "CheckAuth canceled/timeout")
		middleware.WriteJsonError(w, http.StatusRequestTimeout)
		return
	}

	data := map[string]string{
		"status": "authenticated",
		"time":   time.Now().Format(time.RFC3339),
	}

	middleware.WriteJson(w, http.StatusOK, data)
}

func GetBatteryStandardDeviation(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	log := middleware.GetLoggerFromContext(ctx)

	db, err := database.NewSelectRepo()
	if err != nil {
		log.HTTPWarning(req, "Error creating select repository in GetBatteryStandardDeviation: "+err.Error())
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}

	batteryStdDevData, err := db.GetBatteryStandardDeviation(ctx)
	if err != nil {
		log.HTTPWarning(req, "Query error in GetBatteryStandardDeviation: "+err.Error())
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}
	middleware.WriteJson(w, http.StatusOK, batteryStdDevData)
}

func GetAllJobs(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	log := middleware.GetLoggerFromContext(ctx)

	db, err := database.NewSelectRepo()
	if err != nil {
		log.HTTPWarning(req, "Error creating select repository in GetAllJobs: "+err.Error())
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}

	allJobs, err := db.GetAllJobs(ctx)
	if err != nil {
		log.HTTPWarning(req, "Query error in GetAllJobs: "+err.Error())
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}
	middleware.WriteJson(w, http.StatusOK, allJobs)
}

func GetAllLocations(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	log := middleware.GetLoggerFromContext(ctx)

	db, err := database.NewSelectRepo()
	if err != nil {
		log.HTTPWarning(req, "Error creating select repository in GetAllLocations: "+err.Error())
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}

	allLocations, err := db.GetAllLocations(ctx)
	if err != nil {
		log.HTTPWarning(req, "Query error in GetAllLocations: "+err.Error())
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}
	middleware.WriteJson(w, http.StatusOK, allLocations)
}

func GetAllStatuses(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	log := middleware.GetLoggerFromContext(ctx)

	db, err := database.NewSelectRepo()
	if err != nil {
		log.HTTPWarning(req, "Error creating select repository in GetAllStatuses: "+err.Error())
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}
	allStatuses, err := db.GetAllStatuses(ctx)
	if err != nil {
		log.HTTPWarning(req, "Query error in GetAllStatuses: "+err.Error())
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}
	middleware.WriteJson(w, http.StatusOK, allStatuses)
}

func GetAllDeviceTypes(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	log := middleware.GetLoggerFromContext(ctx)

	db, err := database.NewSelectRepo()
	if err != nil {
		log.HTTPWarning(req, "Error creating select repository in GetAllDeviceTypes: "+err.Error())
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}

	allDeviceTypes, err := db.GetAllDeviceTypes(ctx)
	if err != nil {
		log.HTTPWarning(req, "Query error in GetAllDeviceTypes: "+err.Error())
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}
	middleware.WriteJson(w, http.StatusOK, allDeviceTypes)
}
