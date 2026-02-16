package endpoints

import (
	"context"
	"crypto/rand"
	"database/sql"
	"errors"
	"fmt"
	"html/template"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"time"
	config "uit-toolbox/config"
	middleware "uit-toolbox/middleware"
	"unicode/utf8"

	"golang.org/x/crypto/bcrypt"
)

type HttpTemplateResponseData struct {
	JsNonce        string
	WebmasterName  string
	WebmasterEmail string
	ClientTag      string
	CheckoutDate   string
	ReturnDate     string
	CustomerName   string
}

type ServerTime struct {
	Time string `json:"server_time"`
}

func ValidateAuthFormInputSHA256(username, password string) error {
	username = strings.TrimSpace(username)
	usernameLength := utf8.RuneCountInString(username)
	if usernameLength != 64 {
		return errors.New("invalid SHA hash length for username")
	}

	password = strings.TrimSpace(password)
	passwordLength := utf8.RuneCountInString(password)
	if passwordLength != 64 {
		return errors.New("invalid SHA hash length for password")
	}

	if err := middleware.IsSHA256String(username); err != nil {
		return errors.New("username does not match SHA regex")
	}
	if err := middleware.IsSHA256String(password); err != nil {
		return errors.New("password does not match SHA regex")
	}

	authStr := username + ":" + password

	// Check for non-printable ASCII characters
	if !middleware.IsPrintableASCII([]byte(authStr)) {
		return errors.New("credentials contain non-printable ASCII characters")
	}

	return nil
}

func CheckAuthCredentials(ctx context.Context, username, password string) (bool, error) {
	if strings.TrimSpace(username) == "" || strings.TrimSpace(password) == "" {
		return false, errors.New("username or password is empty")
	}

	db, err := config.GetDatabaseConn()
	if err != nil {
		return false, fmt.Errorf("error getting database connection in CheckAuthCredentials: %w", err)
	}

	sqlCode := `SELECT password FROM logins WHERE username = $1 LIMIT 1;`
	var dbBcryptHash sql.NullString
	if err := db.QueryRowContext(ctx, sqlCode, username).Scan(&dbBcryptHash); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			buffer1 := make([]byte, 32)
			_, _ = rand.Read(buffer1)
			buffer2 := make([]byte, 32)
			_, _ = rand.Read(buffer2)
			pass1, _ := bcrypt.GenerateFromPassword(buffer1, bcrypt.DefaultCost)
			pass2, _ := bcrypt.GenerateFromPassword(buffer2, bcrypt.DefaultCost)
			bcrypt.CompareHashAndPassword(pass1, pass2)
			return false, errors.New("invalid credentials")
		}
		return false, fmt.Errorf("query error in CheckAuthCredentials: %w", err)
	}

	// Compare plaintext password versus stored bcrypt
	if bcrypt.CompareHashAndPassword([]byte(dbBcryptHash.String), []byte(password)) != nil {
		return false, errors.New("invalid credentials")
	}
	return true, nil
}

func IsTagnumberInt64Valid(i *int64) error {
	if i == nil {
		return fmt.Errorf("tagnumber is nil")
	}
	if *i < 100000 || *i > 999999 {
		return fmt.Errorf("tagnumber is out of valid numeric range")
	}
	return nil
}

func IsTagnumberStringValid(b []byte) error {
	if len(b) == 0 {
		return fmt.Errorf("tagnumber is nil")
	}
	if !middleware.IsNumericAscii(b) {
		return fmt.Errorf("tagnumber contains non-numeric ASCII characters")
	}
	if utf8.RuneCount(b) != 6 {
		return fmt.Errorf("tagnumber does not contain exactly 6 characters")
	}
	return nil
}

