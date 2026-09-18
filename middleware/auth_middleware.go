package middleware

import (
	"context"
	"net/http"
	"strings"

	"bank-transfer-api/utils"
)

type contextKey string

const UserIDKey contextKey = "userID"

func JWTMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, "authorization header required", http.StatusUnauthorized)
			return
		}

		// Format header yang diharapkan: "Bearer <token>"
		parts := strings.Fields(authHeader)
		if len(parts) != 2 || parts[0] != "Bearer" {
			http.Error(w, "invalid authorization header format", http.StatusUnauthorized)
			return
		}

		tokenString := parts[1]

		claims, err := utils.ValidateToken(tokenString)
		if err != nil {
			http.Error(w, "invalid or expired token", http.StatusUnauthorized)
			return
		}

		// Store the userID in the context so that downstream handlers
		// can know "who is making this request" if necessary.
		ctx := context.WithValue(r.Context(), UserIDKey, claims.UserID)
		next.ServeHTTP(w, r.WithContext(ctx))
	}
}
