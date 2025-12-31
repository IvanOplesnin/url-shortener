package handlers

import (
	"io"
	"net/http"

	repo "github.com/IvanOplesnin/url-shortener/internal/repository"
	"github.com/IvanOplesnin/url-shortener/internal/service/shortener"
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

func InitHandlers(svc *shortener.Service, baseURL string, p Pinger) *chi.Mux {
	router := chi.NewRouter()

	baseP := u.BasePath(baseURL)

	router.Use(WithLogging)
	router.Use(CompressGzip)
	router.Use(UncompressGzip)

	router.Post("/", ShortenLinkHandler(svc))
	router.Post("/api/shorten", ShortenAPIHandler(svc))
	router.Post("/api/shorten/batch", ShortenBatchAPIHandler(svc))
	router.Get("/ping", PingHandler(p))


	router.Route(
		baseP, func(router chi.Router) {
			router.Get("/{id}", RedirectHandler(svc))
		})

	return router
}

func ShortenLinkHandler(svc *shortener.Service) http.HandlerFunc {
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

		ctx := r.Context()
		res, err := svc.Shorten(ctx, repo.URL(raw))
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		if res.Exists {
			w.WriteHeader(http.StatusConflict)
		} else {
			w.WriteHeader(http.StatusCreated)
		}
		_, _ = w.Write([]byte(res.Link))
	}
}

func RedirectHandler(svc *shortener.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		ctx := r.Context()
		url, err := svc.Resolve(ctx, repo.ShortURL(id))
		if err != nil {
			http.NotFound(w, r)
			return
		}
		http.Redirect(w, r, string(url), http.StatusTemporaryRedirect)
	}
}
