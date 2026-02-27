package config

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/netip"
	"os"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
	"uit-toolbox/types"
)

type AppConfiguration struct {
	FormConstraints      atomic.Pointer[types.HTMLFormConstraints]
	FileConstraints      atomic.Pointer[types.FileUploadConstraints]
	LogLevel             string         `json:"UIT_SERVER_LOG_LEVEL"`
	AdminPasswd          string         `json:"UIT_SERVER_ADMIN_PASSWD"`
	DBName               string         `json:"UIT_SERVER_DB_NAME"`
	ServerHostname       string         `json:"UIT_SERVER_HOSTNAME"`
	WANAddr              netip.Addr     `json:"UIT_SERVER_WAN_IP_ADDRESS"`
	LANAddr              netip.Addr     `json:"UIT_SERVER_LAN_IP_ADDRESS"`
	WANIfaceName         string         `json:"UIT_SERVER_WAN_IF"`
	LANIfaceName         string         `json:"UIT_SERVER_LAN_IF"`
	AllowedWANIPs        []netip.Prefix `json:"UIT_SERVER_WAN_ALLOWED_IP"`
	AllowedLANIPs        []netip.Prefix `json:"UIT_SERVER_LAN_ALLOWED_IP"`
	AllAllowedIPs        []netip.Prefix `json:"UIT_SERVER_ANY_ALLOWED_IP"`
	WebUserDefaultPasswd string         `json:"UIT_WEB_USER_DEFAULT_PASSWD"`
	WebDBUsername        string         `json:"UIT_WEB_DB_USERNAME"`
	WebDBPasswd          string         `json:"UIT_WEB_DB_PASSWD"`
	WebDBName            string         `json:"UIT_WEB_DB_NAME"`
	WebDBHost            netip.Addr     `json:"UIT_WEB_DB_HOST"`
	WebDBPort            uint16         `json:"UIT_WEB_DB_PORT"`
	WebHTTPAddr          netip.Addr     `json:"UIT_WEB_HTTP_HOST"`
	WebHTTPPort          uint16         `json:"UIT_WEB_HTTP_PORT"`
	WebHTTPSAddr         netip.Addr     `json:"UIT_WEB_HTTPS_HOST"`
	WebHTTPSPort         uint16         `json:"UIT_WEB_HTTPS_PORT"`
	WebTLSCertFile       string         `json:"UIT_WEB_TLS_CERT_FILE"`
	WebTLSKeyFile        string         `json:"UIT_WEB_TLS_KEY_FILE"`
	APIRequestTimeout    time.Duration  `json:"UIT_WEB_API_REQUEST_TIMEOUT"`
	FileRequestTimeout   time.Duration  `json:"UIT_WEB_FILE_REQUEST_TIMEOUT"`
	RateLimitBurst       int            `json:"UIT_WEB_RATE_LIMIT_BURST"`
	RateLimitInterval    float64        `json:"UIT_WEB_RATE_LIMIT_INTERVAL"`
	RateLimitBanDuration time.Duration  `json:"UIT_WEB_RATE_LIMIT_BAN_DURATION"`
	ClientDBUser         string         `json:"UIT_CLIENT_DB_USER"`
	ClientDBPasswd       string         `json:"UIT_CLIENT_DB_PASSWD"`
	ClientDBName         string         `json:"UIT_CLIENT_DB_NAME"`
	ClientDBHost         netip.Addr     `json:"UIT_CLIENT_DB_HOST"`
	ClientDBPort         uint16         `json:"UIT_CLIENT_DB_PORT"`
	ClientNTPHost        netip.Addr     `json:"UIT_CLIENT_NTP_HOST"`
	ClientPingHost       netip.Addr     `json:"UIT_CLIENT_PING_HOST"`
	WebmasterName        string         `json:"UIT_WEBMASTER_NAME"`
	WebmasterEmail       string         `json:"UIT_WEBMASTER_EMAIL"`
}

