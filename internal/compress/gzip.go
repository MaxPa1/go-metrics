// Package compress содержит вспомогательные функции для работы с gzip.
package compress

import (
	"bytes"
	"compress/gzip"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"
)

var (
	gzipWriterPool = sync.Pool{
		New: func() any { return gzip.NewWriter(io.Discard) },
	}
	gzipReaderPool sync.Pool
)

type compressWriter struct {
	w           http.ResponseWriter
	zw          *gzip.Writer
	wroteHeader bool
}

// NewCompressWriter оборачивает http.ResponseWriter и сжимает ответ gzip, если тип содержимого это позволяет.
func NewCompressWriter(w http.ResponseWriter) *compressWriter {
	return &compressWriter{w: w}
}

func (c *compressWriter) Header() http.Header {
	return c.w.Header()
}

func (c *compressWriter) Write(p []byte) (int, error) {
	if !c.wroteHeader {
		c.WriteHeader(http.StatusOK)
	}
	if c.zw == nil {
		return c.w.Write(p)
	}
	return c.zw.Write(p)
}

func (c *compressWriter) WriteHeader(statusCode int) {
	if c.wroteHeader {
		return
	}
	c.wroteHeader = true

	if !isCompressibleContentType(c.w.Header().Get("Content-Type")) ||
		statusCode == http.StatusNoContent || statusCode == http.StatusNotModified {
		c.w.WriteHeader(statusCode)
		return
	}

	c.zw = gzipWriterPool.Get().(*gzip.Writer)
	c.zw.Reset(c.w)
	c.w.Header().Set("Content-Encoding", "gzip")
	c.w.Header().Del("Content-Length")
	c.w.WriteHeader(statusCode)
}

func (c *compressWriter) Close() error {
	if c.zw == nil {
		return nil
	}
	err := c.zw.Close()
	c.zw.Reset(io.Discard)
	gzipWriterPool.Put(c.zw)
	c.zw = nil
	return err
}

type compressReader struct {
	r  io.ReadCloser
	zr *gzip.Reader
}

// NewCompressReader оборачивает тело запроса и распаковывает его из gzip.
func NewCompressReader(r io.ReadCloser) (*compressReader, error) {
	zr, ok := gzipReaderPool.Get().(*gzip.Reader)
	if !ok {
		var err error
		if zr, err = gzip.NewReader(r); err != nil {
			return nil, err
		}
	} else if err := zr.Reset(r); err != nil {
		gzipReaderPool.Put(zr)
		return nil, err
	}

	return &compressReader{r: r, zr: zr}, nil
}

func (c *compressReader) Read(p []byte) (int, error) {
	if c.zr == nil {
		return 0, io.ErrClosedPipe
	}
	return c.zr.Read(p)
}

func (c *compressReader) Close() error {
	if c.zr == nil {
		return nil
	}
	zerr := c.zr.Close()
	gzipReaderPool.Put(c.zr)
	c.zr = nil
	if err := c.r.Close(); err != nil {
		return err
	}
	return zerr
}

// IsGzipEncoding сообщает, указан ли gzip в заголовке Content-Encoding.
func IsGzipEncoding(value string) bool {
	for part := range strings.SplitSeq(value, ",") {
		if strings.EqualFold(strings.TrimSpace(part), "gzip") {
			return true
		}
	}
	return false
}

// AcceptsGzip сообщает, принимает ли клиент gzip по заголовку Accept-Encoding.
func AcceptsGzip(acceptEncoding string) bool {
	accepted := false
	for part := range strings.SplitSeq(acceptEncoding, ",") {
		coding, params, _ := strings.Cut(strings.TrimSpace(part), ";")
		coding = strings.TrimSpace(coding)
		if !strings.EqualFold(coding, "gzip") && !strings.EqualFold(coding, "x-gzip") {
			continue
		}

		for param := range strings.SplitSeq(params, ";") {
			qVal, found := strings.CutPrefix(strings.TrimSpace(param), "q=")
			if !found {
				continue
			}
			if q, err := strconv.ParseFloat(qVal, 64); err == nil && q <= 0 {
				return false
			}
		}
		accepted = true
	}
	return accepted
}

// Compress сжимает данные gzip.
func Compress(data []byte) (*bytes.Buffer, error) {
	buf := bytes.NewBuffer(make([]byte, 0, len(data)/2+64))
	zw := gzipWriterPool.Get().(*gzip.Writer)
	zw.Reset(buf)
	defer func() {
		zw.Reset(io.Discard)
		gzipWriterPool.Put(zw)
	}()

	if _, err := zw.Write(data); err != nil {
		return nil, err
	}
	if err := zw.Close(); err != nil {
		return nil, err
	}
	return buf, nil
}

func isCompressibleContentType(ct string) bool {
	if ct == "" {
		return false
	}
	parts := strings.SplitN(ct, ";", 2)
	mediaType := strings.TrimSpace(parts[0])
	return mediaType == "application/json" || mediaType == "text/html"
}
