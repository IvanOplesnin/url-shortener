package main

import (
	"io"
	"net/http"
	"strings"

	"github.com/google/uuid"
)

func main() {
	if err := run(); err != nil {
		panic(err)
	}
}

var urls = map[string]string{}

func run() error {
	mux := http.NewServeMux()
	mux.HandleFunc(`POST /`, shortener)
	mux.HandleFunc(`GET /{id}`, goToURL)

	return http.ListenAndServe(`:8080`, mux)
}

func shortener(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost && r.Header.Get("Content-Type") == "text/plain" {
		if contentType := r.Header.Get("Content-Type"); contentType != "text/plain" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		newUrl, err := io.ReadAll(r.Body)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		if sUrl := getURL(string(newUrl)); sUrl == "" {
			id := uuid.New().String()
			urls[id] = string(newUrl)
			if _, err := w.Write([]byte(createURL(id))); err != nil {
				return
			}
		} else {
			if _, err := w.Write([]byte(createURL(sUrl))); err != nil {
				return
			}
		}
	}
}

func goToURL(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		id := strings.TrimPrefix(r.URL.Path, "/")
		if url, ok := urls[id]; ok {
			http.Redirect(w, r, url, http.StatusMovedPermanently)
		} else {
			http.NotFound(w, r)
		}
	}
}

func getURL(url string) string {
	for key, v := range urls {
		if v == url {
			return key
		}
	}
	return ""
}

func createURL(id string) string {
	return "http://localhost:8080/" + id
}
