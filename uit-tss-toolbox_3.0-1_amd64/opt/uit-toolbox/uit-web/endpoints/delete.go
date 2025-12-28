package endpoints

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"
	config "uit-toolbox/config"
	"uit-toolbox/database"
	middleware "uit-toolbox/middleware"
)

func DeleteImage(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	log, ok, err := middleware.GetLoggerFromContext(ctx)
	if !ok || err != nil {
		fmt.Println("Failed to get logger from context for DeleteImage: " + err.Error())
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	requestQueries, ok := middleware.GetRequestQueryFromContext(ctx)
	if !ok {
		log.HTTPWarning(req, "No request query parameters found in context for DeleteImage")
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}

	// Check if required query parameters are set
	if !requestQueries.Has("tagnumber") {
		log.HTTPWarning(req, "No tagnumber query parameter provided for DeleteImage")
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}
	if !requestQueries.Has("uuid") {
		log.HTTPWarning(req, "No uuid query parameter provided for DeleteImage")
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}

	tag, err := ConvertTagnumber(requestQueries.Get("tagnumber"))
	if err != nil {
		log.HTTPWarning(req, "Error parsing tagnumber query parameter for DeleteImage: "+err.Error())
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}

	requestedImageUUID := strings.TrimSpace(requestQueries.Get("uuid"))
	requestedImageUUID = strings.TrimSuffix(requestedImageUUID, ".jpeg")
	requestedImageUUID = strings.TrimSuffix(requestedImageUUID, ".png")
	requestedImageUUID = strings.TrimSuffix(requestedImageUUID, ".mp4")
	requestedImageUUID = strings.TrimSuffix(requestedImageUUID, ".mov")
	// Check if uuid is empty after trimming
	if requestedImageUUID == "" {
		log.HTTPWarning(req, "Invalid/empty uuid query parameter provided for DeleteImage")
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}

	db := config.GetDatabaseConn()
	if db == nil {
		log.HTTPError(req, "No database connection available")
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}
	repo := database.NewRepo(db)
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	err = repo.HideClientImageByUUID(ctx, tag, requestedImageUUID)
	if err != nil {
		log.Error("Failed to delete client image with UUID " + requestedImageUUID + " for tagnumber " + fmt.Sprintf("%d", tag) + ": " + err.Error())
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}

}
