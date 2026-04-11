// Package middleware provides HTTP middleware functions.
package middleware

import (
	"context"
	"net/http"

	"github.com/your-org/go-backend-template/internal/domain"
	http2 "github.com/your-org/go-backend-template/internal/http"
)

// AuthProvider defines the interface for auth service operations
type AuthProvider interface {
	GetUserFromToken(ctx context.Context, token string) (*domain.User, error)
}

// RequireAuth validates JWT Bearer token and injects user into context
func RequireAuth(authSvc AuthProvider) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token := extractBearerToken(r)
			if token == "" {
				http2.RespondError(w, http.StatusUnauthorized, "missing authorization header")
				return
			}

			user, err := authSvc.GetUserFromToken(r.Context(), token)
			if err != nil || user == nil {
				http2.RespondError(w, http.StatusUnauthorized, "could not validate credentials")
				return
			}

			ctx := context.WithValue(r.Context(), CurrentUserKey, user)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RequireAdmin wraps RequireAuth and checks for admin role
func RequireAdmin(authSvc AuthProvider) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return RequireAuth(authSvc)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user := UserFromContext(r.Context())
			if user == nil {
				http2.RespondError(w, http.StatusUnauthorized, "could not validate credentials")
				return
			}

			if !user.HasRole("admin") {
				http2.RespondError(w, http.StatusForbidden, "admin role required")
				return
			}

			next.ServeHTTP(w, r)
		}))
	}
}

// OptionalAuth extracts Bearer token if present but doesn't reject missing tokens
func OptionalAuth(authSvc AuthProvider) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token := extractBearerToken(r)
			if token == "" {
				next.ServeHTTP(w, r)
				return
			}

			user, err := authSvc.GetUserFromToken(r.Context(), token)
			if err != nil || user == nil {
				next.ServeHTTP(w, r)
				return
			}

			ctx := context.WithValue(r.Context(), CurrentUserKey, user)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