type BanList struct {
	bannedClients sync.Map // map[netip.Addr]ClientLimiter
	banPeriod     time.Duration
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

type AppState struct {
	appConfig          atomic.Pointer[AppConfiguration]
	dbConn             atomic.Pointer[sql.DB]
	authMap            sync.Map
	authMapEntryCount  atomic.Int64
	appLogger          atomic.Pointer[slog.Logger]
	webServerLimiter   atomic.Pointer[RateLimiter]
	fileLimiter        atomic.Pointer[RateLimiter]
	apiLimiter         atomic.Pointer[RateLimiter]
	authLimiter        atomic.Pointer[RateLimiter]
	banList            atomic.Pointer[BanList]
	allowedWANIPs      sync.Map
	allowedLANIPs      sync.Map
	allAllowedIPs      sync.Map
	sessionSecret      []byte
	apiRequestTimeout  atomic.Pointer[time.Duration]
	fileRequestTimeout atomic.Pointer[time.Duration]
	webEndpoints       sync.Map
	groupPermissions   sync.Map
	userPermissions    sync.Map
}

var (
	appStateInstance atomic.Pointer[AppState]
)

type levelRangeHandler struct {
	handler  slog.Handler
	minLevel slog.Level
	maxLevel slog.Level
}

func newLevelRangeHandler(handler slog.Handler, minLevel slog.Level, maxLevel slog.Level) slog.Handler {
	return &levelRangeHandler{handler: handler, minLevel: minLevel, maxLevel: maxLevel}
}

func (handler *levelRangeHandler) Enabled(ctx context.Context, level slog.Level) bool {
	if level < handler.minLevel || level > handler.maxLevel {
		return false
	}
	return handler.handler.Enabled(ctx, level)
}

func (handler *levelRangeHandler) Handle(ctx context.Context, record slog.Record) error {
	if record.Level < handler.minLevel || record.Level > handler.maxLevel {
		return nil
	}
	return handler.handler.Handle(ctx, record)
}

func (handler *levelRangeHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &levelRangeHandler{
		handler:  handler.handler.WithAttrs(attrs),
		minLevel: handler.minLevel,
		maxLevel: handler.maxLevel,
	}
}

func (handler *levelRangeHandler) WithGroup(name string) slog.Handler {
	return &levelRangeHandler{
		handler:  handler.handler.WithGroup(name),
		minLevel: handler.minLevel,
		maxLevel: handler.maxLevel,
	}
}

func InitConfig() (*AppConfiguration, error) {
	var appConfig AppConfiguration

	// Decode config file JSON
	mainConfigFile, err := os.ReadFile("/etc/uit-toolbox/uit-toolbox.json")
	if err != nil {
		return nil, fmt.Errorf("failed to read config '/etc/uit-toolbox/uit-toolbox.json': %w", err)
	}
	if err := json.Unmarshal(mainConfigFile, &appConfig); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config JSON: %w", err)
	}

	// Convert durations to seconds
	appConfig.APIRequestTimeout *= time.Second
	appConfig.FileRequestTimeout *= time.Second
	appConfig.RateLimitBanDuration *= time.Second

	// WAN interface, IP, and allowed IPs
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil, fmt.Errorf("failed to get network interfaces: %w", err)
	}
	for _, iface := range ifaces {
		addrs, err := iface.Addrs()
		if err != nil {
			return nil, fmt.Errorf("failed to get addresses for WAN interface: %w", err)
		}
		wanIPFound := false
		lanIPFound := false
		for _, addr := range addrs {
			convIP, ok := addr.(*net.IPNet)
			if !ok {
				return nil, fmt.Errorf("address is not an IPNet: %v", addr)
			}
			if iface.Name == appConfig.WANIfaceName && convIP.IP.String() == appConfig.WANAddr.String() {
				wanIPFound = true
			}
			if iface.Name == appConfig.LANIfaceName && convIP.IP.String() == appConfig.LANAddr.String() {
				lanIPFound = true
			}
		}
		if iface.Name == appConfig.WANIfaceName && !wanIPFound {
			return nil, fmt.Errorf("WAN interface %s does not have the expected IP address %s", appConfig.WANIfaceName, appConfig.WANAddr.String())
		}
		if iface.Name == appConfig.LANIfaceName && !lanIPFound {
			return nil, fmt.Errorf("LAN interface %s does not have the expected IP address %s", appConfig.LANIfaceName, appConfig.LANAddr.String())
		}
	}

	// Set input constraints
	loginFormConstraints := &types.LoginFormConstraints{
		MaxFormBytes:     300,
		UsernameMinChars: 64,
		UsernameMaxChars: 64,
		PasswordMinChars: 64,
		PasswordMaxChars: 64, // All SHA-256, fixed length
	}

	generalNoteConstraints := &types.GeneralNoteConstraints{
		MaxFormBytes:        8192,
		NoteTypeMinChars:    1,
		NoteTypeMaxChars:    256,
		NoteContentMinChars: 0,
		NoteContentMaxChars: 4096,
	}

	inventoryFormConstraints := &types.InventoryUpdateFormConstraints{
		MaxJSONBytes:                 1 << 20,
		TagnumberMinChars:            6,
		TagnumberMaxChars:            6,
		SystemSerialMinChars:         1,
		SystemSerialMaxChars:         128,
		LocationMinChars:             1,
		LocationMaxChars:             128,
		BuildingMinChars:             1,
		BuildingMaxChars:             128,
		RoomMinChars:                 1,
		RoomMaxChars:                 128,
		ManufacturerMinChars:         1,
		ManufacturerMaxChars:         128,
		SystemModelMinChars:          1,
		SystemModelMaxChars:          128,
		DeviceTypeMinChars:           1,
		DeviceTypeMaxChars:           64,
		DepartmentMinChars:           1,
		DepartmentMaxChars:           64,
		DomainMinChars:               1,
		DomainMaxChars:               64,
		PropertyCustodianMinChars:    1,
		PropertyCustodianMaxChars:    64,
		AcquiredDateIsMandatory:      false,
		RetiredDateIsMandatory:       false,
		IsFunctionalIsMandatory:      false,
		DiskRemovedIsMandatory:       false,
		LastHardwareCheckIsMandatory: false,
		ClientStatusMinChars:         1,
		ClientStatusMaxChars:         64,
		CheckoutBoolIsMandatory:      false,
		CheckoutDateIsMandatory:      false,
		ReturnDateIsMandatory:        false,
		ClientNoteMinChars:           0,
		ClientNoteMaxChars:           512,
	}

	// Set form constraints
	formConstraints := &types.HTMLFormConstraints{
		LoginForm:     loginFormConstraints,
		GeneralNote:   generalNoteConstraints,
		InventoryForm: inventoryFormConstraints,
	}
	appConfig.FormConstraints.Store(formConstraints)

	// Set file upload constraints
	imgConstraints := types.ImageUploadConstraints{
		MinFileSize:  512,
		MaxFileSize:  20 << 20,
		MaxFileCount: 20,
		AcceptedImageExtensionsAndMimeTypes: map[string]string{
			".jpg":  "image/jpeg",
			".jpeg": "image/jpeg",
			".png":  "image/png",
			".jfif": "image/jpeg",
		},
	}
	vidConstraints := types.VideoUploadConstraints{
		MinFileSize:  512,
		MaxFileSize:  100 << 20,
		MaxFileCount: 5,
		AcceptedVideoExtensionsAndMimeTypes: map[string]string{
			".mp4": "video/mp4",
		},
	}
	fileConstraints := &types.FileUploadConstraints{
		ImageConstraints:       &imgConstraints,
		VideoConstraints:       &vidConstraints,
		MaxUploadFileSizeLimit: 300 << 20,
	}
	appConfig.FileConstraints.Store(fileConstraints)

	return &appConfig, nil
}

