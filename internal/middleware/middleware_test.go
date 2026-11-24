package middleware

import (
	"bytes"
	"compress/gzip"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/MaxPa1/go-metrics/internal/compress"
	"github.com/MaxPa1/go-metrics/internal/signature"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
)

func echoHandler(t *testing.T) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(body)
	})
}

func gzipBytes(t *testing.T, data []byte) []byte {
	t.Helper()
	buf, err := compress.Compress(data)
	require.NoError(t, err)
	return buf.Bytes()
}

func TestGzipMiddleware(t *testing.T) {
	payload := []byte(`{"id":"PollCount","type":"counter","delta":1}`)

	tests := []struct {
		name            string
		body            []byte
		contentEncoding string
		acceptEncoding  string
		wantStatus      int
		wantEncoding    string
		wantBody        []byte
	}{
		{
			name:            "decompresses request and compresses response",
			body:            gzipBytes(t, payload),
			contentEncoding: "gzip",
			acceptEncoding:  "gzip",
			wantStatus:      http.StatusOK,
			wantEncoding:    "gzip",
			wantBody:        payload,
		},
		{
			name:            "decompresses request, plain response",
			body:            gzipBytes(t, payload),
			contentEncoding: "gzip",
			wantStatus:      http.StatusOK,
			wantBody:        payload,
		},
		{
			name:           "plain request, compressed response",
			body:           payload,
			acceptEncoding: "gzip",
			wantStatus:     http.StatusOK,
			wantEncoding:   "gzip",
			wantBody:       payload,
		},
		{
			name:       "plain request and response",
			body:       payload,
			wantStatus: http.StatusOK,
			wantBody:   payload,
		},
		{
			name:            "invalid gzip body",
			body:            []byte("not gzip"),
			contentEncoding: "gzip",
			wantStatus:      http.StatusBadRequest,
			wantBody:        []byte{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/update", bytes.NewReader(tt.body))
			if tt.contentEncoding != "" {
				req.Header.Set("Content-Encoding", tt.contentEncoding)
			}
			if tt.acceptEncoding != "" {
				req.Header.Set("Accept-Encoding", tt.acceptEncoding)
			}
			rec := httptest.NewRecorder()

			GzipMiddleware(echoHandler(t)).ServeHTTP(rec, req)

			assert.Equal(t, tt.wantStatus, rec.Code)
			assert.Equal(t, tt.wantEncoding, rec.Header().Get("Content-Encoding"))

			got := rec.Body.Bytes()
			if tt.wantEncoding == "gzip" {
				zr, err := gzip.NewReader(rec.Body)
				require.NoError(t, err)
				got, err = io.ReadAll(zr)
				require.NoError(t, err)
			}
			assert.Equal(t, string(tt.wantBody), string(got))
		})
	}
}

func TestHashMiddleware(t *testing.T) {
	const key = "secret"
	payload := []byte(`{"id":"Alloc","type":"gauge","value":2}`)

	statusHandler := func(statuses []int, flush bool, body []byte) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			for _, s := range statuses {
				w.WriteHeader(s)
			}
			if flush {
				w.(http.Flusher).Flush()
			}
			if body != nil {
				_, _ = w.Write(body)
			}
		})
	}

	tests := []struct {
		name       string
		key        string
		body       []byte
		reqHash    string
		handler    http.Handler
		wantStatus int
		wantBody   []byte
		wantHash   string
	}{
		{
			name:       "empty key disables check",
			key:        "",
			body:       payload,
			handler:    echoHandler(t),
			wantStatus: http.StatusOK,
			wantBody:   payload,
		},
		{
			name:       "valid signature",
			key:        key,
			body:       payload,
			reqHash:    signature.Sum(key, payload),
			handler:    echoHandler(t),
			wantStatus: http.StatusOK,
			wantBody:   payload,
			wantHash:   signature.Sum(key, payload),
		},
		{
			name:       "invalid signature",
			key:        key,
			body:       payload,
			reqHash:    signature.Sum("other", payload),
			handler:    echoHandler(t),
			wantStatus: http.StatusBadRequest,
			wantBody:   []byte("Bad Request\n"),
		},
		{
			name:       "explicit status, repeated WriteHeader ignored",
			key:        key,
			reqHash:    signature.Sum(key, nil),
			handler:    statusHandler([]int{http.StatusNotFound, http.StatusOK}, false, nil),
			wantStatus: http.StatusNotFound,
			wantBody:   []byte{},
			wantHash:   signature.Sum(key, nil),
		},
		{
			name:       "flush does not commit response early",
			key:        key,
			reqHash:    signature.Sum(key, nil),
			handler:    statusHandler([]int{http.StatusNotFound}, true, []byte("missing")),
			wantStatus: http.StatusNotFound,
			wantBody:   []byte("missing"),
			wantHash:   signature.Sum(key, []byte("missing")),
		},
		{
			name:       "handler writes nothing",
			key:        key,
			reqHash:    signature.Sum(key, nil),
			handler:    statusHandler(nil, false, nil),
			wantStatus: http.StatusOK,
			wantBody:   []byte{},
			wantHash:   signature.Sum(key, nil),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/update", bytes.NewReader(tt.body))
			if tt.reqHash != "" {
				req.Header.Set(signature.Header, tt.reqHash)
			}
			rec := httptest.NewRecorder()

			HashMiddleware(tt.key)(tt.handler).ServeHTTP(rec, req)

			assert.Equal(t, tt.wantStatus, rec.Code)
			assert.Equal(t, string(tt.wantBody), rec.Body.String())
			assert.Equal(t, tt.wantHash, rec.Header().Get(signature.Header))
		})
	}
}

func TestRequestLogger(t *testing.T) {
	tests := []struct {
		name       string
		method     string
		uri        string
		status     int
		body       string
		wantStatus int
	}{
		{
			name:       "explicit status and body",
			method:     http.MethodPost,
			uri:        "/update/gauge/x/1",
			status:     http.StatusCreated,
			body:       "hello",
			wantStatus: http.StatusCreated,
		},
		{
			name:       "implicit status",
			method:     http.MethodGet,
			uri:        "/value/gauge/x",
			body:       "1.5",
			wantStatus: http.StatusOK,
		},
		{
			name:       "status without body",
			method:     http.MethodGet,
			uri:        "/value/gauge/missing",
			status:     http.StatusNotFound,
			wantStatus: http.StatusNotFound,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			core, logs := observer.New(zap.InfoLevel)
			log := zap.New(core).Sugar()

			h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if tt.status != 0 {
					w.WriteHeader(tt.status)
				}
				if tt.body != "" {
					_, _ = w.Write([]byte(tt.body))
				}
			})
			req := httptest.NewRequest(tt.method, tt.uri, nil)
			rec := httptest.NewRecorder()

			RequestLogger(log)(h).ServeHTTP(rec, req)

			assert.Equal(t, tt.wantStatus, rec.Code)
			entries := logs.FilterMessage("request completed").All()
			require.Len(t, entries, 1)
			fields := entries[0].ContextMap()
			assert.Equal(t, int64(tt.wantStatus), fields["status"])
			assert.Equal(t, int64(len(tt.body)), fields["size"])
			assert.Equal(t, tt.method, fields["method"])
			assert.Equal(t, tt.uri, fields["uri"])
		})
	}
}
