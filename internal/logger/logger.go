// Package logger настраивает zap-логгер и обёртку ResponseWriter для логирования ответов.
package logger

import (
	"net/http"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type (
	// ResponseLogInfo хранит статус и размер HTTP-ответа.
	ResponseLogInfo struct {
		Size   int
		Status int
	}

	// LoggingResponseWriter запоминает статус и размер ответа в ResponseInfo.
	LoggingResponseWriter struct {
		http.ResponseWriter
		ResponseInfo *ResponseLogInfo
	}
)

// Initialize создаёт zap-логгер с указанным уровнем логирования.
func Initialize(logLevel string) (*zap.SugaredLogger, error) {
	level, err := zap.ParseAtomicLevel(logLevel)
	if err != nil {
		return nil, err
	}
	cfg := zap.NewProductionConfig()
	cfg.Level = level
	cfg.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	l, err := cfg.Build()
	if err != nil {
		return nil, err
	}
	return l.Sugar(), nil
}

// Write записывает тело ответа и учитывает его размер.
func (l *LoggingResponseWriter) Write(array []byte) (int, error) {
	if l.ResponseInfo.Status == 0 {
		l.ResponseInfo.Status = http.StatusOK
	}
	size, err := l.ResponseWriter.Write(array)
	l.ResponseInfo.Size += size
	return size, err
}

// WriteHeader записывает статус ответа.
func (l *LoggingResponseWriter) WriteHeader(statusCode int) {
	l.ResponseWriter.WriteHeader(statusCode)
	l.ResponseInfo.Status = statusCode
}
