package handlers

import (
	"errors"
	"io"
	"net/http"
	"net/url"

	st "github.com/IvanOplesnin/url-shortener/internal/repository"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

func InitHandlers(storage st.Storage, baseURL string) *chi.Mux {
	router := chi.NewRouter()
	router.Post("/", ShortenLinkHandler(storage, baseURL))
	router.Get("/{id}", RedirectHandler(storage))
	return router
}

func ShortenLinkHandler(storage st.Storage, baseURL string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost && r.Header.Get("Content-Type") == "text/plain" {
			w.Header().Set("Content-Type", "text/plain")
			newURLRaw, err := io.ReadAll(r.Body)
			if err != nil {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			if _, err := ParseURL(string(newURLRaw)); err != nil {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			newURL := st.URL(newURLRaw)
			sURL, err := storage.Search(newURL)
			switch err {
			case nil:
				w.WriteHeader(http.StatusCreated)
				w.Write([]byte(createURL(baseURL, sURL)))
			case st.ErrNotFoundURL:
				id := st.ShortURL(uuid.New().String())
				err := storage.Add(id, newURL)
				if err != nil {
					w.WriteHeader(http.StatusBadRequest)
					return
				}
				w.WriteHeader(http.StatusCreated)
				w.Write([]byte(createURL(baseURL, id)))
			default:
				w.WriteHeader(http.StatusBadRequest)
			}
			return
		} else if r.Header.Get("Content-Type") != "text/plain" && r.Method == http.MethodPost {
			w.WriteHeader(http.StatusBadRequest)
		}
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

func createURL(base string, id st.ShortURL) string {
	url, err := url.JoinPath(base, string(id))
	if err != nil {
		return ""
	}
	return url
}

func ParseURL(urlRaw string) (st.URL, error) {
	if urlRaw == "" {
		return "", errors.New("empty Body")
	}
	if _, err := url.Parse(urlRaw); err != nil {
		return "", err
	}
	return st.URL(urlRaw), nil
}
