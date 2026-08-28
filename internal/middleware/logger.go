package middleware

import (
	"net/http"
	"time"

	"github.com/MaxPa1/go-metrics/internal/logger"
	"go.uber.org/zap"
)

func RequestLogger(log *zap.SugaredLogger) func(http.Handler) http.Handler {
	log = log.With("component", "http_middleware")

	return func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			respLogInfo := logger.ResponseLogInfo{}
			logWriter := logger.LoggingResponseWriter{
				ResponseWriter: w,
				ResponseInfo:   &respLogInfo,
			}
			h.ServeHTTP(&logWriter, r)

			duration := time.Since(start)

			log.Infow("request completed",
				"uri", r.RequestURI,
				"method", r.Method,
				"duration", duration,
				"status", respLogInfo.Status,
				"size", respLogInfo.Size,
			)
		})
	}
}
