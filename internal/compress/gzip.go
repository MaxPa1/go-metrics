package compress

import (
	"bytes"
	"compress/gzip"
	"io"
	"net/http"
	"strconv"
	"strings"
)

type compressWriter struct {
	w           http.ResponseWriter
	zw          *gzip.Writer
	wroteHeader bool
	disabled    bool
}

func NewCompressWriter(w http.ResponseWriter) *compressWriter {
	return &compressWriter{
		w:  w,
		zw: gzip.NewWriter(w),
	}
}

func (c *compressWriter) Header() http.Header {
	return c.w.Header()
}

func (c *compressWriter) Write(p []byte) (int, error) {
	if !c.wroteHeader {
		c.WriteHeader(http.StatusOK)
	}
	if c.disabled {
		return c.w.Write(p)
	}
	return c.zw.Write(p)
}

func (c *compressWriter) WriteHeader(statusCode int) {
	if c.wroteHeader {
		return
	}
	c.wroteHeader = true

	contentType := c.w.Header().Get("Content-Type")
	if !isCompressibleContentType(contentType) {
		c.disabled = true
		c.w.WriteHeader(statusCode)
		return
	}
	if statusCode == http.StatusNoContent || statusCode == http.StatusNotModified {
		c.disabled = true
		c.w.WriteHeader(statusCode)
		return
	}
	c.w.Header().Set("Content-Encoding", "gzip")
	c.w.Header().Del("Content-Length")
	c.w.WriteHeader(statusCode)
}

func (c *compressWriter) Close() error {
	return c.zw.Close()
}

type compressReader struct {
	r  io.ReadCloser
	zr *gzip.Reader
}

func NewCompressReader(r io.ReadCloser) (*compressReader, error) {
	zr, err := gzip.NewReader(r)
	if err != nil {
		return nil, err
	}

	return &compressReader{
		r:  r,
		zr: zr,
	}, nil
}

func (c compressReader) Read(p []byte) (n int, err error) {
	return c.zr.Read(p)
}

func (c *compressReader) Close() error {
	if err := c.r.Close(); err != nil {
		return err
	}
	return c.zr.Close()
}

func IsGzipEncoding(value string) bool {
	for _, part := range strings.Split(value, ",") {
		if strings.EqualFold(strings.TrimSpace(part), "gzip") {
			return true
		}
	}
	return false
}

func AcceptsGzip(acceptEncoding string) bool {
	if !strings.Contains(strings.ToLower(acceptEncoding), "gzip") {
		return false
	}

	for _, part := range strings.Split(acceptEncoding, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		fields := strings.Split(part, ";")
		coding := strings.TrimSpace(fields[0])
		if !strings.EqualFold(coding, "gzip") && !strings.EqualFold(coding, "x-gzip") {
			continue
		}

		for _, param := range fields[1:] {
			param = strings.TrimSpace(param)
			if strings.HasPrefix(param, "q=") {
				qVal := strings.TrimPrefix(param, "q=")
				q, err := strconv.ParseFloat(qVal, 64)
				if err == nil && q <= 0 {
					return false
				}
			}
		}
	}

	return true
}

func Compress(data []byte) (*bytes.Buffer, error) {
	var buf bytes.Buffer
	zw := gzip.NewWriter(&buf)
	if _, err := zw.Write(data); err != nil {
		return nil, err
	}
	if err := zw.Close(); err != nil {
		return nil, err
	}
	return &buf, nil
}

func isCompressibleContentType(ct string) bool {
	if ct == "" {
		return false
	}
	parts := strings.SplitN(ct, ";", 2)
	mediaType := strings.TrimSpace(parts[0])
	return mediaType == "application/json" || mediaType == "text/html"
}
