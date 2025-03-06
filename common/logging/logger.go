package logging

import (
	"github.com/sirupsen/logrus"
)

// Logger 日志接口
type Logger interface {
	Log(action string, target string, details ...interface{})
	Debug(msg string, args ...interface{})
	Info(msg string, args ...interface{})
	Error(msg string, args ...interface{})
}

// DefaultLogger 默认日志实现
type DefaultLogger struct {
	log *logrus.Logger
}

// NewDefaultLogger 创建默认日志实例
func NewDefaultLogger() *DefaultLogger {
	l := logrus.New()
	// 设置日志格式、级别等
	l.SetFormatter(&logrus.TextFormatter{})
	l.SetLevel(logrus.InfoLevel)
	return &DefaultLogger{log: l}
}

// Log 记录操作日志
func (l *DefaultLogger) Log(action, target string, details ...interface{}) {
	l.log.WithFields(logrus.Fields{
		"action": action,
		"target": target,
	}).Info(details...)
}

// 实现其他接口方法...
