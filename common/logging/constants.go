package logging

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

// StringToLevel 将字符串转换为日志级别
func StringToLevel(s string) LogLevel {
    if level, ok := levelValues[s]; ok {
        return level
    }
    return LevelInfo // 默认为INFO级别
}