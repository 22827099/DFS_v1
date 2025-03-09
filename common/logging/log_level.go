package logging

// 日志级别常量
const (
	LevelDebug LogLevel = iota // 最详细的开发级别
	LevelInfo                  // 一般信息
	LevelWarn                  // 警告信息
	LevelError                 // 错误信息
	LevelFatal                 // 致命错误，通常会导致程序终止
)

// 日志级别名称，便于输出时使用
var levelNames = map[LogLevel]string{
	LevelDebug: "DEBUG",
	LevelInfo:  "INFO",
	LevelWarn:  "WARN",
	LevelError: "ERROR",
	LevelFatal: "FATAL",
}

// String 返回日志级别的字符串表示
func (l LogLevel) String() string {
	if name, ok := levelNames[l]; ok {
		return name
	}
	return "UNKNOWN"
}

// LevelFromString 从字符串解析日志级别
func LevelFromString(level string) LogLevel {
	switch level {
	case "DEBUG":
		return LevelDebug
	case "INFO":
		return LevelInfo
	case "WARN":
		return LevelWarn
	case "ERROR":
		return LevelError
	case "FATAL":
		return LevelFatal
	default:
		return LevelInfo // 默认为INFO级别
	}
}
