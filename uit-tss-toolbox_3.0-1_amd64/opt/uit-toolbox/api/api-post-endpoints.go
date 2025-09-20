package main

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync/atomic"
	"time"
	"unicode/utf8"
)

func authFormEndpoint(authMap *AppState) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		requestIP, ok := GetRequestIP(req)
		if !ok {
			log.Warning("no IP address stored in context")
			http.Error(w, formatHttpError("Internal server error"), http.StatusInternalServerError)
			return
		}
		requestURL, ok := GetRequestURL(req)
		if !ok {
			log.Warning("no URL stored in context")
			http.Error(w, formatHttpError("Internal server error"), http.StatusInternalServerError)
			return
		}

		// Sanitize login POST request
		if req.Method != http.MethodPost || !(strings.HasSuffix(requestURL, "/login.html") || strings.HasSuffix(requestURL, "/login")) {
			log.Warning("Invalid method or URL for auth form sanitization: " + requestIP + " ( " + requestURL + ")")
			http.Error(w, formatHttpError("Bad request"), http.StatusBadRequest)
			return
		}

		body, err := io.ReadAll(req.Body)
		if err != nil {
			log.Warning("Cannot read request body: " + err.Error() + " (" + requestIP + ")")
			http.Error(w, formatHttpError("Bad request"), http.StatusBadRequest)
			return
		}
		defer req.Body.Close()

		base64String := strings.TrimSpace(string(body))
		if base64String == "" {
			log.Warning("No base64 string provided in auth form: " + requestIP)
			http.Error(w, formatHttpError("Bad request"), http.StatusBadRequest)
			return
		}

		decodedBytes, err := base64.StdEncoding.DecodeString(base64String)
		if err != nil {
			log.Warning("Invalid base64 encoding: " + err.Error() + " (" + requestIP + ")")
			http.Error(w, formatHttpError("Bad request"), http.StatusBadRequest)
			return
		}

		if !utf8.Valid(decodedBytes) {
			log.Warning("Invalid UTF-8 in decoded data: " + requestIP)
			http.Error(w, formatHttpError("Bad request"), http.StatusBadRequest)
			return
		}
		var authData struct {
			Username string `json:"username"`
			Password string `json:"password"`
		}
		if err := json.Unmarshal(decodedBytes, &authData); err != nil {
			log.Warning("Invalid JSON structure: " + err.Error() + " (" + requestIP + ")")
			http.Error(w, formatHttpError("Bad request"), http.StatusBadRequest)
			return
		}

		// Validate input data
		if err := validateAuthInput(authData.Username, authData.Password); err != nil {
			log.Warning("Invalid auth input: " + err.Error() + " (" + requestIP + ")")
			http.Error(w, formatHttpError("Bad request"), http.StatusBadRequest)
			return
		}

		// Authenticate with bcrypt
		if err := authenticateUser(authData.Username, authData.Password); err != nil {
			log.Info("Authentication failed: " + requestIP)
			http.Error(w, formatHttpError("Unauthorized"), http.StatusUnauthorized)
			return
		}

		// Store validated auth data in context for downstream handlers
		ctx := context.WithValue(req.Context(), "auth_data", authData)
		req = req.WithContext(ctx)

		w.WriteHeader(http.StatusOK)
		return
	}
}

