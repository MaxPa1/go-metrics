package middleware

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/MaxPa1/go-metrics/internal/compress"
	"github.com/MaxPa1/go-metrics/internal/signature"
	"go.uber.org/zap"
)

var benchBody = bytes.Repeat([]byte(`{"id":"HeapAlloc","type":"gauge","value":12345678.9},`), 40)

func benchEchoHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(body)
	})
}

func benchStatusHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.Copy(io.Discard, r.Body)
		w.WriteHeader(http.StatusOK)
	})
}

func BenchmarkGzipMiddleware(b *testing.B) {
	compressed, err := compress.Compress(benchBody)
	if err != nil {
		b.Fatal(err)
	}

	cases := []struct {
		name            string
		body            []byte
		contentEncoding string
		acceptEncoding  string
		handler         http.Handler
	}{
		{name: "gzip request and response", body: compressed.Bytes(), contentEncoding: "gzip", acceptEncoding: "gzip", handler: benchEchoHandler()},
		{name: "gzip request, status only response", body: compressed.Bytes(), contentEncoding: "gzip", acceptEncoding: "gzip", handler: benchStatusHandler()},
		{name: "plain request and response", body: benchBody, handler: benchEchoHandler()},
	}
	for _, tc := range cases {
		b.Run(tc.name, func(b *testing.B) {
			h := GzipMiddleware(tc.handler)
			b.ReportAllocs()
			for b.Loop() {
				req := httptest.NewRequest(http.MethodPost, "/updates/", bytes.NewReader(tc.body))
				if tc.contentEncoding != "" {
					req.Header.Set("Content-Encoding", tc.contentEncoding)
				}
				if tc.acceptEncoding != "" {
					req.Header.Set("Accept-Encoding", tc.acceptEncoding)
				}
				h.ServeHTTP(httptest.NewRecorder(), req)
			}
		})
	}
}

func BenchmarkHashMiddleware(b *testing.B) {
	const key = "secret"
	sum := signature.Sum(key, benchBody)
	h := HashMiddleware(key)(benchEchoHandler())

	b.ReportAllocs()
	for b.Loop() {
		req := httptest.NewRequest(http.MethodPost, "/updates/", bytes.NewReader(benchBody))
		req.Header.Set(signature.Header, sum)
		h.ServeHTTP(httptest.NewRecorder(), req)
	}
}

func BenchmarkRequestLogger(b *testing.B) {
	h := RequestLogger(zap.NewNop().Sugar())(benchStatusHandler())

	b.ReportAllocs()
	for b.Loop() {
		req := httptest.NewRequest(http.MethodPost, "/updates/", bytes.NewReader(benchBody))
		h.ServeHTTP(httptest.NewRecorder(), req)
	}
}

func BenchmarkMiddlewareChain(b *testing.B) {
	const key = "secret"
	sum := signature.Sum(key, benchBody)
	compressed, err := compress.Compress(benchBody)
	if err != nil {
		b.Fatal(err)
	}
	h := GzipMiddleware(RequestLogger(zap.NewNop().Sugar())(HashMiddleware(key)(benchStatusHandler())))

	b.ReportAllocs()
	for b.Loop() {
		req := httptest.NewRequest(http.MethodPost, "/updates/", bytes.NewReader(compressed.Bytes()))
		req.Header.Set("Content-Encoding", "gzip")
		req.Header.Set("Accept-Encoding", "gzip")
		req.Header.Set(signature.Header, sum)
		h.ServeHTTP(httptest.NewRecorder(), req)
	}
}
