package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"

	// "net/http/httputil"

	"encoding/hex"
	"encoding/json"
	"time"

	// "unicode/utf8"
	"strings"
	// "crypto/sha256"
	"crypto/rand"
	"crypto/sha256"
	"crypto/tls"
	"database/sql"
	"html/template"
	"net/url"
	"runtime/debug"
	"slices"
	"strconv"
	"sync"
	"sync/atomic"
	"uit-toolbox/api/database"
	"uit-toolbox/api/logger"
	"uit-toolbox/api/post"

	_ "net/http/pprof"

	_ "github.com/jackc/pgx/v5/stdlib"
)

type AppConfig struct {
	UIT_WAN_IF                  string
	UIT_WAN_IP_ADDRESS          string
	UIT_WAN_ALLOWED_IP          []string
	UIT_LAN_IF                  string
	UIT_LAN_IP_ADDRESS          string
	UIT_LAN_ALLOWED_IP          []string
	UIT_ALL_ALLOWED_IP          []string
	UIT_WEB_SVC_PASSWD          string
	UIT_DB_CLIENT_PASSWD        string
	UIT_WEB_USER_DEFAULT_PASSWD string
	UIT_WEBMASTER_NAME          string
	UIT_WEBMASTER_EMAIL         string
	UIT_PRINTER_IP              string
	UIT_HTTP_PORT               string
	UIT_HTTPS_PORT              string
	UIT_TLS_CERT_FILE           string
	UIT_TLS_KEY_FILE            string
	UIT_RATE_LIMIT_BURST        int
	UIT_RATE_LIMIT_INTERVAL     float64
	UIT_RATE_LIMIT_BAN_DURATION time.Duration
}

type AppState struct {
	ipRequests   *LimiterMap
	blockedIPs   *BlockedMap
	allowedFiles map[string]bool
}

// Mux handlers
type muxChain []func(http.Handler) http.Handler

func (chain muxChain) thenFunc(handle http.HandlerFunc) http.Handler {
	return chain.then(handle)
}

func (chain muxChain) then(handle http.Handler) http.Handler {
	for _, fn := range slices.Backward(chain) {
		handle = fn(handle)
	}
	return handle
}

type HttpHeaders struct {
	Authorization AuthHeader
	ContentType   *string
	Accept        *string
	Origin        *string
	UserAgent     *string
	Referer       *string
	Host          *string
}

type AuthHeader struct {
	Basic  *string
	Bearer *string
}

type BasicToken struct {
	Token     string    `json:"token"`
	Expiry    time.Time `json:"expiry"`
	NotBefore time.Time `json:"not_before"`
	TTL       float64   `json:"ttl"`
	IP        string    `json:"ip"`
	Valid     bool      `json:"valid"`
}

type BearerToken struct {
	Token     string    `json:"token"`
	Expiry    time.Time `json:"expiry"`
	NotBefore time.Time `json:"not_before"`
	TTL       float64   `json:"ttl"`
	IP        string    `json:"ip"`
	Valid     bool      `json:"valid"`
}

type AuthSession struct {
	Basic  BasicToken
	Bearer BearerToken
	CSRF   string
}

type returnedJsonToken struct {
	Token string  `json:"token"`
	TTL   float64 `json:"ttl"`
	Valid bool    `json:"valid"`
}

type httpErrorCodes struct {
	Message string `json:"message"`
}

type RateLimiter struct {
	Requests       int
	LastSeen       time.Time
	MapLastUpdated time.Time
	BannedUntil    time.Time
	Banned         bool
}

var (
	db                *sql.DB
	authMap           sync.Map
	authMapEntryCount int64
	log               logger.Logger = logger.CreateLogger("console", logger.ParseLogLevel(os.Getenv("UIT_API_LOG_LEVEL")))
)

