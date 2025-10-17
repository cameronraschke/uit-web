package endpoints

import (
	"context"
	"database/sql"
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
		middleware.WriteJsonError(w, http.StatusInternalServerError, "Internal server error")
		return
	}

	curTime := time.Now().Format("2006-01-02 15:04:05.000")

	middleware.WriteJson(w, http.StatusOK, ServerTime{Time: curTime})
}

func GetClientLookup(w http.ResponseWriter, r *http.Request) {
	requestInfo, err := GetRequestInfo(r)
	if err != nil {
		log.Println("Cannot get request info error: " + err.Error())
		middleware.WriteJsonError(w, http.StatusInternalServerError, "Internal server error")
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
		log.Warning("No tagnumber or system_serial provided in request from: " + requestIP + " (" + requestURL + ")")
		middleware.WriteJsonError(w, http.StatusBadRequest, "Bad request")
		return
	}

	db := config.GetDatabaseConn()
	if db == nil {
		log.Warning("no database connection available")
		middleware.WriteJsonError(w, http.StatusInternalServerError, "Internal server error")
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
			log.Info("Client lookup query error: " + requestIP + " (" + requestURL + "): " + err.Error())
			middleware.WriteJsonError(w, http.StatusInternalServerError, "Internal server error")
			return
		}
	}
	middleware.WriteJson(w, http.StatusOK, hardwareData)
}

func GetAllTags(w http.ResponseWriter, r *http.Request) {
	requestInfo, err := GetRequestInfo(r)
	if err != nil {
		log.Println("Cannot get request info error: " + err.Error())
		middleware.WriteJsonError(w, http.StatusInternalServerError, "Internal server error")
		return
	}
	ctx := requestInfo.Ctx
	log := requestInfo.Log
	requestIP := requestInfo.IP
	requestURL := requestInfo.URL

	db := config.GetDatabaseConn()
	if db == nil {
		log.Warning("no database connection available")
		middleware.WriteJsonError(w, http.StatusInternalServerError, "Internal server error")
		return
	}
	repo := database.NewRepo(db)
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	allTags, err := repo.GetAllTags(ctx)
	if err != nil {
		if err != sql.ErrNoRows {
			log.Info("All tags query error: " + requestIP + " (" + requestURL + "): " + err.Error())
			middleware.WriteJsonError(w, http.StatusInternalServerError, "Internal server error")
			return
		}
	}
	middleware.WriteJson(w, http.StatusOK, allTags)
}

func GetHardwareIdentifiers(w http.ResponseWriter, r *http.Request) {
	requestInfo, err := GetRequestInfo(r)
	if err != nil {
		log.Println("Cannot get request info error: " + err.Error())
		middleware.WriteJsonError(w, http.StatusInternalServerError, "Internal server error")
		return
	}
	ctx := requestInfo.Ctx
	log := requestInfo.Log
	requestIP := requestInfo.IP
	requestURL := requestInfo.URL

	tagnumber, ok := ConvertRequestTagnumber(r)
	if tagnumber == 0 || !ok {
		log.Warning("No or invalid tagnumber provided in request from: " + requestIP + " (" + requestURL + ")")
		middleware.WriteJsonError(w, http.StatusBadRequest, "Bad request")
		return
	}

	db := config.GetDatabaseConn()
	if db == nil {
		log.Warning("no database connection available")
		middleware.WriteJsonError(w, http.StatusInternalServerError, "Internal server error")
		return
	}
	repo := database.NewRepo(db)
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	hardwareData, err := repo.GetHardwareIdentifiers(ctx, tagnumber)
	if err != nil {
		if err != sql.ErrNoRows {
			log.Info("Hardware ID query error: " + requestIP + " (" + requestURL + "): " + err.Error())
			middleware.WriteJsonError(w, http.StatusInternalServerError, "Internal server error")
			return
		}
	}
	middleware.WriteJson(w, http.StatusOK, hardwareData)
}

