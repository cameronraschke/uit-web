package config

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"net"
	"net/netip"
	"os"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
	"uit-toolbox/logger"

	"golang.org/x/time/rate"
)

type ConfigFile struct {
	UIT_SERVER_LOG_LEVEL            string `json:"UIT_SERVER_LOG_LEVEL"`
	UIT_SERVER_ADMIN_PASSWD         string `json:"UIT_SERVER_ADMIN_PASSWD"`
	UIT_SERVER_DB_NAME              string `json:"UIT_SERVER_DB_NAME"`
	UIT_SERVER_HOSTNAME             string `json:"UIT_SERVER_HOSTNAME"`
	UIT_SERVER_WAN_IP_ADDRESS       string `json:"UIT_SERVER_WAN_IP_ADDRESS"`
	UIT_SERVER_LAN_IP_ADDRESS       string `json:"UIT_SERVER_LAN_IP_ADDRESS"`
	UIT_SERVER_WAN_IF               string `json:"UIT_SERVER_WAN_IF"`
	UIT_SERVER_LAN_IF               string `json:"UIT_SERVER_LAN_IF"`
	UIT_SERVER_WAN_ALLOWED_IP       string `json:"UIT_SERVER_WAN_ALLOWED_IP"`
	UIT_SERVER_LAN_ALLOWED_IP       string `json:"UIT_SERVER_LAN_ALLOWED_IP"`
	UIT_WEB_USER_DEFAULT_PASSWD     string `json:"UIT_WEB_USER_DEFAULT_PASSWD"`
	UIT_WEB_DB_USERNAME             string `json:"UIT_WEB_DB_USERNAME"`
	UIT_WEB_DB_PASSWD               string `json:"UIT_WEB_DB_PASSWD"`
	UIT_WEB_DB_NAME                 string `json:"UIT_WEB_DB_NAME"`
	UIT_WEB_DB_HOST                 string `json:"UIT_WEB_DB_HOST"`
	UIT_WEB_DB_PORT                 string `json:"UIT_WEB_DB_PORT"`
	UIT_WEB_HTTP_HOST               string `json:"UIT_WEB_HTTP_HOST"`
	UIT_WEB_HTTP_PORT               string `json:"UIT_WEB_HTTP_PORT"`
	UIT_WEB_HTTPS_HOST              string `json:"UIT_WEB_HTTPS_HOST"`
	UIT_WEB_HTTPS_PORT              string `json:"UIT_WEB_HTTPS_PORT"`
	UIT_WEB_TLS_CERT_FILE           string `json:"UIT_WEB_TLS_CERT_FILE"`
	UIT_WEB_TLS_KEY_FILE            string `json:"UIT_WEB_TLS_KEY_FILE"`
	UIT_WEB_MAX_UPLOAD_SIZE_MB      string `json:"UIT_WEB_MAX_UPLOAD_SIZE_MB"`
	UIT_WEB_API_REQUEST_TIMEOUT     string `json:"UIT_WEB_API_REQUEST_TIMEOUT"`
	UIT_WEB_FILE_REQUEST_TIMEOUT    string `json:"UIT_WEB_FILE_REQUEST_TIMEOUT"`
	UIT_WEB_RATE_LIMIT_BURST        string `json:"UIT_WEB_RATE_LIMIT_BURST"`
	UIT_WEB_RATE_LIMIT_INTERVAL     string `json:"UIT_WEB_RATE_LIMIT_INTERVAL"`
	UIT_WEB_RATE_LIMIT_BAN_DURATION string `json:"UIT_WEB_RATE_LIMIT_BAN_DURATION"`
	UIT_CLIENT_DB_USER              string `json:"UIT_CLIENT_DB_USER"`
	UIT_CLIENT_DB_PASSWD            string `json:"UIT_CLIENT_DB_PASSWD"`
	UIT_CLIENT_DB_NAME              string `json:"UIT_CLIENT_DB_NAME"`
	UIT_CLIENT_DB_HOST              string `json:"UIT_CLIENT_DB_HOST"`
	UIT_CLIENT_DB_PORT              string `json:"UIT_CLIENT_DB_PORT"`
	UIT_CLIENT_NTP_HOST             string `json:"UIT_CLIENT_NTP_HOST"`
	UIT_CLIENT_PING_HOST            string `json:"UIT_CLIENT_PING_HOST"`
	UIT_PRINTER_IP                  string `json:"UIT_PRINTER_IP"`
	UIT_WEBMASTER_NAME              string `json:"UIT_WEBMASTER_NAME"`
	UIT_WEBMASTER_EMAIL             string `json:"UIT_WEBMASTER_EMAIL"`
}

