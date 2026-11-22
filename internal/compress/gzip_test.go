package compress

import (
	"bytes"
	"compress/gzip"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func gunzip(t *testing.T, data []byte) []byte {
	t.Helper()
	zr, err := gzip.NewReader(bytes.NewReader(data))
	require.NoError(t, err)
	out, err := io.ReadAll(zr)
	require.NoError(t, err)
	return out
}

func TestCompress(t *testing.T) {
	tests := []struct {
		name string
		data []byte
	}{
		{name: "json payload", data: []byte(`{"id":"Alloc","type":"gauge","value":1.5}`)},
		{name: "empty payload", data: []byte{}},
		{name: "large payload", data: bytes.Repeat([]byte("metrics"), 10_000)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf, err := Compress(tt.data)
			require.NoError(t, err)
			assert.Equal(t, tt.data, gunzip(t, buf.Bytes()))
		})
	}
}

func TestIsGzipEncoding(t *testing.T) {
	tests := []struct {
		name  string
		value string
		want  bool
	}{
		{name: "gzip", value: "gzip", want: true},
		{name: "upper case", value: "GZIP", want: true},
		{name: "list with gzip", value: "deflate, gzip", want: true},
		{name: "other encoding", value: "deflate", want: false},
		{name: "empty", value: "", want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, IsGzipEncoding(tt.value))
		})
	}
}

func TestAcceptsGzip(t *testing.T) {
	tests := []struct {
		name  string
		value string
		want  bool
	}{
		{name: "gzip", value: "gzip", want: true},
		{name: "list with gzip", value: "gzip, deflate, br", want: true},
		{name: "x-gzip", value: "x-gzip", want: true},
		{name: "empty parts and positive q", value: "deflate, , gzip;q=0.8", want: true},
		{name: "zero q", value: "gzip;q=0", want: false},
		{name: "invalid q", value: "gzip;q=abc", want: true},
		{name: "other encoding", value: "deflate", want: false},
		{name: "empty", value: "", want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, AcceptsGzip(tt.value))
		})
	}
}

func TestIsCompressibleContentType(t *testing.T) {
	tests := []struct {
		name string
		ct   string
		want bool
	}{
		{name: "json", ct: "application/json", want: true},
		{name: "html with charset", ct: "text/html; charset=utf-8", want: true},
		{name: "plain text", ct: "text/plain", want: false},
		{name: "empty", ct: "", want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, isCompressibleContentType(tt.ct))
		})
	}
}

func TestCompressWriter(t *testing.T) {
	body := []byte(`{"ok":true}`)

	tests := []struct {
		name         string
		contentType  string
		statuses     []int
		write        bool
		wantStatus   int
		wantEncoding string
		wantBody     []byte
	}{
		{
			name:         "compressible content is gzipped",
			contentType:  "application/json",
			write:        true,
			wantStatus:   http.StatusOK,
			wantEncoding: "gzip",
			wantBody:     body,
		},
		{
			name:        "non-compressible content is passed through",
			contentType: "text/plain",
			statuses:    []int{http.StatusCreated, http.StatusInternalServerError},
			write:       true,
			wantStatus:  http.StatusCreated,
			wantBody:    body,
		},
		{
			name:        "no content status is not compressed",
			contentType: "application/json",
			statuses:    []int{http.StatusNoContent},
			wantStatus:  http.StatusNoContent,
			wantBody:    []byte{},
		},
		{
			name:       "status without content type writes empty body",
			statuses:   []int{http.StatusOK},
			wantStatus: http.StatusOK,
			wantBody:   []byte{},
		},
		{
			name:       "nothing written writes empty body",
			wantStatus: http.StatusOK,
			wantBody:   []byte{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			cw := NewCompressWriter(rec)
			cw.Header().Set("Content-Type", tt.contentType)
			cw.Header().Set("Content-Length", "11")
			for _, s := range tt.statuses {
				cw.WriteHeader(s)
			}
			if tt.write {
				_, err := cw.Write(body)
				require.NoError(t, err)
			}
			require.NoError(t, cw.Close())
			require.NoError(t, cw.Close())

			assert.Equal(t, tt.wantStatus, rec.Code)
			assert.Equal(t, tt.wantEncoding, rec.Header().Get("Content-Encoding"))

			got := rec.Body.Bytes()
			if tt.wantEncoding == "gzip" {
				assert.Empty(t, rec.Header().Get("Content-Length"))
				got = gunzip(t, got)
			}
			assert.Equal(t, string(tt.wantBody), string(got))
		})
	}
}

func TestCompressReader(t *testing.T) {
	data := []byte("hello metrics")
	compressed, err := Compress(data)
	require.NoError(t, err)

	tests := []struct {
		name    string
		input   []byte
		want    []byte
		wantErr bool
	}{
		{name: "valid gzip", input: compressed.Bytes(), want: data},
		{name: "not gzip", input: []byte("not gzip"), wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cr, err := NewCompressReader(io.NopCloser(bytes.NewReader(tt.input)))
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			got, err := io.ReadAll(cr)
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
			assert.NoError(t, cr.Close())
			assert.NoError(t, cr.Close())

			_, err = cr.Read(make([]byte, 1))
			assert.Error(t, err)
		})
	}
}

func TestCompressPoolReuse(t *testing.T) {
	payloads := [][]byte{
		[]byte(`{"id":"Alloc","type":"gauge","value":1}`),
		bytes.Repeat([]byte("x"), 100_000),
		[]byte(`[]`),
		{},
	}
	for range 3 {
		for _, p := range payloads {
			rec := httptest.NewRecorder()
			cw := NewCompressWriter(rec)
			cw.Header().Set("Content-Type", "application/json")
			_, err := cw.Write(p)
			require.NoError(t, err)
			require.NoError(t, cw.Close())

			cr, err := NewCompressReader(io.NopCloser(bytes.NewReader(rec.Body.Bytes())))
			require.NoError(t, err)
			got, err := io.ReadAll(cr)
			require.NoError(t, err)
			require.NoError(t, cr.Close())
			assert.Equal(t, len(p), len(got))
			assert.True(t, bytes.Equal(p, got))

			buf, err := Compress(p)
			require.NoError(t, err)
			assert.True(t, bytes.Equal(p, gunzip(t, buf.Bytes())))
		}
	}
}
