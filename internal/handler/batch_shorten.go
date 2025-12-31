package handlers

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/IvanOplesnin/url-shortener/internal/logger"
	"github.com/IvanOplesnin/url-shortener/internal/model"
	"github.com/IvanOplesnin/url-shortener/internal/service/shortener"
)

func ShortenBatchAPIHandler(svc *shortener.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get(contentTypeKey) != applicationJSONValue {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		defer r.Body.Close()
		w.Header().Set(contentTypeKey, applicationJSONValue)
		body, err := io.ReadAll(r.Body)
		if err != nil {
			logger.Log.Errorf("shorten batch error %s", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		var reqBody []model.RequestBatchBody
		if err := json.Unmarshal(body, &reqBody); err != nil {
			logger.Log.Errorf("shorten batch error %s", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		ctx := r.Context()
		respBatchBody, hadExisting, err := svc.Batch(ctx, reqBody)
		if err != nil {
			logger.Log.Errorf("shorten batch error %s", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		resp, err := json.Marshal(respBatchBody)
		if err != nil {
			logger.Log.Errorf("shorten batch error %s", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		if hadExisting {
			w.WriteHeader(http.StatusConflict)
		} else {
			w.WriteHeader(http.StatusCreated)
		}
		_, _ = w.Write(resp)
	}
}