type AppConfig struct {
	UIT_SERVER_LOG_LEVEL            string         `json:"UIT_SERVER_LOG_LEVEL"`
	UIT_SERVER_ADMIN_PASSWD         string         `json:"UIT_SERVER_ADMIN_PASSWD"`
	UIT_SERVER_DB_NAME              string         `json:"UIT_SERVER_DB_NAME"`
	UIT_SERVER_HOSTNAME             string         `json:"UIT_SERVER_HOSTNAME"`
	UIT_SERVER_WAN_IP_ADDRESS       netip.Addr     `json:"UIT_SERVER_WAN_IP_ADDRESS"`
	UIT_SERVER_LAN_IP_ADDRESS       netip.Addr     `json:"UIT_SERVER_LAN_IP_ADDRESS"`
	UIT_SERVER_WAN_IF               string         `json:"UIT_SERVER_WAN_IF"`
	UIT_SERVER_LAN_IF               string         `json:"UIT_SERVER_LAN_IF"`
	UIT_SERVER_WAN_ALLOWED_IP       []netip.Prefix `json:"UIT_SERVER_WAN_ALLOWED_IP"`
	UIT_SERVER_LAN_ALLOWED_IP       []netip.Prefix `json:"UIT_SERVER_LAN_ALLOWED_IP"`
	UIT_SERVER_ANY_ALLOWED_IP       []netip.Prefix `json:"UIT_SERVER_ANY_ALLOWED_IP"`
	UIT_WEB_USER_DEFAULT_PASSWD     string         `json:"UIT_WEB_USER_DEFAULT_PASSWD"`
	UIT_WEB_DB_USERNAME             string         `json:"UIT_WEB_DB_USERNAME"`
	UIT_WEB_DB_PASSWD               string         `json:"UIT_WEB_DB_PASSWD"`
	UIT_WEB_DB_NAME                 string         `json:"UIT_WEB_DB_NAME"`
	UIT_WEB_DB_HOST                 netip.Addr     `json:"UIT_WEB_DB_HOST"`
	UIT_WEB_DB_PORT                 uint16         `json:"UIT_WEB_DB_PORT"`
	UIT_WEB_HTTP_HOST               netip.Addr     `json:"UIT_WEB_HTTP_HOST"`
	UIT_WEB_HTTP_PORT               uint16         `json:"UIT_WEB_HTTP_PORT"`
	UIT_WEB_HTTPS_HOST              netip.Addr     `json:"UIT_WEB_HTTPS_HOST"`
	UIT_WEB_HTTPS_PORT              uint16         `json:"UIT_WEB_HTTPS_PORT"`
	UIT_WEB_TLS_CERT_FILE           string         `json:"UIT_WEB_TLS_CERT_FILE"`
	UIT_WEB_TLS_KEY_FILE            string         `json:"UIT_WEB_TLS_KEY_FILE"`
	UIT_WEB_MAX_UPLOAD_SIZE_MB      int64          `json:"UIT_WEB_MAX_UPLOAD_SIZE_MB"`
	UIT_WEB_API_REQUEST_TIMEOUT     time.Duration  `json:"UIT_WEB_API_REQUEST_TIMEOUT"`
	UIT_WEB_FILE_REQUEST_TIMEOUT    time.Duration  `json:"UIT_WEB_FILE_REQUEST_TIMEOUT"`
	UIT_WEB_RATE_LIMIT_BURST        int            `json:"UIT_WEB_RATE_LIMIT_BURST"`
	UIT_WEB_RATE_LIMIT_INTERVAL     float64        `json:"UIT_WEB_RATE_LIMIT_INTERVAL"`
	UIT_WEB_RATE_LIMIT_BAN_DURATION time.Duration  `json:"UIT_WEB_RATE_LIMIT_BAN_DURATION"`
	UIT_CLIENT_DB_USER              string         `json:"UIT_CLIENT_DB_USER"`
	UIT_CLIENT_DB_PASSWD            string         `json:"UIT_CLIENT_DB_PASSWD"`
	UIT_CLIENT_DB_NAME              string         `json:"UIT_CLIENT_DB_NAME"`
	UIT_CLIENT_DB_HOST              netip.Addr     `json:"UIT_CLIENT_DB_HOST"`
	UIT_CLIENT_DB_PORT              uint16         `json:"UIT_CLIENT_DB_PORT"`
	UIT_CLIENT_NTP_HOST             netip.Addr     `json:"UIT_CLIENT_NTP_HOST"`
	UIT_CLIENT_PING_HOST            netip.Addr     `json:"UIT_CLIENT_PING_HOST"`
	UIT_PRINTER_IP                  netip.Addr     `json:"UIT_PRINTER_IP"`
	UIT_WEBMASTER_NAME              string         `json:"UIT_WEBMASTER_NAME"`
	UIT_WEBMASTER_EMAIL             string         `json:"UIT_WEBMASTER_EMAIL"`
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

type ClientConfig struct {
	UIT_CLIENT_DB_USER   string `json:"UIT_CLIENT_DB_USER"`
	UIT_CLIENT_DB_PASSWD string `json:"UIT_CLIENT_DB_PASSWD"`
	UIT_CLIENT_DB_NAME   string `json:"UIT_CLIENT_DB_NAME"`
	UIT_CLIENT_DB_HOST   string `json:"UIT_CLIENT_DB_HOST"`
	UIT_CLIENT_DB_PORT   string `json:"UIT_CLIENT_DB_PORT"`
	UIT_CLIENT_NTP_HOST  string `json:"UIT_CLIENT_NTP_HOST"`
	UIT_CLIENT_PING_HOST string `json:"UIT_CLIENT_PING_HOST"`
	UIT_SERVER_HOSTNAME  string `json:"UIT_SERVER_HOSTNAME"`
	UIT_WEB_HTTP_HOST    string `json:"UIT_WEB_HTTP_HOST"`
	UIT_WEB_HTTP_PORT    string `json:"UIT_WEB_HTTP_PORT"`
	UIT_WEB_HTTPS_HOST   string `json:"UIT_WEB_HTTPS_HOST"`
	UIT_WEB_HTTPS_PORT   string `json:"UIT_WEB_HTTPS_PORT"`
	UIT_WEBMASTER_NAME   string `json:"UIT_WEBMASTER_NAME"`
}

type FileList struct {
	Filename string `json:"filename"`
	Allowed  bool   `json:"allowed"`
}

type AppState struct {
	AppConfig          *AppConfig
	DBConn             atomic.Pointer[sql.DB]
	AuthMap            sync.Map
	AuthMapEntryCount  atomic.Int64
	Log                logger.Logger
	WebServerLimiter   *LimiterMap
	FileLimiter        *LimiterMap
	APILimiter         *LimiterMap
	AuthLimiter        *LimiterMap
	BlockedIPs         *BlockedMap
	AllowedFiles       atomic.Value
	AllowedFilesMu     sync.Mutex
	AllowedWANIPs      sync.Map
	AllowedLANIPs      sync.Map
	AllowedIPs         sync.Map
	SessionSecret      []byte
	APIRequestTimeout  atomic.Value
	FileRequestTimeout atomic.Value
}

type AuthHTTPHeader struct {
	CSRFToken   *string
	BasicToken  *string
	BearerToken *string
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

type CSRFToken struct {
	Token     string    `json:"token"`
	Expiry    time.Time `json:"expiry"`
	NotBefore time.Time `json:"not_before"`
	TTL       float64   `json:"ttl"`
	IP        string    `json:"ip"`
	Valid     bool      `json:"valid"`
}

type AuthSession struct {
	SessionID string
	Basic     BasicToken
	Bearer    BearerToken
	CSRF      CSRFToken
}

var (
	appStateInstance *AppState
	appStateMutex    sync.RWMutex
	defaultLogger    logger.Logger = logger.CreateLogger("console", logger.ParseLogLevel(os.Getenv("UIT_SERVER_LOG_LEVEL")))
)

func LoadConfig() (*AppConfig, error) {
	var appConfig AppConfig
	var configFile ConfigFile

	// Decode JSON
	file, err := os.ReadFile("/etc/uit-toolbox/uit-toolbox.json")
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}
	if err := json.Unmarshal(file, &configFile); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config JSON: %w", err)
	}

	// Server section
	appConfig.UIT_SERVER_LOG_LEVEL = configFile.UIT_SERVER_LOG_LEVEL
	appConfig.UIT_SERVER_ADMIN_PASSWD = configFile.UIT_SERVER_ADMIN_PASSWD
	appConfig.UIT_SERVER_DB_NAME = configFile.UIT_SERVER_DB_NAME
	appConfig.UIT_SERVER_HOSTNAME = configFile.UIT_SERVER_HOSTNAME

	// WAN interface, IP, and allowed IPs
	wanIPAddr, err := netip.ParseAddr(configFile.UIT_SERVER_WAN_IP_ADDRESS)
	if err != nil {
		return nil, fmt.Errorf("invalid UIT_SERVER_WAN_IP_ADDRESS: %w", err)
	}
	appConfig.UIT_SERVER_WAN_IP_ADDRESS = wanIPAddr

	lanIPAddr, err := netip.ParseAddr(configFile.UIT_SERVER_LAN_IP_ADDRESS)
	if err != nil {
		return nil, fmt.Errorf("invalid UIT_SERVER_LAN_IP_ADDRESS: %w", err)
	}
	appConfig.UIT_SERVER_LAN_IP_ADDRESS = lanIPAddr

	appConfig.UIT_SERVER_WAN_IF = configFile.UIT_SERVER_WAN_IF
	appConfig.UIT_SERVER_LAN_IF = configFile.UIT_SERVER_LAN_IF

	for wanIPStr := range strings.SplitSeq(configFile.UIT_SERVER_WAN_ALLOWED_IP, ",") {
		ipStr := strings.TrimSpace(wanIPStr)
		if ipStr == "" {
			continue
		}
		ip, err := netip.ParsePrefix(ipStr)
		if err != nil {
			fmt.Println("Skipping invalid UIT_SERVER_WAN_ALLOWED_IP entry: " + ipStr)
		} else {
			appConfig.UIT_SERVER_WAN_ALLOWED_IP = append(appConfig.UIT_SERVER_WAN_ALLOWED_IP, ip)
			appConfig.UIT_SERVER_ANY_ALLOWED_IP = append(appConfig.UIT_SERVER_ANY_ALLOWED_IP, ip)
		}
	}

	for lanIPStr := range strings.SplitSeq(configFile.UIT_SERVER_LAN_ALLOWED_IP, ",") {
		ipStr := strings.TrimSpace(lanIPStr)
		if ipStr == "" {
			continue
		}
		ip, err := netip.ParsePrefix(ipStr)
		if err != nil {
			fmt.Println("Skipping invalid UIT_SERVER_LAN_ALLOWED_IP entry: " + ipStr)
		} else {
			appConfig.UIT_SERVER_LAN_ALLOWED_IP = append(appConfig.UIT_SERVER_LAN_ALLOWED_IP, ip)
			appConfig.UIT_SERVER_ANY_ALLOWED_IP = append(appConfig.UIT_SERVER_ANY_ALLOWED_IP, ip)
		}
	}

	// Webserver section
	appConfig.UIT_WEB_USER_DEFAULT_PASSWD = configFile.UIT_WEB_USER_DEFAULT_PASSWD
	appConfig.UIT_WEB_DB_USERNAME = configFile.UIT_WEB_DB_USERNAME
	appConfig.UIT_WEB_DB_PASSWD = configFile.UIT_WEB_DB_PASSWD
	appConfig.UIT_WEB_DB_NAME = configFile.UIT_WEB_DB_NAME

	dbHostAddr, err := netip.ParseAddr(configFile.UIT_WEB_DB_HOST)
	if err != nil {
		return nil, fmt.Errorf("invalid UIT_WEB_DB_HOST: %w", err)
	}
	appConfig.UIT_WEB_DB_HOST = dbHostAddr

	dbPortAddr, err := netip.ParseAddrPort(configFile.UIT_WEB_DB_HOST + ":" + configFile.UIT_WEB_DB_PORT)
	if err != nil {
		return nil, fmt.Errorf("invalid UIT_WEB_DB_PORT: %w", err)
	}
	appConfig.UIT_WEB_DB_PORT = dbPortAddr.Port()

	httpHostAddr, err := netip.ParseAddr(configFile.UIT_WEB_HTTP_HOST)
	if err != nil {
		return nil, fmt.Errorf("invalid UIT_WEB_HTTP_HOST: %w", err)
	}
	appConfig.UIT_WEB_HTTP_HOST = httpHostAddr

	httpPortAddr, err := netip.ParseAddrPort(configFile.UIT_WEB_HTTP_HOST + ":" + configFile.UIT_WEB_HTTP_PORT)
	if err != nil {
		return nil, fmt.Errorf("invalid UIT_WEB_HTTP_PORT: %w", err)
	}
	appConfig.UIT_WEB_HTTP_PORT = httpPortAddr.Port()

	httpsHostAddr, err := netip.ParseAddr(configFile.UIT_WEB_HTTPS_HOST)
	if err != nil {
		return nil, fmt.Errorf("invalid UIT_WEB_HTTPS_HOST: %w", err)
	}
	appConfig.UIT_WEB_HTTPS_HOST = httpsHostAddr

	httpsPortAddr, err := netip.ParseAddrPort(configFile.UIT_WEB_HTTPS_HOST + ":" + configFile.UIT_WEB_HTTPS_PORT)
	if err != nil {
		return nil, fmt.Errorf("invalid UIT_WEB_HTTPS_PORT: %w", err)
	}
	appConfig.UIT_WEB_HTTPS_PORT = httpsPortAddr.Port()
	appConfig.UIT_WEB_MAX_UPLOAD_SIZE_MB, err = strconv.ParseInt(configFile.UIT_WEB_MAX_UPLOAD_SIZE_MB, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid UIT_WEB_MAX_UPLOAD_SIZE_MB: %w", err)
	}
	requestAPITimeoutSeconds, err := strconv.ParseInt(configFile.UIT_WEB_API_REQUEST_TIMEOUT, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid UIT_WEB_API_REQUEST_TIMEOUT: %w", err)
	}
	appConfig.UIT_WEB_API_REQUEST_TIMEOUT = time.Duration(requestAPITimeoutSeconds) * time.Second
	requestFileTimeoutSeconds, err := strconv.ParseInt(configFile.UIT_WEB_FILE_REQUEST_TIMEOUT, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid UIT_WEB_FILE_REQUEST_TIMEOUT: %w", err)
	}
	appConfig.UIT_WEB_FILE_REQUEST_TIMEOUT = time.Duration(requestFileTimeoutSeconds) * time.Second
	appConfig.UIT_WEB_TLS_CERT_FILE = configFile.UIT_WEB_TLS_CERT_FILE
	appConfig.UIT_WEB_TLS_KEY_FILE = configFile.UIT_WEB_TLS_KEY_FILE

	rateLimitBurst, err := strconv.Atoi(configFile.UIT_WEB_RATE_LIMIT_BURST)
	if err != nil {
		return nil, fmt.Errorf("invalid UIT_WEB_RATE_LIMIT_BURST: %w", err)
	}
	appConfig.UIT_WEB_RATE_LIMIT_BURST = rateLimitBurst

	rateLimitInterval, err := strconv.ParseFloat(configFile.UIT_WEB_RATE_LIMIT_INTERVAL, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid UIT_WEB_RATE_LIMIT_INTERVAL: %w", err)
	}
	appConfig.UIT_WEB_RATE_LIMIT_INTERVAL = rateLimitInterval

	banDurationSeconds, err := strconv.ParseInt(configFile.UIT_WEB_RATE_LIMIT_BAN_DURATION, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid UIT_WEB_RATE_LIMIT_BAN_DURATION: %w", err)
	}
	appConfig.UIT_WEB_RATE_LIMIT_BAN_DURATION = time.Duration(banDurationSeconds) * time.Second

	// Client section
	appConfig.UIT_CLIENT_DB_USER = configFile.UIT_CLIENT_DB_USER
	appConfig.UIT_CLIENT_DB_PASSWD = configFile.UIT_CLIENT_DB_PASSWD
	appConfig.UIT_CLIENT_DB_NAME = configFile.UIT_CLIENT_DB_NAME

	clientDBHostAddr, err := netip.ParseAddr(configFile.UIT_CLIENT_DB_HOST)
	if err != nil {
		return nil, fmt.Errorf("invalid UIT_CLIENT_DB_HOST: %w", err)
	}
	appConfig.UIT_CLIENT_DB_HOST = clientDBHostAddr

	clientDBPortAddr, err := netip.ParseAddrPort(configFile.UIT_CLIENT_DB_HOST + ":" + configFile.UIT_CLIENT_DB_PORT)
	if err != nil {
		return nil, fmt.Errorf("invalid UIT_CLIENT_DB_PORT: %w", err)
	}
	appConfig.UIT_CLIENT_DB_PORT = clientDBPortAddr.Port()

	clientNTPHostAddr, err := netip.ParseAddr(configFile.UIT_CLIENT_NTP_HOST)
	if err != nil {
		return nil, fmt.Errorf("invalid UIT_CLIENT_NTP_HOST: %w", err)
	}
	appConfig.UIT_CLIENT_NTP_HOST = clientNTPHostAddr

	clientPingHostAddr, err := netip.ParseAddr(configFile.UIT_CLIENT_PING_HOST)
	if err != nil {
		return nil, fmt.Errorf("invalid UIT_CLIENT_PING_HOST: %w", err)
	}
	appConfig.UIT_CLIENT_PING_HOST = clientPingHostAddr

	// Printer IP
	printerIPAddr, err := netip.ParseAddr(configFile.UIT_PRINTER_IP)
	if err != nil {
		return nil, fmt.Errorf("invalid UIT_PRINTER_IP: %w", err)
	}
	appConfig.UIT_PRINTER_IP = printerIPAddr

	// Webmaster email
	appConfig.UIT_WEBMASTER_NAME = configFile.UIT_WEBMASTER_NAME
	appConfig.UIT_WEBMASTER_EMAIL = configFile.UIT_WEBMASTER_EMAIL

	return &appConfig, nil
}

