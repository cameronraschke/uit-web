package endpoints

import (
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"uit-toolbox/database"
	"uit-toolbox/middleware"
	"uit-toolbox/types"
)

func DeleteImage(w http.ResponseWriter, req *http.Request) {
	log := middleware.GetLoggerFromContext(req.Context()).With(slog.String("func", "DeleteImage"))

	// Check if required query parameters are set
	if !req.URL.Query().Has("tagnumber") {
		log.Warn("No tagnumber query key provided")
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}
	if !req.URL.Query().Has("uuid") {
		log.Warn("No uuid query key provided")
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}

	tag, err := types.ConvertAndVerifyTagnumber(req.URL.Query().Get("tagnumber"))
	if err != nil {
		log.Warn("Error parsing/converting tagnumber query value: " + err.Error())
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}

	// Check if uuid is empty after trimming
	requestedImageUUID := strings.TrimSpace(req.URL.Query().Get("uuid"))
	if requestedImageUUID == "" {
		log.Warn("Invalid/empty uuid query parameter provided")
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}

	// Get filepath from uuid
	imageManifest, err := database.GetClientImageFilePathFromUUID(req.Context(), &requestedImageUUID)
	if err != nil {
		log.Error("Error retrieving image manifest: " + err.Error())
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}
	// Check for a return value
	if imageManifest == nil {
		log.Warn("No image manifest found for provided uuid: " + requestedImageUUID)
		middleware.WriteJsonError(w, http.StatusNotFound)
		return
	}
	// Check for tagnumber
	if imageManifest.Tagnumber == nil {
		log.Warn("Missing tagnumber in image manifest for provided uuid: " + requestedImageUUID)
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}
	if err := types.IsTagnumberInt64Valid(imageManifest.Tagnumber); err != nil {
		log.Warn("Invalid tagnumber in image manifest for provided uuid: " + requestedImageUUID + ". Error: " + err.Error())
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}
	// Check that tagnumber in manifest matches query tagnumber
	if *imageManifest.Tagnumber != *tag {
		log.Warn("Tagnumber mismatch for provided uuid. Expected tagnumber: " + fmt.Sprintf("%d", *imageManifest.Tagnumber) + ", got: " + fmt.Sprintf("%d", *tag))
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}
	// Check returned file path
	if imageManifest.FilePath == nil || strings.TrimSpace(*imageManifest.FilePath) == "" {
		log.Warn("No file path found for provided uuid: " + requestedImageUUID)
		middleware.WriteJsonError(w, http.StatusNotFound)
		return
	}

	joinedFilePath := filepath.Join(*imageManifest.FilePath)
	cleanedFilePath := filepath.Clean(joinedFilePath)
	resolvedFilePath, err := filepath.EvalSymlinks(cleanedFilePath)
	if err != nil {
		log.Error("Error resolving file path: " + err.Error())
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}

	imageFile, err := os.Open(resolvedFilePath)
	if err != nil {
		if os.IsNotExist(err) {
			log.Warn("No image file found for provided uuid and tagnumber: " + requestedImageUUID)
			middleware.WriteJsonError(w, http.StatusNotFound)
			return
		}
		log.Error("Error reading image file: " + err.Error())
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}
	if imageFile == nil {
		log.Warn("No image found for provided uuid and tagnumber: " + requestedImageUUID)
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
		log.Warn("Resolved image file path is a directory for provided uuid and tagnumber: " + requestedImageUUID)
		middleware.WriteJsonError(w, http.StatusNotFound)
		return
	}
	if fileMetadata.Size() == 0 {
		log.Warn("Resolved image file is empty for provided uuid and tagnumber: " + requestedImageUUID)
		// middleware.WriteJsonError(w, http.StatusNotFound)
		// return
	}
	if !fileMetadata.Mode().IsRegular() {
		log.Warn("Resolved image file is not a regular file for provided uuid and tagnumber: " + requestedImageUUID)
		middleware.WriteJsonError(w, http.StatusNotFound)
		return
	}

	if err := os.Remove(resolvedFilePath); err != nil {
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

	if err := database.HideClientImageByUUID(req.Context(), tag, &requestedImageUUID); err != nil {
		log.Error("DB error while deleting client image with UUID '" + requestedImageUUID + "' and tagnumber '" + fmt.Sprintf("%d", *tag) + "': " + err.Error())
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}

	log.Info("Successfully deleted client image with UUID '" + requestedImageUUID + "' and tagnumber '" + fmt.Sprintf("%d", *tag) + "'")
	middleware.WriteJson(w, http.StatusOK, map[string]string{"message": "Image deleted successfully"})
}
