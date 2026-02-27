package endpoints

import (
	"context"
	"crypto/rand"
	"database/sql"
	"errors"
	"fmt"
	"html/template"
	mathrand "math/rand"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"uit-toolbox/config"
	"uit-toolbox/database"
	"uit-toolbox/middleware"
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

func ValidateAuthFormInputSHA256(username string, password string) error {
	username = strings.TrimSpace(username)
	password = strings.TrimSpace(password)

	if err := middleware.IsSHA256String(username); err != nil {
		return fmt.Errorf("username has invalid SHA256 hash: %w", err)
	}
	if err := middleware.IsSHA256String(password); err != nil {
		return fmt.Errorf("password has invalid SHA256 hash: %w", err)
	}
	return nil
}

func CheckAuthCredentials(ctx context.Context, username string, password string) (bool, error) {
	username = strings.TrimSpace(username)
	password = strings.TrimSpace(password)
	if utf8.RuneCountInString(username) == 0 || utf8.RuneCountInString(password) == 0 {
		return false, fmt.Errorf("username or password is empty")
	}

	selectRepo, err := database.NewSelectRepo()
	if err != nil {
		return false, fmt.Errorf("cannot create select repo: %w", err)
	}
	usernameExists, dbPassword, err := selectRepo.CheckAuthCredentials(ctx, &username, &password)
	if err != nil || !usernameExists || dbPassword == nil {
		if errors.Is(err, sql.ErrNoRows) { // timing attacks
			buffer1 := make([]byte, mathrand.Intn(32)+32)
			_, _ = rand.Read(buffer1)
			buffer2 := make([]byte, mathrand.Intn(32)+32)
			_, _ = rand.Read(buffer2)
			pass1, err := bcrypt.GenerateFromPassword(buffer1, bcrypt.DefaultCost)
			if err != nil {
				return false, fmt.Errorf("error generating bcrypt hash #1 for timing attack: %w", err)
			}
			pass2, err := bcrypt.GenerateFromPassword(buffer2, bcrypt.DefaultCost)
			if err != nil {
				return false, fmt.Errorf("error generating bcrypt hash #2 for timing attack: %w", err)
			}
			bcrypt.CompareHashAndPassword(pass1, pass2)
			return false, fmt.Errorf("invalid username or password")
		}
		return false, fmt.Errorf("query error: %w", err)
	}

	// Compare plaintext password versus stored bcrypt
	if err := bcrypt.CompareHashAndPassword([]byte(*dbPassword), []byte(password)); err != nil {
		return false, fmt.Errorf("invalid password: %w", err)
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
	resolvedPath, err := middleware.GetRequestFileFromContext(ctx)
	if err != nil {
		log.Warn("Error retrieving requested file from context (FileServerHandler): " + err.Error())
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}

	// Previous path and file validation done in middleware
	if ctx.Err() != nil {
		log.Warn("Context error (FileServerHandler): " + ctx.Err().Error())
		middleware.WriteJsonError(w, http.StatusRequestTimeout)
		return
	}
	requestedFile, err := os.Open(resolvedPath)
	if err != nil {
		log.Warn("Error opening file (FileServerHandler): '" + resolvedPath + "': " + err.Error())
		middleware.WriteJsonError(w, http.StatusNotFound)
		return
	}
	defer requestedFile.Close()

	metadata, err := requestedFile.Stat()
	if err != nil {
		log.Error("Error retrieving file metadata (FileServerHandler): '" + resolvedPath + "': " + err.Error())
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}

	// if endpointConfig.EndpointType == "static_file" {
	// 	if endpointConfig.MaxDownloadSizeMB != 0 {
	// 		if metadata.Size() > endpointConfig.MaxDownloadSizeMB {
	// 			log.Warn("Requested file is too large (FileServerHandler): '" + resolvedPath + "' (" + fmt.Sprintf("%.2f", float64(metadata.Size())/1024/1024) + " MB, max allowed: " + fmt.Sprintf("%d", endpointConfig.MaxDownloadSizeMB) + " MB)")
	// 			middleware.WriteJsonError(w, http.StatusRequestEntityTooLarge)
	// 			return
	// 		}
	// 	} else {
	// 		log.Warn("Max download size is not set for static file endpoint (FileServerHandler): '" + resolvedPath + "', rejecting request")
	// 		middleware.WriteJsonError(w, http.StatusInternalServerError)
	// 		return
	// 	}
	// }

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
		log.Warn("Context error while serving file (FileServerHandler): '" + resolvedPath + "': " + ctx.Err().Error())
		return
	}

	// Serve the file
	http.ServeContent(w, req, metadata.Name(), metadata.ModTime(), requestedFile)
	log.Info("Served file '" + resolvedPath + "' (" + fmt.Sprintf("%.2f", float64(metadata.Size())/1024) + " KB)")
}

func WebServerHandler(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	log := middleware.GetLoggerFromContext(ctx)
	requestIP, err := middleware.GetRequestIPFromContext(ctx)
	if err != nil {
		log.Warn("Error retrieving IP address stored in context for WebServerHandler: " + err.Error())
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}
	requestedPath, err := middleware.GetRequestPathFromContext(ctx)
	if err != nil {
		log.Warn("Error retrieving URL stored in context for WebServerHandler: " + err.Error())
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

	log.Debug("Web request from " + requestIP.String() + " for " + requestedPath)

	// Get endpoint config
	endpointConfig, err := config.GetWebEndpointConfig(requestedPath)
	if err != nil {
		log.Warn("Cannot get endpoint config for endpoint " + requestedPath + ": " + err.Error() + "")
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}
	filePath, err := config.GetWebEndpointFilePath(endpointConfig)
	if err != nil {
		log.Warn("Cannot get file path for endpoint " + requestedPath + ": " + err.Error() + "")
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}
	// Open the file
	if ctx.Err() != nil {
		log.Warn("Context error (WebServerHandler): " + ctx.Err().Error())
		middleware.WriteJsonError(w, http.StatusRequestTimeout)
		return
	}
	file, err := os.Open(filePath)
	if err != nil {
		log.Warn("Cannot open file: " + filePath + ": " + err.Error() + "")
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
		log.Warn("Cannot stat file: " + filePath + ": " + err.Error() + "")
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}

	// Set headers
	contentType, err := config.GetWebEndpointContentType(endpointConfig)
	if err != nil {
		log.Warn("Cannot get content type for endpoint " + requestedPath + ": " + err.Error() + "")
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Content-Disposition", "inline; filename=\""+metadata.Name()+"\"")
	w.Header().Set("Last-Modified", metadata.ModTime().UTC().Format(http.TimeFormat))
	w.Header().Set("ETag", fmt.Sprintf(`"%x-%x"`, metadata.ModTime().Unix(), metadata.Size()))
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")

	if ctx.Err() != nil {
		log.Warn("Context cancelled while serving path: " + requestedPath + ": " + ctx.Err().Error())
		return
	}

	if filepath.Ext(filePath) == ".html" {
		parsedHTMLTemplate, err := template.ParseFiles(filePath)
		if err != nil {
			log.Warn("Cannot parse template file (" + filePath + "): " + err.Error())
			middleware.WriteJsonError(w, http.StatusInternalServerError)
			return
		}

		var httpTemplateResponseData = &HttpTemplateResponseData{}
		if endpointConfig.Requires != nil {
			// Generate nonce
			if slices.Contains(endpointConfig.Requires, "nonce") {
				httpTemplateResponseData.JsNonce = nonce
			}

			if slices.Contains(endpointConfig.Requires, "webmaster_contact") {
				webmasterName, webmasterEmail, err := config.GetWebmasterContact()
				if err != nil {
					log.Error("Cannot get webmaster contact info: " + err.Error() + "")
					middleware.WriteJsonError(w, http.StatusInternalServerError)
					return
				}
				httpTemplateResponseData.WebmasterName = webmasterName
				httpTemplateResponseData.WebmasterEmail = webmasterEmail
			}

			if slices.Contains(endpointConfig.Requires, "client_tag") {
				urlTag := req.URL.Query().Get("tagnumber")
				tagnumber, err := ConvertAndVerifyTagnumber(urlTag)
				if err != nil {
					log.Warn("Invalid tagnumber in URL: " + urlTag + " (" + err.Error() + ")")
					middleware.WriteJsonError(w, http.StatusBadRequest)
					return
				}
				if tagnumber == nil {
					log.Warn("No tagnumber provided in URL for endpoint that requires client_tag")
					middleware.WriteJsonError(w, http.StatusBadRequest)
					return
				}
				httpTemplateResponseData.ClientTag = strconv.FormatInt(*tagnumber, 10)
			}

			if slices.Contains(endpointConfig.Requires, "checkout_date") ||
				slices.Contains(endpointConfig.Requires, "return_date") ||
				slices.Contains(endpointConfig.Requires, "customer_name") {
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
			log.Error("Error executing template for " + filePath + ": " + err.Error())
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

	log.Debug("Served file: " + requestedPath + " to " + requestIP.String())
}

func RejectRequest(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	log := middleware.GetLoggerFromContext(ctx)

	log.Warn("Access denied to forbidden endpoint")
	middleware.WriteJsonError(w, http.StatusForbidden)
}
