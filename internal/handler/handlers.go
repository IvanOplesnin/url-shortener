package handlers

import (
	"io"
	"net/http"

	st "github.com/IvanOplesnin/url-shortener/internal/repository"
	u "github.com/IvanOplesnin/url-shortener/internal/service/url"
	"github.com/go-chi/chi/v5"
)

const (
	contentTypeKey       = "Content-Type"
	applicationJSONValue = "application/json"
	textPlainValue       = "text/plain"
)

func InitHandlers(storage st.Storage, baseURL string) *chi.Mux {
	router := chi.NewRouter()

	baseP := u.BasePath(baseURL)

	router.Use(WithLogging)

	router.Post("/", ShortenLinkHandler(storage, baseURL))
	router.Post("/api/shorten", ShortenAPIHandler(storage, baseURL))

	router.Route(
		baseP, func(router chi.Router) {
			router.Get("/{id}", RedirectHandler(storage))
		})

	return router
}

func ShortenLinkHandler(storage st.Storage, baseURL string) http.HandlerFunc {
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
			newURL := st.URL(newURLRaw)
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
			case st.ErrNotFoundURL:
				newPath, err := u.AddRandomString(storage, st.URL(newURL))
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
