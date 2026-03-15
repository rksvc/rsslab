package server

import (
	"compress/gzip"
	"net/http"
)

type gzipResponseWriter struct {
	out *gzip.Writer
	src http.ResponseWriter
}

func (gz *gzipResponseWriter) Header() http.Header {
	return gz.src.Header()
}

func (gz *gzipResponseWriter) Write(p []byte) (int, error) {
	return gz.out.Write(p)
}

func (gz *gzipResponseWriter) WriteHeader(statusCode int) {
	gz.src.WriteHeader(statusCode)
}
