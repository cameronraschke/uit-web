package endpoints

import (
	"errors"
	"fmt"
	"net/http"
	"os"
	"slices"
	"strconv"
	"strings"
	"text/template"
	"time"
	config "uit-toolbox/config"
	database "uit-toolbox/database"
	middleware "uit-toolbox/middleware"
)

type HttpTemplateResponseData struct {
	JsNonce        string
	WebmasterName  string
	WebmasterEmail string
	Departments    map[string]string
	Domains        map[string]string
	ClientTag      string
	Statuses       map[string]string
	Locations      map[string]string
	CheckoutDate   string
	ReturnDate     string
	CustomerName   string
}

type ServerTime struct {
	Time string `json:"server_time"`
}

func ConvertRequestTagnumber(r *http.Request) (int64, bool) {
	tag := r.URL.Query().Get("tagnumber")
	tagnumber, err := strconv.ParseInt(tag, 10, 64)
	if err != nil {
		return 0, false
	}
	if tagnumber < 1 || tagnumber > 999999 {
		return 0, false
	}
	return tagnumber, true
}

func ConvertTagnumber(tag string) (int64, error) {
	tag = strings.TrimSpace(tag)
	if tag == "" {
		return 0, errors.New("tagnumber is empty")
	}
	tagnumber, err := strconv.ParseInt(tag, 10, 64)
	if err != nil {
		return 0, errors.New("invalid tagnumber: " + err.Error())
	}
	if tagnumber < 1 || tagnumber > 999999 {
		return 0, errors.New("invalid tagnumber: out of range")
	}
	return tagnumber, nil
}