func loginEndpoint(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ctx := req.Context()

	requestIP, ok := GetRequestIP(req)
	if !ok {
		log.Warning("no IP address stored in context")
		http.Error(w, formatHttpError("Internal server error"), http.StatusInternalServerError)
		return
	}

	requestURL, ok := GetRequestURL(req)
	if !ok {
		log.Warning("no URL stored in context")
		http.Error(w, formatHttpError("Internal server error"), http.StatusInternalServerError)
		return
	}

	requestCookie := req.Cookies()

	log.Info("Verifying cookie login from IP: " + requestIP + " URL: " + requestURL + " Cookies: " + fmt.Sprintf("%d", len(requestCookie)))

	// Decode POSTed username and password from json body
	var requestBody struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(req.Body).Decode(&requestBody); err != nil {
		log.Warning("Failed to decode JSON body: " + err.Error())
		http.Error(w, formatHttpError("Bad request"), http.StatusBadRequest)
		return
	}

	// Get username from json body
	if strings.TrimSpace(requestBody.Username) == "" || strings.TrimSpace(requestBody.Password) == "" {
		log.Info("No username or password provided for HTML login: " + requestIP)
		http.Error(w, formatHttpError("Bad request"), http.StatusBadRequest)
		return
	}

	hashedUsername := sha256.Sum256([]byte(requestBody.Username))
	hashedUsernameString := hex.EncodeToString(hashedUsername[:])
	hashedPassword := sha256.Sum256([]byte(requestBody.Password))
	hashedPasswordString := hex.EncodeToString(hashedPassword[:])

	var requestBasicToken = hashedUsernameString + ":" + hashedPasswordString

	// If cookie is provided, override the Basic token from json body
	// This allows session persistence via cookies
	// If both are provided, the cookie takes precedence

	for _, cookie := range requestCookie {
		if cookie.Name == "uit_basic_token" {
			requestBasicToken = cookie.Value
		}
	}
	if strings.TrimSpace(requestBasicToken) == "" {
		log.Info("No Basic token cookie provided for HTML login: " + requestIP)
		return
	}
	// Check if DB connection is valid
	if db == nil {
		log.Error("Connection to database failed while attempting API Auth")
		http.Error(w, formatHttpError("Internal server error"), http.StatusInternalServerError)
		return
	}

	// Check if the Basic token exists in the database
	// sqlCode := `SELECT username as token FROM logins WHERE CONCAT(username, ':', password) = $1`
	sqlCode := `SELECT ENCODE(SHA256(CONCAT(username, ':', password)::bytea), 'hex') as token FROM logins WHERE ENCODE(CONCAT(username, ':', password)::bytea, 'hex') = ENCODE(SHA256($1::bytea), 'hex')`
	rows, err := db.QueryContext(ctx, sqlCode, requestBasicToken)
	if err != nil {
		log.Error("Cannot query database for API Auth: " + err.Error())
		http.Error(w, formatHttpError("Internal server error"), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	if !rows.Next() {
		log.Info("No matching Basic token found in database: " + requestIP)
		http.Error(w, formatHttpError("Unauthorized"), http.StatusUnauthorized)
		return
	}

	hash := make([]byte, 32)
	_, err = rand.Read(hash)
	if err != nil {
		log.Error("Cannot generate token: " + err.Error())
		http.Error(w, formatHttpError("Internal server error"), http.StatusInternalServerError)
		return
	}
	bearerToken := fmt.Sprintf("%x", hash)
	if strings.TrimSpace(bearerToken) == "" {
		log.Error("Failed to generate bearer token")
		http.Error(w, formatHttpError("Internal server error"), http.StatusInternalServerError)
		return
	}

	sessionID := fmt.Sprintf("%s:%s", requestIP, bearerToken)

	// Set expiry time
	basicTTL := 20 * time.Minute
	bearerTTL := 20 * time.Minute
	basicExpiry := time.Now().Add(basicTTL)
	bearerExpiry := time.Now().Add(bearerTTL)

	basic := BasicToken{Token: requestBasicToken, Expiry: basicExpiry, NotBefore: time.Now(), TTL: time.Until(basicExpiry).Seconds(), IP: requestIP, Valid: true}
	bearer := BearerToken{Token: bearerToken, Expiry: bearerExpiry, NotBefore: time.Now(), TTL: time.Until(bearerExpiry).Seconds(), IP: requestIP, Valid: true}
	csrfToken, err := generateCSRFToken()
	if err != nil {
		log.Error("Cannot generate CSRF token: " + err.Error())
		http.Error(w, formatHttpError("Internal server error"), http.StatusInternalServerError)
		return
	}

	authSession, exists, err := createOrUpdateAuthSession(&authMap, sessionID, basic, bearer, csrfToken)
	if err != nil {
		log.Error("Error creating or updating auth session: " + err.Error())
		http.Error(w, formatHttpError("Internal server error"), http.StatusInternalServerError)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "uit_basic_token",
		Value:    requestBasicToken,
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
		Secure:   true,
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
	})

	if authSession.Bearer.Token != bearerToken || authSession.Bearer.TTL <= 0 ||
		!authSession.Bearer.Valid || authSession.Bearer.IP != requestIP ||
		time.Now().After(authSession.Bearer.Expiry) || time.Now().Before(authSession.Bearer.NotBefore) {
		log.Error("Error while creating new bearer token: " + requestIP)
		authMap.Delete(sessionID)
		atomic.AddInt64(&authMapEntryCount, -1)
		http.Error(w, formatHttpError("Internal server error"), http.StatusInternalServerError)
		return
	}

	sessionCount := countAuthSessions(&authMap)
	if exists {
		log.Info("Auth session exists: " + requestIP + " (Sessions: " + strconv.Itoa(int(sessionCount)) + " TTL: " + fmt.Sprintf("%.2f", authSession.Bearer.TTL) + "s)")
	} else {
		atomic.AddInt64(&authMapEntryCount, 1)
		log.Info("New auth session created: " + requestIP + " (Sessions: " + strconv.Itoa(int(sessionCount)) + " TTL: " + fmt.Sprintf("%.2f", authSession.Bearer.TTL) + "s)")
	}

	returnedJsonStruct := returnedJsonToken{
		Token: authSession.Bearer.Token,
		TTL:   authSession.Bearer.TTL,
		Valid: authSession.Bearer.Valid,
	}

	jsonData, err := json.Marshal(returnedJsonStruct)
	if err != nil {
		log.Error("Cannot marshal Token to JSON: " + err.Error())
		http.Error(w, formatHttpError("Internal server error"), http.StatusInternalServerError)
		return
	}

	w.Write(jsonData)
}
