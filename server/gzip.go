package server

import (
	"compress/gzip"
	"log"
	"net/http"
	"rsslab/utils"
	"strings"
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

func wrap(handleFunc func(context) error) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
			w.Header().Set("Content-Encoding", "gzip")
			gz := &gzipResponseWriter{out: gzip.NewWriter(w), src: w}
			defer func() {
				if err := gz.out.Close(); err != nil {
					log.Print(err)
				}
			}()
			w = gz
		}
		if err := handleFunc(context{w, r}); err != nil {
			log.Printf("%s %s: %s", r.Method, r.URL.EscapedPath(), err)
			if _, ok := err.(*errBadRequest); ok {
				w.WriteHeader(http.StatusBadRequest)
			} else {
				w.WriteHeader(http.StatusInternalServerError)
			}
			_, err = w.Write(utils.StringToBytes(err.Error()))
			if err != nil {
				log.Print(err)
			}
		}
	}
}
