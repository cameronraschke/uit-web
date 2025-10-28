package endpoints

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"text/template"
	"time"
	config "uit-toolbox/config"
	database "uit-toolbox/database"
	"uit-toolbox/logger"
	middleware "uit-toolbox/middleware"
)

type RequestInfo struct {
	Ctx context.Context
	IP  string
	URL string
	Log logger.Logger
}

type ServerTime struct {
	Time string `json:"server_time"`
}

func GenerateNonce(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func GetRequestInfo(r *http.Request) (RequestInfo, error) {
	log := config.GetLogger()

	ctx := r.Context()
	if ctx == nil {
		return RequestInfo{}, errors.New("no context found in request")
	}

	ip, ok := middleware.GetRequestIP(r)
	if !ok {
		return RequestInfo{}, errors.New("no IP address stored in context")
	}

	url, ok := middleware.GetRequestURL(r)
	if !ok {
		return RequestInfo{}, errors.New("no URL stored in context")
	}

	return RequestInfo{Ctx: ctx, IP: ip, URL: url, Log: log}, nil
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

func FileServerHandler(w http.ResponseWriter, req *http.Request) {
	requestInfo, err := GetRequestInfo(req)
	if err != nil {
		middleware.WriteJsonError(w, http.StatusInternalServerError, "Internal server error")
		return
	}
	ctx := requestInfo.Ctx
	log := requestInfo.Log
	requestIP := requestInfo.IP
	requestURL := requestInfo.URL

	fullPath, resolvedPath, requestedFile, ok := middleware.GetRequestedFile(req)
	if !ok {
		log.Warning("no requested file stored in context")
		middleware.WriteJsonError(w, http.StatusInternalServerError, "Internal server error")
		return
	}

	if resolvedPath != fullPath {
		log.Warning("Resolved path does not match full path (" + requestIP + "): " + resolvedPath + " -> " + fullPath)
		middleware.WriteJsonError(w, http.StatusForbidden, "Forbidden")
		return
	}

	log.Debug("File request from " + requestIP + " for " + requestURL)

	// Previous path and file validation done in middleware
	// Open the file
	f, err := os.Open(fullPath)
	if err != nil {
		log.Warning("File not found: " + fullPath + " (" + err.Error() + ")")
		http.Error(w, "File not found", http.StatusNotFound)
		return
	}
	defer f.Close()

	// err = f.SetDeadline(time.Now().Add(30 * time.Second))
	// if err != nil {
	// 	log.Error("Cannot set file read deadline: " + fullPath + " (" + err.Error() + ")")
	// 	http.Error(w, "Internal server error", http.StatusInternalServerError)
	// 	return
	// }

	metadata, err := f.Stat()
	if err != nil {
		log.Error("Cannot stat file: " + fullPath + " (" + err.Error() + ")")
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	maxFileSize := int64(10) << 30 // 10 GiB
	if metadata.Size() > maxFileSize {
		log.Warning("File too large: " + fullPath + " (" + fmt.Sprintf("%d", metadata.Size()) + " bytes)")
		http.Error(w, "File too large", http.StatusRequestEntityTooLarge)
		return
	}

	// Get file info for headers
	if strings.HasSuffix(fullPath, ".deb") {
		w.Header().Set("Content-Type", "application/vnd.debian.binary-package")
	} else if strings.HasSuffix(fullPath, ".gz") {
		w.Header().Set("Content-Type", "application/gzip")
	} else if strings.HasSuffix(fullPath, ".img") {
		w.Header().Set("Content-Type", "application/vnd.efi.img")
	} else if strings.HasSuffix(fullPath, ".iso") {
		w.Header().Set("Content-Type", "application/vnd.efi.iso")
	} else if strings.HasSuffix(fullPath, ".squashfs") {
		w.Header().Set("Content-Type", "application/octet-stream")
	} else if strings.HasSuffix(fullPath, ".crt") {
		w.Header().Set("Content-Type", "application/x-x509-ca-cert")
	} else if strings.HasSuffix(fullPath, ".pem") {
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

	// Serve the file
	http.ServeContent(w, req, metadata.Name(), metadata.ModTime(), f)

	if ctx.Err() != nil {
		log.Warning("Request cancelled while serving file: " + requestedFile + " to " + requestIP + " (" + ctx.Err().Error() + ")")
		return
	}

	log.Info("Served file: " + requestedFile + " to " + requestIP)
}

func WebServerHandler(w http.ResponseWriter, req *http.Request) {
	requestInfo, err := GetRequestInfo(req)
	if err != nil {
		middleware.WriteJsonError(w, http.StatusInternalServerError, "Internal server error")
		return
	}
	ctx := requestInfo.Ctx
	log := requestInfo.Log
	requestIP := requestInfo.IP
	requestURL := requestInfo.URL

	fullPath, resolvedPath, requestedFile, ok := middleware.GetRequestedFile(req)
	if !ok {
		log.Warning("no requested file stored in context")
		middleware.WriteJsonError(w, http.StatusInternalServerError, "Internal server error")
		return
	}

	if resolvedPath != fullPath {
		log.Warning("Resolved path does not match full path (" + requestIP + "): " + resolvedPath + " -> " + fullPath)
		middleware.WriteJsonError(w, http.StatusForbidden, "Forbidden")
		return
	}

	log.Debug("File request from " + requestIP + " for " + requestURL)

	// Previous path and file validation done in middleware
	// Open the file
	f, err := os.Open(fullPath)
	if err != nil {
		log.Warning("File not found: " + fullPath + " (" + err.Error() + ")")
		http.Error(w, "File not found", http.StatusNotFound)
		return
	}
	defer f.Close()

	// err = f.SetDeadline(time.Now().Add(30 * time.Second))
	// if err != nil {
	// 	log.Error("Cannot set file read deadline: " + fullPath + " (" + err.Error() + ")")
	// 	http.Error(w, "Internal server error", http.StatusInternalServerError)
	// 	return
	// }

	metadata, err := f.Stat()
	if err != nil {
		log.Error("Cannot stat file: " + fullPath + " (" + err.Error() + ")")
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	maxFileSize := int64(128) << 20 // 128 MiB
	if metadata.Size() > maxFileSize {
		log.Warning("File too large: " + fullPath + " (" + fmt.Sprintf("%d", metadata.Size()) + " bytes)")
		http.Error(w, "File too large", http.StatusRequestEntityTooLarge)
		return
	}

	// Set headers
	if strings.HasSuffix(requestedFile, ".html") {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		nonce, err := GenerateNonce(24)
		if err != nil {
			log.Error("Cannot generate CSP nonce: " + err.Error())
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Security-Policy",
			"default-src 'none'; "+
				"script-src 'self' 'nonce-"+nonce+"'; "+
				"style-src 'self'; "+
				"img-src 'self' blob: data:; "+
				"font-src 'self'; "+
				"connect-src 'self'; "+
				"frame-ancestors 'none'; "+
				"form-action 'self'; "+
				"base-uri 'self'")

		htmlTemp, err := template.ParseFiles(resolvedPath)
		if err != nil {
			log.Warning("Cannot parse template file (" + resolvedPath + "): " + err.Error())
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}
		webmasterName, webmasterEmail, err := config.GetWebmasterContact()
		if err != nil {
			log.Error("Cannot get webmaster contact info: " + err.Error())
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		db := database.NewRepo(config.GetDatabaseConn())

		departments, err := db.GetDepartments(ctx)
		if err != nil {
			log.Error("Cannot get department list from database: " + err.Error())
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		domains, err := db.GetDomains(ctx)
		if err != nil {
			log.Error("Cannot get domain list from database: " + err.Error())
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		statuses, err := db.GetStatuses(ctx)
		if err != nil {
			log.Error("Cannot get status list from database: " + err.Error())
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}
		urlTag := req.URL.Query().Get("tagnumber")
		urlTag = strings.TrimSpace(urlTag)

		templateData := struct {
			JsNonce        string
			WebmasterName  string
			WebmasterEmail string
			Departments    map[string]string
			Domains        map[string]string
			ClientTag      string
			Statuses       map[string]string
		}{
			JsNonce:        nonce,
			WebmasterName:  webmasterName,
			WebmasterEmail: webmasterEmail,
			Departments:    departments,
			Domains:        domains,
			ClientTag:      urlTag,
			Statuses:       statuses,
		}

		// Execute the template
		err = htmlTemp.Execute(w, templateData)
		if err != nil {
			log.Error("Error executing template for " + resolvedPath + ": " + err.Error())
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}
		return
	} else if strings.HasSuffix(requestedFile, ".css") {
		w.Header().Set("Content-Type", "text/css; charset=utf-8")
	} else if strings.HasSuffix(requestedFile, ".js") {
		w.Header().Set("Content-Type", "application/javascript; charset=utf-8")
	} else if strings.HasSuffix(requestedFile, ".png") || strings.HasSuffix(requestedFile, ".ico") {
		w.Header().Set("Content-Type", "image/png")
	} else {
		log.Warning("Unknown file type requested: " + requestedFile)
		http.Error(w, "Unsupported Media Type", http.StatusUnsupportedMediaType)
		return
	}

	// Set headers
	w.Header().Set("Content-Security-Policy", "default-src 'none'; script-src 'self'; style-src 'self'; img-src 'self'; font-src 'self'; connect-src 'self'; frame-ancestors 'none'; form-action 'self'; base-uri 'self'")
	w.Header().Set("Content-Length", fmt.Sprintf("%d", metadata.Size()))
	w.Header().Set("Content-Disposition", "inline; filename=\""+metadata.Name()+"\"")
	w.Header().Set("Last-Modified", metadata.ModTime().UTC().Format(http.TimeFormat))
	w.Header().Set("ETag", fmt.Sprintf(`"%x-%x"`, metadata.ModTime().Unix(), metadata.Size()))
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")

	// Serve the file
	http.ServeContent(w, req, metadata.Name(), metadata.ModTime(), f)

	if ctx.Err() != nil {
		log.Warning("Request cancelled while serving file: " + requestedFile + " to " + requestIP + " (" + ctx.Err().Error() + ")")
		return
	}

	log.Debug("Served file: " + requestedFile + " to " + requestIP)
}

func LogoutHandler(w http.ResponseWriter, req *http.Request) {
	requestInfo, err := GetRequestInfo(req)
	if err != nil {
		middleware.WriteJsonError(w, http.StatusInternalServerError, "Internal server error")
		return
	}
	log := requestInfo.Log
	requestIP := requestInfo.IP
	requestURL := requestInfo.URL

	log.Info("Logout request from " + requestIP + " for " + requestURL)
	// Invalidate cookies
	requestSessionIDCookie, err := req.Cookie("uit_session_id")
	if err != nil && err != http.ErrNoCookie {
		log.Warning("Error retrieving session ID cookie for logout: " + err.Error() + " (" + requestIP + ")")
		http.Redirect(w, req, "/login", http.StatusSeeOther)
		return
	}
	if requestSessionIDCookie == nil || strings.TrimSpace(requestSessionIDCookie.Value) == "" {
		log.Info("No session ID cookie provided for logout: " + requestIP)
		http.Redirect(w, req, "/login", http.StatusSeeOther)
		return
	}
	sessionID := strings.TrimSpace(requestSessionIDCookie.Value)
	config.DeleteAuthSession(sessionID)
	log.Info("Deleted auth session for logout: " + sessionID + " (" + requestIP + ")")
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
	requestInfo, err := GetRequestInfo(req)
	if err != nil {
		middleware.WriteJsonError(w, http.StatusInternalServerError, "Internal server error")
		return
	}
	requestIP := requestInfo.IP
	requestURL := requestInfo.URL
	log := requestInfo.Log

	log.Warning("access denied: " + requestIP + " tried to access " + requestURL)
	http.Error(w, "Access denied", http.StatusForbidden)
}
