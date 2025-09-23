package config

import (
	"database/sql"
	"errors"
	"fmt"
	"maps"
	"net"
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
	UIT_LAN_IF                  string
	UIT_LAN_IP_ADDRESS          string
	UIT_WAN_ALLOWED_IP          []string
	UIT_LAN_ALLOWED_IP          []string
	UIT_ALL_ALLOWED_IP          []string
	UIT_WEB_DB_DBNAME           string
	UIT_WEB_DB_HOST             string
	UIT_WEB_DB_PORT             string
	UIT_WEB_DB_USERNAME         string
	UIT_WEB_DB_PASSWD           string
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
	AppConfig         AppConfig
	DBConn            atomic.Pointer[sql.DB]
	AuthMap           sync.Map
	AuthMapEntryCount atomic.Int64
	Log               logger.Logger
	WebServerLimiter  *LimiterMap
	FileLimiter       *LimiterMap
	APILimiter        *LimiterMap
	AuthLimiter       *LimiterMap
	BlockedIPs        *BlockedMap
	AllowedFiles      atomic.Value
	AllowedFilesMu    sync.Mutex
	AllowedWANIPs     sync.Map
	AllowedLANIPs     sync.Map
	AllowedIPs        sync.Map
}

type AuthHeader struct {
	Basic  *string
	Bearer *string
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
	appStateInstance *AppState
	appStateMutex    sync.RWMutex
)

