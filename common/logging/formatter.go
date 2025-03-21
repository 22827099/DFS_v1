package logging

import (
    "encoding/json"
    "fmt"
    "time"
)

// TextFormatter 文本格式化器
type TextFormatter struct {
    TimeFormat string
    Colors     bool
}

// NewTextFormatter 创建新的文本格式化器
func NewTextFormatter() Formatter {
    return &TextFormatter{
        TimeFormat: "2006-01-02 15:04:05.000",
        Colors:     true,
    }
}

// Format 格式化日志条目
func (f *TextFormatter) Format(entry *LogEntry) string {
    levelName := LevelToString(entry.Level)
    timeStr := time.Unix(0, entry.Timestamp).Format(f.TimeFormat)
    
    var colorCode, resetCode string
    if f.Colors {
        colorCode = levelColors[entry.Level]
        resetCode = colorReset
    }
    
    // 基本部分
    output := fmt.Sprintf("[%s] %s%s%s %s", 
        timeStr, 
        colorCode, 
        levelName, 
        resetCode,
        entry.Message)
    
    // 添加字段
    if len(entry.Fields) > 0 {
        output += " "
        for k, v := range entry.Fields {
            output += fmt.Sprintf("%s=%v ", k, v)
        }
    }
    
    // 添加调用者信息
    if entry.Caller != "" {
        output += fmt.Sprintf(" (%s)", entry.Caller)
    }
    
    return output
}

// JSONFormatter JSON格式化器
type JSONFormatter struct {
    TimeFormat string
    Pretty     bool
}

// NewJSONFormatter 创建新的JSON格式化器
func NewJSONFormatter() Formatter {
    return &JSONFormatter{
        TimeFormat: time.RFC3339Nano,
        Pretty:     false,
    }
}

// Format 以JSON格式输出日志
func (f *JSONFormatter) Format(entry *LogEntry) string {
    data := map[string]interface{}{
        "time":    time.Unix(0, entry.Timestamp).Format(f.TimeFormat),
        "level":   LevelToString(entry.Level),
        "message": entry.Message,
    }
    
    // 添加字段
    for k, v := range entry.Fields {
        if _, exists := data[k]; !exists {
            data[k] = v
        }
    }
    
    // 添加调用者信息
    if entry.Caller != "" {
        data["caller"] = entry.Caller
    }
    
    var output []byte
    var err error
    
    if f.Pretty {
        output, err = json.MarshalIndent(data, "", "  ")
    } else {
        output, err = json.Marshal(data)
    }
    
    if err != nil {
        return fmt.Sprintf("{\"error\":\"json格式化失败: %v\", \"message\":\"%s\"}", err, entry.Message)
    }
    
    return string(output)
}