func GetBiosData(w http.ResponseWriter, r *http.Request) {
	requestInfo, err := GetRequestInfo(r)
	if err != nil {
		log.Println("Cannot get request info error: " + err.Error())
		middleware.WriteJsonError(w, http.StatusInternalServerError, "Internal server error")
		return
	}
	ctx := requestInfo.Ctx
	log := requestInfo.Log
	requestIP := requestInfo.IP
	requestURL := requestInfo.URL

	tagnumber, ok := ConvertRequestTagnumber(r)
	if tagnumber == 0 || !ok {
		log.Warning("No or invalid tagnumber provided in request from: " + requestIP + " (" + requestURL + ")")
		middleware.WriteJsonError(w, http.StatusBadRequest, "Bad request")
		return
	}

	db := config.GetDatabaseConn()
	if db == nil {
		log.Warning("no database connection available")
		middleware.WriteJsonError(w, http.StatusInternalServerError, "Internal server error")
		return
	}
	repo := database.NewRepo(db)
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	biosData, err := repo.GetBiosData(ctx, tagnumber)
	if err != nil {
		if err != sql.ErrNoRows {
			log.Info("Bios data query error: " + requestIP + " (" + requestURL + "): " + err.Error())
			middleware.WriteJsonError(w, http.StatusInternalServerError, "Internal server error")
			return
		}
	}

	middleware.WriteJson(w, http.StatusOK, biosData)
}

func GetOSData(w http.ResponseWriter, r *http.Request) {
	requestInfo, err := GetRequestInfo(r)
	if err != nil {
		log.Println("Cannot get request info error: " + err.Error())
		middleware.WriteJsonError(w, http.StatusInternalServerError, "Internal server error")
		return
	}
	ctx := requestInfo.Ctx
	log := requestInfo.Log
	requestIP := requestInfo.IP
	requestURL := requestInfo.URL

	tagnumber, ok := ConvertRequestTagnumber(r)
	if tagnumber == 0 || !ok {
		log.Warning("No or invalid tagnumber provided in request from: " + requestIP + " (" + requestURL + ")")
		middleware.WriteJsonError(w, http.StatusBadRequest, "Bad request")
		return
	}

	db := config.GetDatabaseConn()
	if db == nil {
		log.Warning("no database connection available")
		middleware.WriteJsonError(w, http.StatusInternalServerError, "Internal server error")
		return
	}
	repo := database.NewRepo(db)
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	osData, err := repo.GetOsData(ctx, tagnumber)
	if err != nil {
		if err != sql.ErrNoRows {
			log.Info("OS data query error: " + requestIP + " (" + requestURL + "): " + err.Error())
			middleware.WriteJsonError(w, http.StatusInternalServerError, "Internal server error")
			return
		}
	}

	middleware.WriteJson(w, http.StatusOK, osData)
}

func GetClientQueuedJobs(w http.ResponseWriter, r *http.Request) {
	requestInfo, err := GetRequestInfo(r)
	if err != nil {
		log.Println("Cannot get request info error: " + err.Error())
		middleware.WriteJsonError(w, http.StatusInternalServerError, "Internal server error")
		return
	}
	ctx := requestInfo.Ctx
	log := requestInfo.Log
	requestIP := requestInfo.IP
	requestURL := requestInfo.URL

	tagnumber, ok := ConvertRequestTagnumber(r)
	if tagnumber == 0 || !ok {
		log.Warning("No or invalid tagnumber provided in request from: " + requestIP + " (" + requestURL + ")")
		middleware.WriteJsonError(w, http.StatusBadRequest, "Bad request")
		return
	}

	db := config.GetDatabaseConn()
	if db == nil {
		log.Warning("no database connection available")
		middleware.WriteJsonError(w, http.StatusInternalServerError, "Internal server error")
		return
	}
	repo := database.NewRepo(db)
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	activeJobs, err := repo.GetActiveJobs(ctx, tagnumber)
	if err != nil {
		if err != sql.ErrNoRows {
			log.Info("Queued client jobs query error: " + requestIP + " (" + requestURL + "): " + err.Error())
			middleware.WriteJsonError(w, http.StatusInternalServerError, "Internal server error")
			return
		}
	}

	middleware.WriteJson(w, http.StatusOK, activeJobs)
}

