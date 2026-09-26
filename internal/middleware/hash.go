package middleware

import (
	"bytes"
	"io"
	"net/http"

	"github.com/MaxPa1/go-metrics/internal/signature"
)

func HashMiddleware(key string) func(http.Handler) http.Handler {
	if key == "" {
		return func(next http.Handler) http.Handler { return next }
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			body, err := io.ReadAll(r.Body)
			if err != nil {
				http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
				return
			}

			if got := r.Header.Get(signature.Header); !signature.Valid(key, body, got) {
				http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
				return
			}
			r.Body = io.NopCloser(bytes.NewReader(body))

			hw := &hashResponseWriter{ResponseWriter: w}
			next.ServeHTTP(hw, r)
			hw.flush(key)
		})
	}
}

type hashResponseWriter struct {
	http.ResponseWriter
	status int
	body   bytes.Buffer
}

func (h *hashResponseWriter) WriteHeader(statusCode int) {
	if h.status == 0 {
		h.status = statusCode
	}
}

func (h *hashResponseWriter) Write(p []byte) (int, error) {
	if h.status == 0 {
		h.status = http.StatusOK
	}
	return h.body.Write(p)
}

func (h *hashResponseWriter) Flush() {
	if f, ok := h.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

func (h *hashResponseWriter) flush(key string) {
	if h.status == 0 {
		h.status = http.StatusOK
	}
	h.Header().Set(signature.Header, signature.Sum(key, h.body.Bytes()))
	h.ResponseWriter.WriteHeader(h.status)
	if h.body.Len() > 0 {
		_, _ = h.ResponseWriter.Write(h.body.Bytes())
	}
}
