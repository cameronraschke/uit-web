package config

import (
	"fmt"
	"strings"
)

type WebEndpoints map[string]WebEndpointConfig

type WebEndpointConfig struct {
	FilePath       string   `json:"file_path"`
	AllowedMethods []string `json:"allowed_methods"`
	TLSRequired    *bool    `json:"tls_required"`
	AuthRequired   *bool    `json:"auth_required"`
	Requires       []string `json:"requires"`
	ACLUsers       []string `json:"acl_users"`
	ACLGroups      []string `json:"acl_groups"`
	HTTPVersion    string   `json:"http_version"`
	EndpointType   string   `json:"endpoint_type"`
	ContentType    string   `json:"content_type"`
	StatusCode     int      `json:"status_code"`
	Redirect       *bool    `json:"redirect"`
	RedirectURL    string   `json:"redirect_url"`
}

func GetWebEndpointConfig(endpointPath string) (*WebEndpointConfig, error) {
	appState, err := GetAppState()
	if err != nil {
		return nil, fmt.Errorf("error getting app state in GetWebEndpointConfig: %w", err)
	}
	value, ok := appState.WebEndpoints.Load(endpointPath)
	if !ok {
		return nil, fmt.Errorf("endpoint not found in config: %s", endpointPath)
	}
	endpointData, ok := value.(*WebEndpointConfig)
	if !ok {
		return nil, fmt.Errorf("invalid/missing endpoint data for: %s", endpointPath)
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
