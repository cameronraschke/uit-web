package config

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

type WebEndpointConfig struct {
	FilePath          string   `json:"file_path"`
	AllowedMethods    []string `json:"allowed_methods"`
	TLSRequired       *bool    `json:"tls_required"`
	AuthRequired      *bool    `json:"auth_required"`
	MaxUploadSizeKB   int64    `json:"max_upload_size_kb"`
	MaxDownloadSizeMB int64    `json:"max_download_size_mb"`
	Requires          []string `json:"requires"`
	ACLUsers          []string `json:"acl_users"`
	ACLGroups         []string `json:"acl_groups"`
	HTTPVersion       string   `json:"http_version"`
	EndpointType      string   `json:"endpoint_type"`
	ContentType       string   `json:"content_type"`
	StatusCode        int      `json:"status_code"`
	Redirect          *bool    `json:"redirect"`
	RedirectURL       string   `json:"redirect_url"`
}

func InitWebEndpoints(as *AppState) error {
	if as == nil {
		return fmt.Errorf("app state is nil in InitWebEndpoints")
	}
	populatedDefaultEndpoints := make(map[string]WebEndpointConfig)
	endpointsDirectory := "/etc/uit-toolbox/endpoints/"
	fileInfo, err := os.Stat(endpointsDirectory)
	if err != nil || !fileInfo.IsDir() {
		return fmt.Errorf("endpoints directory does not exist, skipping endpoint loading")
	}
	files, err := os.ReadDir(endpointsDirectory)
	if err != nil || len(files) == 0 {
		return fmt.Errorf("failed to read files in the endpoints directory: %w", err)
	}
	for _, file := range files {
		if file.IsDir() || !strings.HasSuffix(file.Name(), ".json") {
			continue
		}

		endpointsConfig, err := os.ReadFile(endpointsDirectory + file.Name())
		if err != nil {
			return fmt.Errorf("failed to read web endpoints config file %s: %w", file.Name(), err)
		}

		endpoints := make(map[string]WebEndpointConfig)
		if err := json.Unmarshal(endpointsConfig, &endpoints); err != nil {
			return fmt.Errorf("failed to unmarshal web endpoints config JSON: %w", err)
		}

		for endpointPath, endpointData := range endpoints {
			config := WebEndpointConfig{
				FilePath:          endpointData.FilePath,
				AllowedMethods:    endpointData.AllowedMethods,
				TLSRequired:       endpointData.TLSRequired,
				AuthRequired:      endpointData.AuthRequired,
				MaxUploadSizeKB:   endpointData.MaxUploadSizeKB << 10,
				MaxDownloadSizeMB: endpointData.MaxDownloadSizeMB << 20,
				Requires:          endpointData.Requires,
				ACLUsers:          endpointData.ACLUsers,
				ACLGroups:         endpointData.ACLGroups,
				HTTPVersion:       endpointData.HTTPVersion,
				EndpointType:      endpointData.EndpointType,
				ContentType:       endpointData.ContentType,
				StatusCode:        endpointData.StatusCode,
				Redirect:          endpointData.Redirect,
				RedirectURL:       endpointData.RedirectURL,
			}
			if len(config.AllowedMethods) == 0 {
				config.AllowedMethods = []string{"OPTIONS", "GET"}
			}
			if config.TLSRequired == nil {
				config.TLSRequired = new(bool)
				*config.TLSRequired = true
			}
			if config.AuthRequired == nil {
				config.AuthRequired = new(bool)
				*config.AuthRequired = true
			}
			if config.MaxUploadSizeKB == 0 {
				config.MaxUploadSizeKB = 20 << 10 // 20KB default max upload size
			}
			if config.MaxDownloadSizeMB == 0 {
				config.MaxDownloadSizeMB = 20 << 20 // 20MB default max download size
			}
			if config.Requires == nil {
				config.Requires = []string{}
			}
			if config.Redirect == nil {
				config.Redirect = new(bool)
				*config.Redirect = false
			}
			if config.HTTPVersion == "" {
				config.HTTPVersion = "HTTP/2.0"
			}
			if config.EndpointType == "" {
				config.EndpointType = "api"
			}
			if config.ContentType == "" {
				config.ContentType = "application/json; charset=utf-8"
			}
			if config.StatusCode == 0 {
				config.StatusCode = 200
			}
			populatedDefaultEndpoints[endpointPath] = config
			as.webEndpoints.Store(endpointPath, &config)
		}
	}
	return nil
}

