package config

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
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

type ClientLimiter struct {
	IPAddr   netip.Addr
	Limiter  *rate.Limiter
	LastSeen time.Time
}

type RateLimiter struct {
	Type      string
	ClientMap sync.Map // map[netip.Addr]ClientLimiter
	Rate      float64
	Burst     int
}

var (
	webRateLimiter  atomic.Pointer[RateLimiter]
	apiRateLimiter  atomic.Pointer[RateLimiter]
	authRateLimiter atomic.Pointer[RateLimiter]
	fileRateLimiter atomic.Pointer[RateLimiter]
	blockedClients  atomic.Pointer[BlockedClients]
)

type BlockedClients struct {
	ClientMap sync.Map // map[netip.Addr]ClientLimiter
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
	UIT_WEBMASTER_EMAIL  string `json:"UIT_WEBMASTER_EMAIL"`
}

type FileList struct {
	Filename string `json:"filename"`
	Allowed  bool   `json:"allowed"`
}

type AppState struct {
	AppConfig          atomic.Pointer[AppConfig]
	DBConn             atomic.Pointer[sql.DB]
	AuthMap            sync.Map
	AuthMapEntryCount  atomic.Int64
	Log                atomic.Pointer[logger.Logger]
	WebServerLimiter   atomic.Pointer[RateLimiter]
	FileLimiter        atomic.Pointer[RateLimiter]
	APILimiter         atomic.Pointer[RateLimiter]
	AuthLimiter        atomic.Pointer[RateLimiter]
	BlockedIPs         atomic.Pointer[BlockedClients]
	AllowedWANIPs      sync.Map
	AllowedLANIPs      sync.Map
	AllowedIPs         sync.Map
	SessionSecret      []byte
	APIRequestTimeout  atomic.Pointer[time.Duration]
	FileRequestTimeout atomic.Pointer[time.Duration]
	WebEndpoints       sync.Map
	GroupPermissions   sync.Map
	UserPermissions    sync.Map
}

type AuthHTTPHeader struct {
	CSRFToken   *string
	BasicToken  *string
	BearerToken *string
}

type BasicToken struct {
	Token     string     `json:"token"`
	Expiry    time.Time  `json:"expiry"`
	NotBefore time.Time  `json:"not_before"`
	TTL       float64    `json:"ttl"`
	IP        netip.Addr `json:"ip"`
	Valid     bool       `json:"valid"`
}

type BearerToken struct {
	Token     string     `json:"token"`
	Expiry    time.Time  `json:"expiry"`
	NotBefore time.Time  `json:"not_before"`
	TTL       float64    `json:"ttl"`
	IP        netip.Addr `json:"ip"`
	Valid     bool       `json:"valid"`
}

type CSRFToken struct {
	Token     string     `json:"token"`
	Expiry    time.Time  `json:"expiry"`
	NotBefore time.Time  `json:"not_before"`
	TTL       float64    `json:"ttl"`
	IP        netip.Addr `json:"ip"`
	Valid     bool       `json:"valid"`
}

type AuthSession struct {
	SessionID string
	Basic     BasicToken
	Bearer    BearerToken
	CSRF      CSRFToken
}

var (
	appStateInstance atomic.Pointer[AppState]
)