func remoteAPI(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()

	requestURL, ok := GetRequestURL(req)
	if !ok {
		log.Warning("no URL stored in context")
		http.Error(w, formatHttpError("Internal server error"), http.StatusInternalServerError)
		return
	}

	var parsedURL *url.URL
	var err error

	if req.Method != http.MethodGet {
		log.Info("Method " + req.Method + " not allowed")
		http.Error(w, formatHttpError("Method not allowed"), http.StatusMethodNotAllowed)
		return
	}

	// Check database connection
	if db == nil {
		log.Error("Connection to database failed while attempting API Auth")
		http.Error(w, formatHttpError("Internal server error"), http.StatusInternalServerError)
		return
	}

	// Parse URL
	parsedURL, err = url.Parse(requestURL)
	if err != nil {
		log.Warning("Cannot parse URL ( " + req.RemoteAddr + "): " + " " + err.Error() + " (" + requestURL + ")")
		return
	}
	// path := parsedURL.Path
	RawQuery := parsedURL.RawQuery
	queries, _ := url.ParseQuery(RawQuery)

	tag := queries.Get("tagnumber")
	var tagnumber int
	if len(tag) > 0 {
		tagnumber, err = strconv.Atoi(tag)
		if err != nil {
			log.Warning("Tagnumber cannot be converted to integer: " + queries.Get("tagnumber"))
			return
		}
	}
	// if len(tagnumber) != 6 {
	//   log.Warning("Tagnumber not 6 digits long")
	//   http.Error(w, formatHttpError("Bad Request"), http.StatusBadRequest)
	//   return
	// }
	// systemSerial = queries.Get("system_serial")
	// sqlTime = queries.Get("time")
	queryType := queries.Get("type")

	switch queryType {
	case "all_tags":
		var allTagsJson string
		allTagsJson, err = database.GetAllTags(ctx, db)
		if err != nil {
			log.Warning("Query error: " + err.Error())
			return
		}
		io.WriteString(w, allTagsJson)
		return
	case "remote_present":
		var remoteTableJson string
		remoteTableJson, err = database.GetRemoteOnlineTable(ctx, db)
		if err != nil {
			log.Warning("Query error: " + err.Error())
			return
		}
		io.WriteString(w, remoteTableJson)
		return
	case "live_image":
		var liveImageTableJson string
		liveImageTableJson, err = database.GetLiveImage(ctx, db, tagnumber)
		if err != nil {
			log.Warning("Query error: " + err.Error())
			return
		}
		io.WriteString(w, liveImageTableJson)
		return
	case "remote_present_header":
		var remoteTableHeaderJson string
		remoteTableHeaderJson, err = database.GetRemotePresentHeader(ctx, db)
		if err != nil {
			log.Warning("Query error: " + err.Error())
			return
		}
		io.WriteString(w, remoteTableHeaderJson)
		return
	case "remote_offline":
		var remoteOfflineTableJson string
		remoteOfflineTableJson, err = database.GetRemoteOfflineTable(ctx, db)
		if err != nil {
			log.Warning("Query error: " + err.Error())
			return
		}
		io.WriteString(w, remoteOfflineTableJson)
		return
	case "tagnumber_data":
		var tagnumberDataJson string
		tagnumberDataJson, err = database.GetTagnumberData(ctx, db, tagnumber)
		if err != nil {
			log.Warning("Query error: " + err.Error())
			return
		}
		io.WriteString(w, tagnumberDataJson)
		return
	case "job_queue_by_tag":
		var remoteJobQueueByTagJson string
		remoteJobQueueByTagJson, err = database.GetJobQueueByTagnumber(ctx, db, tagnumber)
		if err != nil {
			log.Warning("Query error: " + err.Error())
			return
		}
		io.WriteString(w, remoteJobQueueByTagJson)
		return
	case "available_jobs":
		var availableJobsJson string
		availableJobsJson, err = database.GetAvailableJobs(ctx, db, tagnumber)
		if err != nil {
			log.Warning("Query error: " + err.Error())
			return
		}
		io.WriteString(w, availableJobsJson)
		return
	default:
		log.Warning("No query type defined: " + requestURL)
		return
	}
}

func postAPI(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()

	requestURL, ok := GetRequestURL(req)
	if !ok {
		log.Warning("no URL stored in context")
		http.Error(w, formatHttpError("Internal server error"), http.StatusInternalServerError)
		return
	}

	var parsedURL *url.URL
	var err error

	if req.Method != http.MethodPost {
		log.Info("Method " + req.Method + " not allowed")
		http.Error(w, formatHttpError("Method not allowed"), http.StatusMethodNotAllowed)
		return
	}

	// Check database connection
	if db == nil {
		log.Error("Connection to database failed while attempting API Auth")
		http.Error(w, formatHttpError("Internal server error"), http.StatusInternalServerError)
		return
	}

	// Parse URL
	parsedURL, err = url.Parse(requestURL)
	if err != nil {
		log.Warning("Cannot parse URL ( " + req.RemoteAddr + "): " + " " + err.Error() + " (" + requestURL + ")")
		return
	}
	// path := parsedURL.Path
	RawQuery := parsedURL.RawQuery
	queries, _ := url.ParseQuery(RawQuery)

	// tag := queries.Get("tagnumber")
	// var tagnumber int
	// if len(tag) > 0 {
	//   tagnumber, err = strconv.Atoi(tag)
	//   if err != nil {
	//     log.Warning("Tagnumber cannot be converted to integer: " + queries.Get("tagnumber"))
	//     return
	//   }
	// }

	queryType := queries.Get("type")

	switch queryType {
	case "job_queued":
		err = post.UpdateRemoteJobQueued(ctx, req, db, queryType)
		if err != nil {
			log.Error("Cannot update DB: " + err.Error())
			return
		}
		return
	case "client_image":
		err = post.UpdateClientImages(ctx, req, db, queryType)
		if err != nil {
			log.Error("Cannot update DB: " + err.Error())
			return
		}
		return
	default:
		log.Warning("No POST type defined: " + queryType)
		return
	}
}