func InitApp() (*AppState, error) {
	appConfig, err := InitConfig()
	if err != nil || appConfig == nil {
		return nil, errors.New("failed to load app config: " + err.Error())
	}

	// Initialize rate limiters
	webRateLimiter.Store(&RateLimiter{
		Type:      "webserver",
		ClientMap: sync.Map{},
		Rate:      appConfig.RateLimitInterval,
		Burst:     appConfig.RateLimitBurst,
	})
	apiRateLimiter.Store(&RateLimiter{
		Type:      "api",
		ClientMap: sync.Map{},
		Rate:      appConfig.RateLimitInterval,
		Burst:     appConfig.RateLimitBurst,
	})
	authRateLimiter.Store(&RateLimiter{
		Type:      "auth",
		ClientMap: sync.Map{},
		Rate:      appConfig.RateLimitInterval / 2,
		Burst:     appConfig.RateLimitBurst / 2,
	})
	fileRateLimiter.Store(&RateLimiter{
		Type:      "file",
		ClientMap: sync.Map{},
		Rate:      appConfig.RateLimitInterval / 4,
		Burst:     appConfig.RateLimitBurst / 4,
	})

	appState := new(AppState)

	// Store app config in app state
	appState.appConfig.Store(appConfig)

	// Set DB connection to nil initially
	appState.dbConn.Store(nil)

	// Store rate limiters in app state
	appState.webServerLimiter.Store(webRateLimiter.Load())
	appState.fileLimiter.Store(fileRateLimiter.Load())
	appState.apiLimiter.Store(apiRateLimiter.Load())
	appState.authLimiter.Store(authRateLimiter.Load())
	// Initialize ban list
	banList := &BanList{
		bannedClients: sync.Map{},
		banPeriod:     appConfig.RateLimitBanDuration,
	}
	appState.banList.Store(banList)

	// Initialize logger

	// Set logger to nil initially
	appState.appLogger.Store(nil)

	removeTime := func(groups []string, a slog.Attr) slog.Attr {
		if a.Key == slog.TimeKey && len(groups) == 0 {
			return slog.Attr{}
		}
		return a
	}

	stdoutTextHandler := newLevelRangeHandler(
		slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
			Level:       slog.LevelInfo,
			ReplaceAttr: removeTime,
		}),
		slog.LevelInfo,
		slog.LevelInfo,
	)
	stderrTextHandler := newLevelRangeHandler(
		slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
			Level:       slog.LevelWarn,
			ReplaceAttr: removeTime,
		}),
		slog.LevelWarn,
		slog.LevelError,
	)
	jsonLogFile, err := os.OpenFile("/var/log/uit-web/"+time.Now().Format("2006-01-02_15-04-05")+".json.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o0640)
	if err != nil {
		return nil, fmt.Errorf("failed to open JSON log file: %w", err)
	}
	jsonFileHandler := newLevelRangeHandler(
		slog.NewJSONHandler(jsonLogFile, &slog.HandlerOptions{Level: slog.LevelInfo}),
		slog.LevelInfo,
		slog.LevelError,
	)

	multiHandler := slog.NewMultiHandler(stdoutTextHandler, stderrTextHandler, jsonFileHandler)
	logger := slog.New(multiHandler)
	slog.SetDefault(logger)
	appState.appLogger.Store(logger)

	// Populate allowed IPs
	for _, wanIP := range appConfig.AllowedWANIPs {
		wanIPCopy := wanIP
		appState.allowedWANIPs.Store(&wanIPCopy, true)
		appState.allAllowedIPs.Store(&wanIPCopy, true)
	}

	for _, lanIP := range appConfig.AllowedLANIPs {
		lanIPCopy := lanIP
		appState.allowedLANIPs.Store(&lanIPCopy, true)
		appState.allAllowedIPs.Store(&lanIPCopy, true)
	}

	// Generate server-side secret for HMAC
	sessionSecret := make([]byte, 32)
	if _, err := rand.Read(sessionSecret); err != nil {
		return nil, fmt.Errorf("failed to generate session secret: %w", err)
	}
	appState.sessionSecret = sessionSecret

	// Configure web endpoints
	if err := InitWebEndpoints(appState); err != nil {
		return nil, fmt.Errorf("failed to initialize web endpoints: %w", err)
	}

	// Load permissions
	permissions, err := InitPermissions()
	if err != nil {
		return nil, fmt.Errorf("failed to load permission config: %w", err)
	}

	for _, groupPermissions := range permissions.Groups {
		appState.groupPermissions.Store(groupPermissions.ID, groupPermissions)
	}

	for _, userPermissions := range permissions.Users {
		appState.userPermissions.Store(userPermissions.ID, userPermissions)
	}

	// Set initial timeouts
	appState.apiRequestTimeout.Store(&appConfig.APIRequestTimeout)
	appState.fileRequestTimeout.Store(&appConfig.FileRequestTimeout)

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
		return nil, fmt.Errorf("app state is nil (GetAppState)")
	}
	return appState, nil
}