func ConvertAndVerifyTagnumber(tagStr string) (*int64, error) {
	trimmedTagStr := strings.TrimSpace(tagStr)
	if trimmedTagStr == "" {
		return nil, fmt.Errorf("tagnumber string is empty")
	}
	validStringErr := IsTagnumberStringValid([]byte(trimmedTagStr))
	if validStringErr != nil {
		return nil, fmt.Errorf("invalid tagnumber string: %v", validStringErr)
	}
	tag, err := strconv.ParseInt(trimmedTagStr, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("cannot parse tagnumber: %v", err)
	}
	validInt64Err := IsTagnumberInt64Valid(&tag)
	if validInt64Err != nil {
		return nil, fmt.Errorf("invalid tagnumber: %v", validInt64Err)
	}
	return &tag, nil
}

func FileServerHandler(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	log := middleware.GetLoggerFromContext(ctx)
	requestIP, err := middleware.GetRequestIPFromContext(ctx)
	if err != nil {
		log.Warning("Error retrieving IP address stored in context for FileServerHandler: " + err.Error())
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}
	resolvedPath, resolvedPathExists := middleware.GetRequestFileFromContext(ctx)
	if !resolvedPathExists {
		log.Warning("no requested file stored in context for FileServerHandler")
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}

	log.HTTPDebug(req, "File request received")

	// Previous path and file validation done in middleware
	// Open the file
	f, err := os.Open(resolvedPath)
	if err != nil {
		log.HTTPWarning(req, "File not found for FileServerHandler: "+err.Error())
		middleware.WriteJsonError(w, http.StatusNotFound)
		return
	}
	defer f.Close()

	// err = f.SetDeadline(time.Now().Add(30 * time.Second))
	// if err != nil {
	// 	log.Error("Cannot set file read deadline: " + resolvedPath + " (" + err.Error() + ")")
	// 	http.Error(w, http.StatusInternalServerError)
	// 	return
	// }

	metadata, err := f.Stat()
	if err != nil {
		log.Error("Cannot stat file: " + resolvedPath + " (" + err.Error() + ")")
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}

	maxFileSize := int64(10) << 30 // 10 GiB
	if metadata.Size() > maxFileSize {
		log.Warning("File too large: " + resolvedPath + " (" + fmt.Sprintf("%d", metadata.Size()) + " bytes)")
		middleware.WriteJsonError(w, http.StatusRequestEntityTooLarge)
		return
	}

	// Get file info for headers
	switch filepath.Ext(resolvedPath) {
	case ".deb":
		w.Header().Set("Content-Type", "application/vnd.debian.binary-package")
	case ".gz":
		w.Header().Set("Content-Type", "application/gzip")
	case ".img":
		w.Header().Set("Content-Type", "application/vnd.efi.img")
	case ".iso":
		w.Header().Set("Content-Type", "application/vnd.efi.iso")
	case ".squashfs":
		w.Header().Set("Content-Type", "application/octet-stream")
	case ".crt":
		w.Header().Set("Content-Type", "application/x-x509-ca-cert")
	case ".pem":
		w.Header().Set("Content-Type", "application/pem-certificate-chain")
	default:
		w.Header().Set("Content-Type", "application/octet-stream")
	}

	// Set headers
	w.Header().Set("Content-Security-Policy", "default-src 'none'")
	w.Header().Set("Content-Length", fmt.Sprintf("%d", metadata.Size()))
	w.Header().Set("Content-Disposition", "attachment; filename=\""+metadata.Name()+"\"")
	w.Header().Set("Accept-Ranges", "bytes")
	w.Header().Set("Last-Modified", metadata.ModTime().UTC().Format(http.TimeFormat))
	w.Header().Set("ETag", fmt.Sprintf(`"%x-%x"`, metadata.ModTime().Unix(), metadata.Size()))
	w.Header().Set("Cache-Control", "private, max-age=300")

	if ctx.Err() != nil {
		log.HTTPWarning(req, "Context cancelled while serving file: "+resolvedPath+": "+ctx.Err().Error())
		return
	}

	// Serve the file
	http.ServeContent(w, req, metadata.Name(), metadata.ModTime(), f)

	log.Info("Served file: " + resolvedPath + " to " + requestIP.String())
}

