// Package middleware provides HTTP middleware functions.
package middleware

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/your-org/go-backend-template/internal/domain"
	"golang.org/x/time/rate"
)

type contextKey string

const (
	CurrentUserKey contextKey = "current_user"
	RequestIDKey   contextKey = "request_id"
)

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

// RateLimiter wraps a rate limiter
type RateLimiter struct {
	rate *rate.Limiter
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(requests int, duration time.Duration) *RateLimiter {
	return &RateLimiter{
		rate: rate.NewLimiter(rate.Every(duration), requests),
	}
}

// Allow checks if a request is allowed
func (rl *RateLimiter) Allow() bool {
	return rl.rate.Allow()
}
