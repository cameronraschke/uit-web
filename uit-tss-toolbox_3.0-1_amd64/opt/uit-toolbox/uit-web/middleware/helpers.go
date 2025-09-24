package middleware

import (
	"bufio"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/netip"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	config "uit-toolbox/config"
	webserver "uit-toolbox/webserver"

	"golang.org/x/crypto/bcrypt"
)

func FormatHttpError(errorString string) (jsonErrStr string) {
	jsonStr := webserver.HTTPErrorCodes{Message: errorString}
	jsonErr, err := json.Marshal(jsonStr)
	if err != nil {
		return ""
	}
	return string(jsonErr)
}

func checkValidIP(s string) (isValid bool, isLoopback bool, isLocal bool) {
	log := config.GetLogger()
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

func GetRequestIP(r *http.Request) (string, bool) {
	if ip, ok := r.Context().Value(webserver.CTXClientIP{}).(string); ok {
		return ip, true
	}
	return "", false
}

func GetRequestURL(r *http.Request) (string, bool) {
	if url, ok := r.Context().Value(webserver.CTXURLRequest{}).(string); ok {
		return url, true
	}
	return "", false
}

func GetRequestedFile(req *http.Request) (string, string, string, bool) {
	if fileRequest, ok := req.Context().Value(webserver.CTXFileRequest{}).(webserver.CTXFileRequest); ok {
		return fileRequest.FullPath, fileRequest.ResolvedPath, fileRequest.FileName, true
	}
	return "", "", "", false
}

func CheckAuthCredentials(ctx context.Context, username, password string) (bool, error) {
	db := config.GetDatabaseConn()
	if db == nil {
		return false, errors.New("database is not initialized")
	}
	var tmpToken = username + ":" + password
	authToken := sha256.Sum256([]byte(tmpToken))
	authTokenString := hex.EncodeToString(authToken[:])

	sqlCode := `SELECT ENCODE(SHA256(CONCAT(username, ':', password)::bytea), 'hex') FROM logins WHERE ENCODE(SHA256(CONCAT(username, ':', password)::bytea), 'hex') = $1`
	rows, err := db.QueryContext(ctx, sqlCode, authTokenString)
	for rows.Next() {
		var dbToken string
		if err := rows.Scan(&dbToken); err != nil {
			return false, errors.New("cannot scan database row for API Auth: " + err.Error())
		}
		if dbToken == authTokenString {
			err := bcrypt.CompareHashAndPassword([]byte(dbToken), []byte(authTokenString))
			if err != nil {
				return false, errors.New("invalid credentials - bcrypt mismatch")
			}
			return true, nil
		}
	}
	if err == sql.ErrNoRows {
		buffer1 := make([]byte, 32)
		_, _ = rand.Read(buffer1)
		buffer2 := make([]byte, 32)
		_, _ = rand.Read(buffer2)
		pass1, _ := bcrypt.GenerateFromPassword(buffer1, bcrypt.DefaultCost)
		pass2, _ := bcrypt.GenerateFromPassword(buffer2, bcrypt.DefaultCost)
		bcrypt.CompareHashAndPassword(pass1, pass2)
		return false, errors.New("invalid credentials")
	}
	if err != nil {
		return false, errors.New("cannot query database for API Auth: " + err.Error())
	}
	defer rows.Close()

	if !rows.Next() {
		return false, errors.New("no matching auth token found")
	}

	return false, errors.New("unknown error during authentication")
}

func ValidateAuthFormInput(username, password string) error {
	username = strings.TrimSpace(username)
	if len(username) < 3 || len(username) > 20 {
		return errors.New("invalid username length")
	}

	if len(password) < 8 || len(password) > 64 {
		return errors.New("invalid password length")
	}

	authStr := username + ":" + password

	// ASCII characters except space
	allowedAuthChars := "U+0021-U+007E"

	for _, char := range authStr {
		if char <= 31 || char >= 127 || char > unicode.MaxASCII || char > unicode.MaxLatin1 {
			return errors.New(`auth string contains an invalid control character (beyond ASCII/Latin1): ` + fmt.Sprintf("U+%04X", char))
		}
		if unicode.IsControl(char) {
			return errors.New(`auth string contains an invalid control character: ` + fmt.Sprintf("U+%04X", char))
		}
		if unicode.IsSpace(char) {
			return errors.New(`auth string contains a whitespace character: ` + fmt.Sprintf("U+%04X", char))
		}
		if !strings.ContainsRune(allowedAuthChars, char) {
			return errors.New(`auth string contains a disallowed character: ` + fmt.Sprintf("U+%04X", char))
		}
	}

	return nil
}
