package logging

import (
    "fmt"
    "io"
    "os"
    "path/filepath"

    "github.com/22827099/DFS_v1/common/types"
    "go.uber.org/zap"
    "go.uber.org/zap/zapcore"
    "gopkg.in/natefinch/lumberjack.v2"
)

// ZapLogger 实现基于zap的日志记录器
type ZapLogger struct {
    logger  *zap.Logger
    sugar   *zap.SugaredLogger
    level   zap.AtomicLevel
    config  *LogConfig
    context map[string]interface{}
}

// NewZapLogger 创建新的zap日志记录器
func NewZapLogger(config *LogConfig) Logger {
    // 如果配置为空，使用默认配置
    if config == nil {
        config = NewLogConfig()
    }

    // 创建日志级别原子变量
    var zapLevel zapcore.Level
    switch config.Level {
    case LevelDebug:
        zapLevel = zapcore.DebugLevel
    case LevelInfo:
        zapLevel = zapcore.InfoLevel
    case LevelWarn:
        zapLevel = zapcore.WarnLevel
    case LevelError:
        zapLevel = zapcore.ErrorLevel
    case LevelFatal:
        zapLevel = zapcore.FatalLevel
    default:
        zapLevel = zapcore.InfoLevel
    }

    level := zap.NewAtomicLevelAt(zapLevel)

    // 创建编码器配置
    encoderConfig := zapcore.EncoderConfig{
        TimeKey:        "time",
        LevelKey:       "level",
        NameKey:        "logger",
        CallerKey:      "caller",
        MessageKey:     "msg",
        StacktraceKey:  "stacktrace",
        LineEnding:     zapcore.DefaultLineEnding,
        EncodeLevel:    zapcore.CapitalLevelEncoder,
        EncodeTime:     zapcore.TimeEncoderOfLayout(config.TimeFormat),
        EncodeDuration: zapcore.SecondsDurationEncoder,
        EncodeCaller:   zapcore.ShortCallerEncoder,
    }

    // 根据配置选择编码器
    var encoder zapcore.Encoder
    if config.UseJSON {
        encoder = zapcore.NewJSONEncoder(encoderConfig)
    } else {
        encoder = zapcore.NewConsoleEncoder(encoderConfig)
    }

    // 设置输出
    var output zapcore.WriteSyncer
    if config.Output != nil {
        output = zapcore.AddSync(config.Output)
    } else if config.FilePath != "" {
        // 确保目录存在
        dir := filepath.Dir(config.FilePath)
        if err := os.MkdirAll(dir, 0755); err != nil {
            fmt.Fprintf(os.Stderr, "无法创建日志目录: %v\n", err)
        }
        
        // 创建日志轮转器
        rotator := &lumberjack.Logger{
            Filename:   config.FilePath,
            MaxSize:    config.MaxSize,
            MaxBackups: config.MaxBackups,
            MaxAge:     config.MaxAge,
            Compress:   config.Compress,
            LocalTime:  config.LocalTime,
        }
        output = zapcore.AddSync(rotator)
    } else {
        // 默认输出到控制台
        output = zapcore.AddSync(os.Stdout)
    }

    // 创建核心
    core := zapcore.NewCore(encoder, output, level)

    // 创建选项
    opts := []zap.Option{}
    if config.AddCaller {
        opts = append(opts, zap.AddCaller())
        if config.CallerSkip > 0 {
            opts = append(opts, zap.AddCallerSkip(config.CallerSkip))
        }
    }

    // 添加默认字段
    fields := []zap.Field{}
    for k, v := range config.DefaultTags {
        fields = append(fields, zap.Any(k, v))
    }

    if config.Name != "" {
        opts = append(opts, zap.Fields(zap.String("logger", config.Name)))
    }

    // 创建日志记录器
    logger := zap.New(core, opts...).With(fields...)

    return &ZapLogger{
        logger:  logger,
        sugar:   logger.Sugar(),
        level:   level,
        config:  config,
        context: make(map[string]interface{}),
    }
}

