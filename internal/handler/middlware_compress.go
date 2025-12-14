package handlers

import (
	"compress/gzip"
	"net/http"
	"strings"
)

var ContentSupported []string = []string{applicationJSONValue, textPlainValue}

type gzipWriter struct {
	http.ResponseWriter
	req         *http.Request
	gz          *gzip.Writer
	compress    bool
	wroteHeader bool
	statusCode  int
}

func (w *gzipWriter) shouldCompress() bool {
	if w.Header().Get(contentEncodingKey) != "" {
		return false
	}
	if w.req.Method == http.MethodHead {
		return false
	}
	if w.statusCode >= 100 && w.statusCode < 200 {
		return false
	}
	if w.statusCode == http.StatusNoContent || w.statusCode == http.StatusNotModified {
		return false
	}
	ct := w.Header().Get(contentTypeKey)
	for _, v := range ContentSupported {
		if strings.Contains(ct, v) {
			return true
		}
	}
	return false
}

func (w *gzipWriter) initGzip() {
	if w.gz != nil || w.compress {
		return
	}
	if w.shouldCompress() {
		gz, err := gzip.NewWriterLevel(w.ResponseWriter, gzip.BestSpeed)
		if err != nil {
			return
		}
		w.gz = gz
		w.compress = true

		w.Header().Set(contentEncodingKey, "gzip")
		w.Header().Add("Vary", acceptEncodingKey)
		w.Header().Del("Content-Length")
	}
}

func (w *gzipWriter) WriteHeader(code int) {
	if w.wroteHeader {
		return
	}
	w.wroteHeader = true
	w.statusCode = code

	w.initGzip()
	w.ResponseWriter.WriteHeader(code)
}

func (w *gzipWriter) Write(p []byte) (int, error) {
	if !w.wroteHeader {
        w.WriteHeader(http.StatusOK) 
    }

	if w.compress && w.gz != nil {
		return w.gz.Write(p)
	}
	return w.ResponseWriter.Write(p)
}

func CompressGzip(next http.Handler) http.Handler {
	compressFunc := func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.Header.Get(acceptEncodingKey), "gzip") {
			next.ServeHTTP(w, r)
			return
		}
		newWriter := gzipWriter{ResponseWriter: w, req: r}
		defer func() {
			if newWriter.gz != nil {
				_ = newWriter.gz.Close()
			}
		}()
		next.ServeHTTP(&newWriter, r)
	}
	return http.HandlerFunc(compressFunc)
}
