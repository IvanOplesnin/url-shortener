package handlers

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/IvanOplesnin/url-shortener/internal/service/shortener"
)

type RequestData []string


func UserMarkDeleteHandler(svc *shortener.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		claims, ok  := ClaimsFromContext(ctx)
		if !ok {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		userID := int64(claims.UserID)
		var reqData RequestData
		body, err := io.ReadAll(r.Body)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return 
		}
		defer r.Body.Close()
		if err := json.Unmarshal(body, &reqData); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return 
		}
		ok = svc.MarkDeleteURLs(ctx, userID, reqData)
		if !ok {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusAccepted)
	}
}