package handlers

import (
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"net/url"
	"time"

	st "github.com/IvanOplesnin/url-shortener/internal/repository"
	"github.com/go-chi/chi/v5"
)

func InitHandlers(storage st.Storage, baseURL string) *chi.Mux {
	router := chi.NewRouter()

	baseP := basePath(baseURL)

	router.Post("/", ShortenLinkHandler(storage, baseURL))

	router.Route(
		baseP, func(router chi.Router) {
			router.Get("/{id}", RedirectHandler(storage))
		})

	return router
}

func ShortenLinkHandler(storage st.Storage, baseURL string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Content-Type") == "text/plain" {
			w.Header().Set("Content-Type", "text/plain")
			newURLRaw, err := io.ReadAll(r.Body)
			if err != nil {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			if _, err := parseURL(string(newURLRaw)); err != nil {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			newURL := st.URL(newURLRaw)
			sURL, err := storage.Search(newURL)
			switch err {
			case nil:
				body, err := createURL(baseURL, sURL)
				if err != nil {
					w.WriteHeader(http.StatusBadRequest)
					return
				}
				w.WriteHeader(http.StatusCreated)
				w.Write([]byte(body))
			case st.ErrNotFoundURL:
				newPath, err := addRandomString(storage, st.URL(newURL))
				if err != nil {
					w.WriteHeader(http.StatusBadRequest)
					return
				}
				body, err := createURL(baseURL, newPath)
				if err != nil {
					w.WriteHeader(http.StatusBadRequest)
					return
				}
				w.WriteHeader(http.StatusCreated)
				w.Write([]byte(body))
			default:
				w.WriteHeader(http.StatusBadRequest)
			}
			return
		}
		w.WriteHeader(http.StatusBadRequest)
	}
}

func RedirectHandler(storage st.Storage) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			id := chi.URLParam(r, "id")
			url, err := storage.Get(st.ShortURL(id))
			if err != nil {
				http.NotFound(w, r)
				return
			}
			http.Redirect(w, r, string(url), http.StatusTemporaryRedirect)
		}
	}
}

func createURL(base string, id st.ShortURL) (string, error) {
	url, err := url.JoinPath(base, string(id))
	if err != nil {
		return "", fmt.Errorf("error createUrl: %w", err)
	}
	return url, nil
}

func parseURL(urlRaw string) (st.URL, error) {
	if urlRaw == "" {
		return "", fmt.Errorf("empty body")
	}
	if _, err := url.Parse(urlRaw); err != nil {
		return "", fmt.Errorf("error parseUrl: %w", err)
	}
	return st.URL(urlRaw), nil
}

func basePath(baseURL string) string {
	base, err := url.Parse(baseURL)
	if err != nil {
		log.Fatalf("error base path: %v", err)
	}
	basePath := base.Path
	if basePath == "" {
		basePath = "/"
	}
	return basePath
}

func addRandomString(storage st.Storage, url st.URL) (st.ShortURL, error) {
	const retry = 6

	lettrs := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	b := make([]byte, 6)
	for count := 0; count < retry; count++ {
		for i := range b {
			b[i] = lettrs[r.Intn(len(lettrs))]
		}
		err := storage.Add(st.ShortURL(b), url)
		if err == nil {
			return st.ShortURL(string(b)), nil
		}
	}

	return "", fmt.Errorf("error addRandomString: Can't generate random string")
}
