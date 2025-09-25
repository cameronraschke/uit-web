package endpoints

import (
	"context"
	"database/sql"
	"errors"
	"log"
	"net/http"
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
		http.Error(w, middleware.FormatHttpError("Internal server error"), http.StatusInternalServerError)
		return
	}

	curTime := time.Now().Format("2006-01-02 15:04:05.000")

	writeJSON(w, http.StatusOK, ServerTime{Time: curTime})
}

func GetClientLookup(w http.ResponseWriter, r *http.Request) {
	requestInfo, err := GetRequestInfo(r)
	if err != nil {
		log.Println("Cannot get request info error: " + err.Error())
		http.Error(w, middleware.FormatHttpError("Internal server error"), http.StatusInternalServerError)
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
		http.Error(w, middleware.FormatHttpError("Bad request"), http.StatusBadRequest)
		return
	}

	db := config.GetDatabaseConn()
	if db == nil {
		log.Warning("no database connection available")
		http.Error(w, middleware.FormatHttpError("Internal server error"), http.StatusInternalServerError)
		return
	}

	repo := database.NewRepo(db)

	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	var dbResult *database.ClientLookup
	if tagnumber == 0 && strings.TrimSpace(systemSerial) != "" {
		dbResult, err = repo.ClientLookupBySerial(ctx, systemSerial)
	} else if strings.TrimSpace(systemSerial) == "" && tagnumber > 0 {
		dbResult, err = repo.ClientLookupByTag(ctx, tagnumber)
	} else {
		log.Warning("no tagnumber or system_serial provided")
		http.Error(w, middleware.FormatHttpError("Bad request"), http.StatusBadRequest)
		return
	}
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			http.Error(w, middleware.FormatHttpError("Not found"), http.StatusNotFound)
			return
		}
		log.Warning("DB error: " + err.Error())
		http.Error(w, middleware.FormatHttpError("Internal server error"), http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, dbResult)
}