func FileServerHandler(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	log, loggerExists, err := middleware.GetLoggerFromContext(ctx)
	if !loggerExists || err != nil {
		fmt.Println("Failed to get logger from context for FileServerHandler: " + err.Error())
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	requestIP, requestIPExists := middleware.GetRequestIPFromRequestContext(req)
	if !requestIPExists {
		log.Warning("No IP address stored in context for FileServerHandler")
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}

	resolvedPath, resolvedPathExists := middleware.GetRequestFileFromRequestContext(req)
	if !resolvedPathExists {
		log.Warning("no requested file stored in context")
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}

	log.HTTPDebug(req, "File request received")

	// Previous path and file validation done in middleware
	// Open the file
	f, err := os.Open(resolvedPath)
	if err != nil {
		log.HTTPWarning(req, "File not found for FileServerHandler: "+err.Error())
		http.Error(w, "File not found", http.StatusNotFound)
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
	if strings.HasSuffix(resolvedPath, ".deb") {
		w.Header().Set("Content-Type", "application/vnd.debian.binary-package")
	} else if strings.HasSuffix(resolvedPath, ".gz") {
		w.Header().Set("Content-Type", "application/gzip")
	} else if strings.HasSuffix(resolvedPath, ".img") {
		w.Header().Set("Content-Type", "application/vnd.efi.img")
	} else if strings.HasSuffix(resolvedPath, ".iso") {
		w.Header().Set("Content-Type", "application/vnd.efi.iso")
	} else if strings.HasSuffix(resolvedPath, ".squashfs") {
		w.Header().Set("Content-Type", "application/octet-stream")
	} else if strings.HasSuffix(resolvedPath, ".crt") {
		w.Header().Set("Content-Type", "application/x-x509-ca-cert")
	} else if strings.HasSuffix(resolvedPath, ".pem") {
		w.Header().Set("Content-Type", "application/pem-certificate-chain")
	} else {
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
	log := config.GetLogger()
	requestIP, ok := middleware.GetRequestIPFromRequestContext(req)
	if !ok {
		log.Warning("No IP address stored in context")
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}
	requestURL, ok := middleware.GetRequestURLFromRequestContext(req)
	if !ok {
		log.Warning("No URL stored in context")
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}
	requestedPath, ok := middleware.GetRequestPathFromRequestContext(req)
	if !ok {
		log.Warning("no requested file stored in context")
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}
	requestQueries, _ := middleware.GetRequestQueryFromContext(ctx)

	log.Debug("File request from " + requestIP.String() + " for " + requestURL)

	// Get endpoint config
	endpointData, err := config.GetWebEndpointConfig(requestedPath)
	if err != nil {
		log.Warning("Cannot get endpoint config for endpoint: " + requestURL + " (" + err.Error() + ")")
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}
	filePath, err := config.GetWebEndpointFilePath(endpointData)
	if err != nil {
		log.Warning("Cannot get file path for endpoint: " + requestedPath + " (" + err.Error() + ")")
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}
	// Open the file
	file, err := os.Open(filePath)
	if err != nil {
		log.Warning("Cannot open file: " + filePath + " (" + err.Error() + ")")
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
		log.Error("Cannot stat file: " + filePath + " (" + err.Error() + ")")
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}

	maxFileSize, err := config.GetMaxUploadSize()
	if err != nil {
		log.Error("Cannot get max upload size from config: " + err.Error())
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}
	if metadata.Size() > maxFileSize {
		log.Warning("File too large: " + filePath + " (" + fmt.Sprintf("%d", metadata.Size()) + " bytes)")
		http.Error(w, "File too large", http.StatusRequestEntityTooLarge)
		return
	}

	// Set headers
	contentType, err := config.GetWebEndpointContentType(endpointData)
	if err != nil {
		log.Warning("Cannot get content type for endpoint: " + requestedPath + " (" + err.Error() + ")")
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", contentType)

	var parsedHTMLTemplate *template.Template
	if strings.HasSuffix(filePath, ".html") {

		parsedHTMLTemplate, err = template.ParseFiles(filePath)
		if err != nil {
			log.Warning("Cannot parse template file (" + filePath + "): " + err.Error())
			middleware.WriteJsonError(w, http.StatusInternalServerError)
			return
		}

		db := database.NewRepo(config.GetDatabaseConn())
		if db == nil {
			log.Warning("Cannot get database connection for template endpoint: " + requestedPath)
			middleware.WriteJsonError(w, http.StatusInternalServerError)
			return
		}

		var httpTemplateResponseData = HttpTemplateResponseData{}
		if endpointData.Requires != nil {

			// Generate nonce
			if slices.Contains(endpointData.Requires, "nonce") {
				nonce, ok := middleware.GetNonceFromRequestContext(req)
				if !ok {
					log.Error("Error retrieving CSP nonce from context")
					middleware.WriteJsonError(w, http.StatusInternalServerError)
					return
				}
				httpTemplateResponseData.JsNonce = nonce
			}

			if slices.Contains(endpointData.Requires, "webmaster_contact") {
				webmasterName, webmasterEmail, err := config.GetWebmasterContact()
				if err != nil {
					log.Error("Cannot get webmaster contact info: " + err.Error())
					middleware.WriteJsonError(w, http.StatusInternalServerError)
					return
				}
				httpTemplateResponseData.WebmasterName = webmasterName
				httpTemplateResponseData.WebmasterEmail = webmasterEmail
			}

			if slices.Contains(endpointData.Requires, "departments") {
				departments, err := db.GetDepartments(ctx)
				if err != nil {
					log.Error("Cannot get department list from database: " + err.Error())
					middleware.WriteJsonError(w, http.StatusInternalServerError)
					return
				}
				httpTemplateResponseData.Departments = departments
			}

			if slices.Contains(endpointData.Requires, "domains") {
				domains, err := db.GetDomains(ctx)
				if err != nil {
					log.Error("Cannot get domain list from database: " + err.Error())
					middleware.WriteJsonError(w, http.StatusInternalServerError)
					return
				}
				httpTemplateResponseData.Domains = domains
			}

			if slices.Contains(endpointData.Requires, "statuses") {
				statuses, err := db.GetStatuses(ctx)
				if err != nil {
					log.Error("Cannot get status list from database: " + err.Error())
					middleware.WriteJsonError(w, http.StatusInternalServerError)
					return
				}
				httpTemplateResponseData.Statuses = statuses
			}

			if slices.Contains(endpointData.Requires, "locations") {
				locations, err := db.GetLocations(ctx)
				if err != nil {
					log.Error("Cannot get location list from database: " + err.Error())
					middleware.WriteJsonError(w, http.StatusInternalServerError)
					return
				}
				httpTemplateResponseData.Locations = locations
			}

			if slices.Contains(endpointData.Requires, "client_tag") {
				urlTag := req.URL.Query().Get("tagnumber")
				tagnumber, err := ConvertTagnumber(urlTag)
				if err != nil {
					log.Warning("Invalid tagnumber in URL: " + urlTag + " (" + err.Error() + ")")
					middleware.WriteJsonError(w, http.StatusBadRequest)
					return
				}
				httpTemplateResponseData.ClientTag = strconv.FormatInt(tagnumber, 10)
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
		err = parsedHTMLTemplate.Execute(w, httpTemplateResponseData)
		if err != nil {
			log.Error("Error executing template for " + filePath + ": " + err.Error())
			middleware.WriteJsonError(w, http.StatusInternalServerError)
			return
		}
		return
	}

	// Set headers
	w.Header().Set("Content-Length", fmt.Sprintf("%d", metadata.Size()))
	w.Header().Set("Content-Disposition", "inline; filename=\""+metadata.Name()+"\"")
	w.Header().Set("Last-Modified", metadata.ModTime().UTC().Format(http.TimeFormat))
	w.Header().Set("ETag", fmt.Sprintf(`"%x-%x"`, metadata.ModTime().Unix(), metadata.Size()))
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")

	if ctx.Err() != nil {
		log.HTTPWarning(req, "Context cancelled while serving path: "+requestedPath+": "+ctx.Err().Error())
		return
	}

	// Serve the file
	http.ServeContent(w, req, metadata.Name(), metadata.ModTime(), file)

	log.Debug("Served file: " + requestedPath + " to " + requestIP.String())
}

func LogoutHandler(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	log, loggerExists, err := middleware.GetLoggerFromContext(ctx)
	if !loggerExists || err != nil {
		fmt.Println("Failed to get logger from context for LogoutHandler: " + err.Error())
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	requestIP, requestIPExists := middleware.GetRequestIPFromContext(ctx)
	if !requestIPExists {
		log.HTTPWarning(req, "No IP address stored in context for LogoutHandler")
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
	log.Info("Deleted auth session for logout: " + sessionID + " (" + requestIP.String() + ")")
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
	log, loggerExists, err := middleware.GetLoggerFromContext(ctx)
	if !loggerExists || err != nil {
		fmt.Println("Failed to get logger from context for RejectRequest: " + err.Error())
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}

	log.HTTPWarning(req, "Access denied to forbidden endpoint")
	middleware.WriteJsonError(w, http.StatusForbidden)
}
