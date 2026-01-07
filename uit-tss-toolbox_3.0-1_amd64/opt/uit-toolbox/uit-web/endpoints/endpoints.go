package endpoints

import (
	"fmt"
	"html/template"
	"net/http"
	"os"
	"slices"
	"strconv"
	"strings"
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

func ConvertTagnumber(tagStr string) (int64, error) {
	tagStr = strings.TrimSpace(tagStr)
	if tagStr == "" {
		return 0, fmt.Errorf("tagnumber is empty")
	}
	tag, err := strconv.ParseInt(tagStr, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("error parsing tag: %v", err)
	}
	if tag < 1 || tag > 999999 {
		return 0, fmt.Errorf("invalid tag: out of range")
	}
	return tag, nil
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
	if !nonceExists {
		log.Error("Error retrieving CSP nonce from context")
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}
	if nonce == "" {
		log.Error("CSP nonce is empty")
		middleware.WriteJsonError(w, http.StatusInternalServerError)
		return
	}

	log.HTTPDebug(req, "File request from "+requestIP.String()+" for "+requestedPath)

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

	var parsedHTMLTemplate *template.Template
	if strings.HasSuffix(filePath, ".html") {

		parsedHTMLTemplate, err = template.ParseFiles(filePath)
		if err != nil {
			log.HTTPWarning(req, "Cannot parse template file ("+filePath+"): "+err.Error())
			middleware.WriteJsonError(w, http.StatusInternalServerError)
			return
		}

		db := database.NewRepo(config.GetDatabaseConn())
		if db == nil {
			log.HTTPWarning(req, "Cannot get database connection for template endpoint: "+requestedPath)
			middleware.WriteJsonError(w, http.StatusInternalServerError)
			return
		}

		var httpTemplateResponseData = &HttpTemplateResponseData{
			Departments: make(map[string]string), // Empty maps to avoid nil map errors in templates
			Domains:     make(map[string]string),
			Statuses:    make(map[string]string),
			Locations:   make(map[string]string),
		}
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

			if slices.Contains(endpointData.Requires, "statuses") {
				statuses, err := db.GetStatuses(ctx)
				if err != nil {
					log.HTTPError(req, "Cannot get status list from database: "+err.Error()+"")
					middleware.WriteJsonError(w, http.StatusInternalServerError)
					return
				}
				httpTemplateResponseData.Statuses = statuses
			}

			if slices.Contains(endpointData.Requires, "locations") {
				locations, err := db.GetLocations(ctx)
				if err != nil {
					log.HTTPError(req, "Cannot get location list from database: "+err.Error()+"")
					middleware.WriteJsonError(w, http.StatusInternalServerError)
					return
				}
				httpTemplateResponseData.Locations = locations
			}

			if slices.Contains(endpointData.Requires, "client_tag") {
				urlTag := req.URL.Query().Get("tagnumber")
				tagnumber, err := ConvertTagnumber(urlTag)
				if err != nil {
					log.HTTPWarning(req, "Invalid tagnumber in URL: "+urlTag+" ("+err.Error()+")")
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
			log.HTTPError(req, "Error executing template for "+filePath+": "+err.Error())
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