// Debug 记录调试级别日志
func (l *ZapLogger) Debug(format string, args ...interface{}) {
    l.sugar.Debugf(format, args...)
}

// Info 记录信息级别日志
func (l *ZapLogger) Info(format string, args ...interface{}) {
    l.sugar.Infof(format, args...)
}

// Warn 记录警告级别日志
func (l *ZapLogger) Warn(format string, args ...interface{}) {
    l.sugar.Warnf(format, args...)
}

// Error 记录错误级别日志
func (l *ZapLogger) Error(format string, args ...interface{}) {
    l.sugar.Errorf(format, args...)
}

// Fatal 记录致命错误日志
func (l *ZapLogger) Fatal(format string, args ...interface{}) {
    l.sugar.Fatalf(format, args...)
}

// DebugWithFields 记录带有结构化字段的调试日志
func (l *ZapLogger) DebugWithFields(msg string, fields map[string]interface{}) {
    l.LogWithFields(LevelDebug, msg, fields)
}

// InfoWithFields 记录带有结构化字段的信息日志
func (l *ZapLogger) InfoWithFields(msg string, fields map[string]interface{}) {
    l.LogWithFields(LevelInfo, msg, fields)
}

// WarnWithFields 记录带有结构化字段的警告日志
func (l *ZapLogger) WarnWithFields(msg string, fields map[string]interface{}) {
    l.LogWithFields(LevelWarn, msg, fields)
}

// ErrorWithFields 记录带有结构化字段的错误日志
func (l *ZapLogger) ErrorWithFields(msg string, fields map[string]interface{}) {
    l.LogWithFields(LevelError, msg, fields)
}

// FatalWithFields 记录带有结构化字段的致命错误日志
func (l *ZapLogger) FatalWithFields(msg string, fields map[string]interface{}) {
    l.LogWithFields(LevelFatal, msg, fields)
}

// Log 记录指定级别的日志
func (l *ZapLogger) Log(level LogLevel, format string, args ...interface{}) {
    msg := format
    if len(args) > 0 {
        msg = fmt.Sprintf(format, args...)
    }

    switch level {
    case LevelDebug:
        l.logger.Debug(msg)
    case LevelInfo:
        l.logger.Info(msg)
    case LevelWarn:
        l.logger.Warn(msg)
    case LevelError:
        l.logger.Error(msg)
    case LevelFatal:
        l.logger.Fatal(msg)
    default:
        l.logger.Info(msg)
    }
}

// LogWithFields 记录带有结构化字段的日志
func (l *ZapLogger) LogWithFields(level LogLevel, msg string, fields map[string]interface{}) {
    zapFields := make([]zap.Field, 0, len(fields))
    for k, v := range fields {
        zapFields = append(zapFields, zap.Any(k, v))
    }

    switch level {
    case LevelDebug:
        l.logger.Debug(msg, zapFields...)
    case LevelInfo:
        l.logger.Info(msg, zapFields...)
    case LevelWarn:
        l.logger.Warn(msg, zapFields...)
    case LevelError:
        l.logger.Error(msg, zapFields...)
    case LevelFatal:
        l.logger.Fatal(msg, zapFields...)
    default:
        l.logger.Info(msg, zapFields...)
    }
}

// LogWithNodeID 记录带有节点ID的日志
func (l *ZapLogger) LogWithNodeID(nodeID types.NodeID, level LogLevel, format string, args ...interface{}) {
    // 创建带有nodeID字段的临时logger
    nodeLogger := l.logger.With(zap.String("node_id", string(nodeID)))
    
    msg := format
    if len(args) > 0 {
        msg = fmt.Sprintf(format, args...)
    }

    switch level {
    case LevelDebug:
        nodeLogger.Debug(msg)
    case LevelInfo:
        nodeLogger.Info(msg)
    case LevelWarn:
        nodeLogger.Warn(msg)
    case LevelError:
        nodeLogger.Error(msg)
    case LevelFatal:
        nodeLogger.Fatal(msg)
    default:
        nodeLogger.Info(msg)
    }
}

