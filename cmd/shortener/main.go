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
		newURL, err := io.ReadAll(r.Body)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		if sURL := getURL(string(newURL)); sURL == "" {
			id := uuid.New().String()
			urls[id] = string(newURL)
			if _, err := w.Write([]byte(createURL(id))); err != nil {
				w.WriteHeader(http.StatusCreated)
				return
			}
		} else {
			if _, err := w.Write([]byte(createURL(sURL))); err != nil {
				w.WriteHeader(http.StatusCreated)
				return
			}
		}
	} else {
		w.WriteHeader(http.StatusBadRequest)
	}
}

func goToURL(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		id := strings.TrimPrefix(r.URL.Path, "/")
		if url, ok := urls[id]; ok {
			http.Redirect(w, r, url, http.StatusTemporaryRedirect)
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
