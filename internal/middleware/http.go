// Package middleware provides HTTP middleware functions.
package middleware

import (
	"context"
	"net/http"
	"strings"

	"crypto/rand"
	"encoding/hex"

	"github.com/your-org/go-backend-template/internal/domain"
	http2 "github.com/your-org/go-backend-template/internal/http"
)

type contextKey string

const (
	CurrentUserKey contextKey = "current_user"
	RequestIDKey   contextKey = "request_id"
	ClientIPKey    contextKey = "client_ip"
)

// GenerateRequestID generates a unique request ID
func GenerateRequestID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err == nil {
		return hex.EncodeToString(b)
	}
	// Fallback if crypto/rand fails
	return "unknown-request-id"
}

// GetRealIP extracts the real client IP from headers
func GetRealIP(r *http.Request) string {
	if ip := r.Header.Get("X-Forwarded-For"); ip != "" {
		// X-Forwarded-For may contain multiple IPs, take the first one
		parts := strings.Split(ip, ",")
		return strings.TrimSpace(parts[0])
	}
	if ip := r.Header.Get("X-Real-IP"); ip != "" {
		return ip
	}
	return r.RemoteAddr
}

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

// UserFromContext retrieves the current user from context
func UserFromContext(ctx context.Context) *domain.User {
	u, _ := ctx.Value(CurrentUserKey).(*domain.User)
	return u
}

// RequestIDFromContext retrieves the request ID from context
func RequestIDFromContext(ctx context.Context) string {
	id, _ := ctx.Value(RequestIDKey).(string)
	return id
}

// ClientIPFromContext retrieves the client IP from context
func ClientIPFromContext(ctx context.Context) string {
	ip, _ := ctx.Value(ClientIPKey).(string)
	return ip
}

// extractBearerToken extracts Bearer token from Authorization header
func extractBearerToken(r *http.Request) string {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return ""
	}

	parts := strings.Split(authHeader, " ")
	if len(parts) != 2 || !strings.EqualFold(parts[0], "bearer") {
		return ""
	}

	return parts[1]
}
