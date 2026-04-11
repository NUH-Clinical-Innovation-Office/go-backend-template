// Package router provides HTTP router setup.
package router

import (
	"context"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/your-org/go-backend-template/internal/logging"
	appmiddleware "github.com/your-org/go-backend-template/internal/middleware"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

// RouterConfig holds dependencies for router setup
type RouterConfig struct {
	Logger      *zap.Logger
	Tracer      trace.Tracer
	AuthSvc     appmiddleware.AuthProvider
	TodoHandler interface{} // TODO: Replace with actual handler type
}

// New creates a new Chi router with all middleware and routes configured
func New(cfg RouterConfig) *chi.Mux {
	r := chi.NewMux()

	// Global middleware stack (applied to all routes)
	r.Use(
		requestIDMiddleware(),
		realIPMiddleware(),
		loggerMiddleware(cfg.Logger),
		chimiddleware.Recoverer,
		timeoutMiddleware(30*time.Second),
		corsMiddleware(),
	)

	// Public routes
	r.Get("/", rootHandler())
	r.Get("/health", healthHandler())

	// API routes
	r.Route("/api/v1", func(r chi.Router) {
		// Public endpoints
		r.Post("/auth/register", registerHandler(cfg.AuthSvc))
		r.Post("/auth/login", loginHandler(cfg.AuthSvc))

		// Optional auth endpoints
		r.Group(func(r chi.Router) {
			r.Use(appmiddleware.OptionalAuth(cfg.AuthSvc))
			// Routes that work differently for authenticated users
		})

		// Protected endpoints (require authentication)
		r.Group(func(r chi.Router) {
			r.Use(appmiddleware.RequireAuth(cfg.AuthSvc))

			// User-scoped todo routes
			r.Route("/todos", func(r chi.Router) {
				r.Get("/", listTodosHandler(cfg.TodoHandler))
				r.Post("/", createTodoHandler(cfg.TodoHandler))
				r.Get("/{id}", getTodoHandler(cfg.TodoHandler))
				r.Put("/{id}", updateTodoHandler(cfg.TodoHandler))
				r.Delete("/{id}", deleteTodoHandler(cfg.TodoHandler))
			})

			// User profile routes
			r.Get("/me", meHandler())
		})

		// Admin-only endpoints
		r.Group(func(r chi.Router) {
			r.Use(appmiddleware.RequireAdmin(cfg.AuthSvc))

			// Approved users management
			r.Route("/admin/approved-users", func(r chi.Router) {
				r.Get("/", listApprovedUsersHandler())
				r.Post("/", createApprovedUserHandler())
				r.Post("/bulk", bulkCreateApprovedUsersHandler())
				r.Delete("/{id}", deleteApprovedUserHandler())
			})
		})
	})

	return r
}

// rootHandler returns API version info
func rootHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"version":"1.0.0","status":"running"}`))
	}
}

// healthHandler returns health status
func healthHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"healthy"}`))
	}
}

// requestIDMiddleware generates a unique request ID for each request
func requestIDMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requestID := appmiddleware.GenerateRequestID()
			ctx := context.WithValue(r.Context(), appmiddleware.RequestIDKey, requestID)
			w.Header().Set("X-Request-ID", requestID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// realIPMiddleware extracts the real client IP from X-Forwarded-For or X-Real-IP headers
func realIPMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := appmiddleware.GetRealIP(r)
			ctx := context.WithValue(r.Context(), appmiddleware.ClientIPKey, ip)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// loggerMiddleware logs each request with trace context
func loggerMiddleware(logger *zap.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()

			// Add trace context to logger
			logger := logging.WithTraceContext(ctx, logger)

			// Log request start
			logger.Debug("request started",
				zap.String("method", r.Method),
				zap.String("path", r.URL.Path),
			)

			// Wrap response writer to capture status code
			wrapped := &responseWriter{ResponseWriter: w, status: http.StatusOK}

			next.ServeHTTP(wrapped, r)

			// Log request completion
			logger.Debug("request completed",
				zap.String("method", r.Method),
				zap.String("path", r.URL.Path),
				zap.Int("status", wrapped.status),
			)
		})
	}
}

// timeoutMiddleware sets a timeout for the request
func timeoutMiddleware(timeout time.Duration) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx, cancel := context.WithTimeout(r.Context(), timeout)
			defer cancel()

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// corsMiddleware handles Cross-Origin Resource Sharing
func corsMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Accept, Authorization, Content-Type, X-Request-ID")
			w.Header().Set("Access-Control-Max-Age", "3600")

			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// responseWriter wraps http.ResponseWriter to capture the status code
type responseWriter struct {
	http.ResponseWriter
	status int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.status = code
	rw.ResponseWriter.WriteHeader(code)
}

// Stub handlers - to be replaced with actual implementations
func registerHandler(authSvc appmiddleware.AuthProvider) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotImplemented)
	}
}

func loginHandler(authSvc appmiddleware.AuthProvider) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotImplemented)
	}
}

func listTodosHandler(handler interface{}) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotImplemented)
	}
}

func createTodoHandler(handler interface{}) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotImplemented)
	}
}

func getTodoHandler(handler interface{}) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotImplemented)
	}
}

func updateTodoHandler(handler interface{}) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotImplemented)
	}
}

func deleteTodoHandler(handler interface{}) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotImplemented)
	}
}

func meHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotImplemented)
	}
}

func listApprovedUsersHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotImplemented)
	}
}

func createApprovedUserHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotImplemented)
	}
}

func bulkCreateApprovedUsersHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotImplemented)
	}
}

func deleteApprovedUserHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotImplemented)
	}
}