// Logger access
func GetLogger() *slog.Logger {
	appState, err := GetAppState()
	if err != nil {
		fmt.Println("App state not initialized in GetLogger, using default logger")
		newLogger := slog.New(slog.NewTextHandler(os.Stdout, nil))
		slog.SetDefault(newLogger)
		return newLogger
	}
	if appState.appLogger == (atomic.Pointer[slog.Logger]{}) {
		fmt.Println("Logger not initialized in GetLogger, using default logger")
		newLogger := slog.New(slog.NewTextHandler(os.Stdout, nil))
		return newLogger
	}
	logger := appState.appLogger.Load()
	if logger == nil {
		fmt.Println("Logger is nil in GetLogger, using default logger")
		return slog.New(slog.NewTextHandler(os.Stdout, nil))
	}

	return logger
}

// Database managment
func GetDatabaseCredentials() (dbName string, dbHost string, dbPort string, dbUsername string, dbPassword string, err error) {
	appState, err := GetAppState()
	if err != nil {
		return "", "", "", "", "", fmt.Errorf("error getting app state in GetDatabaseCredentials: %w", err)
	}
	return appState.appConfig.Load().WebDBName, appState.appConfig.Load().WebDBHost.String(), strconv.FormatUint(uint64(appState.appConfig.Load().WebDBPort), 10), appState.appConfig.Load().WebDBUsername, appState.appConfig.Load().WebDBPasswd, nil
}

