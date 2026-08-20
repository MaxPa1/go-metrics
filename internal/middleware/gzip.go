package middleware

import (
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

func newCompressWriter(w http.ResponseWriter) *compressWriter {
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

func newCompressReader(r io.ReadCloser) (*compressReader, error) {
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

func GzipMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ow := w

		if acceptsGzip(r.Header.Get("Accept-Encoding")) {
			cw := newCompressWriter(w)
			ow = cw
			defer cw.Close()
		}

		if isGzipEncoding(r.Header.Get("Content-Encoding")) {
			cr, err := newCompressReader(r.Body)
			if err != nil {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			r.Body = cr
		}

		next.ServeHTTP(ow, r)
	})
}

func isGzipEncoding(value string) bool {
	for _, part := range strings.Split(value, ",") {
		if strings.EqualFold(strings.TrimSpace(part), "gzip") {
			return true
		}
	}
	return false
}

func acceptsGzip(acceptEncoding string) bool {
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

func isCompressibleContentType(ct string) bool {
	if ct == "" {
		return false
	}
	parts := strings.SplitN(ct, ";", 2)
	mediaType := strings.TrimSpace(parts[0])
	return mediaType == "application/json" || mediaType == "text/html"
}