func WebServerHandler(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	log := middleware.GetLoggerFromContext(ctx)
	requestIP, err := middleware.GetRequestIPFromContext(ctx)
	if err != nil {
		log.HTTPWarning(req, "Error retrieving IP address stored in context for WebServerHandler: "+err.Error())
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}
	requestedPath, err := middleware.GetRequestPathFromContext(ctx)
	if err != nil {
		log.HTTPWarning(req, "Error retrieving URL stored in context for WebServerHandler: "+err.Error())
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}
	requestQueries, _ := middleware.GetRequestQueryFromContext(ctx)

	nonce, nonceExists := middleware.GetNonceFromContext(ctx)
	if !nonceExists || strings.TrimSpace(nonce) == "" {
		log.Error("Error retrieving CSP nonce from context")
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}

	log.HTTPDebug(req, "Web request from "+requestIP.String()+" for "+requestedPath)

	// Get endpoint config
	endpointData, err := config.GetWebEndpointConfig(requestedPath)
	if err != nil {
		log.HTTPWarning(req, "Cannot get endpoint config for endpoint "+requestedPath+": "+err.Error()+"")
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}
	filePath, err := config.GetWebEndpointFilePath(endpointData)
	if err != nil {
		log.HTTPWarning(req, "Cannot get file path for endpoint "+requestedPath+": "+err.Error()+"")
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}
	// Open the file
	file, err := os.Open(filePath)
	if err != nil {
		log.HTTPWarning(req, "Cannot open file: "+filePath+": "+err.Error()+"")
		middleware.WriteJsonError(w, http.StatusNotFound)
		return
	}
	defer file.Close()

	// err = file.SetDeadline(time.Now().Add(30 * time.Second))
	// if err != nil {
	// 	log.Error("Cannot set file read deadline: " + filePath + " (" + err.Error() + ")")
	// 	http.Error(w, http.StatusInternalServerError)
	// 	return
	// }

	metadata, err := file.Stat()
	if err != nil {
		log.HTTPWarning(req, "Cannot stat file: "+filePath+": "+err.Error()+"")
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}

	maxFileSize, err := config.GetMaxUploadSize()
	if err != nil {
		log.HTTPError(req, "Cannot get max upload size from config: "+err.Error()+"")
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}
	if metadata.Size() > maxFileSize {
		log.HTTPWarning(req, "File too large: "+filePath+" ("+fmt.Sprintf("%d", metadata.Size())+" bytes)")
		http.Error(w, "File too large", http.StatusRequestEntityTooLarge)
		return
	}

	// Set headers
	contentType, err := config.GetWebEndpointContentType(endpointData)
	if err != nil {
		log.HTTPWarning(req, "Cannot get content type for endpoint "+requestedPath+": "+err.Error()+"")
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Content-Disposition", "inline; filename=\""+metadata.Name()+"\"")
	w.Header().Set("Last-Modified", metadata.ModTime().UTC().Format(http.TimeFormat))
	w.Header().Set("ETag", fmt.Sprintf(`"%x-%x"`, metadata.ModTime().Unix(), metadata.Size()))
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")

	if ctx.Err() != nil {
		log.HTTPWarning(req, "Context cancelled while serving path: "+requestedPath+": "+ctx.Err().Error())
		return
	}

	if filepath.Ext(filePath) == ".html" {
		parsedHTMLTemplate, err := template.ParseFiles(filePath)
		if err != nil {
			log.HTTPWarning(req, "Cannot parse template file ("+filePath+"): "+err.Error())
			middleware.WriteJsonError(w, http.StatusInternalServerError)
			return
		}

		var httpTemplateResponseData = &HttpTemplateResponseData{}
		if endpointData.Requires != nil {
			// Generate nonce
			if slices.Contains(endpointData.Requires, "nonce") {
				httpTemplateResponseData.JsNonce = nonce
			}

			if slices.Contains(endpointData.Requires, "webmaster_contact") {
				webmasterName, webmasterEmail, err := config.GetWebmasterContact()
				if err != nil {
					log.HTTPError(req, "Cannot get webmaster contact info: "+err.Error()+"")
					middleware.WriteJsonError(w, http.StatusInternalServerError)
					return
				}
				httpTemplateResponseData.WebmasterName = webmasterName
				httpTemplateResponseData.WebmasterEmail = webmasterEmail
			}

			if slices.Contains(endpointData.Requires, "client_tag") {
				urlTag := req.URL.Query().Get("tagnumber")
				tagnumber, err := ConvertAndVerifyTagnumber(urlTag)
				if err != nil {
					log.HTTPWarning(req, "Invalid tagnumber in URL: "+urlTag+" ("+err.Error()+")")
					middleware.WriteJsonError(w, http.StatusBadRequest)
					return
				}
				if tagnumber == nil {
					log.HTTPWarning(req, "No tagnumber provided in URL for endpoint that requires client_tag")
					middleware.WriteJsonError(w, http.StatusBadRequest)
					return
				}
				httpTemplateResponseData.ClientTag = strconv.FormatInt(*tagnumber, 10)
			}

			if slices.Contains(endpointData.Requires, "checkout_date") ||
				slices.Contains(endpointData.Requires, "return_date") ||
				slices.Contains(endpointData.Requires, "customer_name") {
				checkoutDate := requestQueries.Get("checkout_date")
				returnDate := requestQueries.Get("return_date")
				customerName := requestQueries.Get("customer_name")
				httpTemplateResponseData.CheckoutDate = checkoutDate
				httpTemplateResponseData.ReturnDate = returnDate
				httpTemplateResponseData.CustomerName = customerName
			}
		}

		// Execute the template
		if err := parsedHTMLTemplate.Execute(w, httpTemplateResponseData); err != nil {
			log.HTTPError(req, "Error executing template for "+filePath+": "+err.Error())
			middleware.WriteJsonError(w, http.StatusInternalServerError)
			return
		}
		return
	} else {
		// Set headers for non-HTML content
		w.Header().Set("Content-Length", fmt.Sprintf("%d", metadata.Size()))
	}

	// Serve the file
	http.ServeContent(w, req, metadata.Name(), metadata.ModTime(), file)

	log.HTTPDebug(req, "Served file: "+requestedPath+" to "+requestIP.String())
}