func InitApp() (*AppState, error) {
	appConfig, err := LoadConfig()
	if err != nil || appConfig == nil {
		return nil, errors.New("failed to load app config: " + err.Error())
	}

	appState := &AppState{
		AppConfig:         appConfig,
		DBConn:            atomic.Pointer[sql.DB]{},
		AuthMap:           sync.Map{},
		AuthMapEntryCount: atomic.Int64{},
		Log:               logger.CreateLogger("console", logger.ParseLogLevel(os.Getenv("UIT_SERVER_LOG_LEVEL"))),
		WebServerLimiter:  &LimiterMap{M: sync.Map{}, Rate: appConfig.UIT_WEB_RATE_LIMIT_INTERVAL, Burst: appConfig.UIT_WEB_RATE_LIMIT_BURST},
		FileLimiter:       &LimiterMap{M: sync.Map{}, Rate: appConfig.UIT_WEB_RATE_LIMIT_INTERVAL / 4, Burst: appConfig.UIT_WEB_RATE_LIMIT_BURST / 4},
		APILimiter:        &LimiterMap{M: sync.Map{}, Rate: appConfig.UIT_WEB_RATE_LIMIT_INTERVAL, Burst: appConfig.UIT_WEB_RATE_LIMIT_BURST},
		AuthLimiter:       &LimiterMap{M: sync.Map{}, Rate: appConfig.UIT_WEB_RATE_LIMIT_INTERVAL / 10, Burst: appConfig.UIT_WEB_RATE_LIMIT_BURST / 10},
		BlockedIPs:        &BlockedMap{M: sync.Map{}, BanPeriod: appConfig.UIT_WEB_RATE_LIMIT_BAN_DURATION},
		AllowedWANIPs:     sync.Map{},
		AllowedLANIPs:     sync.Map{},
		AllowedIPs:        sync.Map{},
	}

	allowedFiles := []FileList{
		{Filename: "filesystem.squashfs", Allowed: true},
		{Filename: "initrd.img", Allowed: true},
		{Filename: "vmlinuz", Allowed: true},
		{Filename: "uit-ca.crt", Allowed: true},
		{Filename: "uit-web.crt", Allowed: true},
		{Filename: "uit-toolbox-client.deb", Allowed: true},
		{Filename: "desktop.css", Allowed: true},
		{Filename: "favicon.png", Allowed: true},
		{Filename: "favicon.ico", Allowed: true},
		{Filename: "header.html", Allowed: true},
		{Filename: "footer.html", Allowed: true},
		{Filename: "dashboard.html", Allowed: true},
		{Filename: "dashboard.js", Allowed: true},
		{Filename: "dashboard.css", Allowed: true},
		{Filename: "login.html", Allowed: true},
		{Filename: "login.css", Allowed: true},
		{Filename: "login.js", Allowed: true},
		{Filename: "auth-webworker.js", Allowed: true},
		{Filename: "footer.js", Allowed: true},
		{Filename: "header.js", Allowed: true},
		{Filename: "init.js", Allowed: true},
		{Filename: "include.js", Allowed: true},
		{Filename: "login.js", Allowed: true},
		{Filename: "logout.js", Allowed: true},
		{Filename: "inventory.html", Allowed: true},
		{Filename: "inventory.js", Allowed: true},
		{Filename: "inventory_table.js", Allowed: true},
		{Filename: "inventory.css", Allowed: true},
		{Filename: "client_images.html", Allowed: true},
		{Filename: "client_images.js", Allowed: true},
		{Filename: "client_images.css", Allowed: true},
		{Filename: "checkouts.html", Allowed: true},
		{Filename: "checkouts.js", Allowed: true},
		{Filename: "job_queue.html", Allowed: true},
		{Filename: "job_queue.js", Allowed: true},
		{Filename: "reports.html", Allowed: true},
		{Filename: "reports.js", Allowed: true},
		{Filename: "go-latest.linux-amd64.tar.gz", Allowed: true},
		{Filename: "uit-client.conf", Allowed: true},
	}

	allowed := make(map[string]bool, len(allowedFiles))
	for _, file := range allowedFiles {
		allowed[file.Filename] = file.Allowed
	}
	appState.AllowedFiles.Store(allowed)

	for _, wanIP := range appConfig.UIT_SERVER_WAN_ALLOWED_IP {
		appState.AllowedWANIPs.Store(wanIP, true)
	}

	for _, lanIP := range appConfig.UIT_SERVER_LAN_ALLOWED_IP {
		appState.AllowedLANIPs.Store(lanIP, true)
	}

	for _, allIP := range appConfig.UIT_SERVER_ANY_ALLOWED_IP {
		appState.AllowedIPs.Store(allIP, true)
	}

	// Generate server-side secret for HMAC
	sessionSecret, err := GenerateSessionToken(64)
	if err != nil {
		return nil, fmt.Errorf("failed to generate session secret: %w", err)
	}
	appState.SessionSecret = []byte(sessionSecret)

	// Set initial timeouts
	appState.APIRequestTimeout.Store(appConfig.UIT_WEB_API_REQUEST_TIMEOUT)
	appState.FileRequestTimeout.Store(appConfig.UIT_WEB_FILE_REQUEST_TIMEOUT)

	SetAppState(appState)
	return appState, nil
}