func LoadConfig() (*AppConfig, error) {
	var appConfig AppConfig
	var configFile ConfigFile

	// Decode JSON
	mainConfigFile, err := os.ReadFile("/etc/uit-toolbox/uit-toolbox.json")
	if err != nil {
		return nil, fmt.Errorf("failed to read config mainConfigFile: %w", err)
	}
	if err := json.Unmarshal(mainConfigFile, &configFile); err != nil {
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
	uploadSizeBytes, err := strconv.ParseInt(configFile.UIT_WEB_MAX_UPLOAD_SIZE_MB, 10, 64)
	uploadSizeMB := uploadSizeBytes << 20
	appConfig.UIT_WEB_MAX_UPLOAD_SIZE_MB = uploadSizeMB
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

	// Initialize rate limiters
	webRateLimiter.Store(&RateLimiter{
		Type:      "webserver",
		ClientMap: sync.Map{},
		Rate:      appConfig.UIT_WEB_RATE_LIMIT_INTERVAL,
		Burst:     appConfig.UIT_WEB_RATE_LIMIT_BURST,
	})
	apiRateLimiter.Store(&RateLimiter{
		Type:      "api",
		ClientMap: sync.Map{},
		Rate:      appConfig.UIT_WEB_RATE_LIMIT_INTERVAL,
		Burst:     appConfig.UIT_WEB_RATE_LIMIT_BURST,
	})
	authRateLimiter.Store(&RateLimiter{
		Type:      "auth",
		ClientMap: sync.Map{},
		Rate:      appConfig.UIT_WEB_RATE_LIMIT_INTERVAL / 2,
		Burst:     appConfig.UIT_WEB_RATE_LIMIT_BURST / 2,
	})
	fileRateLimiter.Store(&RateLimiter{
		Type:      "file",
		ClientMap: sync.Map{},
		Rate:      appConfig.UIT_WEB_RATE_LIMIT_INTERVAL / 4,
		Burst:     appConfig.UIT_WEB_RATE_LIMIT_BURST / 4,
	})
	blockedClients.Store(&BlockedClients{
		ClientMap: sync.Map{},
		BanPeriod: appConfig.UIT_WEB_RATE_LIMIT_BAN_DURATION,
	})

	appState := new(AppState)

	// Store app config in app state
	appState.AppConfig.Store(appConfig)

	// Set DB connection to nil initially
	appState.DBConn.Store(nil)

	// Set logger to nil initially
	appState.Log.Store(nil)

	// Store rate limiters in app state
	appState.WebServerLimiter.Store(webRateLimiter.Load())
	appState.FileLimiter.Store(fileRateLimiter.Load())
	appState.APILimiter.Store(apiRateLimiter.Load())
	appState.AuthLimiter.Store(authRateLimiter.Load())
	appState.BlockedIPs.Store(blockedClients.Load())

	// Initialize logger
	log := logger.CreateLogger("console", logger.ParseLogLevel(os.Getenv("UIT_SERVER_LOG_LEVEL")))
	if log == nil {
		return nil, errors.New("failed to create logger")
	}
	appState.Log.Store(&log)

	// Populate allowed IPs
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

	// Configure web endpoints
	endpointsDirectory := "/etc/uit-toolbox/endpoints/"
	fileInfo, err := os.Stat(endpointsDirectory)
	if err != nil || !fileInfo.IsDir() {
		return appState, fmt.Errorf("endpoints directory does not exist, skipping endpoint loading")
	}
	files, err := os.ReadDir(endpointsDirectory)
	if err != nil || len(files) == 0 {
		return nil, fmt.Errorf("failed to read files in the endpoints directory: %w", err)
	}
	for _, file := range files {
		if file.IsDir() || !strings.HasSuffix(file.Name(), ".json") {
			continue
		}

		endpointsConfig, err := os.ReadFile(endpointsDirectory + file.Name())
		if err != nil {
			return nil, fmt.Errorf("failed to read web endpoints config file %s: %w", file.Name(), err)
		}

		var webEndpoints WebEndpoints
		if err := json.Unmarshal(endpointsConfig, &webEndpoints); err != nil {
			return nil, fmt.Errorf("failed to unmarshal web endpoints config JSON: %w", err)
		}
		for endpointPath, endpointData := range webEndpoints {
			merged := WebEndpointConfig{
				FilePath:       endpointData.FilePath,
				AllowedMethods: endpointData.AllowedMethods,
				TLSRequired:    endpointData.TLSRequired,
				AuthRequired:   endpointData.AuthRequired,
				Requires:       endpointData.Requires,
				ACLUsers:       endpointData.ACLUsers,
				ACLGroups:      endpointData.ACLGroups,
				HTTPVersion:    endpointData.HTTPVersion,
				EndpointType:   endpointData.EndpointType,
				ContentType:    endpointData.ContentType,
				StatusCode:     endpointData.StatusCode,
				Redirect:       endpointData.Redirect,
				RedirectURL:    endpointData.RedirectURL,
			}
			if len(merged.AllowedMethods) == 0 {
				merged.AllowedMethods = []string{"OPTIONS", "GET"}
			}
			if merged.TLSRequired == nil {
				merged.TLSRequired = new(bool)
				*merged.TLSRequired = true
			}
			if merged.AuthRequired == nil {
				merged.AuthRequired = new(bool)
				*merged.AuthRequired = true
			}
			if merged.Requires == nil {
				merged.Requires = []string{}
				// merged.Requires = []string{
				// 	"nonce",
				// 	"webmaster_contact",
				// 	"departments",
				// 	"domains",
				// 	"statuses",
				// 	"locations",
				// 	"client_tag",
				// 	"checkout_date",
				// 	"return_date",
				// 	"customer_name",
				// }
			}
			if merged.Redirect == nil {
				merged.Redirect = new(bool)
				*merged.Redirect = false
			}
			if merged.HTTPVersion == "" {
				merged.HTTPVersion = "HTTP/2.0"
			}
			if merged.EndpointType == "" {
				merged.EndpointType = "api"
			}
			if merged.ContentType == "" {
				merged.ContentType = "application/json; charset=utf-8"
			}
			if merged.StatusCode == 0 {
				merged.StatusCode = 200
			}
			appState.WebEndpoints.Store(endpointPath, &merged)
		}
	}

	permissions, err := InitPermissions()
	if err != nil {
		return nil, fmt.Errorf("failed to load permission config: %w", err)
	}

	for _, groupPermissions := range permissions.Groups {
		appState.GroupPermissions.Store(groupPermissions.ID, groupPermissions)
	}

	for _, userPermissions := range permissions.Users {
		appState.UserPermissions.Store(userPermissions.ID, userPermissions)
	}

	// Set initial timeouts
	appState.APIRequestTimeout.Store(&appConfig.UIT_WEB_API_REQUEST_TIMEOUT)
	appState.FileRequestTimeout.Store(&appConfig.UIT_WEB_FILE_REQUEST_TIMEOUT)

	// Declare endpoints

	if err := SetAppState(appState); err != nil {
		return nil, errors.New("Could not set app state: " + err.Error())
	}
	return appState, nil
}

// App state management
func SetAppState(newState *AppState) error {
	var mu sync.Mutex
	mu.Lock()
	defer mu.Unlock()

	if newState == nil {
		return fmt.Errorf("cannot set app state to nil value")
	}

	appStateInstance.Store(newState)
	return nil
}

func GetAppState() (*AppState, error) {
	appState := appStateInstance.Load()
	if appState == nil {
		return nil, fmt.Errorf("app state is not initialized")
	}
	return appState, nil
}

// Logger access
func GetLogger() logger.Logger {
	appState, err := GetAppState()
	if err != nil {
		fmt.Println("App state not initialized in GetLogger, using default logger")
		return logger.CreateLogger("console", logger.ParseLogLevel("INFO"))
	}

	if appState.Log == (atomic.Pointer[logger.Logger]{}) {
		fmt.Println("Logger not initialized in GetLogger, using default logger")
		return logger.CreateLogger("console", logger.ParseLogLevel("INFO"))
	}

	l := appState.Log.Load()
	if l == nil {
		fmt.Println("Logger is nil in GetLogger, using default logger")
		return logger.CreateLogger("console", logger.ParseLogLevel("INFO"))
	}
	log := *l

	return log
}

// Database managment
func GetDatabaseCredentials() (dbName string, dbHost string, dbPort string, dbUsername string, dbPassword string, err error) {
	appState, err := GetAppState()
	if err != nil {
		return "", "", "", "", "", fmt.Errorf("error getting app state in GetDatabaseCredentials: %w", err)
	}
	return appState.AppConfig.Load().UIT_WEB_DB_NAME, appState.AppConfig.Load().UIT_WEB_DB_HOST.String(), strconv.FormatUint(uint64(appState.AppConfig.Load().UIT_WEB_DB_PORT), 10), appState.AppConfig.Load().UIT_WEB_DB_USERNAME, appState.AppConfig.Load().UIT_WEB_DB_PASSWD, nil
}

func GetWebServerUserDBCredentials() (dbName string, dbHost string, dbPort string, dbUsername string, dbPassword string, err error) {
	appState, err := GetAppState()
	if err != nil {
		return "", "", "", "", "", fmt.Errorf("error getting app state in GetWebServerUserDBCredentials: %w", err)
	}
	return appState.AppConfig.Load().UIT_WEB_DB_NAME, appState.AppConfig.Load().UIT_WEB_DB_HOST.String(), strconv.FormatUint(uint64(appState.AppConfig.Load().UIT_WEB_DB_PORT), 10), appState.AppConfig.Load().UIT_WEB_DB_USERNAME, appState.AppConfig.Load().UIT_WEB_DB_PASSWD, nil
}

func GetDatabaseConn() (*sql.DB, error) {
	appState, err := GetAppState()
	if err != nil {
		return nil, fmt.Errorf("error getting app state in GetDatabaseConn: %w", err)
	}
	db := appState.DBConn.Load()
	if db == nil {
		return nil, fmt.Errorf("database connection is not initialized")
	}
	return db, nil
}

func SetDatabaseConn(newDbConn *sql.DB) error {
	if newDbConn == nil {
		return errors.New("new database connection is nil in SetDatabaseConn")
	}
	appState, err := GetAppState()
	if err != nil {
		return fmt.Errorf("error getting app state in SetDatabaseConn: %w", err)
	}
	if appState == nil {
		return errors.New("app state is not initialized in SetDatabaseConn")
	}
	appState.DBConn.Store(newDbConn)
	return nil
}

// IP address checks
func IsIPAllowed(trafficType string, ipAddr netip.Addr) (allowed bool, err error) {
	appState, err := GetAppState()
	if err != nil {
		return false, fmt.Errorf("error getting app state in IsIPAllowed: %w", err)
	}
	if appState == nil {
		return false, fmt.Errorf("app state is not initialized in IsIPAllowed")
	}

	if !ipAddr.IsValid() || IsIPBlocked(ipAddr) {
		return false, nil
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

func IsIPBlocked(ipAddress netip.Addr) bool {

	appState, err := GetAppState()
	if err != nil {
		return true
	}

	clMap := &appState.BlockedIPs.Load().ClientMap
	val, ok := clMap.Load(ipAddress)
	if !ok {
		return false
	}

	clientLimiter, ok := val.(ClientLimiter)
	if !ok {
		return false
	}

	if time.Now().Before(clientLimiter.LastSeen.Add(appState.BlockedIPs.Load().BanPeriod)) {
		return true
	}

	clMap.Delete(ipAddress)
	return false
}

func CleanupBlockedIPs() {
	appState, err := GetAppState()
	if err != nil {
		return
	}

	blockedIPMap := &appState.BlockedIPs.Load().ClientMap
	blockedIPMap.Range(func(k, v any) bool {
		value := v.(ClientLimiter)
		if time.Now().After(value.LastSeen.Add(appState.BlockedIPs.Load().BanPeriod)) {
			blockedIPMap.Delete(k)
		}
		return true
	})
}

// Webserver config
func GetWebServerIPs() (string, string, error) {
	appState, err := GetAppState()
	if err != nil {
		return "", "", fmt.Errorf("error getting app state in GetWebServerIPs: %w", err)
	}
	return appState.AppConfig.Load().UIT_WEB_HTTP_HOST.String(), appState.AppConfig.Load().UIT_WEB_HTTPS_HOST.String(), nil
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

func GetWebmasterContact() (webmasterName string, webmasterEmail string, err error) {
	appState, err := GetAppState()
	if err != nil {
		return "", "", fmt.Errorf("error getting app state in GetWebmasterContact: %w", err)
	}
	return appState.AppConfig.Load().UIT_WEBMASTER_NAME, appState.AppConfig.Load().UIT_WEBMASTER_EMAIL, nil
}

func GetClientConfig() (*ClientConfig, error) {
	appState, err := GetAppState()
	if err != nil {
		return nil, fmt.Errorf("error getting app state in GetClientConfig: %w", err)
	}
	appConfig := appState.AppConfig.Load()
	if appConfig == nil {
		return nil, fmt.Errorf("app config is not loaded in GetClientConfig")
	}

	clientConfig := &ClientConfig{
		UIT_CLIENT_DB_USER:   appConfig.UIT_CLIENT_DB_USER,
		UIT_CLIENT_DB_PASSWD: appConfig.UIT_CLIENT_DB_PASSWD,
		UIT_CLIENT_DB_NAME:   appConfig.UIT_CLIENT_DB_NAME,
		UIT_CLIENT_DB_HOST:   appConfig.UIT_CLIENT_DB_HOST.String(),
		UIT_CLIENT_DB_PORT:   strconv.FormatUint(uint64(appConfig.UIT_CLIENT_DB_PORT), 10),
		UIT_CLIENT_NTP_HOST:  appConfig.UIT_CLIENT_NTP_HOST.String(),
		UIT_CLIENT_PING_HOST: appConfig.UIT_CLIENT_PING_HOST.String(),
		UIT_SERVER_HOSTNAME:  appConfig.UIT_SERVER_HOSTNAME,
		UIT_WEB_HTTP_HOST:    appConfig.UIT_WEB_HTTP_HOST.String(),
		UIT_WEB_HTTP_PORT:    strconv.FormatUint(uint64(appConfig.UIT_WEB_HTTP_PORT), 10),
		UIT_WEB_HTTPS_HOST:   appConfig.UIT_WEB_HTTPS_HOST.String(),
		UIT_WEB_HTTPS_PORT:   strconv.FormatUint(uint64(appConfig.UIT_WEB_HTTPS_PORT), 10),
		UIT_WEBMASTER_NAME:   appConfig.UIT_WEBMASTER_NAME,
		UIT_WEBMASTER_EMAIL:  appConfig.UIT_WEBMASTER_EMAIL,
	}
	return clientConfig, nil
}

func GetTLSCertFiles() (certFile string, keyFile string, err error) {
	appState, err := GetAppState()
	if err != nil {
		return "", "", fmt.Errorf("error getting app state in GetTLSCertFiles: %w", err)
	}
	appConfig := appState.AppConfig.Load()
	if appConfig == nil {
		return "", "", fmt.Errorf("app config is not loaded in GetTLSCertFiles")
	}
	return appConfig.UIT_WEB_TLS_CERT_FILE, appConfig.UIT_WEB_TLS_KEY_FILE, nil
}

func GetMaxUploadSize() (int64, error) {
	appState, err := GetAppState()
	if err != nil {
		return 0, fmt.Errorf("error getting app state in GetMaxUploadSize: %w", err)
	}
	return appState.AppConfig.Load().UIT_WEB_MAX_UPLOAD_SIZE_MB, nil
}

func GetRequestTimeout(timeoutType string) (time.Duration, error) {
	appState, err := GetAppState()
	if err != nil {
		return 0, fmt.Errorf("error getting app state in GetRequestTimeout: %w", err)
	}
	switch strings.ToLower(timeoutType) {
	case "api":
		apiTimeout := appState.APIRequestTimeout.Load()
		if apiTimeout == nil {
			return 0, fmt.Errorf("cannot get API request timeout in GetRequestTimeout")
		}
		return *apiTimeout, nil
	case "file":
		fileTimeout := appState.FileRequestTimeout.Load()
		if fileTimeout == nil {
			return 0, fmt.Errorf("cannot get file request timeout in GetRequestTimeout")
		}
		return *fileTimeout, nil
	default:
		return 0, fmt.Errorf("invalid timeout type: %s", timeoutType)
	}
}

func SetRequestTimeout(timeoutType string, timeout time.Duration) error {
	appState, err := GetAppState()
	if err != nil {
		return fmt.Errorf("error getting app state in SetRequestTimeout: %w", err)
	}
	if timeout <= 0 {
		return fmt.Errorf("invalid timeout value in SetRequestTimeout: %.2f", timeout.Seconds())
	}
	switch strings.TrimSpace(strings.ToLower(timeoutType)) {
	case "api":
		appState.APIRequestTimeout.Store(&timeout)
		return nil
	case "file":
		appState.FileRequestTimeout.Store(&timeout)
		return nil
	default:
		return fmt.Errorf("invalid timeout type: %s", timeoutType)
	}
}

func GetAllowedLANIPs() ([]netip.Prefix, error) {
	appState, err := GetAppState()
	if err != nil {
		return nil, fmt.Errorf("error getting app state in GetAllowedLANIPs: %w", err)
	}
	var allowedIPs []netip.Prefix
	appState.AllowedLANIPs.Range(func(k, v any) bool {
		ipRange, ok := k.(netip.Prefix)
		if !ok || ipRange == (netip.Prefix{}) {
			return true
		}
		allowedIPs = append(allowedIPs, ipRange)
		return true
	})
	return allowedIPs, nil
}
