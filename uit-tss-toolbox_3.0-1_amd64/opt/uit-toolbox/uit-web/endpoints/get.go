package endpoints

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"image"
	"net/http"
	"net/url"
	"os"
	"path"
	"strconv"
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
	var tagStr = strings.TrimSpace(urlQueries.Get("tagnumber"))
	var systemSerial = strings.TrimSpace(urlQueries.Get("system_serial"))
	var tagnumber int64

	if tagStr == "" && systemSerial == "" {
		log.HTTPWarning(req, "No tagnumber or system_serial provided in client lookup request")
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}

	if tagStr != "" {
		tagnumber, err = ConvertTagnumber(urlQueries.Get("tagnumber"))
		if err != nil {
			log.HTTPWarning(req, "Cannot convert tagnumber to int64 in GetClientLookup: "+err.Error())
			middleware.WriteJsonError(w, http.StatusBadRequest)
			return
		}
	}

	db := config.GetDatabaseConn()
	if db == nil {
		log.HTTPWarning(req, "No database connection available for GetClientLookup")
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}
	repo := database.NewRepo(db)

	var hardwareData *database.ClientLookup
	if tagnumber != 0 {
		hardwareData, err = repo.ClientLookupByTag(ctx, tagnumber)
	} else if systemSerial != "" {
		hardwareData, err = repo.ClientLookupBySerial(ctx, systemSerial)
	}
	if err != nil {
		if err != sql.ErrNoRows {
			log.HTTPWarning(req, "Client lookup query error: "+err.Error())
			middleware.WriteJsonError(w, http.StatusInternalServerError)
			return
		}
	}
	middleware.WriteJson(w, http.StatusOK, hardwareData)
}

func GetAllTags(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	log := middleware.GetLoggerFromContext(ctx)

	db := config.GetDatabaseConn()
	if db == nil {
		log.HTTPWarning(req, "No database connection available for GetAllTags")
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}
	repo := database.NewRepo(db)

	allTags, err := repo.GetAllTags(ctx)
	if err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) || errors.Is(ctx.Err(), context.Canceled) || errors.Is(ctx.Err(), context.DeadlineExceeded) {
			log.HTTPInfo(req, "GetAllTags canceled/timeout")
			middleware.WriteJsonError(w, http.StatusRequestTimeout)
			return
		}
		if err != sql.ErrNoRows {
			log.HTTPWarning(req, "Query error in GetAllTags: "+err.Error())
			middleware.WriteJsonError(w, http.StatusInternalServerError)
			return
		}
	}
	middleware.WriteJson(w, http.StatusOK, allTags)
}