// App state management
func SetAppState(newState *AppState) {
	appStateMutex.Lock()
	defer appStateMutex.Unlock()
	appStateInstance = newState
}

func GetAppState() *AppState {
	appStateMutex.RLock()
	defer appStateMutex.RUnlock()
	return appStateInstance
}

// Logger access
func GetLogger() logger.Logger {
	appStateMutex.RLock()
	asi := appStateInstance
	defer appStateMutex.RUnlock()
	if asi == nil || asi.Log == nil {
		fmt.Println("Logger not initialized, using default logger")
		return defaultLogger
	}
	return asi.Log
}

// Database managment
func GetDatabaseCredentials() (dbName string, dbHost string, dbPort string, dbUsername string, dbPassword string, err error) {
	appState := GetAppState()
	if appState == nil {
		return "", "", "", "", "", errors.New("app state is not initialized")
	}
	return appState.AppConfig.UIT_WEB_DB_NAME, appState.AppConfig.UIT_WEB_DB_HOST.String(), strconv.FormatUint(uint64(appState.AppConfig.UIT_WEB_DB_PORT), 10), appState.AppConfig.UIT_WEB_DB_USERNAME, appState.AppConfig.UIT_WEB_DB_PASSWD, nil
}

func GetWebServerUserDBCredentials() (dbName string, dbHost string, dbPort string, dbUsername string, dbPassword string, err error) {
	appState := GetAppState()
	if appState == nil {
		return "", "", "", "", "", errors.New("app state is not initialized")
	}
	return appState.AppConfig.UIT_WEB_DB_NAME, appState.AppConfig.UIT_WEB_DB_HOST.String(), strconv.FormatUint(uint64(appState.AppConfig.UIT_WEB_DB_PORT), 10), appState.AppConfig.UIT_WEB_DB_USERNAME, appState.AppConfig.UIT_WEB_DB_PASSWD, nil
}

