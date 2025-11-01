package endpoints

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"
	config "uit-toolbox/config"
	"uit-toolbox/database"
	middleware "uit-toolbox/middleware"
)

func DeleteImage(w http.ResponseWriter, r *http.Request) {
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
		log.Warning("No or invalid tagnumber provided in request from: " + requestIP + " (" + requestURL + ")")
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}

	requestFilePath := strings.TrimPrefix(r.URL.Path, "/api/images/")
	requestFilePath = strings.TrimSuffix(requestFilePath, ".jpeg")
	requestFilePath = strings.TrimSuffix(requestFilePath, ".png")
	requestFilePath = strings.TrimSuffix(requestFilePath, ".mp4")
	requestFilePath = strings.TrimSuffix(requestFilePath, ".mov")
	if requestFilePath == "" {
		log.Warning("No image path provided in request from: " + requestIP + " (" + requestURL + ")")
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}

	uuid := strings.TrimSpace(requestFilePath)
	if uuid == "" {
		log.Warning("No UUID provided in delete image request from: " + requestIP + " (" + requestURL + ")")
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}
	uuid = strings.SplitN(uuid, "/", 2)[1]

	db := config.GetDatabaseConn()
	if db == nil {
		log.Warning("no database connection available")
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}
	repo := database.NewRepo(db)
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	err = repo.HideClientImageByUUID(ctx, tagnumber, uuid)
	if err != nil {
		log.Error("Failed to delete client image with UUID " + uuid + " for tagnumber " + fmt.Sprintf("%d", tagnumber) + ": " + err.Error())
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}

}
