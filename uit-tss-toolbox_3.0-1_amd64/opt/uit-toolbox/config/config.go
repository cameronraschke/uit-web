package config

import (
	"database/sql"
	"errors"
	"log"
	"os"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
	"uit-toolbox/logger"

	"golang.org/x/time/rate"
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

type LimiterEntry struct {
	Limiter  *rate.Limiter
	LastSeen time.Time
}

type LimiterMap struct {
	M     sync.Map
	Rate  float64
	Burst int
}

type BlockedMap struct {
	M         sync.Map
	BanPeriod time.Duration
}

type FileList struct {
	Filename string `json:"filename"`
	Allowed  bool   `json:"allowed"`
}

type AppState struct {
	DB                *sql.DB
	AuthMap           sync.Map
	AuthMapEntryCount int64
	Log               logger.Logger
	WebServerLimiter  *LimiterMap
	FileLimiter       *LimiterMap
	APILimiter        *LimiterMap
	AuthLimiter       *LimiterMap
	BlockedIPs        *BlockedMap
	AllowedFiles      sync.Map
}

type AuthHeader struct {
	Basic  *string
	Bearer *string
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

var (
	rateLimit            float64
	rateLimitBurst       int
	rateLimitBanDuration time.Duration
	WebServerLimiter     *LimiterMap
	blockedIPs           *BlockedMap
	appStateInstance     *AppState
	appStateOnce         sync.Once
	appStateMutex        sync.RWMutex
)

func LoadConfig() (AppConfig, error) {
	// WAN interface, IP, and allowed IPs
	wanIf, ok := os.LookupEnv("UIT_WAN_IF")
	if !ok {
		return AppConfig{}, errors.New("error getting UIT_WAN_IF: not found")
	}
	wanIP, ok := os.LookupEnv("UIT_WAN_IP_ADDRESS")
	if !ok {
		return AppConfig{}, errors.New("error getting UIT_WAN_IP_ADDRESS: not found")
	}
	envWanAllowedIPStr, ok := os.LookupEnv("UIT_WAN_ALLOWED_IP")
	if !ok {
		return AppConfig{}, errors.New("error getting UIT_WAN_ALLOWED_IP: not found")
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
		return AppConfig{}, errors.New("error getting UIT_LAN_IF: not found")
	}
	lanIP, ok := os.LookupEnv("UIT_LAN_IP_ADDRESS")
	if !ok {
		return AppConfig{}, errors.New("error getting UIT_LAN_IP_ADDRESS: not found")
	}
	envAllowedLanIPStr, ok := os.LookupEnv("UIT_LAN_ALLOWED_IP")
	if !ok {
		return AppConfig{}, errors.New("error getting UIT_LAN_ALLOWED_IP: not found")
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
		return AppConfig{}, errors.New("error getting UIT_WEB_SVC_PASSWD: not found")
	}
	uitWebSvcPasswd = strings.TrimSpace(uitWebSvcPasswd)

	dbClientPasswd, ok := os.LookupEnv("UIT_DB_CLIENT_PASSWD")
	if !ok {
		return AppConfig{}, errors.New("error getting UIT_DB_CLIENT_PASSWD: not found")
	}
	webUserDefaultPasswd, ok := os.LookupEnv("UIT_WEB_USER_DEFAULT_PASSWD")
	if !ok {
		return AppConfig{}, errors.New("error getting UIT_WEB_USER_DEFAULT_PASSWD: not found")
	}

	// Website config
	webmasterName, ok := os.LookupEnv("UIT_WEBMASTER_NAME")
	if !ok {
		return AppConfig{}, errors.New("error getting UIT_WEBMASTER_NAME: not found")
	}
	webmasterEmail, ok := os.LookupEnv("UIT_WEBMASTER_EMAIL")
	if !ok {
		return AppConfig{}, errors.New("error getting UIT_WEBMASTER_EMAIL: not found")
	}

	// Printer IP
	printerIP, ok := os.LookupEnv("UIT_PRINTER_IP")
	if !ok {
		return AppConfig{}, errors.New("error getting UIT_PRINTER_IP: not found")
	}

	// Webserver config
	httpPort, ok := os.LookupEnv("UIT_HTTP_PORT")
	if !ok {
		return AppConfig{}, errors.New("error getting UIT_HTTP_PORT: not found")
	}
	httpsPort, ok := os.LookupEnv("UIT_HTTPS_PORT")
	if !ok {
		return AppConfig{}, errors.New("error getting UIT_HTTPS_PORT: not found")
	}
	tlsCertFile, ok := os.LookupEnv("UIT_TLS_CERT_FILE")
	if !ok {
		return AppConfig{}, errors.New("error getting UIT_TLS_CERT_FILE: not found")
	}
	tlsKeyFile, ok := os.LookupEnv("UIT_TLS_KEY_FILE")
	if !ok {
		return AppConfig{}, errors.New("error getting UIT_TLS_KEY_FILE: not found")
	}

	// Rate limiting config
	rateLimitBurstStr, ok := os.LookupEnv("UIT_RATE_LIMIT_BURST")
	if !ok {
		return AppConfig{}, errors.New("error getting UIT_RATE_LIMIT_BURST: not found")
	}
	var rateLimitBurstErr error
	rateLimitBurst, rateLimitBurstErr = strconv.Atoi(rateLimitBurstStr)
	if rateLimitBurstErr != nil || rateLimitBurst <= 0 {
		rateLimitBurst = 100
		return AppConfig{}, errors.New("error converting UIT_RATE_LIMIT_BURST to integer: " + rateLimitBurstErr.Error())
	}
	rateLimitIntervalStr, ok := os.LookupEnv("UIT_RATE_LIMIT_INTERVAL")
	if !ok {
		return AppConfig{}, errors.New("error getting UIT_RATE_LIMIT_INTERVAL: not found")
	}
	var rateLimitErr error
	rateLimit, rateLimitErr = strconv.ParseFloat(rateLimitIntervalStr, 64)
	if rateLimitErr != nil || rateLimit <= 0 {
		rateLimit = 1
		return AppConfig{}, errors.New("error converting UIT_RATE_LIMIT_INTERVAL to float: " + rateLimitErr.Error())
	}
	rateLimitBanDurationStr, ok := os.LookupEnv("UIT_RATE_LIMIT_BAN_DURATION")
	if !ok {
		return AppConfig{}, errors.New("error getting UIT_RATE_LIMIT_BAN_DURATION: not found")
	}
	banDurationInt, err := strconv.ParseInt(rateLimitBanDurationStr, 10, 64)
	if err != nil || banDurationInt <= 0 {
		banDurationInt = 30
		return AppConfig{}, errors.New("error converting UIT_RATE_LIMIT_BAN_DURATION to integer: " + err.Error())
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
	}, nil
}

func InitApp(appConfig AppConfig) (*AppState, error) {
	var dbConn *sql.DB
	var dbErr error
	appStateOnce.Do(func() {
		dbConn, dbErr = NewDBConnection(appConfig)
	})

	if dbErr != nil {
		log.Printf("%s", "Failed to initialize database connection: "+dbErr.Error())
		return nil, dbErr
	}

	appStateMutex.Lock()
	defer appStateMutex.Unlock()

	appState := &AppState{
		DB:                dbConn,
		AuthMap:           sync.Map{},
		AuthMapEntryCount: int64(0),
		Log:               logger.CreateLogger("console", logger.ParseLogLevel(os.Getenv("UIT_API_LOG_LEVEL"))),
		WebServerLimiter:  &LimiterMap{Rate: appConfig.UIT_RATE_LIMIT_INTERVAL, Burst: appConfig.UIT_RATE_LIMIT_BURST},
		FileLimiter:       &LimiterMap{Rate: appConfig.UIT_RATE_LIMIT_INTERVAL / 4, Burst: appConfig.UIT_RATE_LIMIT_BURST / 4},
		APILimiter:        &LimiterMap{Rate: appConfig.UIT_RATE_LIMIT_INTERVAL, Burst: appConfig.UIT_RATE_LIMIT_BURST},
		AuthLimiter:       &LimiterMap{Rate: appConfig.UIT_RATE_LIMIT_INTERVAL / 10, Burst: appConfig.UIT_RATE_LIMIT_BURST / 10},
		BlockedIPs:        &BlockedMap{BanPeriod: appConfig.UIT_RATE_LIMIT_BAN_DURATION},
		AllowedFiles:      sync.Map{},
	}

	allowedFiles := []FileList{
		{Filename: "filesystem.squashfs", Allowed: true},
		{Filename: "initrd.img", Allowed: true},
		{Filename: "vmlinuz", Allowed: true},
		{Filename: "uit-ca.crt", Allowed: true},
		{Filename: "uit-web.crt", Allowed: true},
		{Filename: "uit-toolbox-client.deb", Allowed: true},
		{Filename: "desktop.css", Allowed: true},
		{Filename: "favicon.ico", Allowed: true},
		{Filename: "header.html", Allowed: true},
		{Filename: "footer.html", Allowed: true},
		{Filename: "index.html", Allowed: true},
		{Filename: "login.html", Allowed: true},
		{Filename: "auth-webworker.js", Allowed: true},
		{Filename: "footer.js", Allowed: true},
		{Filename: "header.js", Allowed: true},
		{Filename: "init.js", Allowed: true},
		{Filename: "include.js", Allowed: true},
		{Filename: "login.js", Allowed: true},
		{Filename: "logout.js", Allowed: true},
		{Filename: "inventory.html", Allowed: true},
		{Filename: "inventory.js", Allowed: true},
		{Filename: "checkouts.html", Allowed: true},
		{Filename: "checkouts.js", Allowed: true},
		{Filename: "job_queue.html", Allowed: true},
		{Filename: "job_queue.js", Allowed: true},
		{Filename: "reports.html", Allowed: true},
		{Filename: "reports.js", Allowed: true},
		{Filename: "go-latest.linux-amd64.tar.gz", Allowed: true},
	}

	for _, file := range allowedFiles {
		appState.AllowedFiles.Store(file.Filename, file.Allowed)
	}

	appStateInstance = appState
	return appState, nil
}

func GetLogger() logger.Logger {
	appStateMutex.RLock()
	defer appStateMutex.RUnlock()
	if appStateInstance != nil {
		return appStateInstance.Log
	}
	return nil
}

func SetDatabaseConn(db *sql.DB) {
	appStateMutex.Lock()
	defer appStateMutex.Unlock()
	if appStateInstance != nil {
		appStateInstance.DB = db
	}
}

func GetAllowedFiles() map[string]bool {
	appStateMutex.RLock()
	defer appStateMutex.RUnlock()
	if appStateInstance != nil {
		result := make(map[string]bool)
		appStateInstance.AllowedFiles.Range(func(key, value any) bool {
			keyStr, keyExists := key.(string)
			valueBool, valueExists := value.(bool)
			if keyExists && valueExists {
				result[keyStr] = valueBool
			}
			return true
		})
		return result
	}
	return nil
}

func IsFileAllowed(filename string) bool {
	appStateMutex.RLock()
	defer appStateMutex.RUnlock()
	if appStateInstance == nil {
		return false
	}
	v, ok := appStateInstance.AllowedFiles.Load(filename)
	if !ok {
		return false
	}
	allowed, ok := v.(bool)
	return ok && allowed
}

func AddAllowedFile(filename string) {
	appStateMutex.Lock()
	defer appStateMutex.Unlock()
	if appStateInstance == nil {
		return
	}
	appStateInstance.AllowedFiles.Store(filename, true)
}

func RemoveAllowedFile(filename string) {
	appStateMutex.Lock()
	defer appStateMutex.Unlock()
	if appStateInstance == nil {
		return
	}
	appStateInstance.AllowedFiles.Delete(filename)
}

func GetAuthMap() map[string]AuthSession {
	appStateMutex.RLock()
	defer appStateMutex.RUnlock()
	if appStateInstance != nil {
		result := make(map[string]AuthSession)
		appStateInstance.AuthMap.Range(func(key, value any) bool {
			keyStr, keyExists := key.(string)
			authSession, valueExists := value.(AuthSession)

			if keyExists && valueExists {
				result[keyStr] = authSession
			}
			return true
		})

		return result
	}
	return map[string]AuthSession{}
}

func CreateAuthSession(sessionID string, authSession AuthSession) {
	appStateMutex.Lock()
	defer appStateMutex.Unlock()
	if appStateInstance == nil {
		return
	}
	_, exists := appStateInstance.AuthMap.LoadOrStore(sessionID, authSession)
	if !exists {
		atomic.AddInt64(&appStateInstance.AuthMapEntryCount, 1)
	} else {
		appStateInstance.AuthMap.Store(sessionID, authSession)
	}
}

func DeleteAuthSession(sessionID string) {
	appStateMutex.Lock()
	defer appStateMutex.Unlock()
	if appStateInstance == nil {
		return
	}
	if _, exists := appStateInstance.AuthMap.Load(sessionID); exists {
		appStateInstance.AuthMap.Delete(sessionID)
		newVal := atomic.AddInt64(&appStateInstance.AuthMapEntryCount, -1)
		if newVal < 0 {
			atomic.StoreInt64(&appStateInstance.AuthMapEntryCount, 0)
		}
	}
}

func GetAuthSessionCount() int64 {
	// No need to lock mutex for atomic read
	if appStateInstance == nil {
		return 0
	}
	return atomic.LoadInt64(&appStateInstance.AuthMapEntryCount)
}

func RefreshAndGetAuthSessionCount() int64 {
	if appStateInstance == nil {
		return 0
	}
	var entries int64
	appStateInstance.AuthMap.Range(func(_, _ any) bool {
		entries++
		return true
	})
	atomic.StoreInt64(&appStateInstance.AuthMapEntryCount, entries)
	return entries
}

func CheckAuthSession(sessionID string, ipAddress string, basicToken string, bearerToken string, csrfToken string) (bool, error) {
	appStateMutex.RLock()
	defer appStateMutex.RUnlock()
	if appStateInstance == nil {
		return false, errors.New("app state is not initialized")
	}
	v, exists := appStateInstance.AuthMap.Load(sessionID)
	if !exists {
		return false, nil
	}
	authSession, exists := v.(AuthSession)
	if !exists {
		return false, errors.New("invalid auth session type")
	}

	if authSession.Basic.IP != ipAddress || authSession.Bearer.IP != ipAddress {
		return false, errors.New("IP address mismatch for session ID: " + sessionID)
	}

	if strings.TrimSpace(ipAddress) == "" || strings.TrimSpace(basicToken) == "" || strings.TrimSpace(bearerToken) == "" {
		return false, errors.New("empty IP address or token for session ID: " + sessionID)
	}

	if authSession.Basic.Token != basicToken || authSession.Bearer.Token != bearerToken {
		return false, nil
	}

	return true, nil
}