func GetDatabaseConn() *sql.DB {
	appState := GetAppState()
	if appState == nil {
		return nil
	}
	return appState.DBConn.Load()
}

func SetDatabaseConn(newDbConn *sql.DB) {
	appState := GetAppState()
	if appState != nil {
		appState.DBConn.Store(newDbConn)
	}
}

func SwapDatabaseConn(newDbConn *sql.DB) (oldDbConn *sql.DB) {
	appState := GetAppState()
	if appState == nil {
		return nil
	}
	return appState.DBConn.Swap(newDbConn)
}

// Allowed file checks
func GetAllowedFiles() map[string]bool {
	appState := GetAppState()
	if appState == nil {
		return nil
	}
	allowedFiles, _ := appState.AllowedFiles.Load().(map[string]bool)
	if allowedFiles == nil {
		return nil
	}

	return maps.Clone(allowedFiles)
}

func IsFileAllowed(filename string) bool {
	appState := GetAppState()
	if appState == nil || filename == "" {
		return false
	}
	cur, _ := appState.AllowedFiles.Load().(map[string]bool)
	if cur == nil {
		return false
	}
	allowed, ok := cur[filename]
	return ok && allowed
}

func AddAllowedFile(filename string) error {
	appState := GetAppState()
	if appState == nil {
		return errors.New("app state is not initialized")
	}

	appState.AllowedFilesMu.Lock()
	defer appState.AllowedFilesMu.Unlock()

	oldMap, _ := appState.AllowedFiles.Load().(map[string]bool)
	if oldMap == nil {
		oldMap = map[string]bool{}
	}
	if oldMap[filename] {
		return nil
	}

	newMap := make(map[string]bool, len(oldMap)+1)
	maps.Copy(newMap, oldMap)
	newMap[filename] = true
	appState.AllowedFiles.Store(newMap)
	return nil
}

