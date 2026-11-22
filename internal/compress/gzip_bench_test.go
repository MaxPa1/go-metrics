package compress

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

var benchPayload = bytes.Repeat([]byte(`{"id":"HeapAlloc","type":"gauge","value":12345678.9},`), 40)

func BenchmarkCompress(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		if _, err := Compress(benchPayload); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkCompressWriter(b *testing.B) {
	cases := []struct {
		name        string
		contentType string
		status      int
		body        []byte
	}{
		{name: "json response", contentType: "application/json", status: http.StatusOK, body: benchPayload},
		{name: "empty response", status: http.StatusOK},
		{name: "plain text response", contentType: "text/plain", status: http.StatusOK, body: []byte("12345.6")},
	}
	for _, tc := range cases {
		b.Run(tc.name, func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				rec := httptest.NewRecorder()
				cw := NewCompressWriter(rec)
				if tc.contentType != "" {
					cw.Header().Set("Content-Type", tc.contentType)
				}
				cw.WriteHeader(tc.status)
				if tc.body != nil {
					if _, err := cw.Write(tc.body); err != nil {
						b.Fatal(err)
					}
				}
				if err := cw.Close(); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func BenchmarkCompressReader(b *testing.B) {
	compressed, err := Compress(benchPayload)
	if err != nil {
		b.Fatal(err)
	}
	data := compressed.Bytes()

	b.ReportAllocs()
	for b.Loop() {
		cr, err := NewCompressReader(io.NopCloser(bytes.NewReader(data)))
		if err != nil {
			b.Fatal(err)
		}
		if _, err := io.Copy(io.Discard, cr); err != nil {
			b.Fatal(err)
		}
		if err := cr.Close(); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkAcceptsGzip(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		AcceptsGzip("gzip, deflate, br;q=0.9")
	}
}
