package logger

import (
	"net/http"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type (
	ResponseLogInfo struct {
		Size   int
		Status int
	}

	LoggingResponseWriter struct {
		http.ResponseWriter
		ResponseInfo *ResponseLogInfo
	}
)

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

func (l *LoggingResponseWriter) Write(array []byte) (int, error) {
	if l.ResponseInfo.Status == 0 {
		l.ResponseInfo.Status = http.StatusOK
	}
	size, err := l.ResponseWriter.Write(array)
	l.ResponseInfo.Size += size
	return size, err
}

func (l *LoggingResponseWriter) WriteHeader(statusCode int) {
	l.ResponseWriter.WriteHeader(statusCode)
	l.ResponseInfo.Status = statusCode
}
