package handlers

import (
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
		if r.Header.Get(contentTypeKey) == textPlainValue {
			w.Header().Set(contentTypeKey, textPlainValue)
			newURLRaw, err := io.ReadAll(r.Body)
			defer r.Body.Close()
			if err != nil {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			if _, err := u.ParseURL(string(newURLRaw)); err != nil {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			newURL := repo.URL(newURLRaw)
			sURL, err := storage.Search(newURL)
			switch err {
			case nil:
				body, err := u.CreateURL(baseURL, sURL)
				if err != nil {
					w.WriteHeader(http.StatusBadRequest)
					return
				}
				w.WriteHeader(http.StatusCreated)
				w.Write([]byte(body))
			case repo.ErrNotFoundURL:
				newPath, err := u.AddRandomString(storage, repo.URL(newURL))
				if err != nil {
					w.WriteHeader(http.StatusBadRequest)
					return
				}
				body, err := u.CreateURL(baseURL, newPath)
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