// WithContext 创建带有上下文的日志记录器
func (l *ZapLogger) WithContext(ctx map[string]interface{}) Logger {
    if len(ctx) == 0 {
        return l
    }

    fields := make([]zap.Field, 0, len(ctx))
    for k, v := range ctx {
        fields = append(fields, zap.Any(k, v))
    }

    // 创建新的日志记录器
    newLogger := &ZapLogger{
        logger:  l.logger.With(fields...),
        level:   l.level,
        config:  l.config,
        context: make(map[string]interface{}),
    }
    
    // 复制上下文
    for k, v := range l.context {
        newLogger.context[k] = v
    }
    
    // 添加新上下文
    for k, v := range ctx {
        newLogger.context[k] = v
    }
    
    newLogger.sugar = newLogger.logger.Sugar()
    
    return newLogger
}

// WithNodeID 创建带有节点ID的日志记录器
func (l *ZapLogger) WithNodeID(nodeID types.NodeID) Logger {
    return l.WithContext(map[string]interface{}{"node_id": string(nodeID)})
}

// WithName 创建带有名称的日志记录器
func (l *ZapLogger) WithName(name string) Logger {
    if name == "" {
        return l
    }
    
    newLogger := &ZapLogger{
        logger:  l.logger.Named(name),
        level:   l.level,
        config:  l.config,
        context: make(map[string]interface{}),
    }
    
    // 复制上下文
    for k, v := range l.context {
        newLogger.context[k] = v
    }
    
    newLogger.sugar = newLogger.logger.Sugar()
    
    return newLogger
}

// SetLevel 设置日志级别
func (l *ZapLogger) SetLevel(level LogLevel) {
    var zapLevel zapcore.Level
    switch level {
    case LevelDebug:
        zapLevel = zapcore.DebugLevel
    case LevelInfo:
        zapLevel = zapcore.InfoLevel
    case LevelWarn:
        zapLevel = zapcore.WarnLevel
    case LevelError:
        zapLevel = zapcore.ErrorLevel
    case LevelFatal:
        zapLevel = zapcore.FatalLevel
    default:
        zapLevel = zapcore.InfoLevel
    }
    
    l.level.SetLevel(zapLevel)
    l.config.Level = level
}

// SetOutput 设置输出位置
func (l *ZapLogger) SetOutput(w io.Writer) {
    if w == nil {
        return
    }
    
    // 创建新的syncer
    output := zapcore.AddSync(w)
    
    // 获取当前的编码器
    var encoder zapcore.Encoder
    if l.config.UseJSON {
        encoder = zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig())
    } else {
        encoder = zapcore.NewConsoleEncoder(zap.NewDevelopmentEncoderConfig())
    }
    
    // 创建新的Core
    core := zapcore.NewCore(encoder, output, l.level)
    
    // 创建选项
    opts := []zap.Option{}
    if l.config.AddCaller {
        opts = append(opts, zap.AddCaller())
        if l.config.CallerSkip > 0 {
            opts = append(opts, zap.AddCallerSkip(l.config.CallerSkip))
        }
    }
    
    // 重新创建Logger
    l.logger = zap.New(core, opts...)
    l.sugar = l.logger.Sugar()
    
    // 应用上下文
    fields := make([]zap.Field, 0, len(l.context))
    for k, v := range l.context {
        fields = append(fields, zap.Any(k, v))
    }
    if len(fields) > 0 {
        l.logger = l.logger.With(fields...)
        l.sugar = l.logger.Sugar()
    }
}

// Sync 将缓冲的日志刷新到输出
func (l *ZapLogger) Sync() error {
    return l.logger.Sync()
}