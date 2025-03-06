package formatter

import (
	"fmt"
)

// 格式化为Text格式
func FormatText(logData map[string]interface{}) string {
	return fmt.Sprintf("%v", logData)
}
