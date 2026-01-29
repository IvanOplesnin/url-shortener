package handlers

import (
	"context"
	"errors"
	"net/http"

	"github.com/IvanOplesnin/url-shortener/internal/logger"
	"github.com/IvanOplesnin/url-shortener/internal/service/shortener"
)

const tokenCookieName = "token"

type contextKey int

const claimsKey contextKey = iota

type TokenService interface {
	CreateToken(ctx context.Context) (string, *shortener.Claims, error)
	VerifyToken(ctx context.Context, token string) (*shortener.Claims, error)
}

func ClaimsFromContext(ctx context.Context) (*shortener.Claims, bool) {
	c, ok := ctx.Value(claimsKey).(*shortener.Claims)
	return c, ok
}

func CheckCookieJWTAndSet(svc TokenService) func(http.Handler) http.Handler {
	CheckCookie := func(next http.Handler) http.Handler {
		checkCookieFunc := func(w http.ResponseWriter, r *http.Request) {
			var token string
			ctx := r.Context()
			c, err := r.Cookie(tokenCookieName)
			if err == nil && c != nil {
				token = c.Value
			} else if err != nil && !errors.Is(err, http.ErrNoCookie) {
				http.Error(w, "failed to read cookie", http.StatusInternalServerError)
				return
			}
			if token != "" {
				claims, err := svc.VerifyToken(ctx, token)
				if err == nil {
					ctx := context.WithValue(ctx, claimsKey, claims)
					next.ServeHTTP(w, r.WithContext(ctx))
					return
				} else if errors.Is(err, shortener.ErrNotUserFound) || errors.Is(err, shortener.ErrNotUserID) {
					logger.Log.Errorf("mwCokie:verify token error: %s", err.Error())
					w.WriteHeader(http.StatusUnauthorized)
					return
				} else {
					logger.Log.Errorf("mwCokie:verify token error: %s", err.Error())
				}
			}
			newToken, newClaims, err := svc.CreateToken(ctx)
			if err != nil {
				logger.Log.Errorf("mwCokie: error create token: %s", err.Error())
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			if newToken != "" {
				http.SetCookie(w, &http.Cookie{
					Name:     tokenCookieName,
					Value:    newToken,
					Path:     "/",
					HttpOnly: true,
					SameSite: http.SameSiteLaxMode,
				})
			}
			ctx = context.WithValue(ctx, claimsKey, newClaims)
			next.ServeHTTP(w, r.WithContext(ctx))
		}
		return http.HandlerFunc(checkCookieFunc)
	}
	return CheckCookie
}
