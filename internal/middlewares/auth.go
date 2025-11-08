package middlewares

import (
	"context"
	"net/http"
	"strings"

	"license-service/internal/utils"
)

type contextKey string

const UserLoginKey = contextKey("user_login")


func AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func (w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, "Missing authorization field", http.StatusUnauthorized)
			return
		}

		if !strings.HasPrefix(authHeader, "Bearer ") {
			http.Error(w, "Invalid authorization field", http.StatusUnauthorized)
			return
		}

		tokenStr := strings.TrimPrefix(authHeader, "Bearer ")
		login, err := utils.ValidateAccessToken(tokenStr)

		if err != nil {
			http.Error(w, "Invalid or expired access key", http.StatusUnauthorized)
			return
		}

		ctx := context.WithValue(r.Context(), UserLoginKey, login)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}