func RemoveAllowedFile(filename string) {
	appState := GetAppState()
	if appState == nil {
		return
	}

	appState.AllowedFilesMu.Lock()
	defer appState.AllowedFilesMu.Unlock()

	oldMap, _ := appState.AllowedFiles.Load().(map[string]bool)
	if oldMap == nil {
		return
	}
	if _, exists := oldMap[filename]; !exists {
		return
	}
	newMap := make(map[string]bool, len(oldMap)-1)
	for k, v := range oldMap {
		if k != filename {
			newMap[k] = v
		}
	}
	appState.AllowedFiles.Store(newMap)
}

// IP address checks
func IsIPAllowed(trafficType string, ip string) (allowed bool, err error) {
	appState := GetAppState()
	if appState == nil {
		return false, fmt.Errorf("app state is not initialized")
	}
	ipAddr, err := netip.ParseAddr(ip)
	if err != nil {
		return false, fmt.Errorf("invalid IP address: %w", err)
	}

	allowed = false
	switch trafficType {
	case "wan":
		appState.AllowedWANIPs.Range(func(k, v any) bool {
			ipRange, ok := k.(netip.Prefix)
			if !ok || ipRange == (netip.Prefix{}) {
				return true
			}
			if ipRange.Contains(ipAddr) {
				allowed = true
				return false
			}
			return true
		})
	case "lan":
		appState.AllowedLANIPs.Range(func(k, v any) bool {
			ipRange, ok := k.(netip.Prefix)
			if !ok || ipRange == (netip.Prefix{}) {
				return true
			}
			if ipRange.Contains(ipAddr) {
				allowed = true
				return false
			}
			return true
		})
	case "any":
		appState.AllowedIPs.Range(func(k, v any) bool {
			ipRange, ok := k.(netip.Prefix)
			if !ok || ipRange == (netip.Prefix{}) {
				return true
			}
			if ipRange.Contains(ipAddr) {
				allowed = true
				return false
			}
			return true
		})
	default:
		return false, errors.New("invalid traffic type, must be 'wan', 'lan', or 'any'")
	}
	return allowed, nil
}

