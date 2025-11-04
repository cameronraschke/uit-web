package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/netip"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

type PermissionConfig struct {
	Groups []GroupConfig `json:"groups"`
	Users  []UserConfig  `json:"users"`
}

type GroupConfig struct {
	ID               string               `json:"id"`
	Name             string               `json:"name"`
	Type             string               `json:"type"`
	IPRanges         []netip.Addr         `json:"ip_ranges"`
	AllowedServices  []ServicePermissions `json:"allowed_services"`
	AllowedEndpoints []string             `json:"allowed_endpoints"`
}

type UserConfig struct {
	ID               string               `json:"id"`
	Name             string               `json:"name"`
	Type             string               `json:"type"`
	IPRanges         []netip.Addr         `json:"ip_ranges"`
	AllowedServices  []ServicePermissions `json:"allowed_services"`
	AllowedEndpoints []string             `json:"allowed_endpoints"`
	IsAdmin          bool                 `json:"is_admin"`
	InGroups         []string             `json:"in_groups"`
}

type ServicePermissions int

const (
	EndpointPermissionAPIRead ServicePermissions = iota
	EndpointPermissionAPIWrite
	EndpointPermissionWebAccess
	EndpointPermissionFileRead
	EndpointPermissionFileWrite
	EndpointPermissionAdmin
)

var (
	ErrInvalidEndpointPermission = errors.New("invalid endpoint permission")
	mu                           sync.RWMutex
)

func InitPermissions() (*PermissionConfig, error) {
	permissionsDirectory := "/etc/uit-toolbox/acls/"
	fileInfo, err := os.Stat(permissionsDirectory)
	if err != nil || !fileInfo.IsDir() {
		return nil, fmt.Errorf("acls directory does not exist, skipping endpoint loading")
	}
	files, err := os.ReadDir(permissionsDirectory)
	if err != nil || len(files) == 0 {
		return nil, fmt.Errorf("failed to read files in the endpoints directory: %w", err)
	}

	permissionConfig := &PermissionConfig{}
	for _, file := range files {
		if file.IsDir() || !strings.HasSuffix(file.Name(), ".json") {
			continue
		}
		fileData, err := os.ReadFile(filepath.Join(permissionsDirectory + file.Name()))
		if err != nil {
			return nil, fmt.Errorf("failed to read permissions file %s: %w", file.Name(), err)
		}
		var fileConfig PermissionConfig
		if err := json.Unmarshal(fileData, &fileConfig); err != nil {
			return nil, fmt.Errorf("failed to unmarshal %s: %w", file.Name(), err)
		}

		permissionConfig.Groups = append(permissionConfig.Groups, fileConfig.Groups...)
		permissionConfig.Users = append(permissionConfig.Users, fileConfig.Users...)
	}
	return permissionConfig, nil
}

func (ep ServicePermissions) String() string {
	return [...]string{"API Read", "API Write", "Web Access", "File Read", "File Write", "Admin"}[ep]
}

func ParseEndpointPermission(permissionStr string) (ServicePermissions, error) {
	switch permissionStr {
	case "api_read":
		return EndpointPermissionAPIRead, nil
	case "api_write":
		return EndpointPermissionAPIWrite, nil
	case "web_access":
		return EndpointPermissionWebAccess, nil
	case "file_read":
		return EndpointPermissionFileRead, nil
	case "file_write":
		return EndpointPermissionFileWrite, nil
	case "admin":
		return EndpointPermissionAdmin, nil
	default:
		return -1, ErrInvalidEndpointPermission
	}
}

func (servicePermissions *ServicePermissions) UnmarshalJSON(data []byte) error {
	var servicePermissionString string
	if err := json.Unmarshal(data, &servicePermissionString); err != nil {
		return err
	}
	parsedServicePermission, err := ParseEndpointPermission(servicePermissionString)
	if err != nil {
		return fmt.Errorf("invalid service permission %q: %w", servicePermissionString, err)
	}
	*servicePermissions = parsedServicePermission
	return nil
}
