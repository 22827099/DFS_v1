package logging

import (
	"strings"
	"strconv"
)

// 日志级别名称映射
var levelNames = map[LogLevel]string{
    LevelDebug: "DEBUG",
    LevelInfo:  "INFO",
    LevelWarn:  "WARN",
    LevelError: "ERROR",
    LevelFatal: "FATAL",
}

// 日志级别名称反向映射
var levelValues = map[string]LogLevel{
    "DEBUG": LevelDebug,
    "INFO":  LevelInfo,
    "WARN":  LevelWarn,
    "ERROR": LevelError,
    "FATAL": LevelFatal,
    "debug": LevelDebug,
    "info":  LevelInfo,
    "warn":  LevelWarn,
    "error": LevelError,
    "fatal": LevelFatal,
	"warning": LevelWarn,
    "WARNING": LevelWarn,
}

// 颜色常量
const (
    colorReset  = "\033[0m"
    colorRed    = "\033[31m"
    colorGreen  = "\033[32m"
    colorYellow = "\033[33m"
    colorBlue   = "\033[34m"
    colorPurple = "\033[35m"
)

// 日志颜色映射
var levelColors = map[LogLevel]string{
    LevelDebug: colorBlue,
    LevelInfo:  colorGreen,
    LevelWarn:  colorYellow,
    LevelError: colorRed,
    LevelFatal: colorPurple,
}

// LevelToString 将日志级别转换为字符串
func LevelToString(level LogLevel) string {
    if name, ok := levelNames[level]; ok {
        return name
    }
    return "UNKNOWN"
}

// StringToLevel 将字符串转换为日志级别 - 增强错误处理
func StringToLevel(s string) LogLevel {
    // 优先精确匹配
    if level, ok := levelValues[s]; ok {
        return level
    }
    
    // 将输入转换为小写以进行不区分大小写的匹配
    lowerS := strings.ToLower(s)
    
    // 检查常见前缀匹配
    if strings.HasPrefix(lowerS, "debug") {
        return LevelDebug
    }
    if strings.HasPrefix(lowerS, "info") {
        return LevelInfo
    }
    if strings.HasPrefix(lowerS, "warn") {
        return LevelWarn
    }
    if strings.HasPrefix(lowerS, "error") {
        return LevelError
    }
    if strings.HasPrefix(lowerS, "fatal") || strings.HasPrefix(lowerS, "crit") {
        return LevelFatal
    }
    
    // 尝试转换数字字符串
    if val, err := strconv.Atoi(s); err == nil {
        if val >= int(LevelDebug) && val <= int(LevelFatal) {
            return LogLevel(val)
        }
    }
    
    // 默认返回Info级别
    return LevelInfo
}