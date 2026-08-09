package logger

import (
	"net/http"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var Log *zap.SugaredLogger = zap.NewNop().Sugar()

type (
	responseLogInfo struct {
		size   int
		status int
	}

	loggingResponseWriter struct {
		http.ResponseWriter
		responseInfo *responseLogInfo
	}
)

func Initialize(logLevel string) error {
	level, err := zap.ParseAtomicLevel(logLevel)
	if err != nil {
		return err
	}
	cfg := zap.NewProductionConfig()
	cfg.Level = level
	cfg.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	logger, err := cfg.Build()
	if err != nil {
		return err
	}
	Log = logger.Sugar()
	return nil
}

func (l *loggingResponseWriter) Write(array []byte) (int, error) {
	size, err := l.ResponseWriter.Write(array)
	l.responseInfo.size += size
	return size, err
}

func (l *loggingResponseWriter) WriteHeader(statusCode int) {
	l.ResponseWriter.WriteHeader(statusCode)
	l.responseInfo.status = statusCode
}

func RequestLogger(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		respLogInfo := responseLogInfo{}
		logWriter := loggingResponseWriter{
			ResponseWriter: w,
			responseInfo:   &respLogInfo,
		}
		h.ServeHTTP(&logWriter, r)

		duration := time.Since(start)

		Log.Infow("request completed",
			"uri", r.RequestURI,
			"method", r.Method,
			"duration", duration,
			"status", respLogInfo.status,
			"size", respLogInfo.size,
		)
	})
}
