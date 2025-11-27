package handlers

import (
	"io"
	"net/http"
	"net/url"

	st "github.com/IvanOplesnin/url-shortener/internal/repository"
	"github.com/google/uuid"
)

func InitHandlers(storage st.Storage, baseURL string) *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /", ShortenLinkHandler(storage, baseURL))
	mux.HandleFunc("GET /{id}", RedirectHandler(storage))
	return mux
}

func ShortenLinkHandler(storage st.Storage, baseURL string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost && r.Header.Get("Content-Type") == "text/plain" {
			newURLRaw, err := io.ReadAll(r.Body)
			if err != nil {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			newURL := st.URL(newURLRaw)
			w.Header().Set("Content-Type", "text/plain")
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
			id := r.PathValue("id")
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