func GetClientAvailableJobs(w http.ResponseWriter, r *http.Request) {
	requestInfo, err := GetRequestInfo(r)
	if err != nil {
		log.Println("Cannot get request info error: " + err.Error())
		middleware.WriteJsonError(w, http.StatusInternalServerError, "Internal server error")
		return
	}
	ctx := requestInfo.Ctx
	log := requestInfo.Log
	requestIP := requestInfo.IP
	requestURL := requestInfo.URL

	tagnumber, ok := ConvertRequestTagnumber(r)
	if tagnumber == 0 || !ok {
		log.Warning("No or invalid tagnumber provided in request from: " + requestIP + " (" + requestURL + ")")
		middleware.WriteJsonError(w, http.StatusBadRequest, "Bad request")
		return
	}

	db := config.GetDatabaseConn()
	if db == nil {
		log.Warning("no database connection available")
		middleware.WriteJsonError(w, http.StatusInternalServerError, "Internal server error")
		return
	}
	repo := database.NewRepo(db)
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	availableJobs, err := repo.GetAvailableJobs(ctx, tagnumber)
	if err != nil {
		if err != sql.ErrNoRows {
			log.Info("Available jobs query error: " + requestIP + " (" + requestURL + "): " + err.Error())
			middleware.WriteJsonError(w, http.StatusInternalServerError, "Internal server error")
			return
		}
	}

	middleware.WriteJson(w, http.StatusOK, availableJobs)
}

func GetNotes(w http.ResponseWriter, r *http.Request) {
	requestInfo, err := GetRequestInfo(r)
	if err != nil {
		log.Println("Cannot get request info error: " + err.Error())
		middleware.WriteJsonError(w, http.StatusInternalServerError, "Internal server error")
		return
	}
	ctx := requestInfo.Ctx
	log := requestInfo.Log
	requestIP := requestInfo.IP
	requestURL := requestInfo.URL

	db := config.GetDatabaseConn()
	if db == nil {
		log.Warning("no database connection available")
		middleware.WriteJsonError(w, http.StatusInternalServerError, "Internal server error")
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
			log.Info("Get notes query error: " + requestIP + " (" + requestURL + "): " + err.Error())
			middleware.WriteJsonError(w, http.StatusInternalServerError, "Internal server error")
			return
		}
	}
	middleware.WriteJson(w, http.StatusOK, notesData)
}

func GetLocationFormData(w http.ResponseWriter, r *http.Request) {
	requestInfo, err := GetRequestInfo(r)
	if err != nil {
		log.Println("Cannot get request info error: " + err.Error())
		middleware.WriteJsonError(w, http.StatusInternalServerError, "Internal server error")
		return
	}
	ctx := requestInfo.Ctx
	log := requestInfo.Log
	requestIP := requestInfo.IP
	requestURL := requestInfo.URL
	tagnumber, ok := ConvertRequestTagnumber(r)
	if tagnumber == 0 || !ok {
		log.Warning("No or invalid tagnumber provided in request from: " + requestIP + " (" + requestURL + ")")
		middleware.WriteJsonError(w, http.StatusBadRequest, "Bad request")
		return
	}

	db := config.GetDatabaseConn()
	if db == nil {
		log.Warning("no database connection available")
		middleware.WriteJsonError(w, http.StatusInternalServerError, "Internal server error")
		return
	}
	repo := database.NewRepo(db)
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	locationData, err := repo.GetLocationFormData(ctx, tagnumber)
	if err != nil {
		if err != sql.ErrNoRows {
			log.Info("Location form data query error: " + requestIP + " (" + requestURL + "): " + err.Error())
			middleware.WriteJsonError(w, http.StatusInternalServerError, "Internal server error")
			return
		}
	}
	middleware.WriteJson(w, http.StatusOK, locationData)
}

