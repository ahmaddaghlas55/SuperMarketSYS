package middleware

import (
	"context"
	"net/http"
	"strings"

	"supermarket/internal/models"
	"supermarket/internal/services"
)

type contextKey string

const userContextKey contextKey = "authenticated_user"

func UserFromContext(ctx context.Context) (models.User, bool) {
	user, ok := ctx.Value(userContextKey).(models.User)
	return user, ok
}

func RequireAuth(auth *services.AuthService, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		const prefix = "Bearer "
		header := r.Header.Get("Authorization")
		if !strings.HasPrefix(header, prefix) {
			WriteError(w, http.StatusUnauthorized, "authentication_required")
			return
		}
		user, err := auth.UserForToken(r.Context(), strings.TrimSpace(strings.TrimPrefix(header, prefix)))
		if err != nil {
			WriteError(w, http.StatusUnauthorized, "invalid_session")
			return
		}
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), userContextKey, user)))
	})
}

func RequireRole(roles ...string) func(http.Handler) http.Handler {
	allowed := make(map[string]struct{}, len(roles))
	for _, role := range roles {
		allowed[role] = struct{}{}
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user, ok := UserFromContext(r.Context())
			if !ok {
				WriteError(w, http.StatusUnauthorized, "authentication_required")
				return
			}
			if _, ok := allowed[user.Role]; !ok {
				WriteError(w, http.StatusForbidden, "insufficient_role")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func WriteError(w http.ResponseWriter, status int, code string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write([]byte(`{"error":{"code":"` + code + `"}}`))
}