func getNewBearerToken(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	var basicToken string

	requestIP, ok := GetRequestIP(req)
	if !ok {
		log.Warning("no IP address stored in context")
		http.Error(w, formatHttpError("Internal server error"), http.StatusInternalServerError)
		return
	}

	headers := ParseHeaders(req.Header)
	if headers.Authorization.Basic != nil {
		basicToken = *headers.Authorization.Basic
	}

	if strings.TrimSpace(basicToken) == "" {
		log.Warning("Empty value for Basic Authorization header")
		http.Error(w, formatHttpError("Empty Basic Authorization header"), http.StatusUnauthorized)
		return
	}

	// Check if DB connection is valid
	if db == nil {
		log.Error("Connection to database failed while attempting API Auth")
		http.Error(w, formatHttpError("Internal server error"), http.StatusInternalServerError)
		return
	}

	// Check if the Basic token exists in the database
	sqlCode := `SELECT ENCODE(SHA256(CONCAT(username, ':', password)::bytea), 'hex') as token FROM logins WHERE ENCODE(SHA256(CONCAT(username, ':', password)::bytea), 'hex') = $1`
	rows, err := db.QueryContext(ctx, sqlCode, basicToken)
	if err != nil {
		log.Error("Cannot query database for API Auth: " + err.Error())
		http.Error(w, formatHttpError("Internal server error"), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	for rows.Next() {
		var dbToken string

		if err := rows.Scan(&dbToken); err != nil {
			log.Error("Error scanning token: " + err.Error())
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}
		if ctx.Err() != nil {
			log.Error("Context error: " + ctx.Err().Error())
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}
		if strings.TrimSpace(dbToken) == "" {
			log.Info("DB token has 0 length")
			http.Error(w, formatHttpError("Internal server error"), http.StatusUnauthorized)
			return
		}

		if dbToken != basicToken {
			log.Info("Incorrect credentials provided for token refresh: " + req.RemoteAddr)
			http.Error(w, formatHttpError("Forbidden"), http.StatusUnauthorized)
			return
		}

		// hash := sha256.New()
		// hash.Write([]byte(dbToken))
		// bearerToken := fmt.Sprintf("%x", hash.Sum(nil))
		hash := make([]byte, 32)
		_, err := rand.Read(hash)
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

		sessionID := fmt.Sprintf("%s:%s", requestIP, basicToken)

		// Set expiry time
		basicTTL := 60 * time.Minute
		bearerTTL := 60 * time.Second
		basicExpiry := time.Now().Add(basicTTL)
		bearerExpiry := time.Now().Add(bearerTTL)

		basic := BasicToken{Token: basicToken, Expiry: basicExpiry, NotBefore: time.Now(), TTL: time.Until(basicExpiry).Seconds(), IP: requestIP, Valid: true}
		bearer := BearerToken{Token: bearerToken, Expiry: bearerExpiry, NotBefore: time.Now(), TTL: time.Until(bearerExpiry).Seconds(), IP: requestIP, Valid: true}
		csrfToken, err := generateCSRFToken()
		if err != nil {
			log.Error("Cannot generate CSRF token: " + err.Error())
			http.Error(w, formatHttpError("Internal server error"), http.StatusInternalServerError)
			return
		}
		_, exists := authMap.Load(sessionID)
		authMap.Store(sessionID, AuthSession{Basic: basic, Bearer: bearer, CSRF: csrfToken})

		value, ok := authMap.Load(sessionID)
		if !ok {
			log.Error("Cannot load auth session from authMap")
			http.Error(w, formatHttpError("Internal server error"), http.StatusInternalServerError)
			return
		}

		http.SetCookie(w, &http.Cookie{
			Name:     "uit_basic_token",
			Value:    basicToken,
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
			HttpOnly: false,
			SameSite: http.SameSiteStrictMode,
		})

		authSession := value.(AuthSession)

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
}

func checkAuthSession(authMap *sync.Map, requestIP string, requestBasicToken string, requestBearerToken string) (basicValid bool, bearerValid bool, basicTTL float64, bearerTTL float64, matchedSession *AuthSession) {
	basicValid = false
	bearerValid = false
	basicTTL = 0.0
	bearerTTL = 0.0
	matchedSession = nil

	authMap.Range(func(k, v any) bool {
		sessionID := k.(string)
		authSession := v.(AuthSession)
		sessionIP := strings.SplitN(sessionID, ":", 2)[0]

		basicExists := strings.TrimSpace(authSession.Basic.Token) != ""
		bearerExists := strings.TrimSpace(authSession.Bearer.Token) != ""

		if strings.TrimSpace(sessionID) == "" || strings.TrimSpace(sessionIP) == "" {
			authMap.Delete(sessionID)
			atomic.AddInt64(&authMapEntryCount, -1)
			return true
		}

		if !basicExists && !bearerExists {
			authMap.Delete(sessionID)
			atomic.AddInt64(&authMapEntryCount, -1)
			return true
		}

		if basicExists &&
			strings.TrimSpace(requestBasicToken) == strings.TrimSpace(authSession.Basic.Token) &&
			requestIP == authSession.Basic.IP {
			if time.Now().Before(authSession.Basic.Expiry) && authSession.Basic.Valid {
				basicValid = true
				basicTTL = time.Until(authSession.Basic.Expiry).Seconds()
				matchedSession = &authSession
			} else {
				log.Debug("Basic token found but expired/invalid for IP: " + sessionIP)
			}
		}

		if bearerExists &&
			strings.TrimSpace(requestBearerToken) == strings.TrimSpace(authSession.Bearer.Token) &&
			requestIP == authSession.Bearer.IP {
			if time.Now().Before(authSession.Bearer.Expiry) && authSession.Bearer.Valid {
				bearerValid = true
				bearerTTL = time.Until(authSession.Bearer.Expiry).Seconds()
				matchedSession = &authSession
			} else {
				log.Debug("Bearer token found but expired/invalid for IP: " + sessionIP)
			}
		}

		if basicValid || bearerValid {
			return false
		}
		return true
	})

	return
}

func verifyCookieLogin(w http.ResponseWriter, req *http.Request) {
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
	sqlCode := `SELECT username as token FROM logins WHERE CONCAT(username, ':', password) = $1`
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
	_, exists := authMap.Load(sessionID)
	authMap.Store(sessionID, AuthSession{Basic: basic, Bearer: bearer, CSRF: csrfToken})

	value, ok := authMap.Load(sessionID)
	if !ok {
		log.Error("Cannot load auth session from authMap")
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

	authSession := value.(AuthSession)

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

func redirectToHTTPSHandler(httpsPort string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		requestURL, ok := GetRequestURL(req)
		if !ok {
			log.Warning("no URL stored in context")
			http.Error(w, formatHttpError("Internal server error"), http.StatusInternalServerError)
			return
		}

		host := req.Host
		if colon := strings.LastIndex(host, ":"); colon != -1 {
			host = host[:colon]
		}
		httpsURL := "https://" + host
		if httpsPort != "443" && httpsPort != "" {
			httpsURL += ":" + httpsPort
		}
		httpsURL += host + requestURL
		http.Redirect(w, req, httpsURL, http.StatusMovedPermanently)
	})
}

func fileServerHandler(appState *AppState) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		// ctx := req.Context()
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

		basePath := "/srv/uit-toolbox/"

		fullPath := path.Join(basePath, requestURL)
		_, fileRequested := path.Split(fullPath)

		if len(appState.allowedFiles) > 0 && !appState.allowedFiles[fileRequested] {
			log.Warning("File not in whitelist: " + fileRequested)
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}

		log.Debug("File request from " + requestIP + " for " + fileRequested)

		resolvedPath, err := filepath.EvalSymlinks(fullPath)
		if err != nil || !strings.HasPrefix(resolvedPath, basePath) {
			log.Warning("File request error from " + requestIP + " (" + resolvedPath + "): Error resolving symlink: " + err.Error())
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}

		// Open the file
		f, err := os.Open(resolvedPath)
		if err != nil {
			log.Warning("File not found: " + resolvedPath + " (" + err.Error() + ")")
			http.Error(w, "File not found", http.StatusNotFound)
			return
		}
		defer f.Close()

		// Get file info for headers
		stat, err := f.Stat()
		if err != nil {
			log.Error("Cannot stat file: " + resolvedPath + " (" + err.Error() + ")")
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}
		if stat.IsDir() {
			log.Warning("Attempt to access directory as file: " + resolvedPath)
			http.Error(w, "Not a file", http.StatusForbidden)
			return
		}

		// Set headers
		w.Header().Set("Content-Type", "application/octet-stream")
		w.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate, proxy-revalidate")
		w.Header().Set("Pragma", "no-cache")
		w.Header().Set("Expires", "0")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Content-Length", fmt.Sprintf("%d", stat.Size()))
		w.Header().Set("Content-Disposition", "attachment; filename=\""+stat.Name()+"\"")

		// Serve the file
		_, err = io.Copy(w, f)
		if err != nil {
			log.Error("Error sending file (" + resolvedPath + "): " + err.Error())
			return
		}
		log.Info("Served file: " + resolvedPath + " to " + requestIP)
	}
}

func webServerHandler(appState *AppState) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		// ctx := req.Context()
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

		basePath := "/var/www/html/uit-web/"

		fullPath := path.Join(basePath, requestURL)
		_, fileRequested := path.Split(fullPath)

		if len(appState.allowedFiles) > 0 && !appState.allowedFiles[fileRequested] {
			log.Warning("File not in whitelist: " + fileRequested)
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}

		log.Debug("File request from " + requestIP + " for " + fileRequested)

		resolvedPath, err := filepath.EvalSymlinks(fullPath)
		if err != nil || !strings.HasPrefix(resolvedPath, basePath) {
			log.Warning("File request error from " + requestIP + " (" + resolvedPath + "): Error resolving symlink: " + err.Error())
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}

		// Open the file
		f, err := os.Open(resolvedPath)
		if err != nil {
			log.Warning("File not found: " + resolvedPath + " (" + err.Error() + ")")
			http.Error(w, "File not found", http.StatusNotFound)
			return
		}
		defer f.Close()

		// Get file info for headers
		stat, err := f.Stat()
		if err != nil {
			log.Error("Cannot stat file: " + resolvedPath + " (" + err.Error() + ")")
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}
		if stat.IsDir() {
			log.Warning("Attempt to access directory as file: " + resolvedPath)
			http.Error(w, "Not a file", http.StatusForbidden)
			return
		}

		w.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate, proxy-revalidate")
		w.Header().Set("Pragma", "no-cache")
		w.Header().Set("Expires", "0")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Content-Disposition", "inline; filename=\""+stat.Name()+"\"")
		w.Header().Set("Content-Length", fmt.Sprintf("%d", stat.Size()))

		// Set headers
		if strings.HasSuffix(fileRequested, ".html") {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			// Parse the template
			htmlTemp, err := template.ParseFiles(resolvedPath)
			if err != nil {
				log.Warning("Cannot parse template file (" + resolvedPath + "): " + err.Error())
				http.Error(w, "Internal server error", http.StatusInternalServerError)
				return
			}
			// Execute the template
			err = htmlTemp.Execute(w, nil)
			if err != nil {
				log.Error("Error executing template for " + resolvedPath + ": " + err.Error())
				http.Error(w, "Internal server error", http.StatusInternalServerError)
				return
			}
		} else if strings.HasSuffix(fileRequested, ".css") {
			w.Header().Set("Content-Type", "text/css; charset=utf-8")
			// Serve the CSS file
			_, err = io.Copy(w, f)
			if err != nil {
				log.Error("Error sending file (" + resolvedPath + "): " + err.Error())
				return
			}
		} else if strings.HasSuffix(fileRequested, ".js") {
			w.Header().Set("Content-Type", "application/javascript; charset=utf-8")
			// Serve the JS file
			_, err = io.Copy(w, f)
			if err != nil {
				log.Error("Error sending file (" + resolvedPath + "): " + err.Error())
				return
			}
		} else if strings.HasSuffix(fileRequested, ".ico") {
			w.Header().Set("Content-Type", "image/x-icon")
			// Serve the favicon
			_, err = io.Copy(w, f)
			if err != nil {
				log.Error("Error sending file (" + resolvedPath + "): " + err.Error())
				return
			}
		} else {
			log.Warning("Unknown file type requested: " + fileRequested)
			http.Error(w, "Unsupported Media Type", http.StatusUnsupportedMediaType)
			return
		}

		log.Debug("Served file: " + resolvedPath + " to " + requestIP)
	}
}