// Overview section
func GetJobQueueOverview(w http.ResponseWriter, r *http.Request) {
	requestInfo, err := GetRequestInfo(r)
	if err != nil {
		log.Println("Cannot get request info error: " + err.Error())
		middleware.WriteJsonError(w, http.StatusInternalServerError, "Internal server error")
		return
	}
	ctx := requestInfo.Ctx
	log := requestInfo.Log
	requestIP := requestInfo.IP
	requestURL := requestInfo.URL

	db := config.GetDatabaseConn()
	if db == nil {
		log.Warning("no database connection available")
		middleware.WriteJsonError(w, http.StatusInternalServerError, "Internal server error")
		return
	}
	repo := database.NewRepo(db)
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	jobQueueOverview, err := repo.GetJobQueueOverview(ctx)
	if err != nil {
		if err != sql.ErrNoRows {
			log.Info("Job queue overview query error: " + requestIP + " (" + requestURL + "): " + err.Error())
			middleware.WriteJsonError(w, http.StatusInternalServerError, "Internal server error")
			return
		}
	}

	middleware.WriteJson(w, http.StatusOK, jobQueueOverview)
}

func GetDashboardInventorySummary(w http.ResponseWriter, r *http.Request) {
	requestInfo, err := GetRequestInfo(r)
	if err != nil {
		log.Println("Cannot get request info error: " + err.Error())
		middleware.WriteJsonError(w, http.StatusInternalServerError, "Internal server error")
		return
	}
	ctx := requestInfo.Ctx
	log := requestInfo.Log
	requestIP := requestInfo.IP
	requestURL := requestInfo.URL

	db := config.GetDatabaseConn()
	if db == nil {
		log.Warning("no database connection available")
		middleware.WriteJsonError(w, http.StatusInternalServerError, "Internal server error")
		return
	}
	repo := database.NewRepo(db)
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	inventorySummary, err := repo.GetDashboardInventorySummary(ctx)
	if err != nil {
		if err != sql.ErrNoRows {
			log.Info("Inventory summary query error: " + requestIP + " (" + requestURL + "): " + err.Error())
			middleware.WriteJsonError(w, http.StatusInternalServerError, "Internal server error")
			return
		}
	}
	middleware.WriteJson(w, http.StatusOK, inventorySummary)
}

type ImageConfig struct {
	Name   string
	URL    string
	Width  int
	Height int
	Size   int64
}

func GetClientImagesManifest(w http.ResponseWriter, r *http.Request) {
	requestInfo, err := GetRequestInfo(r)
	if err != nil {
		log.Println("Cannot get request info error: " + err.Error())
		middleware.WriteJsonError(w, http.StatusInternalServerError, "Internal server error")
		return
	}
	ctx := requestInfo.Ctx
	log := requestInfo.Log
	requestIP := requestInfo.IP
	requestURL := requestInfo.URL
	tagnumber, ok := ConvertRequestTagnumber(r)
	if tagnumber == 0 || !ok {
		log.Warning("No or invalid tagnumber provided in request from: " + requestIP + " (" + requestURL + ")")
		middleware.WriteJsonError(w, http.StatusBadRequest, "Bad request")
		return
	}
	db := config.GetDatabaseConn()
	if db == nil {
		log.Warning("no database connection available")
		middleware.WriteJsonError(w, http.StatusInternalServerError, "Internal server error")
		return
	}
	repo := database.NewRepo(db)
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	images, err := repo.GetClientImagePaths(ctx, tagnumber)
	if err != nil && err != sql.ErrNoRows {
		log.Info("Client images query error: " + requestIP + " (" + requestURL + "): " + err.Error())
		middleware.WriteJsonError(w, http.StatusInternalServerError, "Internal server error")
		return
	}

	var imageList []ImageConfig
	for _, imageFilePath := range images {
		img, err := os.Open(imageFilePath)
		if err != nil {
			log.Info("Client image open error: " + requestIP + " (" + requestURL + "): " + err.Error())
			middleware.WriteJsonError(w, http.StatusInternalServerError, "Internal server error")
			return
		}
		defer img.Close()

		imageStat, err := img.Stat()
		if err != nil {
			log.Info("Client image stat error: " + requestIP + " (" + requestURL + "): " + err.Error())
			middleware.WriteJsonError(w, http.StatusInternalServerError, "Internal server error")
			return
		}

		imageReader := http.MaxBytesReader(w, img, 64<<20)
		decodedImage, imageType, err := image.DecodeConfig(imageReader)
		if err != nil {
			log.Info("Client image decode error: " + requestIP + " (" + requestURL + "): " + err.Error())
			middleware.WriteJsonError(w, http.StatusInternalServerError, "Internal server error")
			return
		}
		if imageType != "jpeg" && imageType != "png" {
			log.Info("Client image has invalid type: " + requestIP + " (" + requestURL + "): " + imageType)
			middleware.WriteJsonError(w, http.StatusInternalServerError, "Internal server error")
			return
		}

		var imageConfig ImageConfig
		imageConfig.Name = imageStat.Name()
		imageConfig.URL, err = url.JoinPath("/api/images/", strconv.Itoa(tagnumber), imageStat.Name())
		if err != nil {
			log.Info("Client image URL join error: " + requestIP + " (" + requestURL + "): " + err.Error())
			middleware.WriteJsonError(w, http.StatusInternalServerError, "Internal server error")
			return
		}
		imageConfig.Width = decodedImage.Width
		imageConfig.Height = decodedImage.Height
		imageConfig.Size = imageStat.Size()

		if imageConfig.Width == 0 || imageConfig.Height == 0 {
			log.Info("Client image has invalid dimensions: " + requestIP + " (" + requestURL + ")")
			middleware.WriteJsonError(w, http.StatusInternalServerError, "Internal server error")
			return
		}
		imageList = append(imageList, imageConfig)
	}
	w.Header().Set("Content-Type", "application/json")
	middleware.WriteJson(w, http.StatusOK, imageList)
}