func GetWebEndpointConfig(endpointPath string) (*WebEndpointConfig, error) {
	appState, err := GetAppState()
	if err != nil {
		return nil, fmt.Errorf("error getting app state in GetWebEndpointConfig: %w", err)
	}
	value, ok := appState.webEndpoints.Load(endpointPath)
	if !ok {
		return nil, fmt.Errorf("endpoint not found in config: %s", endpointPath)
	}
	endpointData, ok := value.(*WebEndpointConfig)
	if !ok {
		return nil, fmt.Errorf("invalid/missing endpoint data for: %s", endpointPath)
	}
	if endpointData == nil {
		return nil, fmt.Errorf("endpoint data is nil for: %s", endpointPath)
	}
	return endpointData, nil // return a copy
}

func GetWebEndpointFilePath(webEndpoint *WebEndpointConfig) (string, error) {
	if webEndpoint == nil {
		return "", fmt.Errorf("web endpoint config is nil in GetWebEndpointFilePath")
	}
	if strings.TrimSpace(webEndpoint.FilePath) == "" {
		return "", fmt.Errorf("file path field is empty in web endpoint config")
	}
	return webEndpoint.FilePath, nil
}

func IsWebEndpointAuthRequired(webEndpoint *WebEndpointConfig) (bool, error) {
	if webEndpoint == nil {
		return false, fmt.Errorf("web endpoint config is nil in IsWebEndpointAuthRequired")
	}
	if webEndpoint.TLSRequired == nil {
		return false, fmt.Errorf("auth required field is nil for endpoint")
	}
	return *webEndpoint.AuthRequired, nil
}

func IsWebEndpointHTTPSRequired(webEndpoint *WebEndpointConfig) (bool, error) {
	if webEndpoint == nil {
		return false, fmt.Errorf("web endpoint config is nil in IsWebEndpointHTTPSRequired")
	}
	if webEndpoint.TLSRequired == nil {
		return false, fmt.Errorf("TLS required field is nil for endpoint")
	}
	return *webEndpoint.TLSRequired, nil
}

func GetWebEndpointAllowedMethods(webEndpoint *WebEndpointConfig) ([]string, error) {
	if webEndpoint == nil {
		return nil, fmt.Errorf("web endpoint config is nil in GetWebEndpointAllowedMethods")
	}
	if len(webEndpoint.AllowedMethods) == 0 {
		return nil, fmt.Errorf("allowed methods field is empty for endpoint")
	}
	return webEndpoint.AllowedMethods, nil
}

func GetWebEndpointContentType(webEndpoint *WebEndpointConfig) (string, error) {
	if webEndpoint == nil {
		return "", fmt.Errorf("web endpoint config is nil in GetWebEndpointContentType")
	}
	if strings.TrimSpace(webEndpoint.ContentType) == "" {
		return "", fmt.Errorf("content type field is empty for endpoint")
	}
	return webEndpoint.ContentType, nil
}

func GetWebEndpointType(webEndpoint *WebEndpointConfig) (string, error) {
	if webEndpoint == nil {
		return "", fmt.Errorf("web endpoint config is nil in GetWebEndpointType")
	}
	if strings.TrimSpace(webEndpoint.EndpointType) == "" {
		return "", fmt.Errorf("endpoint type field is empty for endpoint")
	}
	return webEndpoint.EndpointType, nil
}

func GetWebEndpointRedirectURL(webEndpoint *WebEndpointConfig) (string, error) {
	if webEndpoint == nil {
		return "", fmt.Errorf("web endpoint config is nil in GetWebEndpointRedirectURL")
	}
	if strings.TrimSpace(webEndpoint.RedirectURL) == "" {
		return "", fmt.Errorf("redirect URL field is empty for endpoint")
	}
	if webEndpoint.Redirect == nil || !*webEndpoint.Redirect {
		return "", fmt.Errorf("redirect field is not set to true for endpoint")
	}
	return webEndpoint.RedirectURL, nil
}
