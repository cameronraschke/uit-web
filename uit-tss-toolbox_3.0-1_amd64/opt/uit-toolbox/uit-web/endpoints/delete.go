package endpoints

import (
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"uit-toolbox/database"
	"uit-toolbox/middleware"
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

	clientUUID := strings.TrimSpace(req.URL.Query().Get("client_uuid"))
	fileUUID := strings.TrimSpace(req.URL.Query().Get("file_uuid"))

	// Check if uuid is empty after trimming
	if clientUUID == "" || fileUUID == "" {
		log.Warn("Invalid/empty uuid query parameter provided")
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}

	// Get filepath from uuid
	imageManifest, err := database.GetClientImageManifestByFileUUID(req.Context(), fileUUID)
	if err != nil {
		log.Error("Error retrieving image manifest: " + err.Error())
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}
	// Check for a non-nil response from DB
	if imageManifest == nil {
		log.Warn("No image manifest found for provided uuid: " + fileUUID)
		middleware.WriteJsonError(w, http.StatusNotFound)
		return
	}

	// client UUID
	if imageManifest.ClientUUID == nil || strings.TrimSpace(*imageManifest.ClientUUID) == "" {
		log.Warn("No client UUID found in image manifest for provided file uuid: " + fileUUID)
		middleware.WriteJsonError(w, http.StatusNotFound)
		return
	}
	clientUUIDFromManifest := strings.TrimSpace(*imageManifest.ClientUUID)
	if clientUUIDFromManifest != clientUUID {
		log.Warn("Client UUID from manifest does not match client_uuid query parameter for provided file uuid: " + fileUUID)
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}

	// file uuid
	if imageManifest.FileUUID == nil || strings.TrimSpace(*imageManifest.FileUUID) == "" {
		log.Warn("No file found in image manifest for provided file uuid: " + fileUUID)
		middleware.WriteJsonError(w, http.StatusNotFound)
		return
	}
	fileUUIDFromManifest := strings.TrimSpace(*imageManifest.FileUUID)
	if fileUUIDFromManifest != fileUUID {
		log.Warn("File UUID from manifest does not match file_uuid query parameter for provided file uuid: " + fileUUID)
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}

	filePath := filepath.Join("/opt/inventory_images", clientUUIDFromManifest, fileUUID)
	filePath = filepath.Clean(filePath)

	imageFile, err := os.Open(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			log.Warn("No image file found for provided file uuid: " + fileUUID)
			middleware.WriteJsonError(w, http.StatusNotFound)
			return
		}
		log.Error("Error reading image file: " + err.Error())
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}
	if imageFile == nil {
		log.Warn("No image found for provided uuid and file name: " + fileUUID)
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
		log.Warn("Resolved image file path is a directory for provided uuid and file name: " + fileUUID)
		middleware.WriteJsonError(w, http.StatusNotFound)
		return
	}
	if fileMetadata.Size() == 0 {
		log.Warn("Resolved image file is empty for provided uuid and file name: " + fileUUID)
		// middleware.WriteJsonError(w, http.StatusNotFound)
		// return
	}
	if !fileMetadata.Mode().IsRegular() {
		log.Warn("Resolved image file is not a regular file for provided uuid and file name: " + fileUUID)
		middleware.WriteJsonError(w, http.StatusNotFound)
		return
	}

	if err := os.Remove(filePath); err != nil {
		log.Error("Error deleting image file: " + err.Error())
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}

	if imageManifest.ThumbnailFilePath != nil && strings.TrimSpace(*imageManifest.ThumbnailFilePath) != "" {
		joinedThumbnailPath := filepath.Join(*imageManifest.ThumbnailFilePath)
		cleanedThumbnailPath := filepath.Clean(joinedThumbnailPath)
		resolvedThumbnailPath, err := filepath.EvalSymlinks(cleanedThumbnailPath)
		if err != nil {
			log.Error("Error resolving thumbnail file path: " + err.Error())
			middleware.WriteJsonError(w, http.StatusInternalServerError)
			return
		}
		thumbnailFile, err := os.Open(resolvedThumbnailPath)
		if err != nil {
			log.Error("Error opening thumbnail file: " + err.Error())
			middleware.WriteJsonError(w, http.StatusInternalServerError)
			return
		}
		thumbnailFile.Close()
		if err := os.Remove(resolvedThumbnailPath); err != nil {
			log.Error("Error deleting thumbnail file: " + err.Error())
			middleware.WriteJsonError(w, http.StatusInternalServerError)
			return
		}
	}

	if err := database.HideClientImageByUUID(req.Context(), &fileUUID); err != nil {
		log.Error("DB error while deleting client image with UUID '" + fileUUID + "': " + err.Error())
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}

	log.Info("Successfully deleted client image with UUID '" + fileUUID + "'")
	middleware.WriteJson(w, http.StatusOK, map[string]string{"message": "Image deleted successfully"})
}
