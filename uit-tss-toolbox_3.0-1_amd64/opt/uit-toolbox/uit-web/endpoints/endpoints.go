package endpoints

import "encoding/json"

func jsonEncode(v any) (jsonStr string, err error) {
	jsonBytes, err := json.Marshal(v)
	if err != nil {
		return "", err
	}
	return string(jsonBytes), nil
}