func IsIPBlocked(ipAddress string) bool {
	appState := GetAppState()
	if appState == nil {
		return false
	}
	value, ok := appState.BlockedIPs.M.Load(ipAddress)
	if !ok {
		return false
	}
	blockedEntry, ok := value.(LimiterEntry)
	if !ok {
		return false
	}
	if time.Now().Before(blockedEntry.LastSeen.Add(appState.BlockedIPs.BanPeriod)) {
		return true
	}
	appState.BlockedIPs.M.Delete(ipAddress)
	return false
}

func CleanupBlockedIPs() {
	appState := GetAppState()
	if appState == nil {
		return
	}
	now := time.Now()
	appState.BlockedIPs.M.Range(func(key, value any) bool {
		blockedEntry, ok := value.(LimiterEntry)
		if !ok {
			return true
		}
		if now.After(blockedEntry.LastSeen.Add(appState.BlockedIPs.BanPeriod)) {
			appState.BlockedIPs.M.Delete(key)
		}
		return true
	})
}

// Webserver config
func GetWebServerIPs() (string, string, error) {
	appState := GetAppState()
	if appState == nil {
		return "", "", errors.New("app state is not initialized")
	}
	return appState.AppConfig.UIT_WEB_HTTP_HOST.String(), appState.AppConfig.UIT_WEB_HTTPS_HOST.String(), nil
}

