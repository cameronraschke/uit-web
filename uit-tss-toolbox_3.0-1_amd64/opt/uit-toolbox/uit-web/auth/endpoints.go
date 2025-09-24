package auth

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
	config "uit-toolbox/config"
	middleware "uit-toolbox/middleware"
	"unicode/utf8"
)

type ReturnedJsonToken struct {
	Token string  `json:"token"`
	TTL   float64 `json:"ttl"`
	Valid bool    `json:"valid"`
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
	var authData struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := json.Unmarshal(decodedBytes, &authData); err != nil {
		log.Warning("Invalid JSON structure: " + err.Error() + " (" + requestIP + ")")
		http.Error(w, middleware.FormatHttpError("Bad request"), http.StatusBadRequest)
		return
	}

	// Validate input data
	if err := middleware.ValidateAuthFormInput(authData.Username, authData.Password); err != nil {
		log.Warning("Invalid auth input: " + err.Error() + " (" + requestIP + ")")
		http.Error(w, middleware.FormatHttpError("Bad request"), http.StatusBadRequest)
		return
	}

	// Authenticate with bcrypt
	authenticated, err := middleware.CheckAuthCredentials(ctx, authData.Username, authData.Password)
	if err != nil || !authenticated {
		log.Info("Authentication failed for " + requestIP + ": " + err.Error())
		http.Error(w, middleware.FormatHttpError("Unauthorized"), http.StatusUnauthorized)
		return
	}

	bearerValue, csrfValue, err := middleware.GenerateAuthTokens()
	if err != nil {
		log.Error("Failed to generate tokens: " + err.Error())
		http.Error(w, middleware.FormatHttpError("Internal middleware error"), http.StatusInternalServerError)
		return
	}

	basicExpiry := time.Now().Add(20 * time.Minute)
	bearerExpiry := time.Now().Add(20 * time.Minute)
	basic := config.BasicToken{Token: authData.Username + ":" + authData.Password, Expiry: basicExpiry, NotBefore: time.Now(), TTL: time.Until(basicExpiry).Seconds(), IP: requestIP, Valid: true}
	bearer := config.BearerToken{Token: bearerValue, Expiry: bearerExpiry, NotBefore: time.Now(), TTL: time.Until(bearerExpiry).Seconds(), IP: requestIP, Valid: true}
	csrfToken := csrfValue

	sessionID := fmt.Sprintf("%s:%s", requestIP, bearer.Token)
	authSession := config.AuthSession{Basic: basic, Bearer: bearer, CSRF: csrfToken}

	if err := config.CreateAuthSession(sessionID, authSession); err != nil {
		log.Error("Failed to create auth session: " + err.Error())
		http.Error(w, middleware.FormatHttpError("Internal middleware error"), http.StatusInternalServerError)
		return
	}

	sessionValid, sessionExists, err := config.CheckAuthSession(sessionID, requestIP, basic.Token, bearer.Token, csrfToken)
	if err != nil {
		log.Error("Failed to create or update auth session: " + err.Error())
		http.Error(w, middleware.FormatHttpError("Internal middleware error"), http.StatusInternalServerError)
		return
	}
	if !sessionValid {
		log.Warning("Auth session invalid after creation/update: " + requestIP)
		http.Error(w, middleware.FormatHttpError("Internal middleware error"), http.StatusInternalServerError)
		return
	}

	sessionCount := config.GetAuthSessionCount()
	if sessionExists {
		log.Info("Auth session exists: " + requestIP + " (Sessions: " + strconv.Itoa(int(sessionCount)) + " TTL: " + fmt.Sprintf("%.2f", bearer.TTL) + "s)")
	} else {
		config.DeleteAuthSession(sessionID)
		sessionCount = config.GetAuthSessionCount()
		log.Info("New auth session created: " + requestIP + " (Sessions: " + strconv.Itoa(int(sessionCount)) + " TTL: " + fmt.Sprintf("%.2f", bearer.TTL) + "s)")
	}

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
