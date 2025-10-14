package endpoints

import (
	"context"
	"database/sql"
	"image/jpeg"
	"log"
	"net/http"
	"os"
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

func GetClientImages(w http.ResponseWriter, r *http.Request) {
	requestInfo, err := GetRequestInfo(r)
	if err != nil {
		log.Println("Cannot get request info error: " + err.Error())
		middleware.WriteJsonError(w, http.StatusInternalServerError, "Internal server error")
		return
	}
	// ctx := requestInfo.Ctx
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
	// repo := database.NewRepo(db)
	// ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	// defer cancel()
	// images, err := repo.GetClientImages(ctx, tagnumber)
	// if err != nil {
	// 	if err != sql.ErrNoRows {
	// 		log.Info("Client images query error: " + requestIP + " (" + requestURL + "): " + err.Error())
	// 		middleware.WriteJsonError(w, http.StatusInternalServerError, "Internal server error")
	// 		return
	// 	}
	// }

	image, err := os.Open("./inventory-images/625830/2025-08-11-163114-image-689a61236e8844.70187376.jpeg")
	if err != nil {
		log.Info("Client image open error: " + requestIP + " (" + requestURL + "): " + err.Error())
		middleware.WriteJsonError(w, http.StatusInternalServerError, "Internal server error")
		return
	}
	defer image.Close()

	imageStat, err := image.Stat()
	if err != nil {
		log.Info("Client image stat error: " + requestIP + " (" + requestURL + "): " + err.Error())
		middleware.WriteJsonError(w, http.StatusInternalServerError, "Internal server error")
		return
	}

	imageConfig, err := jpeg.DecodeConfig(image)
	if err != nil {
		log.Info("Client image decode config error: " + requestIP + " (" + requestURL + "): " + err.Error())
		middleware.WriteJsonError(w, http.StatusInternalServerError, "Internal server error")
		return
	}
	log.Info("Serving image " + imageStat.Name() + " (" + strconv.Itoa(imageConfig.Width) + "x" + strconv.Itoa(imageConfig.Height) + ") to " + requestIP + " (" + requestURL + ")")
	w.Header().Set("Content-Type", "image/jpeg")
	http.ServeContent(w, r, imageStat.Name(), imageStat.ModTime(), image)
}
