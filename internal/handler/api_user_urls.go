package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/IvanOplesnin/url-shortener/internal/logger"
	"github.com/IvanOplesnin/url-shortener/internal/service/shortener"
)

func UserUrlsHandler(svc *shortener.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		urls, err := svc.GetUserURLs(ctx)
		if err != nil {
			logger.Log.Errorf("UserUrlsHandler: get user urls: %v", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		if len(urls) == 0 {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		
		b, err := json.Marshal(urls)
		if err != nil {
			logger.Log.Errorf("UserUrlsHandler: encode json: %v", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		w.Header().Set(contentTypeKey, applicationJSONValue)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(b)
	}
}