func LogoutHandler(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	log := middleware.GetLoggerFromContext(ctx)
	requestIP, err := middleware.GetRequestIPFromContext(ctx)
	if err != nil {
		log.HTTPWarning(req, "Error retrieving IP address stored in context for LogoutHandler: "+err.Error())
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}

	log.HTTPInfo(req, "Logout request received")
	// Invalidate cookies
	requestSessionIDCookie, err := req.Cookie("uit_session_id")
	if err != nil && err != http.ErrNoCookie {
		log.HTTPWarning(req, "Error retrieving session ID cookie for logout: "+err.Error())
		http.Redirect(w, req, "/login", http.StatusSeeOther)
		return
	}
	if err == http.ErrNoCookie || requestSessionIDCookie == nil || strings.TrimSpace(requestSessionIDCookie.Value) == "" {
		log.HTTPInfo(req, "No session ID cookie provided for logout")
		http.Redirect(w, req, "/login", http.StatusSeeOther)
		return
	}
	sessionID := strings.TrimSpace(requestSessionIDCookie.Value)
	config.DeleteAuthSession(sessionID)
	log.HTTPInfo(req, "Deleted auth session for logout: "+sessionID+" ("+requestIP.String()+")")
	// Clear cookies
	sessionIDCookie, basicCookie, bearerCookie, csrfCookie := middleware.GetAuthCookiesForResponse("", "", "", "", -time.Hour)
	http.SetCookie(w, sessionIDCookie)
	http.SetCookie(w, basicCookie)
	http.SetCookie(w, bearerCookie)
	http.SetCookie(w, csrfCookie)

	// Redirect to login page
	http.Redirect(w, req, "/login", http.StatusSeeOther)
}

func RejectRequest(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	log := middleware.GetLoggerFromContext(ctx)

	log.HTTPWarning(req, "Access denied to forbidden endpoint")
	middleware.WriteJsonError(w, http.StatusForbidden)
}
