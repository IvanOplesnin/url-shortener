package handlers

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/IvanOplesnin/url-shortener/internal/logger"
	"github.com/IvanOplesnin/url-shortener/internal/model"
	"github.com/IvanOplesnin/url-shortener/internal/service/shortener"
)

func ShortenAPIHandler(svc *shortener.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get(contentTypeKey) != applicationJSONValue {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		defer r.Body.Close()

		w.Header().Set(contentTypeKey, applicationJSONValue)

		body, err := io.ReadAll(r.Body)
		if err != nil {
			logger.Log.Errorf("shorten error %s", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		var req model.RequestBody
		if err := json.Unmarshal(body, &req); err != nil {
			logger.Log.Errorf("shorten error %s", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		ctx := r.Context()
		res, err := svc.Shorten(ctx, req.URL)
		if err != nil {
			logger.Log.Errorf("shorten error %s", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		resp := model.ResponseBody{Result: res.Link}
		b, err := json.Marshal(resp)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		if res.Exists {
			w.WriteHeader(http.StatusConflict)
		} else {
			w.WriteHeader(http.StatusCreated)
		}
		w.Write(b)
	}
}