func GetImage(w http.ResponseWriter, r *http.Request) {
	requestInfo, err := GetRequestInfo(r)
	if err != nil {
		log.Println("Cannot get request info error: " + err.Error())
		middleware.WriteJsonError(w, http.StatusInternalServerError, "Internal server error")
		return
	}
	ctx := requestInfo.Ctx
	log := requestInfo.Log
	requestIP := requestInfo.IP
	requestURL := requestInfo.URL

	requestFilePath := strings.TrimPrefix(r.URL.Path, "/api/images/")
	if requestFilePath == "" {
		log.Warning("No image path provided in request from: " + requestIP + " (" + requestURL + ")")
		middleware.WriteJsonError(w, http.StatusBadRequest, "Bad request")
		return
	}
	db := config.GetDatabaseConn()
	if db == nil {
		log.Warning("no database connection available")
		middleware.WriteJsonError(w, http.StatusInternalServerError, "Internal server error")
		return
	}
	repo := database.NewRepo(db)
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	requestFileName := path.Base(requestFilePath)
	if requestFileName == "" {
		log.Warning("No image file name provided in request from: " + requestIP + " (" + requestURL + ")")
		middleware.WriteJsonError(w, http.StatusBadRequest, "Bad request")
		return
	}
	log.Info("Serving image request for: " + requestFileName + " from " + requestIP + " (" + requestURL + ")")
	imagePath, _, err := repo.GetClientImageFilePathByFileName(ctx, requestFileName)
	if err != nil {
		if err == sql.ErrNoRows {
			log.Info("Image not found: " + requestIP + " (" + requestURL + "): " + err.Error())
			middleware.WriteJsonError(w, http.StatusNotFound, "Image not found")
			return
		}
		log.Info("Client image query error: " + requestIP + " (" + requestURL + "): " + err.Error())
		middleware.WriteJsonError(w, http.StatusInternalServerError, "Internal server error")
		return
	}

	imageFile, err := os.Open(*imagePath)
	if err != nil {
		if os.IsNotExist(err) {
			log.Info("Image not found: " + requestIP + " (" + requestURL + "): " + err.Error())
			middleware.WriteJsonError(w, http.StatusNotFound, "Image not found")
			return
		}
		log.Info("Client image open error: " + requestIP + " (" + requestURL + "): " + err.Error())
		middleware.WriteJsonError(w, http.StatusInternalServerError, "Internal server error")
		return
	}
	defer imageFile.Close()

	http.ServeContent(w, r, imageFile.Name(), time.Time{}, imageFile)
}