func GetWebServerUserDBCredentials() (dbName string, dbHost string, dbPort string, dbUsername string, dbPassword string, err error) {
	appState, err := GetAppState()
	if err != nil {
		return "", "", "", "", "", fmt.Errorf("error getting app state in GetWebServerUserDBCredentials: %w", err)
	}
	return appState.appConfig.Load().WebDBName, appState.appConfig.Load().WebDBHost.String(), strconv.FormatUint(uint64(appState.appConfig.Load().WebDBPort), 10), appState.appConfig.Load().WebDBUsername, appState.appConfig.Load().WebDBPasswd, nil
}

func GetDatabaseConn() (*sql.DB, error) {
	appState, err := GetAppState()
	if err != nil {
		return nil, fmt.Errorf("error getting app state in GetDatabaseConn: %w", err)
	}
	db := appState.dbConn.Load()
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
	appState.dbConn.Store(newDbConn)
	return nil
}

// IP address checks
func IsIPAllowed(trafficType string, ipAddr netip.Addr) (allowed bool, err error) {
	appState, err := GetAppState()
	if err != nil {
		return allowed, fmt.Errorf("cannot retrieve app state: %w", err)
	}

	if !ipAddr.IsValid() || RequestIPBlocked(ipAddr) {
		return allowed, fmt.Errorf("request IP is invalid or blocked")
	}

	switch strings.TrimSpace(strings.ToLower(trafficType)) {
	case "wan":
		allowed, err = ipAllowedInRanges(ipAddr, &appState.allowedWANIPs)
		if err != nil {
			return allowed, fmt.Errorf("error checking WAN IP ranges: %w", err)
		}
	case "lan":
		allowed, err = ipAllowedInRanges(ipAddr, &appState.allowedLANIPs)
		if err != nil {
			return allowed, fmt.Errorf("error checking LAN IP ranges: %w", err)
		}
	case "any":
		allowed, err = ipAllowedInRanges(ipAddr, &appState.allAllowedIPs)
		if err != nil {
			return allowed, fmt.Errorf("error checking IP ranges: %w", err)
		}
	default:
		return allowed, errors.New("invalid traffic type, must be 'wan', 'lan', or 'any'")
	}
	return allowed, nil
}

func ipAllowedInRanges(ipAddr netip.Addr, ranges *sync.Map) (allowed bool, err error) {
	if ranges == nil {
		return false, fmt.Errorf("IP range map is nil")
	}

	allowed = false
	ranges.Range(func(k, v any) bool {
		ipRangePtr, ok := k.(*netip.Prefix)
		if !ok || ipRangePtr == nil {
			return true
		}
		ipRange := *ipRangePtr
		if ipRange == (netip.Prefix{}) {
			return true
		}
		if ipRange.Contains(ipAddr) {
			allowed = true
			return false
		}
		return true
	})
	return allowed, nil
}

func RequestIPBlocked(ip netip.Addr) bool {
	if !ip.IsValid() {
		return true
	}
	as, err := GetAppState()
	if err != nil || as == nil {
		return true
	}

	bannedClient, ok := as.banList.Load().bannedClients.Load(ip)
	if !ok || bannedClient == nil {
		return false
	}

	bannedClientLimiter, ok := bannedClient.(ClientLimiter)
	if !ok || bannedClientLimiter == (ClientLimiter{}) {
		return false
	}

	if time.Now().Before(bannedClientLimiter.LastSeen.Add(as.banList.Load().banPeriod)) {
		return true
	}

	as.banList.Load().bannedClients.Delete(ip)
	return false
}

