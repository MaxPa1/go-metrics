package middleware

import (
	"net/http"

	"github.com/MaxPa1/go-metrics/internal/compress"
)

func GzipMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ow := w

		if compress.AcceptsGzip(r.Header.Get("Accept-Encoding")) {
			cw := compress.NewCompressWriter(w)
			ow = cw
			defer cw.Close()
		}

		if compress.IsGzipEncoding(r.Header.Get("Content-Encoding")) {
			cr, err := compress.NewCompressReader(r.Body)
			if err != nil {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			r.Body = cr
		}

		next.ServeHTTP(ow, r)
	})
}
