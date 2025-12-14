package handlers

import (
	"compress/gzip"
	"net/http"
	"strings"
)

func UncompressGzip(next http.Handler) http.Handler {
	f := func(w http.ResponseWriter, r *http.Request) {
		ce := strings.ToLower(r.Header.Get(contentEncodingKey))
		if !strings.Contains(ce, "gzip") || r.Body == nil || r.Body == http.NoBody {
			next.ServeHTTP(w, r)
			return
		}
		originalBody := r.Body

		gz, err := gzip.NewReader(originalBody)
		if err != nil {
			_ = originalBody.Close()
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		defer originalBody.Close()
		r.Body = gz

		r.Header.Del(contentEncodingKey)
		next.ServeHTTP(w, r)
	}
	return http.HandlerFunc(f)
}
