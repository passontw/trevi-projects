// 自定義簡易日誌記錄器
package logger

import "log"

// SimpleLogger 提供基本的日誌功能
type SimpleLogger struct {
	prefix string
}

// NewSimpleLogger 創建一個新的簡易日誌記錄器
func NewSimpleLogger(prefix string) *SimpleLogger {
	return &SimpleLogger{prefix: prefix}
}

// Info 記錄信息級別的日誌
func (l *SimpleLogger) Info(format string, args ...interface{}) {
	log.Printf(l.prefix+"INFO: "+format, args...)
}

// Warn 記錄警告級別的日誌
func (l *SimpleLogger) Warn(format string, args ...interface{}) {
	log.Printf(l.prefix+"WARN: "+format, args...)
}

// Error 記錄錯誤級別的日誌
func (l *SimpleLogger) Error(format string, args ...interface{}) {
	log.Printf(l.prefix+"ERROR: "+format, args...)
}

// Debug 記錄調試級別的日誌
func (l *SimpleLogger) Debug(format string, args ...interface{}) {
	log.Printf(l.prefix+"DEBUG: "+format, args...)
}
