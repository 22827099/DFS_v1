package formatter

import (
	"encoding/json"
)

// 格式化为JSON格式
func FormatJSON(logData map[string]interface{}) (string, error) {
	jsonLog, err := json.Marshal(logData)
	if err != nil {
		return "", err
	}
	return string(jsonLog), nil
}
