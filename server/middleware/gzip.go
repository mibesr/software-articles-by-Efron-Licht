package middleware

import (
	"compress/gzip"
	"net/http"
	"strings"
)

// WriteGzip compresses the response body with GZip when it encounters an Accept-Encoding: gzip header.
func WriteGzip(h http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		for _, v := range r.Header.Values("Accept-Encoding") {
			if strings.Contains(v, "gzip") {
				gz := gzip.NewWriter(w)
				w.Header().Set("Content-Encoding", "gzip")
				defer gz.Close()
				h.ServeHTTP(&gzipWriter{zip: gz, ResponseWriter: w}, r)
				return
			}
		}
		h.ServeHTTP(w, r)
	}
}

type gzipWriter struct {
	zip *gzip.Writer
	http.ResponseWriter
}

func (gz *gzipWriter) Write(p []byte) (n int, err error) { return gz.zip.Write(p) }