func GetHardwareIdentifiers(w http.ResponseWriter, r *http.Request) {
	requestInfo, err := GetRequestInfo(r)
	if err != nil {
		log.Println("Cannot get request info error: " + err.Error())
		http.Error(w, middleware.FormatHttpError("Internal server error"), http.StatusInternalServerError)
		return
	}
	ctx := requestInfo.Ctx
	log := requestInfo.Log
	requestIP := requestInfo.IP
	requestURL := requestInfo.URL

	tagnumber, ok := ConvertRequestTagnumber(r)
	if tagnumber == 0 || !ok {
		log.Warning("No or invalid tagnumber provided in request from: " + requestIP + " (" + requestURL + ")")
		http.Error(w, middleware.FormatHttpError("Bad request"), http.StatusBadRequest)
		return
	}

	db := config.GetDatabaseConn()
	if db == nil {
		log.Warning("no database connection available")
		http.Error(w, middleware.FormatHttpError("Internal server error"), http.StatusInternalServerError)
		return
	}
	repo := database.NewRepo(db)
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	hardwareData, err := repo.GetHardwareIdentifiers(ctx, tagnumber)
	if err != nil {
		log.Warning("Database lookup failed for: " + requestIP + " (" + requestURL + "): " + err.Error())
		http.Error(w, middleware.FormatHttpError("Internal server error"), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, hardwareData)
}

func GetBiosData(w http.ResponseWriter, r *http.Request) {
	requestInfo, err := GetRequestInfo(r)
	if err != nil {
		log.Println("Cannot get request info error: " + err.Error())
		http.Error(w, middleware.FormatHttpError("Internal server error"), http.StatusInternalServerError)
		return
	}
	ctx := requestInfo.Ctx
	log := requestInfo.Log
	requestIP := requestInfo.IP
	requestURL := requestInfo.URL

	tagnumber, ok := ConvertRequestTagnumber(r)
	if tagnumber == 0 || !ok {
		log.Warning("No or invalid tagnumber provided in request from: " + requestIP + " (" + requestURL + ")")
		http.Error(w, middleware.FormatHttpError("Bad request"), http.StatusBadRequest)
		return
	}

	db := config.GetDatabaseConn()
	if db == nil {
		log.Warning("no database connection available")
		http.Error(w, middleware.FormatHttpError("Internal server error"), http.StatusInternalServerError)
		return
	}
	repo := database.NewRepo(db)
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	biosData, err := repo.GetBiosData(ctx, tagnumber)
	if err != nil {
		log.Warning("Database lookup failed for: " + requestIP + " (" + requestURL + "): " + err.Error())
		http.Error(w, middleware.FormatHttpError("Internal server error"), http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, biosData)
}

func GetOSData(w http.ResponseWriter, r *http.Request) {
	requestInfo, err := GetRequestInfo(r)
	if err != nil {
		log.Println("Cannot get request info error: " + err.Error())
		http.Error(w, middleware.FormatHttpError("Internal server error"), http.StatusInternalServerError)
		return
	}
	ctx := requestInfo.Ctx
	log := requestInfo.Log
	requestIP := requestInfo.IP
	requestURL := requestInfo.URL

	tagnumber, ok := ConvertRequestTagnumber(r)
	if tagnumber == 0 || !ok {
		log.Warning("No or invalid tagnumber provided in request from: " + requestIP + " (" + requestURL + ")")
		http.Error(w, middleware.FormatHttpError("Bad request"), http.StatusBadRequest)
		return
	}

	db := config.GetDatabaseConn()
	if db == nil {
		log.Warning("no database connection available")
		http.Error(w, middleware.FormatHttpError("Internal server error"), http.StatusInternalServerError)
		return
	}
	repo := database.NewRepo(db)
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	osData, err := repo.GetOsData(ctx, tagnumber)
	if err != nil {
		log.Warning("Database lookup failed for: " + requestIP + " (" + requestURL + "): " + err.Error())
		http.Error(w, middleware.FormatHttpError("Internal server error"), http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, osData)
}

func GetClientQueuedJobs(w http.ResponseWriter, r *http.Request) {
	requestInfo, err := GetRequestInfo(r)
	if err != nil {
		log.Println("Cannot get request info error: " + err.Error())
		http.Error(w, middleware.FormatHttpError("Internal server error"), http.StatusInternalServerError)
		return
	}
	ctx := requestInfo.Ctx
	log := requestInfo.Log
	requestIP := requestInfo.IP
	requestURL := requestInfo.URL

	tagnumber, ok := ConvertRequestTagnumber(r)
	if tagnumber == 0 || !ok {
		log.Warning("No or invalid tagnumber provided in request from: " + requestIP + " (" + requestURL + ")")
		http.Error(w, middleware.FormatHttpError("Bad request"), http.StatusBadRequest)
		return
	}

	db := config.GetDatabaseConn()
	if db == nil {
		log.Warning("no database connection available")
		http.Error(w, middleware.FormatHttpError("Internal server error"), http.StatusInternalServerError)
		return
	}
	repo := database.NewRepo(db)
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	activeJobs, err := repo.GetActiveJobs(ctx, tagnumber)
	if err != nil {
		log.Warning("Database lookup failed for: " + requestIP + " (" + requestURL + "): " + err.Error())
		http.Error(w, middleware.FormatHttpError("Internal server error"), http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, activeJobs)
}

func GetClientAvailableJobs(w http.ResponseWriter, r *http.Request) {
	requestInfo, err := GetRequestInfo(r)
	if err != nil {
		log.Println("Cannot get request info error: " + err.Error())
		http.Error(w, middleware.FormatHttpError("Internal server error"), http.StatusInternalServerError)
		return
	}
	ctx := requestInfo.Ctx
	log := requestInfo.Log
	requestIP := requestInfo.IP
	requestURL := requestInfo.URL

	tagnumber, ok := ConvertRequestTagnumber(r)
	if tagnumber == 0 || !ok {
		log.Warning("No or invalid tagnumber provided in request from: " + requestIP + " (" + requestURL + ")")
		http.Error(w, middleware.FormatHttpError("Bad request"), http.StatusBadRequest)
		return
	}

	db := config.GetDatabaseConn()
	if db == nil {
		log.Warning("no database connection available")
		http.Error(w, middleware.FormatHttpError("Internal server error"), http.StatusInternalServerError)
		return
	}
	repo := database.NewRepo(db)
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	availableJobs, err := repo.GetAvailableJobs(ctx, tagnumber)
	if err != nil {
		log.Warning("Database lookup failed for: " + requestIP + " (" + requestURL + "): " + err.Error())
		http.Error(w, middleware.FormatHttpError("Internal server error"), http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, availableJobs)
}

// Overview section
func GetJobQueueOverview(w http.ResponseWriter, r *http.Request) {
	requestInfo, err := GetRequestInfo(r)
	if err != nil {
		log.Println("Cannot get request info error: " + err.Error())
		http.Error(w, middleware.FormatHttpError("Internal server error"), http.StatusInternalServerError)
		return
	}
	ctx := requestInfo.Ctx
	log := requestInfo.Log
	requestIP := requestInfo.IP
	requestURL := requestInfo.URL

	db := config.GetDatabaseConn()
	if db == nil {
		log.Warning("no database connection available")
		http.Error(w, middleware.FormatHttpError("Internal server error"), http.StatusInternalServerError)
		return
	}
	repo := database.NewRepo(db)
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	jobQueueOverview, err := repo.GetJobQueueOverview(ctx)
	if err != nil {
		log.Warning("Database lookup failed for: " + requestIP + " (" + requestURL + "): " + err.Error())
		http.Error(w, middleware.FormatHttpError("Internal server error"), http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, jobQueueOverview)
}
