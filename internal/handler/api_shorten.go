package handlers

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/IvanOplesnin/url-shortener/internal/model"
	st "github.com/IvanOplesnin/url-shortener/internal/repository"
	u "github.com/IvanOplesnin/url-shortener/internal/service/url"
)

func ShortenApiHandler(storage st.Storage, baseURL string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get(contentTypeKey) == applicationJSONValue {
			w.Header().Set(contentTypeKey, applicationJSONValue)
			body, err := io.ReadAll(r.Body)
			defer r.Body.Close()
			if err != nil {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			reqBody := model.RequestBody{}
			if err := json.Unmarshal(body, &reqBody); err != nil {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			if _, err := u.ParseURL(string(reqBody.Url)); err != nil {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			sURL, err := storage.Search(reqBody.Url)
			switch err {
			case nil:
				link, err := u.CreateURL(baseURL, sURL)
				if err != nil {
					w.WriteHeader(http.StatusBadRequest)
					return
				}
				w.WriteHeader(http.StatusCreated)
				body, err := marshallResponse(link)
				if err != nil {
					w.WriteHeader(http.StatusInternalServerError)
					return
				}
				w.Write(body)
			case st.ErrNotFoundURL:
				newPath, err := u.AddRandomString(storage, reqBody.Url)
				if err != nil {
					w.WriteHeader(http.StatusBadRequest)
					return
				}
				link, err := u.CreateURL(baseURL, newPath)
				if err != nil {
					w.WriteHeader(http.StatusBadRequest)
					return
				}
				w.WriteHeader(http.StatusCreated)
				body, err := marshallResponse(link)
				if err != nil {
					w.WriteHeader(http.StatusInternalServerError)
					return
				}
				w.Write(body)
			default:
				w.WriteHeader(http.StatusBadRequest)
			}
			return
		}
		w.WriteHeader(http.StatusBadRequest)
	}
}

func marshallResponse(link string) ([]byte, error) {
	responseBody := model.ResponseBody{Result: link}
	return json.Marshal(responseBody)
}
