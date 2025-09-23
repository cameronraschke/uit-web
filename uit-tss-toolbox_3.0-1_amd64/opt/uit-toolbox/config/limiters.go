package config

func GetLimiter(limiterType string) *LimiterMap {
	appState := GetAppState()
	if appState == nil {
		return nil
	}

	switch limiterType {
	case "file":
		return appState.FileLimiter
	case "web":
		return appState.WebServerLimiter
	case "api":
		return appState.APILimiter
	case "auth":
		return appState.AuthLimiter
	default:
		return nil
	}
}