func logoutHandler(appState *AppState) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		requestIP, ok := GetRequestIP(req)
		if !ok {
			log.Warning("no IP address stored in context")
			http.Error(w, formatHttpError("Internal server error"), http.StatusInternalServerError)
			return
		}
		// Invalidate cookies
		cookie, err := req.Cookie("uit_basic_token")
		if err != nil && err != http.ErrNoCookie {
			log.Warning("Error retrieving Basic token cookie for logout: " + err.Error() + " (" + requestIP + ")")
			http.Redirect(w, req, "/login.html", http.StatusSeeOther)
			return
		}
		if cookie == nil || strings.TrimSpace(cookie.Value) == "" {
			log.Info("No Basic token cookie provided for logout: " + requestIP)
			http.Redirect(w, req, "/login.html", http.StatusSeeOther)
			return
		}
		http.SetCookie(w, &http.Cookie{
			Name:     "uit_basic_token",
			Value:    "",
			Path:     "/",
			Expires:  time.Unix(0, 0),
			MaxAge:   -1,
			HttpOnly: true,
		})
		http.SetCookie(w, &http.Cookie{
			Name:     "csrf_token",
			Value:    "",
			Path:     "/",
			Expires:  time.Unix(0, 0),
			MaxAge:   -1,
			HttpOnly: true,
		})
		authMap.Range(func(k, v any) bool {
			sessionID := k.(string)
			authSession := v.(AuthSession)
			// Diagnostic log to help determine why sessions may not match
			if authSession.Basic.Token == cookie.Value && authSession.Basic.IP == requestIP {
				log.Debug("Logout scan: sessionID=" + sessionID + " basic.token=" + authSession.Basic.Token + " basic.ip=" + authSession.Basic.IP + " cookie=" + cookie.Value + " requestIP=" + requestIP)
				authMap.Delete(sessionID)
				atomic.AddInt64(&authMapEntryCount, -1)
				log.Info("Invalidated session: " + sessionID)
				return false
			}
			return true
		})
		// Redirect to login page
		http.Redirect(w, req, "/login.html", http.StatusSeeOther)
	}
}

