package endpoints

import (
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"uit-toolbox/database"
	"uit-toolbox/middleware"
	"uit-toolbox/types"
)

func DeleteImage(w http.ResponseWriter, req *http.Request) {
	log := middleware.GetLoggerFromContext(req.Context()).With(slog.String("func", "DeleteImage"))

	// Check if required query parameters are set
	if !req.URL.Query().Has("client_uuid") {
		log.Warn("No client_uuid query key provided")
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}
	if !req.URL.Query().Has("file_uuid") {
		log.Warn("No file_uuid query key provided")
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}

	clientUUID, err := middleware.GetUUIDFromQuery(req.URL.Query(), "client_uuid")
	if err != nil {
		log.Warn("Invalid client_uuid query parameter: " + err.Error())
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}
	fileUUID := middleware.GetStrQuery(req.URL.Query(), "file_uuid")
	if fileUUID == nil || strings.TrimSpace(*fileUUID) == "" {
		log.Warn("Invalid file_uuid query parameter")
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}

	// Get filepath from uuid
	imageManifest, err := database.GetClientImageManifestByFileUUID(req.Context(), *fileUUID)
	if err != nil {
		log.Error("Error retrieving image manifest: " + err.Error())
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}
	// Check for a non-nil response from DB
	if imageManifest == nil {
		log.Warn("No image manifest found for provided uuid: " + *fileUUID)
		middleware.WriteJsonError(w, http.StatusNotFound)
		return
	}

	// client UUID
	if imageManifest.ClientUUID == nil || strings.TrimSpace(*imageManifest.ClientUUID) == "" {
		log.Warn("No client UUID found in image manifest for provided file uuid: " + *fileUUID)
		middleware.WriteJsonError(w, http.StatusNotFound)
		return
	}
	clientUUIDFromManifest := strings.TrimSpace(*imageManifest.ClientUUID)
	if clientUUIDFromManifest != clientUUID.String() {
		log.Warn("Client UUID from manifest does not match client_uuid query parameter for provided file uuid: " + *fileUUID)
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}

	// file uuid
	if imageManifest.FileUUID == nil || strings.TrimSpace(*imageManifest.FileUUID) == "" {
		log.Warn("No file found in image manifest for provided file uuid: " + *fileUUID)
		middleware.WriteJsonError(w, http.StatusNotFound)
		return
	}
	fileUUIDFromManifest := strings.TrimSpace(*imageManifest.FileUUID)
	if fileUUIDFromManifest != *fileUUID {
		log.Warn("File UUID from manifest does not match file_uuid query parameter for provided file uuid: " + *fileUUID)
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}

	// file name
	if imageManifest.FileName == nil || strings.TrimSpace(*imageManifest.FileName) == "" {
		log.Warn("No file name found in image manifest for provided file uuid: " + *fileUUID)
		middleware.WriteJsonError(w, http.StatusNotFound)
		return
	}
	fileNameFromManifest := strings.TrimSpace(*imageManifest.FileName)
	if fileNameFromManifest == "" {
		log.Warn("File name is empty in image manifest for provided file uuid: " + *fileUUID)
		middleware.WriteJsonError(w, http.StatusNotFound)
		return
	}

	filePath := filepath.Join("/opt/inventory_images", clientUUIDFromManifest, fileNameFromManifest)
	filePath = filepath.Clean(filePath)

	imageFile, err := os.Open(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			log.Warn("No image file found for provided file uuid: " + *fileUUID)
			middleware.WriteJsonError(w, http.StatusNotFound)
			return
		}
		log.Error("Error reading image file: " + err.Error())
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}
	if imageFile == nil {
		log.Warn("No image found for provided uuid and file name: " + *fileUUID)
		middleware.WriteJsonError(w, http.StatusNotFound)
		return
	}
	defer imageFile.Close()

	fileMetadata, err := imageFile.Stat()
	if err != nil {
		log.Error("Error getting image file metadata: " + err.Error())
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}
	if fileMetadata.IsDir() {
		log.Warn("Resolved image file path is a directory for provided uuid and file name: " + *fileUUID)
		middleware.WriteJsonError(w, http.StatusNotFound)
		return
	}
	if fileMetadata.Size() == 0 {
		log.Warn("Resolved image file is empty for provided uuid and file name: " + *fileUUID)
		// middleware.WriteJsonError(w, http.StatusNotFound)
		// return
	}
	if !fileMetadata.Mode().IsRegular() {
		log.Warn("Resolved image file is not a regular file for provided uuid and file name: " + *fileUUID)
		middleware.WriteJsonError(w, http.StatusNotFound)
		return
	}

	if err := os.Remove(filePath); err != nil {
		log.Error("Error deleting image file: " + err.Error())
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}

	if imageManifest.ThumbnailFileName != nil && strings.TrimSpace(*imageManifest.ThumbnailFileName) != "" {
		thumbnailPath := filepath.Join("/opt/inventory_images", clientUUIDFromManifest, *imageManifest.ThumbnailFileName)
		thumbnailPath = filepath.Clean(thumbnailPath)
		resolvedThumbnailPath, err := filepath.EvalSymlinks(thumbnailPath)
		if err != nil {
			log.Error("Error resolving thumbnail file path, continuing: " + err.Error())
			// middleware.WriteJsonError(w, http.StatusInternalServerError)
			// return
		}
		thumbnailFile, err := os.Open(resolvedThumbnailPath)
		if err != nil {
			log.Error("Error opening thumbnail file " + resolvedThumbnailPath + ", continuing: " + err.Error())
			// middleware.WriteJsonError(w, http.StatusInternalServerError)
			// return
		}
		if thumbnailFile != nil {
			thumbnailFile.Close()
		}
		if err := os.Remove(resolvedThumbnailPath); err != nil {
			log.Error("Error deleting thumbnail file " + resolvedThumbnailPath + ", continuing: " + err.Error())
			// middleware.WriteJsonError(w, http.StatusInternalServerError)
			// return
		}
	}

	if err := database.HideClientImageByUUID(req.Context(), *fileUUID); err != nil {
		log.Error("DB error while deleting client image with UUID '" + *fileUUID + "': " + err.Error())
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}

	log.Info("Successfully deleted client image with UUID '" + *fileUUID + "'")
	middleware.WriteJson(w, http.StatusOK, map[string]string{"message": "Image deleted successfully"})
}

