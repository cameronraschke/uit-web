package config

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"uit-toolbox/types"
)

type WebEndpointConfig struct {
	FilePath        string   `json:"file_path"`
	AllowedMethods  []string `json:"allowed_methods"`
	TLSRequired     *bool    `json:"tls_required"`
	AuthRequired    *bool    `json:"auth_required"`
	Requires        []string `json:"requires"`
	ACLUsers        []string `json:"acl_users"`
	ACLGroups       []string `json:"acl_groups"`
	HTTPVersion     string   `json:"http_version"`
	EndpointType    string   `json:"endpoint_type"`
	ContentType     string   `json:"content_type"`
	StatusCode      int      `json:"status_code"`
	Redirect        *bool    `json:"redirect"`
	RedirectURL     string   `json:"redirect_url"`
	MaxUploadSize   *int64   `json:"max_upload_size"`
	MaxDownloadSize *int64   `json:"max_download_size"`
}

func (as *AppState) GetFormConstraints() (*types.HTMLFormConstraints, error) {
	if as == nil {
		return nil, fmt.Errorf("nil AppState (GetFormConstraints)")
	}
	return as.appConfig.Load().FormConstraints.Load(), nil
}

func (as *AppState) GetFileUploadConstraints() (*types.FileUploadConstraints, error) {
	if as == nil {
		return nil, fmt.Errorf("nil AppState (GetFileUploadConstraints)")
	}
	return as.appConfig.Load().FileConstraints.Load(), nil
}

func InitWebEndpoints(as *AppState) error {
	if as == nil {
		return fmt.Errorf("app state is nil in InitWebEndpoints")
	}
	endpointConfigMap := make(map[string]WebEndpointConfig)
	configDir := "/etc/uit-toolbox/endpoints/"
	configDirMetadata, err := os.Stat(configDir)
	if err != nil || !configDirMetadata.IsDir() {
		return fmt.Errorf("endpoints directory does not exist, skipping endpoint loading")
	}
	allFiles, err := os.ReadDir(configDir)
	if err != nil || len(allFiles) == 0 {
		return fmt.Errorf("failed to read allFiles in the endpoints directory: %w", err)
	}
	for _, file := range allFiles {
		if file.IsDir() || !strings.HasSuffix(file.Name(), ".json") {
			continue
		}
		endpointConfigFiles, err := os.ReadFile(configDir + file.Name())
		if err != nil {
			return fmt.Errorf("failed to read web endpoints config file %s: %w", file.Name(), err)
		}

		endpoints := make(map[string]WebEndpointConfig)
		if err := json.Unmarshal(endpointConfigFiles, &endpoints); err != nil {
			return fmt.Errorf("failed to unmarshal web endpoints config JSON: %w", err)
		}

		for endpointPath, endpointData := range endpoints {
			merged := WebEndpointConfig{
				FilePath:        endpointData.FilePath,
				AllowedMethods:  endpointData.AllowedMethods,
				TLSRequired:     endpointData.TLSRequired,
				AuthRequired:    endpointData.AuthRequired,
				Requires:        endpointData.Requires,
				ACLUsers:        endpointData.ACLUsers,
				ACLGroups:       endpointData.ACLGroups,
				HTTPVersion:     endpointData.HTTPVersion,
				EndpointType:    endpointData.EndpointType,
				ContentType:     endpointData.ContentType,
				StatusCode:      endpointData.StatusCode,
				Redirect:        endpointData.Redirect,
				RedirectURL:     endpointData.RedirectURL,
				MaxUploadSize:   endpointData.MaxUploadSize,
				MaxDownloadSize: endpointData.MaxDownloadSize,
			}
			if len(merged.AllowedMethods) == 0 {
				merged.AllowedMethods = []string{"OPTIONS", "GET"}
			}
			if merged.TLSRequired == nil {
				merged.TLSRequired = new(bool)
				*merged.TLSRequired = true
			}
			if merged.AuthRequired == nil {
				merged.AuthRequired = new(bool)
				*merged.AuthRequired = true
			}
			if merged.Requires == nil {
				merged.Requires = []string{}
			}
			if merged.Redirect == nil {
				merged.Redirect = new(bool)
				*merged.Redirect = false
			}
			if merged.HTTPVersion == "" {
				merged.HTTPVersion = "HTTP/2.0"
			}
			if merged.EndpointType == "" {
				merged.EndpointType = "api"
			}
			if merged.ContentType == "" {
				merged.ContentType = "application/json; charset=utf-8"
			}
			if merged.StatusCode == 0 {
				merged.StatusCode = 200
			}
			if merged.MaxDownloadSize == nil || merged.MaxUploadSize == nil {
				switch endpointPath {
				case "/login", "/api/check_auth":
					merged.MaxUploadSize = new(int64)
					*merged.MaxUploadSize += as.appConfig.Load().FormConstraints.Load().LoginForm.MaxFormBytes
				case "/api/overview/note":
					merged.MaxUploadSize = new(int64)
					*merged.MaxUploadSize += as.appConfig.Load().FormConstraints.Load().GeneralNote.MaxFormBytes
				case "/api/inventory/update":
					maxOverallJSONSize := as.appConfig.Load().FormConstraints.Load().InventoryForm.MaxJSONBytes
					maxOverallImageSize := as.appConfig.Load().FileConstraints.Load().ImageConstraints.MaxFileSize * int64(as.appConfig.Load().FileConstraints.Load().ImageConstraints.MaxFileCount)
					maxOverallVideoSize := as.appConfig.Load().FileConstraints.Load().VideoConstraints.MaxFileSize * int64(as.appConfig.Load().FileConstraints.Load().VideoConstraints.MaxFileCount)
					merged.MaxUploadSize = new(int64)
					*merged.MaxUploadSize += maxOverallJSONSize + maxOverallImageSize + maxOverallVideoSize
				default:
					merged.MaxUploadSize = new(int64)
					*merged.MaxUploadSize = 0
				}
			}
			endpointConfigMap[endpointPath] = merged
			as.webEndpoints.Store(endpointPath, &merged)
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