func rejectRequest(w http.ResponseWriter, req *http.Request) {
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

	log.Warning("access denied: " + requestIP + " tried to access " + requestURL)
	http.Error(w, "Access denied", http.StatusForbidden)
}

func configureEnvironment() AppConfig {
	// WAN interface, IP, and allowed IPs
	wanIf, ok := os.LookupEnv("UIT_WAN_IF")
	if !ok {
		log.Error("Error getting UIT_WAN_IF: not found")
	}
	wanIP, ok := os.LookupEnv("UIT_WAN_IP_ADDRESS")
	if !ok {
		log.Error("Error getting UIT_WAN_IP_ADDRESS: not found")
	}
	envWanAllowedIPStr, ok := os.LookupEnv("UIT_WAN_ALLOWED_IP")
	if !ok {
		log.Error("Error getting UIT_WAN_ALLOWED_IP: not found")
	}

	envWanAllowedIPs := strings.Split(envWanAllowedIPStr, ",")
	wanAllowedIP := make([]string, 0, len(envWanAllowedIPs))
	for _, cidr := range envWanAllowedIPs {
		cidr = strings.TrimSpace(cidr)
		if cidr != "" {
			wanAllowedIP = append(wanAllowedIP, cidr)
		}
	}

	// LAN interface, IP, and allowed IPs
	lanIf, ok := os.LookupEnv("UIT_LAN_IF")
	if !ok {
		log.Error("Error getting UIT_LAN_IF: not found")
	}
	lanIP, ok := os.LookupEnv("UIT_LAN_IP_ADDRESS")
	if !ok {
		log.Error("Error getting UIT_LAN_IP_ADDRESS: not found")
	}
	envAllowedLanIPStr, ok := os.LookupEnv("UIT_LAN_ALLOWED_IP")
	if !ok {
		log.Error("Error getting UIT_LAN_ALLOWED_IP: not found")
	}

	envLanAllowedIPs := strings.Split(envAllowedLanIPStr, ",")
	lanAllowedIP := make([]string, 0, len(envLanAllowedIPs))
	for _, cidr := range envLanAllowedIPs {
		cidr = strings.TrimSpace(cidr)
		if cidr != "" {
			lanAllowedIP = append(lanAllowedIP, cidr)
		}
	}

	envAllAllowedIPStr := envAllowedLanIPStr + "," + envWanAllowedIPStr
	envAllAllowedIPs := strings.Split(envAllAllowedIPStr, ",")
	allAllowedIPs := make([]string, 0, len(envAllAllowedIPs))
	for _, cidr := range envAllAllowedIPs {
		cidr = strings.TrimSpace(cidr)
		if cidr != "" {
			allAllowedIPs = append(allAllowedIPs, cidr)
		}
	}

	// Database credentials
	uitWebSvcPasswd, ok := os.LookupEnv("UIT_WEB_SVC_PASSWD")
	if !ok {
		log.Error("Error getting UIT_WEB_SVC_PASSWD: not found")
	}
	uitWebSvcPasswd = strings.TrimSpace(uitWebSvcPasswd)

	dbClientPasswd, ok := os.LookupEnv("UIT_DB_CLIENT_PASSWD")
	if !ok {
		log.Error("Error getting UIT_DB_CLIENT_PASSWD: not found")
	}
	webUserDefaultPasswd, ok := os.LookupEnv("UIT_WEB_USER_DEFAULT_PASSWD")
	if !ok {
		log.Error("Error getting UIT_WEB_USER_DEFAULT_PASSWD: not found")
	}

	// Website config
	webmasterName, ok := os.LookupEnv("UIT_WEBMASTER_NAME")
	if !ok {
		log.Error("Error getting UIT_WEBMASTER_NAME: not found")
	}
	webmasterEmail, ok := os.LookupEnv("UIT_WEBMASTER_EMAIL")
	if !ok {
		log.Error("Error getting UIT_WEBMASTER_EMAIL: not found")
	}

	// Printer IP
	printerIP, ok := os.LookupEnv("UIT_PRINTER_IP")
	if !ok {
		log.Error("Error getting UIT_PRINTER_IP: not found")
	}

	// Webserver config
	httpPort, ok := os.LookupEnv("UIT_HTTP_PORT")
	if !ok {
		log.Error("Error getting UIT_HTTP_PORT: not found")
	}
	httpsPort, ok := os.LookupEnv("UIT_HTTPS_PORT")
	if !ok {
		log.Error("Error getting UIT_HTTPS_PORT: not found")
	}
	tlsCertFile, ok := os.LookupEnv("UIT_TLS_CERT_FILE")
	if !ok {
		log.Error("Error getting UIT_TLS_CERT_FILE: not found")
	}
	tlsKeyFile, ok := os.LookupEnv("UIT_TLS_KEY_FILE")
	if !ok {
		log.Error("Error getting UIT_TLS_KEY_FILE: not found")
	}

	// Rate limiting config
	rateLimitBurstStr, ok := os.LookupEnv("UIT_RATE_LIMIT_BURST")
	if !ok {
		log.Error("Error getting UIT_RATE_LIMIT_BURST: not found")
	}
	var rateLimitBurstErr error
	rateLimitBurst, rateLimitBurstErr = strconv.Atoi(rateLimitBurstStr)
	if rateLimitBurstErr != nil || rateLimitBurst <= 0 {
		log.Error("Error converting UIT_RATE_LIMIT_BURST to integer: " + rateLimitBurstErr.Error())
		rateLimitBurst = 100
	}
	rateLimitIntervalStr, ok := os.LookupEnv("UIT_RATE_LIMIT_INTERVAL")
	if !ok {
		log.Error("Error getting UIT_RATE_LIMIT_INTERVAL: not found")
	}
	var rateLimitErr error
	rateLimit, rateLimitErr = strconv.ParseFloat(rateLimitIntervalStr, 64)
	if rateLimitErr != nil || rateLimit <= 0 {
		log.Error("Error converting UIT_RATE_LIMIT_INTERVAL to float: " + rateLimitErr.Error())
		rateLimit = 1
	}
	rateLimitBanDurationStr, ok := os.LookupEnv("UIT_RATE_LIMIT_BAN_DURATION")
	if !ok {
		log.Error("Error getting UIT_RATE_LIMIT_BAN_DURATION: not found")
	}
	banDurationInt, err := strconv.ParseInt(rateLimitBanDurationStr, 10, 64)
	if err != nil || banDurationInt <= 0 {
		log.Error("Error converting UIT_RATE_LIMIT_BAN_DURATION to integer: " + err.Error())
		banDurationInt = 30
	}
	rateLimitBanDuration = time.Duration(banDurationInt) * time.Second

	return AppConfig{
		UIT_WAN_IF:                  wanIf,
		UIT_WAN_IP_ADDRESS:          wanIP,
		UIT_WAN_ALLOWED_IP:          wanAllowedIP,
		UIT_LAN_IF:                  lanIf,
		UIT_LAN_IP_ADDRESS:          lanIP,
		UIT_LAN_ALLOWED_IP:          lanAllowedIP,
		UIT_ALL_ALLOWED_IP:          allAllowedIPs,
		UIT_WEB_SVC_PASSWD:          uitWebSvcPasswd,
		UIT_DB_CLIENT_PASSWD:        dbClientPasswd,
		UIT_WEB_USER_DEFAULT_PASSWD: webUserDefaultPasswd,
		UIT_WEBMASTER_NAME:          webmasterName,
		UIT_WEBMASTER_EMAIL:         webmasterEmail,
		UIT_PRINTER_IP:              printerIP,
		UIT_HTTP_PORT:               httpPort,
		UIT_HTTPS_PORT:              httpsPort,
		UIT_TLS_CERT_FILE:           tlsCertFile,
		UIT_TLS_KEY_FILE:            tlsKeyFile,
		UIT_RATE_LIMIT_BURST:        rateLimitBurst,
		UIT_RATE_LIMIT_INTERVAL:     rateLimit,
		UIT_RATE_LIMIT_BAN_DURATION: rateLimitBanDuration,
	}
}

