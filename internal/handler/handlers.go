package handlers

import (
	"errors"
	"fmt"
	"io"
	"net/http"

	repo "github.com/IvanOplesnin/url-shortener/internal/repository"
	u "github.com/IvanOplesnin/url-shortener/internal/service/url"
	"github.com/go-chi/chi/v5"
)

const (
	contentTypeKey       = "Content-Type"
	acceptEncodingKey    = "Accept-Encoding"
	contentEncodingKey   = "Content-Encoding"
	applicationJSONValue = "application/json"
	textPlainValue       = "text/plain"
)

func InitHandlers(storage repo.Repository, baseURL string) *chi.Mux {
	router := chi.NewRouter()

	baseP := u.BasePath(baseURL)

	router.Use(WithLogging)
	router.Use(CompressGzip)
	router.Use(UncompressGzip)

	router.Post("/", ShortenLinkHandler(storage, baseURL))
	router.Post("/api/shorten", ShortenAPIHandler(storage, baseURL))

	router.Route(
		baseP, func(router chi.Router) {
			router.Get("/{id}", RedirectHandler(storage))
		})

	return router
}

func ShortenLinkHandler(storage repo.Repository, baseURL string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get(contentTypeKey) != textPlainValue {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		defer r.Body.Close()

		w.Header().Set(contentTypeKey, textPlainValue)
		
		raw, err := io.ReadAll(r.Body)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		if _, err := u.ParseURL(string(raw)); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		url := repo.URL(raw)

		short, _, err := getOrCreateShort(storage, url)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		link, err := u.CreateURL(baseURL, short)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(link))
	}
}


func RedirectHandler(storage repo.Repository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			id := chi.URLParam(r, "id")
			url, err := storage.Get(repo.ShortURL(id))
			if err != nil {
				http.NotFound(w, r)
				return
			}
			http.Redirect(w, r, string(url), http.StatusTemporaryRedirect)
		}
	}
}

func getOrCreateShort(r repo.Repository, url repo.URL) (repo.ShortURL, bool, error) {
	// bool = existed (true если уже была)
	sURL, err := r.Search(url)
	if err == nil {
		return sURL, true, nil
	}
	if errors.Is(err, repo.ErrNotFoundURL) {
		newPath, err := u.AddRandomString(r, url)
		if err != nil {
			return "", false, fmt.Errorf("add random: %w", err)
		}
		return newPath, false, nil
	}
	return "", false, err
}
