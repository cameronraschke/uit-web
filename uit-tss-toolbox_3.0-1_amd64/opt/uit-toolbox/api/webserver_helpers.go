package main

import (
	"bufio"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/netip"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"golang.org/x/time/rate"
)

type limiterEntry struct {
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

var (
	rateLimit            float64
	rateLimitBurst       int
	rateLimitBanDuration time.Duration
	ipRequests           *LimiterMap
	blockedIPs           *BlockedMap
)

func countAuthSessions(m *sync.Map) int {
	authSessionCount := 0
	m.Range(func(_, _ any) bool {
		authSessionCount++
		return true
	})
	return authSessionCount
}

func formatHttpError(errorString string) (jsonErrStr string) {
	jsonStr := httpErrorCodes{Message: errorString}
	jsonErr, err := json.Marshal(jsonStr)
	if err != nil {
		log.Error("Cannot parse JSON: " + err.Error())
		return
	}
	return string(jsonErr)
}

func (lm *LimiterMap) Get(ip string) *rate.Limiter {
	curTime := time.Now()
	newEntry := &limiterEntry{
		limiter:  rate.NewLimiter(rate.Limit(lm.rate), lm.burst),
		lastSeen: curTime,
	}
	queriedLimiter, exists := lm.m.LoadOrStore(ip, newEntry)
	entry := queriedLimiter.(*limiterEntry)
	entry.lastSeen = curTime
	if !exists {
		log.Debug("Created new limiter for IP: " + ip + " rate=" + fmt.Sprint(lm.rate) + " burst=" + fmt.Sprint(lm.burst))
	}

	return entry.limiter
}

func (lm *LimiterMap) Delete(ip string) {
	lm.m.Delete(ip)
}

func (bm *BlockedMap) IsBlocked(ip string) bool {
	blockTime, ok := bm.m.Load(ip)
	if !ok {
		return false
	}
	unblockTime, ok := blockTime.(time.Time)
	if !ok {
		bm.m.Delete(ip)
		return false
	}
	if time.Now().After(unblockTime) {
		bm.m.Delete(ip)
		return false
	}
	return true
}

func (bm *BlockedMap) Block(ip string) {
	bm.m.Store(ip, time.Now().Add(bm.banPeriod))
}

func GetLimiter(requestIP string) *rate.Limiter {
	return ipRequests.Get(requestIP)
}

func IsBlocked(requestIP string) bool {
	return blockedIPs.IsBlocked(requestIP)
}

func BlockIP(requestIP string) {
	blockedIPs.Block(requestIP)
}

func (as *AppState) Cleanup() {
	ttl := time.Now().Add(-10 * time.Minute)
	as.ipRequests.m.Range(func(key, value any) bool {
		entry, ok := value.(*limiterEntry)
		if !ok || entry.lastSeen.Before(ttl) {
			as.ipRequests.m.Delete(key)
		}
		return true
	})

	curTime := time.Now()
	as.blockedIPs.m.Range(func(key, value any) bool {
		unblockTime, ok := value.(time.Time)
		if !ok || curTime.After(unblockTime) {
			as.blockedIPs.m.Delete(key)
		}
		return true
	})
}

func (as *AppState) GetAllBlockedIPs() string {
	var blocked []string
	as.blockedIPs.m.Range(func(key, value any) bool {
		ip, ok := key.(string)
		if ok {
			blocked = append(blocked, ip)
		}
		return true
	})
	return strings.Join(blocked, ", ")
}

func checkValidIP(s string) (isValid bool, isLoopback bool, isLocal bool) {
	maxStringSize := int64(128)
	maxCharSize := int(4)

	ipBytes := &io.LimitedReader{
		R: strings.NewReader(s),
		N: maxStringSize,
	}
	reader := bufio.NewReader(ipBytes)

	var totalBytes int64
	var b strings.Builder
	for {
		char, charSize, err := reader.ReadRune()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Warning("read error in checkValidIP" + err.Error())
			return false, false, false
		}
		if charSize > maxCharSize {
			log.Warning("IP address contains an invalid Unicode character")
			return false, false, false
		}
		if char == utf8.RuneError && charSize == 1 {
			return false, false, false
		}
		if (char >= '0' && char <= '9') && (char == '.' || char == ':') {
			log.Warning("IP address contains an invalid character")
			return false, false, false
		}
		totalBytes += int64(charSize)
		if totalBytes > maxStringSize {
			log.Warning("IP length exceeded " + strconv.FormatInt(maxStringSize, 10) + " bytes")
			return false, false, false
		}
		b.WriteRune(char)
	}

	ip := strings.TrimSpace(b.String())
	if ip == "" {
		return false, false, false
	}

	// Reset string builder so GC can get rid of it
	b.Reset()

	parsedIP, err := netip.ParseAddr(ip)
	if err != nil {
		return false, false, false
	}

	// If unspecified, empty, or wrong byte size
	if parsedIP.BitLen() != 32 && parsedIP.BitLen() != 128 {
		log.Warning("IP Address is the incorrect length")
		return false, false, false
	}

	if parsedIP.IsUnspecified() || !parsedIP.IsValid() {
		log.Warning("IP address is unspecified or invalid: " + string(parsedIP.String()))
		return false, false, false
	}

	if !parsedIP.Is4() || parsedIP.Is4In6() || parsedIP.Is6() {
		log.Warning("IP address is not IPv4: " + string(parsedIP.String()))
		return false, false, false
	}

	if parsedIP.IsInterfaceLocalMulticast() || parsedIP.IsLinkLocalMulticast() || parsedIP.IsMulticast() {
		log.Warning("IP address is multicast: " + string(parsedIP.String()))
		return false, false, false
	}

	return true, parsedIP.IsLoopback(), parsedIP.IsPrivate()
}

func generateCSRFToken() (string, error) {
	b := make([]byte, 32)
	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(b), nil
}

func GetRequestIP(r *http.Request) (string, bool) {
	ip, ok := r.Context().Value(ctxClientIP{}).(string)
	return ip, ok
}

func GetRequestURL(r *http.Request) (string, bool) {
	ip, ok := r.Context().Value(ctxURLRequest{}).(string)
	return ip, ok
}

func ParseHeaders(header http.Header) HttpHeaders {
	var headers HttpHeaders
	var authHeader AuthHeader
	for _, value := range header.Values("Authorization") {
		value = strings.TrimSpace(value)
		if strings.HasPrefix(value, "Basic ") {
			basic := strings.TrimSpace(strings.TrimPrefix(value, "Basic "))
			authHeader.Basic = &basic
		}
		if strings.HasPrefix(value, "Bearer ") {
			bearer := strings.TrimSpace(strings.TrimPrefix(value, "Bearer "))
			authHeader.Bearer = &bearer
		}
	}
	headers.Authorization = authHeader
	return headers
}
