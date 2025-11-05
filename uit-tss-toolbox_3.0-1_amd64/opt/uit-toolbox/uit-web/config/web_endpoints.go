package config

import (
	"fmt"
	"strings"
)

type WebEndpoints map[string]WebEndpoint

type WebEndpoint struct {
	FilePath       string   `json:"file_path"`
	AllowedMethods []string `json:"allowed_methods"`
	TLSRequired    *bool    `json:"tls_required"`
	AuthRequired   *bool    `json:"auth_required"`
	ACLUsers       []string `json:"acl_users"`
	ACLGroups      []string `json:"acl_groups"`
	HTTPVersion    string   `json:"http_version"`
	EndpointType   string   `json:"endpoint_type"`
	ContentType    string   `json:"content_type"`
	StatusCode     int      `json:"status_code"`
	Redirect       *bool    `json:"redirect"`
	RedirectURL    string   `json:"redirect_url"`
}

func GetWebEndpointConfig(endpointPath string) (WebEndpoint, error) {
	appState := GetAppState()
	if appState == nil {
		return WebEndpoint{}, fmt.Errorf("cannot get web endpoint, app state is not initialized")
	}
	value, ok := appState.WebEndpoints.Load(endpointPath)
	if !ok {
		return WebEndpoint{}, fmt.Errorf("endpoint not found: %s", endpointPath)
	}
	endpointData, ok := value.(*WebEndpoint)
	if !ok {
		return WebEndpoint{}, fmt.Errorf("invalid endpoint data for: %s", endpointPath)
	}
	return *endpointData, nil // return copy
}

func GetWebEndpointFilePath(webEndpoint WebEndpoint) (string, error) {
	if strings.TrimSpace(webEndpoint.FilePath) == "" {
		return "", fmt.Errorf("file path is empty for endpoint")
	}
	return webEndpoint.FilePath, nil
}

func IsWebEndpointAuthRequired(webEndpoint WebEndpoint) (bool, error) {
	if webEndpoint.AuthRequired == nil {
		return false, fmt.Errorf("auth required field is nil for endpoint")
	}
	return true, nil
}

func IsWebEndpointHTTPSRequired(webEndpoint WebEndpoint) (bool, error) {
	if webEndpoint.TLSRequired == nil {
		return false, fmt.Errorf("TLS required field is nil for endpoint")
	}
	return true, nil
}

func GetWebEndpointAllowedMethods(webEndpoint WebEndpoint) ([]string, error) {
	if len(webEndpoint.AllowedMethods) == 0 {
		return nil, fmt.Errorf("allowed methods list is empty for endpoint")
	}
	return webEndpoint.AllowedMethods, nil
}

func GetWebEndpointContentType(webEndpoint WebEndpoint) (string, error) {
	return webEndpoint.ContentType, nil
}

func GetWebEndpointType(webEndpoint WebEndpoint) (string, error) {
	if strings.TrimSpace(webEndpoint.EndpointType) == "" {
		return "", fmt.Errorf("endpoint type is empty for endpoint")
	}
	return webEndpoint.EndpointType, nil
}