func CleanupBlockedIPs() {
	appState, err := GetAppState()
	if err != nil {
		return
	}

	blockedIPMap := &appState.banList.Load().bannedClients
	blockedIPMap.Range(func(k, v any) bool {
		value := v.(ClientLimiter)
		if time.Now().After(value.LastSeen.Add(appState.banList.Load().banPeriod)) {
			blockedIPMap.Delete(k)
		}
		return true
	})
}

// Webserver config
func GetWebServerIPs() (httpIP string, httpsIP string, err error) {
	appState, err := GetAppState()
	if err != nil {
		return "", "", fmt.Errorf("error getting app state in GetWebServerIPs: %w", err)
	}
	return appState.appConfig.Load().WebHTTPAddr.String(), appState.appConfig.Load().WebHTTPSAddr.String(), nil
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
	return appState.appConfig.Load().WebmasterName, appState.appConfig.Load().WebmasterEmail, nil
}

func GetClientConfig() (*ClientConfig, error) {
	appState, err := GetAppState()
	if err != nil {
		return nil, fmt.Errorf("error getting app state in GetClientConfig: %w", err)
	}
	appConfig := appState.appConfig.Load()
	if appConfig == nil {
		return nil, fmt.Errorf("app config is not loaded in GetClientConfig")
	}

	clientConfig := &ClientConfig{
		UIT_CLIENT_DB_USER:   appConfig.ClientDBUser,
		UIT_CLIENT_DB_PASSWD: appConfig.ClientDBPasswd,
		UIT_CLIENT_DB_NAME:   appConfig.ClientDBName,
		UIT_CLIENT_DB_HOST:   appConfig.ClientDBHost.String(),
		UIT_CLIENT_DB_PORT:   strconv.FormatUint(uint64(appConfig.ClientDBPort), 10),
		UIT_CLIENT_NTP_HOST:  appConfig.ClientNTPHost.String(),
		UIT_CLIENT_PING_HOST: appConfig.ClientPingHost.String(),
		UIT_SERVER_HOSTNAME:  appConfig.ServerHostname,
		UIT_WEB_HTTP_HOST:    appConfig.WebHTTPAddr.String(),
		UIT_WEB_HTTP_PORT:    strconv.FormatUint(uint64(appConfig.WebHTTPPort), 10),
		UIT_WEB_HTTPS_HOST:   appConfig.WebHTTPSAddr.String(),
		UIT_WEB_HTTPS_PORT:   strconv.FormatUint(uint64(appConfig.WebHTTPSPort), 10),
		UIT_WEBMASTER_NAME:   appConfig.WebmasterName,
		UIT_WEBMASTER_EMAIL:  appConfig.WebmasterEmail,
	}
	return clientConfig, nil
}

func GetTLSCertFiles() (certFile string, keyFile string, err error) {
	appState, err := GetAppState()
	if err != nil {
		return "", "", fmt.Errorf("error getting app state in GetTLSCertFiles: %w", err)
	}
	appConfig := appState.appConfig.Load()
	if appConfig == nil {
		return "", "", fmt.Errorf("app config is not loaded in GetTLSCertFiles")
	}
	return appConfig.WebTLSCertFile, appConfig.WebTLSKeyFile, nil
}

func GetRequestTimeout(timeoutType string) (time.Duration, error) {
	appState, err := GetAppState()
	if err != nil {
		return 0, fmt.Errorf("error getting app state in GetRequestTimeout: %w", err)
	}
	switch strings.ToLower(timeoutType) {
	case "api":
		apiTimeout := appState.apiRequestTimeout.Load()
		if apiTimeout == nil {
			return 0, fmt.Errorf("cannot get API request timeout in GetRequestTimeout")
		}
		return *apiTimeout, nil
	case "file":
		fileTimeout := appState.fileRequestTimeout.Load()
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
		appState.apiRequestTimeout.Store(&timeout)
		return nil
	case "file":
		appState.fileRequestTimeout.Store(&timeout)
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
	appState.allowedLANIPs.Range(func(k, v any) bool {
		ipRangePtr, ok := k.(*netip.Prefix)
		if !ok || ipRangePtr == nil {
			return true
		}
		ipRange := *ipRangePtr
		if ipRange == (netip.Prefix{}) {
			return true
		}
		allowedIPs = append(allowedIPs, ipRange)
		return true
	})
	return allowedIPs, nil
}