func main() {
	debug.PrintStack()
	log.Info("Server time: " + time.Now().Format("01-02-2006 15:04:05"))
	log.Info("UIT API Starting...")

	// Recover from panics
	defer func() {
		if pan := recover(); pan != nil {
			log.Error("Recovered. Error: \n" + fmt.Sprintf("%v", pan))
			log.Error("Trace: \n" + string(debug.Stack()))
		}
	}()

	appConfig := configureEnvironment()

	appState := &AppState{
		ipRequests: &LimiterMap{rate: rateLimit, burst: rateLimitBurst},
		blockedIPs: &BlockedMap{banPeriod: rateLimitBanDuration},
		allowedFiles: map[string]bool{
			"filesystem.squashfs":    true,
			"initrd.img":             true,
			"vmlinuz":                true,
			"uit-ca.crt":             true,
			"uit-web.crt":            true,
			"uit-toolbox-client.deb": true,
			"desktop.css":            true,
			"favicon.ico":            true,
			"header.html":            true,
			"footer.html":            true,
			"index.html":             true,
			"login.html":             true,
			"auth-webworker.js":      true,
			"footer.js":              true,
			"header.js":              true,
			"init.js":                true,
			"include.js":             true,
			"login.js":               true,
			"logout.js":              true,
			"inventory.html":         true,
			"inventory.js":           true,
			"checkouts.html":         true,
			"checkouts.js":           true,
			"job_queue.html":         true,
			"job_queue.js":           true,
			"reports.html":           true,
			"reports.js":             true,
		},
	}

	backgroundProcesses(appState)

	// Connect to db with pgx
	log.Info("Attempting connection to database...")
	dbConnScheme := "postgres"
	dbConnHost := "127.0.0.1"
	dbConnPort := "5432"
	dbConnUser := "uitweb"
	dbConnDBName := "uitdb"
	dbConnPass := appConfig.UIT_WEB_SVC_PASSWD
	dbConnString := dbConnScheme + "://" + dbConnUser + ":" + dbConnPass + "@" + dbConnHost + ":" + dbConnPort + "/" + dbConnDBName + "?sslmode=disable"
	var dbConnErr error
	db, dbConnErr = sql.Open("pgx", dbConnString)
	if dbConnErr != nil {
		log.Error("Unable to connect to database: \n" + dbConnErr.Error())
		os.Exit(1)
	}
	defer db.Close()

	// Check if the database connection is valid
	if err := db.Ping(); err != nil {
		log.Error("Cannot ping database: \n" + err.Error())
		os.Exit(1)
	}
	log.Info("Connected to database successfully")

	// Set defaults for db connection
	db.SetMaxOpenConns(30)
	db.SetMaxIdleConns(10)
	db.SetConnMaxIdleTime(1 * time.Minute)
	db.SetConnMaxLifetime(5 * time.Minute)

	fileServerMuxChain := muxChain{
		limitRequestSizeMiddleware,
		timeoutMiddleware,
		storeClientIPMiddleware,
		checkValidURLMiddleware,
		allowIPRangeMiddleware(appConfig.UIT_LAN_ALLOWED_IP),
		rateLimitMiddleware(appState),
	}

	httpRedirectToHttps := muxChain{
		limitRequestSizeMiddleware,
		timeoutMiddleware,
		storeClientIPMiddleware,
		checkValidURLMiddleware,
		allowIPRangeMiddleware(appConfig.UIT_ALL_ALLOWED_IP),
		rateLimitMiddleware(appState),
	}

	httpMux := http.NewServeMux()
	httpMux.Handle("/client/", fileServerMuxChain.then(fileServerHandler(appState)))
	httpMux.Handle("/client", fileServerMuxChain.thenFunc(rejectRequest))
	httpMux.Handle("/", httpRedirectToHttps.then(redirectToHTTPSHandler("31411")))

	httpServer := &http.Server{
		Addr:         appConfig.UIT_LAN_IP_ADDRESS + ":8080",
		Handler:      httpMux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 1 * time.Minute,
		IdleTimeout:  2 * time.Minute,
	}

	go func() {
		log.Info("HTTP server listening on http://" + appConfig.UIT_LAN_IP_ADDRESS + ":8080")
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error("HTTP server error: " + err.Error())
		}
	}()
	defer httpServer.Close()

	// go func() {
	//   err := http.ListenAndServe("localhost:6060", nil)
	//   if err != nil {
	//     log.Error("Profiler error: " + err.Error())
	//   }
	// }()

	// Route to correct function
	httpsNoAuth := muxChain{
		limitRequestSizeMiddleware,
		timeoutMiddleware,
		storeClientIPMiddleware,
		checkValidURLMiddleware,
		allowIPRangeMiddleware(appConfig.UIT_ALL_ALLOWED_IP),
		rateLimitMiddleware(appState),
		tlsMiddleware,
		httpMethodMiddleware,
		checkHeadersMiddleware,
		setHeadersMiddleware,
	}

	httpsApiAuth := muxChain{
		limitRequestSizeMiddleware,
		timeoutMiddleware,
		storeClientIPMiddleware,
		checkValidURLMiddleware,
		allowIPRangeMiddleware(appConfig.UIT_ALL_ALLOWED_IP),
		rateLimitMiddleware(appState),
		tlsMiddleware,
		httpMethodMiddleware,
		checkHeadersMiddleware,
		setHeadersMiddleware,
		apiAuth,
	}

	httpsCookieAuth := muxChain{
		limitRequestSizeMiddleware,
		timeoutMiddleware,
		storeClientIPMiddleware,
		checkValidURLMiddleware,
		allowIPRangeMiddleware(appConfig.UIT_ALL_ALLOWED_IP),
		rateLimitMiddleware(appState),
		tlsMiddleware,
		httpMethodMiddleware,
		checkHeadersMiddleware,
		setHeadersMiddleware,
		httpCookieAuth,
	}

	httpsMux := http.NewServeMux()
	httpsMux.Handle("GET /api/server_time", httpsApiAuth.thenFunc(getServerTime))
	httpsMux.Handle("GET /api/lookup", httpsApiAuth.thenFunc(getClientLookup))
	httpsMux.Handle("GET /api/client/hardware", httpsApiAuth.thenFunc(getHardwareData))
	httpsMux.Handle("GET /api/client/bios", httpsApiAuth.thenFunc(getBiosData))
	httpsMux.Handle("GET /api/client/os", httpsApiAuth.thenFunc(getOSData))
	httpsMux.Handle("GET /api/job_queue/overview", httpsApiAuth.thenFunc(getJobQueueOverview))
	httpsMux.Handle("GET /api/job_queue/client/queued_job", httpsApiAuth.thenFunc(getClientQueuedJobs))
	httpsMux.Handle("GET /api/job_queue/client/job_available", httpsApiAuth.thenFunc(getClientAvailableJobs))

	httpsMux.Handle("GET /login.html", httpsNoAuth.then(webServerHandler(appState)))
	httpsMux.Handle("POST /login.html", httpsNoAuth.thenFunc(verifyCookieLogin))
	httpsMux.Handle("/js/login.js", httpsNoAuth.then(webServerHandler(appState)))
	httpsMux.Handle("/css/desktop.css", httpsNoAuth.then(webServerHandler(appState)))
	httpsMux.Handle("/favicon.ico", httpsNoAuth.then(webServerHandler(appState)))

	httpsMux.Handle("GET /logout", httpsCookieAuth.then(logoutHandler(appState)))

	httpsMux.Handle("/js/", httpsCookieAuth.then(webServerHandler(appState)))
	httpsMux.Handle("/css/", httpsCookieAuth.then(webServerHandler(appState)))
	httpsMux.Handle("/", httpsCookieAuth.then(webServerHandler(appState)))
	// httpsMux.HandleFunc("/dbstats/", GetInfoHandler)

	log.Info("Starting web server")

	tlsConfig := &tls.Config{
		// MinVersion: tls.VersionTLS12, //0x0303
		MinVersion: tls.VersionTLS13, //0x0304
		CurvePreferences: []tls.CurveID{
			tls.X25519,
			tls.CurveP256,
		},
		CipherSuites: []uint16{
			tls.TLS_AES_128_GCM_SHA256,
			tls.TLS_AES_256_GCM_SHA384,
			tls.TLS_CHACHA20_POLY1305_SHA256,
		},
		PreferServerCipherSuites: true,
		SessionTicketsDisabled:   true,
	}

	httpsServer := http.Server{
		Addr:           ":31411",
		Handler:        httpsMux,
		TLSConfig:      tlsConfig,
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		IdleTimeout:    120 * time.Second,
		MaxHeaderBytes: 1 << 20, // 1MB header size max
	}

	httpsServer.Protocols = new(http.Protocols)
	httpsServer.Protocols.SetHTTP1(false)
	httpsServer.Protocols.SetHTTP2(true)

	log.Info("Web server ready and listening for requests on https://*:31411")

	webCertFile, ok := os.LookupEnv("UIT_TLS_CERT_FILE")
	if !ok {
		log.Error("Error getting UIT_TLS_CERT_FILE: variable not set")
		os.Exit(1)
	}
	webKeyFile, ok := os.LookupEnv("UIT_TLS_KEY_FILE")
	if !ok {
		log.Error("Error getting UIT_TLS_KEY_FILE: variable not set")
		os.Exit(1)
	}

	// Start HTTPS server
	if err := httpsServer.ListenAndServeTLS(webCertFile, webKeyFile); err != nil {
		log.Error("Cannot start web server: " + err.Error())
		os.Exit(1)
	}
	defer httpsServer.Close()
}