func GetServerIPAddressByInterface(ifName string) (string, error) {
	if ifName == "" {
		return "", errors.New("interface name is empty")
	}
	iface, err := net.InterfaceByName(ifName)
	if err != nil {
		return "", fmt.Errorf("failed to get interface %s: %w", ifName, err)
	}
	addrs, err := iface.Addrs()
	if err != nil {
		return "", fmt.Errorf("failed to get addresses for interface %s: %w", ifName, err)
	}
	for _, addr := range addrs {
		var ip net.IP
		switch v := addr.(type) {
		case *net.IPNet:
			ip = v.IP
		case *net.IPAddr:
			ip = v.IP
		}
		if ip != nil {
			return ip.String(), nil
		}
	}
	return "", fmt.Errorf("no valid IP address found for interface %s", ifName)
}

func GetWebmasterContact() (string, string, error) {
	appState := GetAppState()
	if appState == nil {
		return "", "", errors.New("app state is not initialized")
	}
	return appState.AppConfig.UIT_WEBMASTER_NAME, appState.AppConfig.UIT_WEBMASTER_EMAIL, nil
}

func GetClientConfig() (*ClientConfig, error) {
	appState := GetAppState()
	if appState == nil {
		return nil, errors.New("app state is not initialized")
	}
	clientConfig := &ClientConfig{
		UIT_CLIENT_DB_USER:   appState.AppConfig.UIT_CLIENT_DB_USER,
		UIT_CLIENT_DB_PASSWD: appState.AppConfig.UIT_CLIENT_DB_PASSWD,
		UIT_CLIENT_DB_NAME:   appState.AppConfig.UIT_CLIENT_DB_NAME,
		UIT_CLIENT_DB_HOST:   appState.AppConfig.UIT_CLIENT_DB_HOST.String(),
		UIT_CLIENT_DB_PORT:   strconv.FormatUint(uint64(appState.AppConfig.UIT_CLIENT_DB_PORT), 10),
		UIT_CLIENT_NTP_HOST:  appState.AppConfig.UIT_CLIENT_NTP_HOST.String(),
		UIT_CLIENT_PING_HOST: appState.AppConfig.UIT_CLIENT_PING_HOST.String(),
		UIT_SERVER_HOSTNAME:  appState.AppConfig.UIT_SERVER_HOSTNAME,
		UIT_WEB_HTTP_HOST:    appState.AppConfig.UIT_WEB_HTTP_HOST.String(),
		UIT_WEB_HTTP_PORT:    strconv.FormatUint(uint64(appState.AppConfig.UIT_WEB_HTTP_PORT), 10),
		UIT_WEB_HTTPS_HOST:   appState.AppConfig.UIT_WEB_HTTPS_HOST.String(),
		UIT_WEB_HTTPS_PORT:   strconv.FormatUint(uint64(appState.AppConfig.UIT_WEB_HTTPS_PORT), 10),
		UIT_WEBMASTER_NAME:   appState.AppConfig.UIT_WEBMASTER_NAME,
	}
	return clientConfig, nil
}

func GetTLSCertFiles() (certFile string, keyFile string, err error) {
	appState := GetAppState()
	if appState == nil {
		return "", "", fmt.Errorf("%s", "cannot retrieve TLS cert files, app state is not initialized")
	}
	return appState.AppConfig.UIT_WEB_TLS_CERT_FILE, appState.AppConfig.UIT_WEB_TLS_KEY_FILE, nil
}

func GetMaxUploadSize() (int64, error) {
	appState := GetAppState()
	if appState == nil {
		return 0, fmt.Errorf("%s", "cannot retrieve max upload size, app state is not initialized")
	}
	return appState.AppConfig.UIT_WEB_MAX_UPLOAD_SIZE_MB, nil
}

func GetRequestTimeout(timeoutType string) (time.Duration, error) {
	appState := GetAppState()
	if appState == nil {
		return 0, fmt.Errorf("%s", "cannot get request timeout, app state is not initialized")
	}
	switch strings.ToLower(timeoutType) {
	case "api":
		timeout, ok := appState.APIRequestTimeout.Load().(time.Duration)
		if !ok {
			return 0, fmt.Errorf("%s", "cannot get API request timeout, invalid type stored")
		}
		return timeout, nil
	case "file":
		timeout, ok := appState.FileRequestTimeout.Load().(time.Duration)
		if !ok {
			return 0, fmt.Errorf("%s", "cannot get file request timeout, invalid type stored")
		}
		return timeout, nil
	default:
		return 0, fmt.Errorf("invalid timeout type: %s", timeoutType)
	}
}

func SetRequestTimeout(timeoutType string, timeout time.Duration) error {
	appState := GetAppState()
	if appState == nil {
		return fmt.Errorf("%s", "cannot set request timeout, app state is not initialized")
	}
	if timeout <= 0 {
		return fmt.Errorf("invalid timeout value: %v", timeout)
	}
	switch strings.ToLower(timeoutType) {
	case "api":
		appState.APIRequestTimeout.Store(timeout)
		return nil
	case "file":
		appState.FileRequestTimeout.Store(timeout)
		return nil
	default:
		return fmt.Errorf("invalid timeout type: %s", timeoutType)
	}
}
