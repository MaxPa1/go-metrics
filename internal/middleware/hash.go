package middleware

import (
	"bytes"
	"io"
	"net/http"
	"sync"

	"github.com/MaxPa1/go-metrics/internal/signature"
)

const maxPooledBufferSize = 64 << 10

var (
	requestBufferPool = sync.Pool{New: func() any { return new(bytes.Buffer) }}
	hashWriterPool    = sync.Pool{New: func() any { return new(hashResponseWriter) }}
)

func HashMiddleware(key string) func(http.Handler) http.Handler {
	if key == "" {
		return func(next http.Handler) http.Handler { return next }
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			reqBuf := requestBufferPool.Get().(*bytes.Buffer)
			reqBuf.Reset()
			defer putRequestBuffer(reqBuf)

			if _, err := reqBuf.ReadFrom(r.Body); err != nil {
				http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
				return
			}
			body := reqBuf.Bytes()

			if got := r.Header.Get(signature.Header); !signature.Valid(key, body, got) {
				http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
				return
			}
			r.Body = io.NopCloser(bytes.NewReader(body))

			hw := hashWriterPool.Get().(*hashResponseWriter)
			hw.reset(w)
			defer hw.release()

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

func putRequestBuffer(buf *bytes.Buffer) {
	if buf.Cap() > maxPooledBufferSize {
		return
	}
	requestBufferPool.Put(buf)
}

func (h *hashResponseWriter) reset(w http.ResponseWriter) {
	h.ResponseWriter = w
	h.status = 0
	h.body.Reset()
}

func (h *hashResponseWriter) release() {
	h.ResponseWriter = nil
	if h.body.Cap() > maxPooledBufferSize {
		h.body = bytes.Buffer{}
	}
	hashWriterPool.Put(h)
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

func (h *hashResponseWriter) Flush() {}

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
