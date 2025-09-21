package config

import (
	"database/sql"
	"errors"
	"log"
	"maps"
	"os"
	"strconv"
	"strings"
	"sync"
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
	limiter  *rate.Limiter
	lastSeen time.Time
}

type LimiterMap struct {
	m     sync.Map
	rate  float64
	burst int
}

type BlockedMap struct {
	m         sync.Map
	banPeriod time.Duration
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
	AllowedFiles      map[string]bool
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
		AuthMapEntryCount: 0,
		Log:               logger.CreateLogger("console", logger.ParseLogLevel(os.Getenv("UIT_API_LOG_LEVEL"))),
		WebServerLimiter:  &LimiterMap{rate: appConfig.UIT_RATE_LIMIT_INTERVAL, burst: appConfig.UIT_RATE_LIMIT_BURST},
		FileLimiter:       &LimiterMap{rate: appConfig.UIT_RATE_LIMIT_INTERVAL / 4, burst: appConfig.UIT_RATE_LIMIT_BURST / 4},
		APILimiter:        &LimiterMap{rate: appConfig.UIT_RATE_LIMIT_INTERVAL, burst: appConfig.UIT_RATE_LIMIT_BURST},
		AuthLimiter:       &LimiterMap{rate: appConfig.UIT_RATE_LIMIT_INTERVAL / 10, burst: appConfig.UIT_RATE_LIMIT_BURST / 10},
		BlockedIPs:        &BlockedMap{banPeriod: appConfig.UIT_RATE_LIMIT_BAN_DURATION},
		AllowedFiles: map[string]bool{
			"filesystem.squashfs":          true,
			"initrd.img":                   true,
			"vmlinuz":                      true,
			"uit-ca.crt":                   true,
			"uit-web.crt":                  true,
			"uit-toolbox-client.deb":       true,
			"desktop.css":                  true,
			"favicon.ico":                  true,
			"header.html":                  true,
			"footer.html":                  true,
			"index.html":                   true,
			"login.html":                   true,
			"auth-webworker.js":            true,
			"footer.js":                    true,
			"header.js":                    true,
			"init.js":                      true,
			"include.js":                   true,
			"login.js":                     true,
			"logout.js":                    true,
			"inventory.html":               true,
			"inventory.js":                 true,
			"checkouts.html":               true,
			"checkouts.js":                 true,
			"job_queue.html":               true,
			"job_queue.js":                 true,
			"reports.html":                 true,
			"reports.js":                   true,
			"go-latest.linux-amd64.tar.gz": true,
		},
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
		maps.Copy(result, appStateInstance.AllowedFiles)
		return result
	}
	return nil
}

func IsFileAllowed(filename string) bool {
	appStateMutex.RLock()
	defer appStateMutex.RUnlock()
	if appStateInstance != nil && appStateInstance.AllowedFiles != nil {
		return appStateInstance.AllowedFiles[filename]
	}
	return false
}

func AddAllowedFile(filename string) {
	appStateMutex.Lock()
	defer appStateMutex.Unlock()
	if appStateInstance != nil && appStateInstance.AllowedFiles != nil {
		appStateInstance.AllowedFiles[filename] = true
	}
}

func RemoveAllowedFile(filename string) {
	appStateMutex.Lock()
	defer appStateMutex.Unlock()
	if appStateInstance != nil && appStateInstance.AllowedFiles != nil {
		delete(appStateInstance.AllowedFiles, filename)
	}
}