func LoadConfig() (AppConfig, error) {
	var appConfig AppConfig

	// WAN interface, IP, and allowed IPs
	wanIf, ok := os.LookupEnv("UIT_WAN_IF")
	if !ok {
		return appConfig, errors.New("error getting UIT_WAN_IF: not found")
	}
	wanIP, ok := os.LookupEnv("UIT_WAN_IP_ADDRESS")
	if !ok {
		return appConfig, errors.New("error getting UIT_WAN_IP_ADDRESS: not found")
	}
	envWanAllowedIPStr, ok := os.LookupEnv("UIT_WAN_ALLOWED_IP")
	if !ok {
		return appConfig, errors.New("error getting UIT_WAN_ALLOWED_IP: not found")
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
		return appConfig, errors.New("error getting UIT_LAN_IF: not found")
	}
	lanIP, ok := os.LookupEnv("UIT_LAN_IP_ADDRESS")
	if !ok {
		return appConfig, errors.New("error getting UIT_LAN_IP_ADDRESS: not found")
	}
	envAllowedLanIPStr, ok := os.LookupEnv("UIT_LAN_ALLOWED_IP")
	if !ok {
		return appConfig, errors.New("error getting UIT_LAN_ALLOWED_IP: not found")
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
	dbName, ok := os.LookupEnv("UIT_WEB_DB_DBNAME")
	if !ok {
		return appConfig, errors.New("error getting UIT_WEB_DB_DBNAME: not found")
	}
	dbHost, ok := os.LookupEnv("UIT_WEB_DB_HOST")
	if !ok {
		return appConfig, errors.New("error getting UIT_WEB_DB_HOST: not found")
	}
	dbPort, ok := os.LookupEnv("UIT_WEB_DB_PORT")
	if !ok {
		return appConfig, errors.New("error getting UIT_WEB_DB_PORT: not found")
	}
	dbUser, ok := os.LookupEnv("UIT_WEB_DB_USERNAME")
	if !ok {
		return appConfig, errors.New("error getting UIT_WEB_DB_USERNAME: not found")
	}
	uitWebDbPasswd, ok := os.LookupEnv("UIT_WEB_DB_PASSWD")
	if !ok {
		return appConfig, errors.New("error getting UIT_WEB_DB_PASSWD: not found")
	}
	uitWebDbPasswd = strings.TrimSpace(uitWebDbPasswd)

	dbClientPasswd, ok := os.LookupEnv("UIT_DB_CLIENT_PASSWD")
	if !ok {
		return appConfig, errors.New("error getting UIT_DB_CLIENT_PASSWD: not found")
	}
	webUserDefaultPasswd, ok := os.LookupEnv("UIT_WEB_USER_DEFAULT_PASSWD")
	if !ok {
		return appConfig, errors.New("error getting UIT_WEB_USER_DEFAULT_PASSWD: not found")
	}

	// Website config
	webmasterName, ok := os.LookupEnv("UIT_WEBMASTER_NAME")
	if !ok {
		return appConfig, errors.New("error getting UIT_WEBMASTER_NAME: not found")
	}
	webmasterEmail, ok := os.LookupEnv("UIT_WEBMASTER_EMAIL")
	if !ok {
		return appConfig, errors.New("error getting UIT_WEBMASTER_EMAIL: not found")
	}

	// Printer IP
	printerIP, ok := os.LookupEnv("UIT_PRINTER_IP")
	if !ok {
		return appConfig, errors.New("error getting UIT_PRINTER_IP: not found")
	}

	// Webserver config
	httpPort, ok := os.LookupEnv("UIT_HTTP_PORT")
	if !ok {
		return appConfig, errors.New("error getting UIT_HTTP_PORT: not found")
	}
	httpsPort, ok := os.LookupEnv("UIT_HTTPS_PORT")
	if !ok {
		return appConfig, errors.New("error getting UIT_HTTPS_PORT: not found")
	}
	tlsCertFile, ok := os.LookupEnv("UIT_TLS_CERT_FILE")
	if !ok {
		return appConfig, errors.New("error getting UIT_TLS_CERT_FILE: not found")
	}
	tlsKeyFile, ok := os.LookupEnv("UIT_TLS_KEY_FILE")
	if !ok {
		return appConfig, errors.New("error getting UIT_TLS_KEY_FILE: not found")
	}

	// Rate limiting config
	//Burst
	rateLimitBurstStr, ok := os.LookupEnv("UIT_RATE_LIMIT_BURST")
	if !ok {
		return appConfig, errors.New("error getting UIT_RATE_LIMIT_BURST: not found")
	}
	parsedBurst, err := strconv.Atoi(rateLimitBurstStr)
	if err != nil {
		return appConfig, fmt.Errorf("invalid UIT_RATE_LIMIT_BURST: %w", err)
	}
	if parsedBurst <= 0 {
		return appConfig, errors.New("UIT_RATE_LIMIT_BURST must be > 0")
	}

	// Interval
	rateLimitIntervalStr, ok := os.LookupEnv("UIT_RATE_LIMIT_INTERVAL")
	if !ok {
		return appConfig, errors.New("error getting UIT_RATE_LIMIT_INTERVAL: not found")
	}
	parsedInterval, err := strconv.ParseFloat(rateLimitIntervalStr, 64)
	if err != nil {
		return appConfig, fmt.Errorf("invalid UIT_RATE_LIMIT_INTERVAL: %w", err)
	}
	if parsedInterval <= 0 {
		return appConfig, errors.New("UIT_RATE_LIMIT_INTERVAL must be > 0")
	}

	// Ban duration
	rateLimitBanDurationStr, ok := os.LookupEnv("UIT_RATE_LIMIT_BAN_DURATION")
	if !ok {
		return appConfig, errors.New("error getting UIT_RATE_LIMIT_BAN_DURATION: not found")
	}
	banSeconds, err := strconv.ParseInt(rateLimitBanDurationStr, 10, 64)
	if err != nil {
		return appConfig, fmt.Errorf("invalid UIT_RATE_LIMIT_BAN_DURATION: %w", err)
	}
	if banSeconds <= 0 {
		return appConfig, errors.New("UIT_RATE_LIMIT_BAN_DURATION must be > 0")
	}
	rateLimitBanDuration := time.Duration(banSeconds) * time.Second

	appConfig.UIT_WAN_IF = wanIf
	appConfig.UIT_WAN_IP_ADDRESS = wanIP
	appConfig.UIT_LAN_IF = lanIf
	appConfig.UIT_LAN_IP_ADDRESS = lanIP
	appConfig.UIT_WAN_ALLOWED_IP = wanAllowedIP
	appConfig.UIT_LAN_ALLOWED_IP = lanAllowedIP
	appConfig.UIT_ALL_ALLOWED_IP = allAllowedIPs
	appConfig.UIT_WEB_DB_DBNAME = dbName
	appConfig.UIT_WEB_DB_HOST = dbHost
	appConfig.UIT_WEB_DB_PORT = dbPort
	appConfig.UIT_WEB_DB_USERNAME = dbUser
	appConfig.UIT_WEB_DB_PASSWD = uitWebDbPasswd
	appConfig.UIT_DB_CLIENT_PASSWD = dbClientPasswd
	appConfig.UIT_WEB_USER_DEFAULT_PASSWD = webUserDefaultPasswd
	appConfig.UIT_WEBMASTER_NAME = webmasterName
	appConfig.UIT_WEBMASTER_EMAIL = webmasterEmail
	appConfig.UIT_PRINTER_IP = printerIP
	appConfig.UIT_HTTP_PORT = httpPort
	appConfig.UIT_HTTPS_PORT = httpsPort
	appConfig.UIT_TLS_CERT_FILE = tlsCertFile
	appConfig.UIT_TLS_KEY_FILE = tlsKeyFile
	appConfig.UIT_RATE_LIMIT_BURST = parsedBurst
	appConfig.UIT_RATE_LIMIT_INTERVAL = parsedInterval
	appConfig.UIT_RATE_LIMIT_BAN_DURATION = rateLimitBanDuration

	return appConfig, nil
}

func InitApp() (*AppState, error) {
	appConfig, err := LoadConfig()
	if err != nil {
		return nil, errors.New("failed to load app config: " + err.Error())
	}

	appState := &AppState{
		AppConfig:         appConfig,
		DBConn:            atomic.Pointer[sql.DB]{},
		AuthMap:           sync.Map{},
		AuthMapEntryCount: atomic.Int64{},
		Log:               logger.CreateLogger("console", logger.ParseLogLevel(os.Getenv("UIT_API_LOG_LEVEL"))),
		WebServerLimiter:  &LimiterMap{M: sync.Map{}, Rate: appConfig.UIT_RATE_LIMIT_INTERVAL, Burst: appConfig.UIT_RATE_LIMIT_BURST},
		FileLimiter:       &LimiterMap{M: sync.Map{}, Rate: appConfig.UIT_RATE_LIMIT_INTERVAL / 4, Burst: appConfig.UIT_RATE_LIMIT_BURST / 4},
		APILimiter:        &LimiterMap{M: sync.Map{}, Rate: appConfig.UIT_RATE_LIMIT_INTERVAL, Burst: appConfig.UIT_RATE_LIMIT_BURST},
		AuthLimiter:       &LimiterMap{M: sync.Map{}, Rate: appConfig.UIT_RATE_LIMIT_INTERVAL / 10, Burst: appConfig.UIT_RATE_LIMIT_BURST / 10},
		BlockedIPs:        &BlockedMap{M: sync.Map{}, BanPeriod: appConfig.UIT_RATE_LIMIT_BAN_DURATION},
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

	allowed := make(map[string]bool, len(allowedFiles))
	for _, file := range allowedFiles {
		allowed[file.Filename] = file.Allowed
	}
	appState.AllowedFiles.Store(allowed)

	// for _, file := range allowedFiles {
	// 	appState.AllowedFiles.Store(file.Filename, file.Allowed)
	// }

	for _, wanIP := range appConfig.UIT_WAN_ALLOWED_IP {
		appState.AllowedWANIPs.Store(wanIP, true)
	}
	for _, lanIP := range appConfig.UIT_LAN_ALLOWED_IP {
		appState.AllowedLANIPs.Store(lanIP, true)
	}
	for _, allIP := range appConfig.UIT_ALL_ALLOWED_IP {
		appState.AllowedIPs.Store(allIP, true)
	}

	SetAppState(appState)
	return appState, nil
}

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

func GetLogger() logger.Logger {
	appStateMutex.RLock()
	defer appStateMutex.RUnlock()
	if appStateInstance == nil {
		return nil
	}
	return appStateInstance.Log
}

func GetDatabaseCredentials() (string, string, string, string, string) {
	appState := GetAppState()
	if appState == nil {
		return "", "", "", "", ""
	}
	return appState.AppConfig.UIT_WEB_DB_DBNAME, appState.AppConfig.UIT_WEB_DB_HOST, appState.AppConfig.UIT_WEB_DB_PORT, appState.AppConfig.UIT_WEB_DB_USERNAME, appState.AppConfig.UIT_WEB_DB_PASSWD
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
	oldDbConn = appState.DBConn.Load()
	appState.DBConn.Store(newDbConn)
	return oldDbConn
}

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

func GetAuthSessions() map[string]AuthSession {
	appState := GetAppState()
	if appState == nil {
		return nil
	}
	authSessionsMap := make(map[string]AuthSession)
	appState.AuthMap.Range(func(k, v any) bool {
		key, keyExists := k.(string)
		value, valueExists := v.(AuthSession)
		if keyExists && valueExists {
			authSessionsMap[key] = value
		}
		return true
	})
	return authSessionsMap
}

func CreateAuthSession(sessionID string, authSession AuthSession) error {
	appState := GetAppState()
	if appState == nil {
		return errors.New("app state is not initialized")
	}
	_, ok := appState.AuthMap.LoadOrStore(sessionID, authSession)
	if !ok {
		appState.AuthMapEntryCount.Add(1)
	} else {
		appState.AuthMap.Store(sessionID, authSession)
	}
	return nil
}

func DeleteAuthSession(sessionID string) {
	appState := GetAppState()
	if appState == nil {
		return
	}
	if _, ok := appState.AuthMap.LoadAndDelete(sessionID); ok {
		newVal := appState.AuthMapEntryCount.Add(-1)
		if newVal < 0 {
			appState.AuthMapEntryCount.Store(0)
		}
	}
}

func GetAuthSessionCount() int64 {
	appState := GetAppState()
	if appState == nil {
		return 0
	}
	return appState.AuthMapEntryCount.Load()
}

func RefreshAndGetAuthSessionCount() int64 {
	appState := GetAppState()
	if appState == nil {
		return 0
	}
	var entries int64
	appState.AuthMap.Range(func(_, _ any) bool {
		entries++
		return true
	})
	appState.AuthMapEntryCount.Store(entries)
	return entries
}

func CheckAuthSessionExists(sessionID string, ipAddress string, basicToken string, bearerToken string, csrfToken string) (bool, bool, error) {
	sessionValid := false
	sessionExists := false

	appState := GetAppState()
	if appState == nil {
		return sessionValid, sessionExists, errors.New("app state is not initialized")
	}

	value, ok := appState.AuthMap.Load(sessionID)
	if !ok {
		return sessionValid, sessionExists, nil
	}
	sessionExists = true

	authSession, ok := value.(AuthSession)
	if !ok {
		return sessionValid, sessionExists, errors.New("invalid auth session type")
	}

	curTime := time.Now()

	if authSession.Basic.IP != ipAddress || authSession.Bearer.IP != ipAddress {
		return sessionValid, sessionExists, errors.New("IP address mismatch for session ID: " + sessionID)
	}

	if strings.TrimSpace(ipAddress) == "" || strings.TrimSpace(basicToken) == "" || strings.TrimSpace(bearerToken) == "" {
		return sessionValid, sessionExists, errors.New("empty IP address or token for session ID: " + sessionID)
	}

	if authSession.Basic.Token != basicToken || authSession.Bearer.Token != bearerToken {
		return sessionValid, sessionExists, nil
	}

	if authSession.Basic.Expiry.Before(curTime) || authSession.Bearer.Expiry.Before(curTime) {
		return sessionValid, sessionExists, nil
	}

	sessionValid = true
	return sessionValid, sessionExists, nil
}

func IsIPAllowed(source string, ipAddress string) bool {
	appState := GetAppState()
	if appState == nil {
		return false
	}
	ip := net.ParseIP(ipAddress)
	if ip == nil {
		return false
	}
	allowed := false
	appState.AllowedIPs.Range(func(k, v any) bool {
		cidr, ok := k.(string)
		if !ok || strings.TrimSpace(cidr) == "" {
			return true
		}
		_, ipNet, err := net.ParseCIDR(cidr)
		if err != nil {
			return true
		}
		if ipNet.Contains(ip) {
			allowed = true
			return false
		}
		return true
	})
	return allowed
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
