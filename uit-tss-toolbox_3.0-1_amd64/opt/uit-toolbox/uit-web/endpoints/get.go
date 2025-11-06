package endpoints

import (
	"context"
	"database/sql"
	"fmt"
	"image"
	"log"
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
func GetServerTime(w http.ResponseWriter, r *http.Request) {
	_, err := GetRequestInfo(r)
	if err != nil {
		log.Println("Cannot get request info error: " + err.Error())
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}

	curTime := time.Now().Format("2006-01-02 15:04:05.000")

	middleware.WriteJson(w, http.StatusOK, ServerTime{Time: curTime})
}

func GetClientLookup(w http.ResponseWriter, r *http.Request) {
	requestInfo, err := GetRequestInfo(r)
	if err != nil {
		log.Println("Cannot get request info error: " + err.Error())
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}
	ctx := requestInfo.Ctx
	log := requestInfo.Log
	requestIP := requestInfo.IP
	requestURL := requestInfo.URL

	// No consequence for missing tag, acceptable if lookup by serial
	tagnumber, _ := ConvertRequestTagnumber(r)

	systemSerial := strings.TrimSpace(r.URL.Query().Get("system_serial"))
	if tagnumber == 0 && systemSerial == "" {
		log.Warning("No tagnumber or system_serial provided in request from: " + requestIP.String() + " (" + requestURL + ")")
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
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	var hardwareData *database.ClientLookup
	if tagnumber != 0 {
		hardwareData, err = repo.ClientLookupByTag(ctx, tagnumber)
	} else if systemSerial != "" {
		hardwareData, err = repo.ClientLookupBySerial(ctx, systemSerial)
	}
	if err != nil {
		if err != sql.ErrNoRows {
			log.Info("Client lookup query error: " + requestIP.String() + " (" + requestURL + "): " + err.Error())
			middleware.WriteJsonError(w, http.StatusInternalServerError)
			return
		}
	}
	middleware.WriteJson(w, http.StatusOK, hardwareData)
}

func GetAllTags(w http.ResponseWriter, r *http.Request) {
	requestInfo, err := GetRequestInfo(r)
	if err != nil {
		log.Println("Cannot get request info error: " + err.Error())
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}
	ctx := requestInfo.Ctx
	log := requestInfo.Log
	requestIP := requestInfo.IP
	requestURL := requestInfo.URL

	db := config.GetDatabaseConn()
	if db == nil {
		log.Warning("no database connection available")
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}
	repo := database.NewRepo(db)
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	allTags, err := repo.GetAllTags(ctx)
	if err != nil {
		if err != sql.ErrNoRows {
			log.Info("All tags query error: " + requestIP.String() + " (" + requestURL + "): " + err.Error())
			middleware.WriteJsonError(w, http.StatusInternalServerError)
			return
		}
	}
	middleware.WriteJson(w, http.StatusOK, allTags)
}

func GetHardwareIdentifiers(w http.ResponseWriter, r *http.Request) {
	requestInfo, err := GetRequestInfo(r)
	if err != nil {
		log.Println("Cannot get request info error: " + err.Error())
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}
	ctx := requestInfo.Ctx
	log := requestInfo.Log
	requestIP := requestInfo.IP
	requestURL := requestInfo.URL

	tagnumber, ok := ConvertRequestTagnumber(r)
	if tagnumber == 0 || !ok {
		log.Warning("No or invalid tagnumber provided in request from: " + requestIP.String() + " (" + requestURL + ")")
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
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	hardwareData, err := repo.GetHardwareIdentifiers(ctx, tagnumber)
	if err != nil {
		if err != sql.ErrNoRows {
			log.Info("Hardware ID query error: " + requestIP.String() + " (" + requestURL + "): " + err.Error())
			middleware.WriteJsonError(w, http.StatusInternalServerError)
			return
		}
	}
	middleware.WriteJson(w, http.StatusOK, hardwareData)
}

func GetBiosData(w http.ResponseWriter, r *http.Request) {
	requestInfo, err := GetRequestInfo(r)
	if err != nil {
		log.Println("Cannot get request info error: " + err.Error())
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}
	ctx := requestInfo.Ctx
	log := requestInfo.Log
	requestIP := requestInfo.IP
	requestURL := requestInfo.URL

	tagnumber, ok := ConvertRequestTagnumber(r)
	if tagnumber == 0 || !ok {
		log.Warning("No or invalid tagnumber provided in request from: " + requestIP.String() + " (" + requestURL + ")")
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
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	biosData, err := repo.GetBiosData(ctx, tagnumber)
	if err != nil {
		if err != sql.ErrNoRows {
			log.Info("Bios data query error: " + requestIP.String() + " (" + requestURL + "): " + err.Error())
			middleware.WriteJsonError(w, http.StatusInternalServerError)
			return
		}
	}

	middleware.WriteJson(w, http.StatusOK, biosData)
}

func GetOSData(w http.ResponseWriter, r *http.Request) {
	requestInfo, err := GetRequestInfo(r)
	if err != nil {
		log.Println("Cannot get request info error: " + err.Error())
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}
	ctx := requestInfo.Ctx
	log := requestInfo.Log
	requestIP := requestInfo.IP
	requestURL := requestInfo.URL

	tagnumber, ok := ConvertRequestTagnumber(r)
	if tagnumber == 0 || !ok {
		log.Warning("No or invalid tagnumber provided in request from: " + requestIP.String() + " (" + requestURL + ")")
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
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	osData, err := repo.GetOsData(ctx, tagnumber)
	if err != nil {
		if err != sql.ErrNoRows {
			log.Info("OS data query error: " + requestIP.String() + " (" + requestURL + "): " + err.Error())
			middleware.WriteJsonError(w, http.StatusInternalServerError)
			return
		}
	}

	middleware.WriteJson(w, http.StatusOK, osData)
}

func GetClientQueuedJobs(w http.ResponseWriter, r *http.Request) {
	requestInfo, err := GetRequestInfo(r)
	if err != nil {
		log.Println("Cannot get request info error: " + err.Error())
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}
	ctx := requestInfo.Ctx
	log := requestInfo.Log
	requestIP := requestInfo.IP
	requestURL := requestInfo.URL

	tagnumber, ok := ConvertRequestTagnumber(r)
	if tagnumber == 0 || !ok {
		log.Warning("No or invalid tagnumber provided in request from: " + requestIP.String() + " (" + requestURL + ")")
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
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	activeJobs, err := repo.GetActiveJobs(ctx, tagnumber)
	if err != nil {
		if err != sql.ErrNoRows {
			log.Info("Queued client jobs query error: " + requestIP.String() + " (" + requestURL + "): " + err.Error())
			middleware.WriteJsonError(w, http.StatusInternalServerError)
			return
		}
	}

	middleware.WriteJson(w, http.StatusOK, activeJobs)
}

func GetClientAvailableJobs(w http.ResponseWriter, r *http.Request) {
	requestInfo, err := GetRequestInfo(r)
	if err != nil {
		log.Println("Cannot get request info error: " + err.Error())
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}
	ctx := requestInfo.Ctx
	log := requestInfo.Log
	requestIP := requestInfo.IP
	requestURL := requestInfo.URL

	tagnumber, ok := ConvertRequestTagnumber(r)
	if tagnumber == 0 || !ok {
		log.Warning("No or invalid tagnumber provided in request from: " + requestIP.String() + " (" + requestURL + ")")
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
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	availableJobs, err := repo.GetAvailableJobs(ctx, tagnumber)
	if err != nil {
		if err != sql.ErrNoRows {
			log.Info("Available jobs query error: " + requestIP.String() + " (" + requestURL + "): " + err.Error())
			middleware.WriteJsonError(w, http.StatusInternalServerError)
			return
		}
	}

	middleware.WriteJson(w, http.StatusOK, availableJobs)
}

func GetNotes(w http.ResponseWriter, r *http.Request) {
	requestInfo, err := GetRequestInfo(r)
	if err != nil {
		log.Println("Cannot get request info error: " + err.Error())
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}
	ctx := requestInfo.Ctx
	log := requestInfo.Log
	requestIP := requestInfo.IP
	requestURL := requestInfo.URL

	db := config.GetDatabaseConn()
	if db == nil {
		log.Warning("no database connection available")
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}
	repo := database.NewRepo(db)
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	noteType := strings.TrimSpace(r.URL.Query().Get("note_type"))
	if noteType == "" {
		log.Info("No note_type provided, defaulting to 'general'")
		noteType = "general"
	}

	notesData, err := repo.GetNotes(ctx, noteType)
	if err != nil {
		if err != sql.ErrNoRows {
			log.Info("Get notes query error: " + requestIP.String() + " (" + requestURL + "): " + err.Error())
			middleware.WriteJsonError(w, http.StatusInternalServerError)
			return
		}
	}
	middleware.WriteJson(w, http.StatusOK, notesData)
}

func GetLocationFormData(w http.ResponseWriter, r *http.Request) {
	requestInfo, err := GetRequestInfo(r)
	if err != nil {
		log.Println("Cannot get request info error: " + err.Error())
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}
	ctx := requestInfo.Ctx
	log := requestInfo.Log
	requestIP := requestInfo.IP
	requestURL := requestInfo.URL
	tagnumber, ok := ConvertRequestTagnumber(r)
	if tagnumber == 0 || !ok {
		log.Warning("No or invalid tagnumber provided in request from: " + requestIP.String() + " (" + requestURL + ")")
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
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	locationData, err := repo.GetLocationFormData(ctx, tagnumber)
	if err != nil {
		if err != sql.ErrNoRows {
			log.Info("Location form data query error: " + requestIP.String() + " (" + requestURL + "): " + err.Error())
			middleware.WriteJsonError(w, http.StatusInternalServerError)
			return
		}
	}
	middleware.WriteJson(w, http.StatusOK, locationData)
}

func GetClientImagesManifest(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	log, ok, err := middleware.GetLoggerFromRequestContext(req)
	if err != nil || !ok {
		fmt.Println("Cannot get logger for GetClientImagesManifest from context: " + err.Error())
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}
	requestIP, ok := middleware.GetRequestIPFromRequestContext(req)
	if !ok {
		log.Warning("Cannot get request IP for GetClientImagesManifest from context")
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}
	requestURL, ok := middleware.GetRequestPathFromRequestContext(req)
	if !ok {
		log.Warning("Cannot get request URL for GetClientImagesManifest from context")
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}
	requestQueries, ok := middleware.GetRequestQueryFromRequestContext(req)
	if !ok {
		log.Warning("Cannot get request queries for GetClientImagesManifest from context")
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}
	tagnumber, err := ConvertTagnumber(requestQueries.Get("tagnumber"))
	if err != nil {
		log.HTTPWarning(req, "No or invalid tagnumber provided in request")
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
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	imageUUIDs, err := repo.GetClientImageUUIDs(ctx, tagnumber)
	if err != nil && err != sql.ErrNoRows {
		log.Info("Client images query error: " + requestIP.String() + " (" + requestURL + "): " + err.Error())
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}

	var imageManifests []database.ImageManifest
	for _, imageUUID := range imageUUIDs {
		var imageData database.ImageManifest
		time, tag, filepath, _, hidden, primaryImage, note, err := repo.GetClientImageManifestByUUID(ctx, imageUUID)
		if err != nil {
			if err == sql.ErrNoRows {
				log.Info("Image not found in database: " + requestIP.String() + " (" + requestURL + "): " + err.Error())
			} else {
				log.Info("Client image query error (manifest): " + requestIP.String() + " (" + requestURL + "): " + err.Error())
				middleware.WriteJsonError(w, http.StatusInternalServerError)
				return
			}
		}

		if hidden != nil && *hidden {
			continue
		}

		if filepath == nil || strings.TrimSpace(*filepath) == "" {
			log.Info("Client image filepath is empty: " + requestIP.String() + " (" + requestURL + ")")
			middleware.WriteJsonError(w, http.StatusInternalServerError)
			return
		}

		img, err := os.Open(*filepath)
		if err != nil {
			log.Info("Client image open error (manifest): " + requestIP.String() + " (" + requestURL + "): " + err.Error())
			middleware.WriteJsonError(w, http.StatusInternalServerError)
			return
		}
		defer img.Close()

		imageStat, err := img.Stat()
		if err != nil {
			log.Info("Client image stat error: " + requestIP.String() + " (" + requestURL + "): " + err.Error())
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
				log.Info("Client image decode error: " + requestIP.String() + " (" + requestURL + "): " + err.Error())
				middleware.WriteJsonError(w, http.StatusInternalServerError)
				return
			}
			if imageType != "jpeg" && imageType != "png" {
				_ = img.Close()
				log.Info("Client image has invalid type: " + requestIP.String() + " (" + requestURL + "): " + imageType)
				middleware.WriteJsonError(w, http.StatusInternalServerError)
				return
			}
			if (imageType == "jpeg" && fileExtension != ".jpg" && fileExtension != ".jpeg") ||
				(imageType == "png" && fileExtension != ".png") {
				_ = img.Close()
				log.Info("Client image file extension does not match image type: " + requestIP.String() + " (" + requestURL + "): " + imageStat.Name())
				middleware.WriteJsonError(w, http.StatusInternalServerError)
				return
			}
			imageData.Width = &imageConfig.Width
			imageData.Height = &imageConfig.Height
			if imageData.Width == nil || imageData.Height == nil || *imageData.Width <= 0 || *imageData.Height <= 0 {
				log.Info("Client image has invalid dimensions: " + requestIP.String() + " (" + requestURL + ")")
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
			log.Warning("Client image has no or invalid tag in database: " + requestIP.String() + " (" + requestURL + ")")
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
			log.Info("Client image URL join error: " + requestIP.String() + " (" + requestURL + "): " + err.Error())
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
	log, ok, err := middleware.GetLoggerFromContext(ctx)
	if err != nil || !ok {
		fmt.Println("Cannot get logger for GetImage from context: " + err.Error())
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}
	requestedQueries, ok := middleware.GetRequestQueryFromContext(ctx)
	if !ok {
		log.HTTPWarning(req, "Cannot get request queries for GetImage from context")
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
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

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
func GetJobQueueOverview(w http.ResponseWriter, r *http.Request) {
	requestInfo, err := GetRequestInfo(r)
	if err != nil {
		log.Println("Cannot get request info error: " + err.Error())
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}
	ctx := requestInfo.Ctx
	log := requestInfo.Log
	requestIP := requestInfo.IP
	requestURL := requestInfo.URL

	db := config.GetDatabaseConn()
	if db == nil {
		log.Warning("no database connection available")
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}
	repo := database.NewRepo(db)
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	jobQueueOverview, err := repo.GetJobQueueOverview(ctx)
	if err != nil {
		if err != sql.ErrNoRows {
			log.Info("Job queue overview query error: " + requestIP.String() + " (" + requestURL + "): " + err.Error())
			middleware.WriteJsonError(w, http.StatusInternalServerError)
			return
		}
	}

	middleware.WriteJson(w, http.StatusOK, jobQueueOverview)
}

func GetDashboardInventorySummary(w http.ResponseWriter, r *http.Request) {
	requestInfo, err := GetRequestInfo(r)
	if err != nil {
		log.Println("Cannot get request info error: " + err.Error())
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}
	ctx := requestInfo.Ctx
	log := requestInfo.Log
	requestIP := requestInfo.IP
	requestURL := requestInfo.URL

	db := config.GetDatabaseConn()
	if db == nil {
		log.Warning("no database connection available")
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}
	repo := database.NewRepo(db)
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	inventorySummary, err := repo.GetDashboardInventorySummary(ctx)
	if err != nil {
		if err != sql.ErrNoRows {
			log.Info("Inventory summary query error: " + requestIP.String() + " (" + requestURL + "): " + err.Error())
			middleware.WriteJsonError(w, http.StatusInternalServerError)
			return
		}
	}
	middleware.WriteJson(w, http.StatusOK, inventorySummary)
}

func GetInventoryTableData(w http.ResponseWriter, r *http.Request) {
	requestInfo, err := GetRequestInfo(r)
	if err != nil {
		log.Println("Cannot get request info error: " + err.Error())
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}
	ctx := requestInfo.Ctx
	log := requestInfo.Log
	requestIP := requestInfo.IP
	requestURL := requestInfo.URL
	db := config.GetDatabaseConn()
	if db == nil {
		log.Warning("no database connection available")
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}
	repo := database.NewRepo(db)
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	getStr := func(key string) *string {
		s := strings.TrimSpace(r.URL.Query().Get(key))
		if s == "" {
			return nil
		}
		return &s
	}
	getInt64 := func(key string) *int64 {
		raw := strings.TrimSpace(r.URL.Query().Get(key))
		if raw == "" {
			return nil
		}
		v, err := strconv.ParseInt(raw, 10, 64)
		if err != nil {
			log.Info("Invalid " + key + " filter provided: " + requestIP.String() + " (" + requestURL + "): " + err.Error())
			return nil
		}
		return &v
	}
	getBool := func(key string) *bool {
		raw := strings.TrimSpace(r.URL.Query().Get(key))
		if raw == "" {
			return nil
		}
		v, err := strconv.ParseBool(raw)
		if err != nil {
			log.Info("Invalid " + key + " filter provided: " + requestIP.String() + " (" + requestURL + "): " + err.Error())
			return nil
		}
		return &v
	}

	filterOptions := &database.InventoryFilterOptions{
		Tagnumber:          getInt64("tagnumber"),
		SystemSerial:       getStr("system_serial"),
		Location:           getStr("location"),
		SystemManufacturer: getStr("manufacturer"),
		SystemModel:        getStr("model"),
		Department:         getStr("department"),
		Domain:             getStr("domain"),
		Status:             getStr("status"),
		Broken:             getBool("broken"),
		HasImages:          getBool("has_images"),
	}

	inventoryTableData, err := repo.GetInventoryTableData(ctx, filterOptions)
	if err != nil {
		if err != sql.ErrNoRows {
			log.Info("Inventory table data query error: " + requestIP.String() + " (" + requestURL + "): " + err.Error())
			middleware.WriteJsonError(w, http.StatusInternalServerError)
			return
		}
	}
	middleware.WriteJson(w, http.StatusOK, inventoryTableData)
}

func GetClientConfig(w http.ResponseWriter, r *http.Request) {
	requestInfo, err := GetRequestInfo(r)
	if err != nil {
		log.Println("Cannot get request info error: " + err.Error())
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}
	// ctx := requestInfo.Ctx
	log := requestInfo.Log
	requestIP := requestInfo.IP
	requestURL := requestInfo.URL

	clientConfig, err := config.GetClientConfig()
	if err != nil {
		log.Error("Error getting client config: " + requestIP.String() + " (" + requestURL + "): " + err.Error())
		middleware.WriteJsonError(w, http.StatusInternalServerError)
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
			log.Warning("Client config value for " + k + " is empty: " + requestIP.String() + " (" + requestURL + ")")
		}
		response += fmt.Sprintf("%s=%s\n", k, v)
	}

	middleware.WriteJson(w, http.StatusOK, response)
}
