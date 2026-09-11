package handlers

import (
	"context"
	"net/http"
	"strings"

	"ticket-system/internal/auth"
)

type contextKey string

const userIDContextKey contextKey = "userID"

// RequireAuth validates the Authorization: Bearer <token> header and, on
// success, stores the authenticated user's ID in the request context.
func RequireAuth(jwtSecret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				writeError(w, http.StatusUnauthorized, "missing authorization header")
				return
			}

			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") || parts[1] == "" {
				writeError(w, http.StatusUnauthorized, "authorization header must be 'Bearer <token>'")
				return
			}

			claims, err := auth.ParseToken(jwtSecret, parts[1])
			if err != nil {
				writeError(w, http.StatusUnauthorized, "invalid or expired token")
				return
			}

			ctx := context.WithValue(r.Context(), userIDContextKey, claims.UserID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func userIDFromContext(r *http.Request) (string, bool) {
	id, ok := r.Context().Value(userIDContextKey).(string)
	return id, ok
}
