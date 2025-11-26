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
	mux.HandleFunc(`GET /{id}`, goToUrl)

	return http.ListenAndServe(`:8080`, mux)
}

func shortener(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost && r.Header.Get("Content-Type") == "text/plain" {
		if content_type := r.Header.Get("Content-Type"); content_type != "text/plain" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		new_url, err := io.ReadAll(r.Body)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		if s_url := getUrl(string(new_url)); s_url == "" {
			id := uuid.New().String()
			urls[id] = string(new_url)
			if _, err := w.Write([]byte(createUrl(id))); err != nil {
				return
			}
		} else {
			if _, err := w.Write([]byte(createUrl(s_url))); err != nil {
				return
			}
		}
	}
}

func goToUrl(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		id := strings.TrimPrefix(r.URL.Path, "/")
		if url, ok := urls[id]; ok {
			http.Redirect(w, r, url, http.StatusMovedPermanently)
		} else {
			http.NotFound(w, r)
		}
	}
}

func getUrl(url string) string {
	for key, v := range urls {
		if v == url {
			return key
		}
	}
	return ""
}

func createUrl(id string) string {
	return "http://localhost:8080/" + id
}
