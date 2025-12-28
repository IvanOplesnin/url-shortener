package handlers

import (
	"context"
	"net/http"
	"time"
)

func PingHandler(p Pinger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 1*time.Second)
		defer cancel()
		if p == nil {
			w.WriteHeader(http.StatusInternalServerError)
			return 
		}
		if err := p.PingContext(ctx); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	}
}
