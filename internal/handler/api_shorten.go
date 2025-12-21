package handlers

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/IvanOplesnin/url-shortener/internal/model"
	repo "github.com/IvanOplesnin/url-shortener/internal/repository"
	u "github.com/IvanOplesnin/url-shortener/internal/service/url"
)

func ShortenAPIHandler(storage repo.Repository, baseURL string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get(contentTypeKey) != applicationJSONValue {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		defer r.Body.Close()

		w.Header().Set(contentTypeKey, applicationJSONValue)
		body, err := io.ReadAll(r.Body)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		var reqBody model.RequestBody
		if err := json.Unmarshal(body, &reqBody); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		if _, err := u.ParseURL(string(reqBody.URL)); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		short, _, err := getOrCreateShort(storage, reqBody.URL)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		link, err := u.CreateURL(baseURL, short)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		resp, err := marshallResponse(link)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write(resp)
	}
}

func marshallResponse(link string) ([]byte, error) {
	responseBody := model.ResponseBody{Result: link}
	return json.Marshal(responseBody)
}