func GetHardwareIdentifiers(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	log := middleware.GetLoggerFromContext(ctx)

	urlQueries, err := middleware.GetRequestQueryFromContext(ctx)
	if err != nil {
		log.HTTPWarning(req, "Error retrieving query parameters from context for GetHardwareIdentifiers: "+err.Error())
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}
	tagnumber, err := ConvertTagnumber(urlQueries.Get("tagnumber"))
	if err != nil {
		log.HTTPWarning(req, "Invalid tagnumber provided in GetHardwareIdentifiers: "+err.Error())
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}

	db := config.GetDatabaseConn()
	if db == nil {
		log.HTTPWarning(req, "No database connection available for GetHardwareIdentifiers")
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}
	repo := database.NewRepo(db)
	hardwareData, err := repo.GetHardwareIdentifiers(ctx, tagnumber)
	if err != nil {
		if err != sql.ErrNoRows {
			log.HTTPInfo(req, "Query error in GetHardwareIdentifiers: "+err.Error())
			middleware.WriteJsonError(w, http.StatusInternalServerError)
			return
		}
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
	tagnumber, err := ConvertTagnumber(urlQueries.Get("tagnumber"))
	if err != nil {
		log.HTTPWarning(req, "Invalid tagnumber provided in GetBiosData: "+err.Error())
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}

	db := config.GetDatabaseConn()
	if db == nil {
		log.HTTPWarning(req, "No database connection available in GetBiosData")
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}
	repo := database.NewRepo(db)

	biosData, err := repo.GetBiosData(ctx, tagnumber)
	if err != nil {
		if err != sql.ErrNoRows {
			log.HTTPWarning(req, "Query error in GetBiosData: "+err.Error())
			middleware.WriteJsonError(w, http.StatusInternalServerError)
			return
		}
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
	tagnumber, err := ConvertTagnumber(urlQueries.Get("tagnumber"))
	if err != nil {
		log.HTTPWarning(req, "Invalid tagnumber provided in GetOSData: "+err.Error())
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}

	db := config.GetDatabaseConn()
	if db == nil {
		log.HTTPWarning(req, "No database connection available in GetOSData")
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}
	repo := database.NewRepo(db)

	osData, err := repo.GetOsData(ctx, tagnumber)
	if err != nil {
		if err != sql.ErrNoRows {
			log.Info("Query error in GetOSData: " + err.Error())
			middleware.WriteJsonError(w, http.StatusInternalServerError)
			return
		}
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
	tagnumber, err := ConvertTagnumber(urlQueries.Get("tagnumber"))
	if err != nil {
		log.HTTPWarning(req, "Invalid tagnumber provided in GetClientQueuedJobs: "+err.Error())
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}

	db := config.GetDatabaseConn()
	if db == nil {
		log.HTTPWarning(req, "No database connection available in GetClientQueuedJobs")
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}
	repo := database.NewRepo(db)

	activeJobs, err := repo.GetActiveJobs(ctx, tagnumber)
	if err != nil {
		if err != sql.ErrNoRows {
			log.HTTPWarning(req, "Query error in GetClientQueuedJobs: "+err.Error())
			middleware.WriteJsonError(w, http.StatusInternalServerError)
			return
		}
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
	tagnumber, err := ConvertTagnumber(urlQueries.Get("tagnumber"))
	if err != nil {
		log.HTTPWarning(req, "Invalid tagnumber provided in GetClientAvailableJobs: "+err.Error())
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}

	db := config.GetDatabaseConn()
	if db == nil {
		log.HTTPWarning(req, "No database connection available in GetClientAvailableJobs")
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}
	repo := database.NewRepo(db)

	availableJobs, err := repo.GetAvailableJobs(ctx, tagnumber)
	if err != nil {
		if err != sql.ErrNoRows {
			log.HTTPWarning(req, "Query error in GetClientAvailableJobs: "+err.Error())
			middleware.WriteJsonError(w, http.StatusInternalServerError)
			return
		}
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

	db := config.GetDatabaseConn()
	if db == nil {
		log.HTTPWarning(req, "No database connection available in GetNotes")
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}
	repo := database.NewRepo(db)

	notesData, err := repo.GetNotes(ctx, noteType)
	if err != nil {
		if err != sql.ErrNoRows {
			log.HTTPWarning(req, "Query error in GetNotes: "+err.Error())
			middleware.WriteJsonError(w, http.StatusInternalServerError)
			return
		}
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
	tagnumber, err := ConvertTagnumber(requestQueries.Get("tagnumber"))
	if err != nil {
		log.HTTPWarning(req, "Invalid tagnumber provided in GetLocationFormData: "+err.Error())
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}

	db := config.GetDatabaseConn()
	if db == nil {
		log.HTTPWarning(req, "No database connection available for GetLocationFormData")
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}
	repo := database.NewRepo(db)
	locationData, err := repo.GetLocationFormData(ctx, tagnumber)
	if err != nil {
		if err != sql.ErrNoRows {
			log.HTTPWarning(req, "Query error in GetLocationFormData: "+err.Error())
			middleware.WriteJsonError(w, http.StatusInternalServerError)
			return
		}
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
	tagnumber, err := ConvertTagnumber(requestQueries.Get("tagnumber"))
	if err != nil {
		log.HTTPWarning(req, "No or invalid tagnumber provided in request to GetClientImagesManifest: "+err.Error())
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}

	db := config.GetDatabaseConn()
	if db == nil {
		log.HTTPWarning(req, "No database connection available for GetClientImagesManifest")
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}
	repo := database.NewRepo(db)
	imageUUIDs, err := repo.GetClientImageUUIDs(ctx, tagnumber)
	if err != nil && err != sql.ErrNoRows {
		log.HTTPWarning(req, "Query error in GetClientImagesManifest: "+err.Error())
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}

	var imageManifests []database.ImageManifest
	for _, imageUUID := range imageUUIDs {
		var imageData database.ImageManifest
		time, tag, filepath, _, hidden, primaryImage, note, err := repo.GetClientImageManifestByUUID(ctx, imageUUID)
		if err != nil {
			if err == sql.ErrNoRows {
				log.HTTPInfo(req, "Image not found in database: "+imageUUID+" "+err.Error())
			} else {
				log.HTTPWarning(req, "Query error in GetClientImagesManifest: "+imageUUID+" "+err.Error())
				middleware.WriteJsonError(w, http.StatusInternalServerError)
				return
			}
		}

		if hidden != nil && *hidden {
			continue
		}

		if filepath == nil || strings.TrimSpace(*filepath) == "" {
			log.HTTPInfo(req, "Filepath provided to GetClientImagesManifest is nil")
			middleware.WriteJsonError(w, http.StatusInternalServerError)
			return
		}

		img, err := os.Open(*filepath)
		if err != nil {
			log.HTTPWarning(req, "Cannot open image file in GetClientImagesManifest: "+*filepath+" "+err.Error())
			middleware.WriteJsonError(w, http.StatusInternalServerError)
			return
		}
		defer img.Close()

		imageStat, err := img.Stat()
		if err != nil {
			log.HTTPWarning(req, "Cannot stat image in GetClientImagesManifest: "+*filepath+" "+err.Error())
			middleware.WriteJsonError(w, http.StatusInternalServerError)
			return
		}
		fileExtension := strings.ToLower(path.Ext(imageStat.Name()))

		filePathLower := strings.ToLower(*filepath)
		if strings.HasSuffix(filePathLower, ".jpg") ||
			strings.HasSuffix(filePathLower, ".jpeg") ||
			strings.HasSuffix(filePathLower, ".png") {

			imageReader := http.MaxBytesReader(w, img, 64<<20)
			imageConfig, imageType, err := image.DecodeConfig(imageReader)
			if err != nil {
				_ = img.Close()
				log.HTTPWarning(req, "Cannot decode image in GetClientImagesManifest: "+*filepath+" "+err.Error())
				middleware.WriteJsonError(w, http.StatusInternalServerError)
				return
			}
			if imageType != "jpeg" && imageType != "png" {
				_ = img.Close()
				log.HTTPWarning(req, "Image has invalid type in GetClientImagesManifest: "+*filepath+" -> "+imageType)
				middleware.WriteJsonError(w, http.StatusInternalServerError)
				return
			}
			if (imageType == "jpeg" && fileExtension != ".jpg" && fileExtension != ".jpeg") ||
				(imageType == "png" && fileExtension != ".png") {
				_ = img.Close()
				log.HTTPWarning(req, "Image file extension does not match image type in GetClientImagesManifest: "+imageStat.Name()+" != "+imageType)
				middleware.WriteJsonError(w, http.StatusInternalServerError)
				return
			}
			imageData.Width = &imageConfig.Width
			imageData.Height = &imageConfig.Height
			if imageData.Width == nil || imageData.Height == nil || *imageData.Width <= 0 || *imageData.Height <= 0 {
				log.HTTPWarning(req, "Image has invalid dimensions in GetClientImagesManifest: "+*filepath)
				middleware.WriteJsonError(w, http.StatusInternalServerError)
				return
			}
			fileType := "image/" + imageType
			imageData.FileType = &fileType
		} else if strings.HasSuffix(filePathLower, ".mp4") {
			fileType := "video/mp4"
			imageData.FileType = &fileType
		} else if strings.HasSuffix(filePathLower, ".mov") {
			fileType := "video/quicktime"
			imageData.FileType = &fileType
		}

		_ = img.Close()

		var tagStr string
		if tag != nil && *tag >= 1 {
			tagStr = fmt.Sprintf("%d", *tag)
		} else {
			log.HTTPWarning(req, "Image has no or invalid tag in database in GetClientImagesManifest: "+imageUUID)
			middleware.WriteJsonError(w, http.StatusInternalServerError)
			return
		}
		imageData.Tagnumber = tag
		imageData.Time = time
		imageData.Hidden = hidden
		imageData.PrimaryImage = primaryImage

		imgFileName := imageStat.Name()
		imageData.Name = &imgFileName

		clientImgUUIDPath := imageUUID + fileExtension
		imageData.UUID = &clientImgUUIDPath

		imageData.Note = note

		imgSize := imageStat.Size()
		imageData.Size = &imgSize

		urlStr, err := url.JoinPath("/api/images/", tagStr, clientImgUUIDPath)
		if err != nil {
			log.HTTPWarning(req, "Error joining URL paths in GetClientImagesManifest: "+err.Error())
			middleware.WriteJsonError(w, http.StatusInternalServerError)
			return
		}
		imageData.URL = &urlStr

		imageManifests = append(imageManifests, imageData)
	}
	w.Header().Set("Content-Type", "application/json")
	middleware.WriteJson(w, http.StatusOK, imageManifests)
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
	requestedImageUUID := strings.TrimSpace(requestedQueries.Get("uuid"))
	requestedImageUUID = strings.TrimSuffix(requestedImageUUID, ".jpeg")
	requestedImageUUID = strings.TrimSuffix(requestedImageUUID, ".png")
	requestedImageUUID = strings.TrimSuffix(requestedImageUUID, ".mp4")
	requestedImageUUID = strings.TrimSuffix(requestedImageUUID, ".mov")
	if requestedImageUUID == "" {
		log.HTTPWarning(req, "No image path provided in request")
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}

	db := config.GetDatabaseConn()
	if db == nil {
		log.Warning("no database connection available")
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}
	repo := database.NewRepo(db)

	log.HTTPDebug(req, "Serving image request for: "+requestedImageUUID)
	_, _, imagePath, _, hidden, _, _, err := repo.GetClientImageManifestByUUID(ctx, requestedImageUUID)
	if err != nil {
		if err == sql.ErrNoRows {
			log.HTTPInfo(req, "Image not found: "+requestedImageUUID+" "+err.Error())
			middleware.WriteJsonError(w, http.StatusNotFound)
			return
		}
		log.HTTPInfo(req, "Client image query error: "+requestedImageUUID+" "+err.Error())
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}

	if strings.TrimSpace(*imagePath) == "" {
		log.HTTPInfo(req, "Image path from database is empty for: "+requestedImageUUID)
		middleware.WriteJsonError(w, http.StatusNotFound)
		return
	}
	if *hidden {
		log.HTTPWarning(req, "Attempt to access hidden image: "+requestedImageUUID)
		middleware.WriteJsonError(w, http.StatusNotFound)
		return
	}
	imageFile, err := os.Open(*imagePath)
	if err != nil {
		if os.IsNotExist(err) {
			log.HTTPWarning(req, "Image not found on disk: "+requestedImageUUID+" "+err.Error())
			middleware.WriteJsonError(w, http.StatusNotFound)
			return
		}
		log.HTTPWarning(req, "Image cannot be opened: "+requestedImageUUID+" "+err.Error())
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}
	defer imageFile.Close()

	http.ServeContent(w, req, imageFile.Name(), time.Time{}, imageFile)
}

// Overview section
func GetJobQueueOverview(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	log := middleware.GetLoggerFromContext(ctx)

	db := config.GetDatabaseConn()
	if db == nil {
		log.HTTPWarning(req, "No database connection available for GetJobQueueOverview")
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}
	repo := database.NewRepo(db)

	jobQueueOverview, err := repo.GetJobQueueOverview(ctx)
	if err != nil {
		if err != sql.ErrNoRows {
			log.HTTPWarning(req, "Query error in GetJobQueueOverview: "+err.Error())
			middleware.WriteJsonError(w, http.StatusInternalServerError)
			return
		}
	}

	middleware.WriteJson(w, http.StatusOK, jobQueueOverview)
}

func GetDashboardInventorySummary(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	log := middleware.GetLoggerFromContext(ctx)

	db := config.GetDatabaseConn()
	if db == nil {
		log.HTTPWarning(req, "No database connection available for GetDashboardInventorySummary")
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}
	repo := database.NewRepo(db)

	inventorySummary, err := repo.GetDashboardInventorySummary(ctx)
	if err != nil {
		if err != sql.ErrNoRows {
			log.HTTPWarning(req, "Query error in GetDashboardInventorySummary: "+err.Error())
			middleware.WriteJsonError(w, http.StatusInternalServerError)
			return
		}
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

	if requestQueries == nil {
		requestQueries = &url.Values{}
	}

	getStr := func(key string) *string {
		s := strings.TrimSpace(requestQueries.Get(key))
		if s == "" {
			return nil
		}
		return &s
	}
	getInt64 := func(key string) *int64 {
		raw := strings.TrimSpace(requestQueries.Get(key))
		if raw == "" {
			return nil
		}
		v, err := strconv.ParseInt(raw, 10, 64)
		if err != nil {
			log.HTTPWarning(req, "Invalid '"+key+"' filter provided in GetInventoryTableData: "+err.Error())
			return nil
		}
		return &v
	}
	getBool := func(key string) *bool {
		raw := strings.TrimSpace(requestQueries.Get(key))
		if raw == "" {
			return nil
		}
		v, err := strconv.ParseBool(raw)
		if err != nil {
			log.HTTPWarning(req, "Invalid '"+key+"' filter provided in GetInventoryTableData: "+err.Error())
			return nil
		}
		return &v
	}

	filterOptions := &database.InventoryFilterOptions{
		Tagnumber:          getInt64("tagnumber"),
		SystemSerial:       getStr("system_serial"),
		Location:           getStr("location"),
		SystemManufacturer: getStr("system_manufacturer"),
		SystemModel:        getStr("system_model"),
		Department:         getStr("department_name"),
		Domain:             getStr("ad_domain"),
		Status:             getStr("status"),
		Broken:             getBool("is_broken"),
		HasImages:          getBool("has_images"),
	}

	// log.HTTPDebug(req, fmt.Sprintf("Filter options: %+v", filterOptions))

	db := config.GetDatabaseConn()
	if db == nil {
		log.HTTPWarning(req, "No database connection available for GetInventoryTableData")
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}
	repo := database.NewRepo(db)

	inventoryTableData, err := repo.GetInventoryTableData(ctx, filterOptions)
	if err != nil {
		if err != sql.ErrNoRows {
			log.HTTPWarning(req, "Query error in GetInventoryTableData: "+err.Error())
			middleware.WriteJsonError(w, http.StatusInternalServerError)
			return
		}
	}
	if requestQueries.Get("csv") == "true" {
		log.HTTPDebug(req, "CSV file requested in GetInventoryTableData")
		csvData, err := database.ConvertInventoryTableDataToCSV(ctx, inventoryTableData)
		if err != nil {
			log.HTTPWarning(req, "Error converting inventory table data to CSV in GetInventoryTableData: "+err.Error())
			middleware.WriteJsonError(w, http.StatusInternalServerError)
			return
		}
		if csvData == "" {
			log.HTTPWarning(req, "No CSV data generated in GetInventoryTableData")
			middleware.WriteJsonError(w, http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "text/csv; charset=utf-8")
		w.Header().Set("Content-Disposition", "attachment; filename=\"inventory_table_data-"+time.Now().Format("01-02-2006-150405")+".csv\"")
		if _, err = w.Write([]byte(csvData)); err != nil {
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

	clientConfigMap := map[string]string{
		"UIT_CLIENT_DB_USER":   clientConfig.UIT_CLIENT_DB_USER,
		"UIT_CLIENT_DB_PASSWD": clientConfig.UIT_CLIENT_DB_PASSWD,
		"UIT_CLIENT_DB_NAME":   clientConfig.UIT_CLIENT_DB_NAME,
		"UIT_CLIENT_DB_HOST":   clientConfig.UIT_CLIENT_DB_HOST,
		"UIT_CLIENT_DB_PORT":   clientConfig.UIT_CLIENT_DB_PORT,
		"UIT_CLIENT_NTP_HOST":  clientConfig.UIT_CLIENT_NTP_HOST,
		"UIT_CLIENT_PING_HOST": clientConfig.UIT_CLIENT_PING_HOST,
		"UIT_SERVER_HOSTNAME":  clientConfig.UIT_SERVER_HOSTNAME,
		"UIT_WEB_HTTP_HOST":    clientConfig.UIT_WEB_HTTP_HOST,
		"UIT_WEB_HTTP_PORT":    clientConfig.UIT_WEB_HTTP_PORT,
		"UIT_WEB_HTTPS_HOST":   clientConfig.UIT_WEB_HTTPS_HOST,
		"UIT_WEB_HTTPS_PORT":   clientConfig.UIT_WEB_HTTPS_PORT,
	}

	var response string
	for k, v := range clientConfigMap {
		if v == "" {
			log.HTTPWarning(req, "Client config value for '"+k+"' is empty: using default empty string")
		}
		response += fmt.Sprintf("%s=%s\n", k, v)
	}

	middleware.WriteJson(w, http.StatusOK, response)
}

func GetManufacturersAndModels(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	log := middleware.GetLoggerFromContext(ctx)

	db := config.GetDatabaseConn()
	if db == nil {
		log.HTTPWarning(req, "No database connection available for GetManufacturersAndModels")
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}
	repo := database.NewRepo(db)

	manufacturersAndModels, err := repo.GetManufacturersAndModels(ctx)
	if err != nil {
		if err != sql.ErrNoRows {
			log.HTTPWarning(req, "Query error in GetManufacturersAndModels: "+err.Error())
			middleware.WriteJsonError(w, http.StatusInternalServerError)
			return
		}
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
	tagnumber, err := ConvertTagnumber(urlQueries.Get("tagnumber"))
	if err != nil {
		log.HTTPWarning(req, "Invalid tagnumber provided in GetClientBatteryHealth: "+err.Error())
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}
	db := config.GetDatabaseConn()
	if db == nil {
		log.HTTPWarning(req, "No database connection available in GetClientBatteryHealth")
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}
	repo := database.NewRepo(db)

	batteryHealthData, err := repo.GetClientBatteryHealth(ctx, tagnumber)
	if err != nil {
		if err != sql.ErrNoRows {
			log.HTTPWarning(req, "Query error in GetClientBatteryHealth: "+err.Error())
			middleware.WriteJsonError(w, http.StatusInternalServerError)
			return
		}
	}

	middleware.WriteJson(w, http.StatusOK, batteryHealthData)
}

func GetDomains(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	log := middleware.GetLoggerFromContext(ctx)
	db := config.GetDatabaseConn()
	if db == nil {
		log.HTTPWarning(req, "No database connection available in GetDomains")
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}
	repo := database.NewRepo(db)

	domains, err := repo.GetDomains(ctx)
	if err != nil {
		if err != sql.ErrNoRows {
			log.HTTPWarning(req, "Query error in GetDomains: "+err.Error())
			middleware.WriteJsonError(w, http.StatusInternalServerError)
			return
		}
	}
	middleware.WriteJson(w, http.StatusOK, domains)
}

func GetDepartments(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	log := middleware.GetLoggerFromContext(ctx)
	db := config.GetDatabaseConn()
	if db == nil {
		log.HTTPWarning(req, "No database connection available in GetDepartments")
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}
	repo := database.NewRepo(db)

	departments, err := repo.GetDepartments(ctx)
	if err != nil {
		if err != sql.ErrNoRows {
			log.HTTPWarning(req, "Query error in GetDepartments: "+err.Error())
			middleware.WriteJsonError(w, http.StatusInternalServerError)
			return
		}
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
