package endpoints

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"uit-toolbox/database"
	middleware "uit-toolbox/middleware"
)

func DeleteImage(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	log := middleware.GetLoggerFromContext(ctx)
	requestQueries, err := middleware.GetRequestQueryFromContext(ctx)
	if err != nil {
		log.HTTPWarning(req, "Error retrieving request query parameters from context for DeleteImage")
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

	tag, err := ConvertAndVerifyTagnumber(requestQueries.Get("tagnumber"))
	if err != nil {
		log.HTTPWarning(req, "Error parsing tagnumber query parameter for DeleteImage: "+err.Error())
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}
	if tag == nil || *tag <= 0 {
		log.HTTPWarning(req, "No tagnumber provided in URL for DeleteImage")
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}

	// Check if uuid is empty after trimming
	requestedImageUUID := strings.TrimSpace(requestQueries.Get("uuid"))
	if requestedImageUUID == "" {
		log.HTTPWarning(req, "Invalid/empty uuid query parameter provided for DeleteImage")
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}

	// Get filepath from uuid
	selectRepo, err := database.NewSelectRepo()
	if err != nil {
		log.HTTPError(req, "No database connection available")
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}
	imageManifest, err := selectRepo.GetClientImageFilePathFromUUID(ctx, &requestedImageUUID)
	if err != nil {
		log.HTTPError(req, "Error retrieving image file path for DeleteImage: "+err.Error())
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}
	// Check returned file path
	if imageManifest == nil || imageManifest.FilePath == nil || strings.TrimSpace(*imageManifest.FilePath) == "" {
		log.HTTPWarning(req, "No image found for provided uuid in DeleteImage: "+requestedImageUUID)
		middleware.WriteJsonError(w, http.StatusNotFound)
		return
	}
	// Checked that returned tagnumber matches tagnumber from queries
	if imageManifest.Tagnumber == nil || *imageManifest.Tagnumber != *tag {
		log.HTTPWarning(req, "Tagnumber mismatch for provided uuid in DeleteImage. Expected tagnumber: "+fmt.Sprintf("%d", *imageManifest.Tagnumber)+" provided tagnumber: "+fmt.Sprintf("%d", *tag))
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}

	dbFilePath := filepath.Join(*imageManifest.FilePath)
	resolvedFilePath, err := filepath.EvalSymlinks(dbFilePath)
	if err != nil {
		log.HTTPError(req, "Error resolving file path for DeleteImage: "+err.Error())
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}
	if filepath.Base(requestedImageUUID) != filepath.Base(resolvedFilePath) || !strings.HasPrefix(resolvedFilePath, filepath.Join(filepath.Clean("inventory-images"), fmt.Sprintf("%06d", *tag))) {
		log.HTTPWarning(req, "Invalid uuid query parameter provided for DeleteImage: "+requestedImageUUID)
		middleware.WriteJsonError(w, http.StatusBadRequest)
		return
	}

	imageFile, err := os.Open(resolvedFilePath)
	if err != nil {
		if os.IsNotExist(err) {
			log.HTTPWarning(req, "No image file found for provided uuid and tagnumber in DeleteImage: "+requestedImageUUID)
			middleware.WriteJsonError(w, http.StatusNotFound)
			return
		}
		log.HTTPError(req, "Error reading image file for DeleteImage: "+err.Error())
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}
	if imageFile == nil {
		log.HTTPWarning(req, "No image found for provided uuid and tagnumber in DeleteImage: "+requestedImageUUID)
		middleware.WriteJsonError(w, http.StatusNotFound)
		return
	}
	imageFile.Close()

	if err := os.Remove(resolvedFilePath); err != nil {
		log.HTTPError(req, "Error deleting image file for DeleteImage: "+err.Error())
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}

	// Update database to mark image as deleted
	deleteRepo, err := database.NewUpdateRepo()
	if err != nil {
		log.HTTPError(req, "No database connection available")
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}

	if err := deleteRepo.HideClientImageByUUID(ctx, tag, &requestedImageUUID); err != nil {
		log.Error("Failed to delete client image with UUID '" + requestedImageUUID + "' and tagnumber '" + fmt.Sprintf("%d", *tag) + "': " + err.Error())
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}

	log.Info("Successfully deleted client image with UUID '" + requestedImageUUID + "' and tagnumber '" + fmt.Sprintf("%d", *tag) + "'")
	middleware.WriteJson(w, http.StatusOK, map[string]string{"message": "Image deleted successfully"})
}
