package main

import (
	"context"
	"github.com/ziliscite/bard_narate/auth/internal/service"
	"golang.org/x/oauth2"
	"net/http"
)

type OAuthMiddleware struct {
	cfg service.OAuthProvider
}

func NewOAuthMiddleware(cfg service.OAuthProvider) *OAuthMiddleware {
	return &OAuthMiddleware{
		cfg: cfg,
	}
}

func (o *OAuthMiddleware) AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check if the request is authenticated
		if r.Header.Get("Authorization") == "" {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// Extract bearer token
		token := r.Header.Get("Authorization")[7:]
		if token == "" {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// Exchange the token for user info
		userInfo, err := o.cfg.User(r.Context(), &oauth2.Token{
			AccessToken: token,
		})
		if err != nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), "user", userInfo)))
	})
}
