package endpoints

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	_ "image/png"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"
	config "uit-toolbox/config"
	"uit-toolbox/database"
	middleware "uit-toolbox/middleware"
	"unicode/utf8"
)

type RemoteTable struct {
	Tagnumber         *int       `sql:"tagnumber"`
	JobQueued         *string    `sql:"job_queued"`
	JobQueuedPosition *int       `sql:"job_queued_position"`
	JobActive         *bool      `sql:"job_queued_position"`
	CloneMode         *string    `sql:"clone_mode"`
	EraseMode         *string    `sql:"erase_mode"`
	LastJobTime       *time.Time `sql:"last_job_time"`
	Present           *time.Time `sql:"present"`
	PresentBool       *bool      `sql:"present_bool"`
	Status            *string    `sql:"status"`
	KernelUpdated     *bool      `sql:"kernel_updated"`
	BatteryCharge     *int       `sql:"battery_charge"`
	BatteryStatus     *string    `sql:"battery_status"`
	Uptime            *int       `sql:"uptime"`
	CpuTemp           *int       `sql:"cpu_temp"`
	DiskTemp          *int       `sql:"disk_temp"`
	MaxDiskTemp       *int       `sql:"max_disk_temp"`
	WattsNow          *int       `sql:"watts_now"`
	NetworkSpeed      *int       `sql:"network_speed"`
}

type FormJobQueue struct {
	Tagnumber string `json:"job_queued_tagnumber"`
	JobQueued string `json:"job_queued_select"`
}

func UpdateRemoteJobQueued(ctx context.Context, req *http.Request, key string) error {
	// Parse request body JSON
	var j FormJobQueue
	err := json.NewDecoder(req.Body).Decode(&j)
	if err != nil {
		return errors.New("Cannot parse request body JSON: " + err.Error())
	}
	defer req.Body.Close()

	tag := j.Tagnumber
	var tagnumber int
	if len(tag) > 0 {
		tagnumber, err = strconv.Atoi(tag)
		if err != nil {
			return errors.New("Tagnumber cannot be converted to integer: " + j.Tagnumber)
		}
	}
	value := j.JobQueued

	log.Println("Updating job_queued for tagnumber " + j.Tagnumber + " to value " + j.JobQueued)

	// Commit to DB
	if key == "job_queued" {
		err := database.UpdateDB(ctx, "UPDATE remote SET job_queued = $1 WHERE tagnumber = $2", value, tagnumber)
		if err != nil {
			return errors.New("Database error: " + err.Error())
		}
		return nil
	}

	return errors.New("Unknown key: " + key)
}

func WebAuthEndpoint(w http.ResponseWriter, req *http.Request) {
	log := config.GetLogger()
	ctx := req.Context()
	requestIP, ok := middleware.GetRequestIP(req)
	if !ok {
		log.Warning("no IP address stored in context")
		http.Error(w, middleware.FormatHttpError("Internal middleware error"), http.StatusInternalServerError)
		return
	}
	requestURL, ok := middleware.GetRequestURL(req)
	if !ok {
		log.Warning("no URL stored in context")
		http.Error(w, middleware.FormatHttpError("Internal middleware error"), http.StatusInternalServerError)
		return
	}

	// Sanitize login POST request
	if req.Method != http.MethodPost || !(strings.HasSuffix(requestURL, "/login.html") || strings.HasSuffix(requestURL, "/login")) {
		log.Warning("Invalid method or URL for auth form sanitization: " + requestIP + " ( " + requestURL + ")")
		http.Error(w, middleware.FormatHttpError("Bad request"), http.StatusBadRequest)
		return
	}

	body, err := io.ReadAll(req.Body)
	if err != nil {
		log.Warning("Cannot read request body: " + err.Error() + " (" + requestIP + ")")
		http.Error(w, middleware.FormatHttpError("Bad request"), http.StatusBadRequest)
		return
	}
	defer req.Body.Close()

	base64String := strings.TrimSpace(string(body))
	if base64String == "" {
		log.Warning("No base64 string provided in auth form: " + requestIP)
		http.Error(w, middleware.FormatHttpError("Bad request"), http.StatusBadRequest)
		return
	}

	decodedBytes, err := base64.StdEncoding.DecodeString(base64String)
	if err != nil {
		log.Warning("Invalid base64 encoding: " + err.Error() + " (" + requestIP + ")")
		http.Error(w, middleware.FormatHttpError("Bad request"), http.StatusBadRequest)
		return
	}

	if !utf8.Valid(decodedBytes) {
		log.Warning("Invalid UTF-8 in decoded data: " + requestIP)
		http.Error(w, middleware.FormatHttpError("Bad request"), http.StatusBadRequest)
		return
	}
	var clientFormAuthData struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := json.Unmarshal(decodedBytes, &clientFormAuthData); err != nil {
		log.Warning("Invalid JSON structure: " + err.Error() + " (" + requestIP + ")")
		http.Error(w, middleware.FormatHttpError("Bad request"), http.StatusBadRequest)
		return
	}

	// Validate input data
	if err := middleware.ValidateAuthFormInput(clientFormAuthData.Username, clientFormAuthData.Password); err != nil {
		log.Warning("Invalid auth input: " + err.Error() + " (" + requestIP + ")")
		http.Error(w, middleware.FormatHttpError("Bad request"), http.StatusBadRequest)
		return
	}

	// Authenticate with bcrypt
	authenticated, err := middleware.CheckAuthCredentials(ctx, clientFormAuthData.Username, clientFormAuthData.Password)
	if err != nil || !authenticated {
		log.Info("Authentication failed for " + requestIP + ": " + err.Error())
		http.Error(w, middleware.FormatHttpError("Unauthorized"), http.StatusUnauthorized)
		return
	}

	sessionID, basicToken, bearerToken, csrfToken, err := config.CreateAuthSession(requestIP)
	if err != nil {
		log.Error("Failed to generate tokens: " + err.Error())
		http.Error(w, middleware.FormatHttpError("Internal middleware error"), http.StatusInternalServerError)
		return
	}

	sessionCount := config.GetAuthSessionCount()
	log.Info("New auth session created: " + requestIP + " (Sessions: " + strconv.Itoa(int(sessionCount)) + " TTL: " + fmt.Sprintf("%.2f", bearerToken) + "s)")

	http.SetCookie(w, &http.Cookie{
		Name:     "uit_basic_token",
		Value:    basic.Token,
		Path:     "/",
		Expires:  time.Now().Add(20 * time.Minute),
		MaxAge:   20 * 60,
		Secure:   true,
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
	})

	http.SetCookie(w, &http.Cookie{
		Name:     "csrf_token",
		Value:    csrfToken,
		Path:     "/",
		Expires:  time.Now().Add(20 * time.Minute),
		Secure:   true,
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
	})

	returnedJsonStruct := returnedJsonToken{
		Token: bearer.Token,
		TTL:   bearer.TTL,
		Valid: bearer.Valid,
	}

	jsonData, err := json.Marshal(returnedJsonStruct)
	if err != nil {
		log.Error("Cannot marshal Token to JSON: " + err.Error())
		http.Error(w, middleware.FormatHttpError("Internal middleware error"), http.StatusInternalServerError)
		return
	}

	w.Write(jsonData)

}