func DeleteOSInfoByTagnumber(w http.ResponseWriter, req *http.Request) {
	log := middleware.GetLoggerFromContext(req.Context()).With(slog.String("func", "DeleteOSInfoByTagnumber"))

	querySerialValPtr := middleware.GetStrQuery(req.URL.Query(), "system_serial")
	if querySerialValPtr == nil || strings.TrimSpace(*querySerialValPtr) == "" {
		log.Warn("No system_serial query key provided")
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}
	if err := types.IsSystemSerialValid(querySerialValPtr); err != nil {
		log.Warn("Invalid system_serial query parameter: " + err.Error())
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}

	queryTagValPtr := middleware.GetStrQuery(req.URL.Query(), "tagnumber")
	if queryTagValPtr == nil || strings.TrimSpace(*queryTagValPtr) == "" {
		log.Warn("No tagnumber query key provided")
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(*queryTagValPtr) == "" {
		log.Warn("No tagnumber query key provided")
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}
	queryTagVal := strings.TrimSpace(*queryTagValPtr)
	tagnumber, err := strconv.ParseInt(queryTagVal, 10, 64)
	if err != nil {
		log.Warn("Invalid tagnumber query parameter: " + err.Error())
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}

	if err := types.IsTagnumberInt64Valid(&tagnumber); err != nil {
		log.Warn("Invalid tagnumber: " + err.Error())
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}

	if err := database.DeleteOSInfoByTagnumber(req.Context(), tagnumber, *querySerialValPtr); err != nil {
		log.Error("error deleting OS info for tagnumber '" + queryTagVal + "': " + err.Error())
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}

	log.Info("successfully deleted OS info for tagnumber '" + queryTagVal + "'")
}
