package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/your-org/go-backend-template/internal/domain"
)

func TestGenerateRequestID(t *testing.T) {
	id1 := GenerateRequestID()
	id2 := GenerateRequestID()

	assert.NotEmpty(t, id1)
	assert.NotEmpty(t, id2)
	assert.NotEqual(t, id1, id2)
}

func TestGetRealIP(t *testing.T) {
	t.Run("X-Forwarded-For header", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/", nil)
		req.Header.Set("X-Forwarded-For", "192.168.1.1, 10.0.0.1")
		ip := GetRealIP(req)
		assert.Equal(t, "192.168.1.1", ip)
	})

	t.Run("X-Real-IP header", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/", nil)
		req.Header.Set("X-Real-IP", "192.168.1.1")
		ip := GetRealIP(req)
		assert.Equal(t, "192.168.1.1", ip)
	})

	t.Run("no headers - use RemoteAddr", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/", nil)
		ip := GetRealIP(req)
		assert.NotEmpty(t, ip)
	})
}

func TestUserFromContext(t *testing.T) {
	user := &domain.User{
		ID:       uuid.New(),
		IsActive: true,
	}

	// Test with user in context
	ctx := context.WithValue(context.Background(), CurrentUserKey, user)
	result := UserFromContext(ctx)
	assert.NotNil(t, result)
	assert.Equal(t, user.ID, result.ID)

	// Test without user in context
	ctx = context.Background()
	result = UserFromContext(ctx)
	assert.Nil(t, result)

	// Test with wrong type in context
	ctx = context.WithValue(context.Background(), CurrentUserKey, "not a user")
	result = UserFromContext(ctx)
	assert.Nil(t, result)
}

func TestRequestIDFromContext(t *testing.T) {
	// Test with request ID in context
	ctx := context.WithValue(context.Background(), RequestIDKey, "test-request-id")
	result := RequestIDFromContext(ctx)
	assert.Equal(t, "test-request-id", result)

	// Test without request ID in context
	ctx = context.Background()
	result = RequestIDFromContext(ctx)
	assert.Empty(t, result)
}

func TestClientIPFromContext(t *testing.T) {
	// Test with client IP in context
	ctx := context.WithValue(context.Background(), ClientIPKey, "192.168.1.1")
	result := ClientIPFromContext(ctx)
	assert.Equal(t, "192.168.1.1", result)

	// Test without client IP in context
	ctx = context.Background()
	result = ClientIPFromContext(ctx)
	assert.Empty(t, result)
}

func TestRequireAuth(t *testing.T) {
	mockAuthProvider := &mockAuthProvider{
		user: &domain.User{ID: uuid.New(), IsActive: true},
		err:  nil,
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	wrappedHandler := RequireAuth(mockAuthProvider)(handler)

	// Test with valid token
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "Bearer valid-token")
	w := httptest.NewRecorder()

	wrappedHandler.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	// Test with missing token
	req = httptest.NewRequest("GET", "/", nil)
	w = httptest.NewRecorder()

	wrappedHandler.ServeHTTP(w, req)
	assert.Equal(t, http.StatusUnauthorized, w.Code)

	// Test with invalid token
	req = httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "Bearer invalid-token")
	mockAuthProvider.err = assert.AnError
	w = httptest.NewRecorder()

	wrappedHandler.ServeHTTP(w, req)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestOptionalAuth(t *testing.T) {
	mockAuthProvider := &mockAuthProvider{
		user: &domain.User{ID: uuid.New(), IsActive: true},
		err:  nil,
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user := UserFromContext(r.Context())
		if user != nil {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("authenticated"))
		} else {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("anonymous"))
		}
	})

	wrappedHandler := OptionalAuth(mockAuthProvider)(handler)

	// Test with valid token
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "Bearer valid-token")
	w := httptest.NewRecorder()

	wrappedHandler.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "authenticated", w.Body.String())

	// Test without token
	req = httptest.NewRequest("GET", "/", nil)
	w = httptest.NewRecorder()

	wrappedHandler.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "anonymous", w.Body.String())
}

func TestRequireAdmin(t *testing.T) {
	adminUser := &domain.User{
		ID:       uuid.New(),
		IsActive: true,
		Roles:    []domain.Role{{Name: "admin"}},
	}

	regularUser := &domain.User{
		ID:       uuid.New(),
		IsActive: true,
		Roles:    []domain.Role{{Name: "user"}},
	}

	// Test with admin user
	mockAdminProvider := &mockAuthProvider{user: adminUser, err: nil}
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	wrappedHandler := RequireAdmin(mockAdminProvider)(handler)

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "Bearer valid-token")
	w := httptest.NewRecorder()

	wrappedHandler.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	// Test with regular user (should be forbidden)
	mockUserProvider := &mockAuthProvider{user: regularUser, err: nil}
	wrappedHandler = RequireAdmin(mockUserProvider)(handler)

	req = httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "Bearer valid-token")
	w = httptest.NewRecorder()

	wrappedHandler.ServeHTTP(w, req)
	assert.Equal(t, http.StatusForbidden, w.Code)
}

type mockAuthProvider struct {
	user *domain.User
	err  error
}

func (m *mockAuthProvider) GetUserFromToken(ctx context.Context, token string) (*domain.User, error) {
	if token == "invalid-token" {
		return nil, assert.AnError
	}
	return m.user, m.err
}

func TestUser_HasRole(t *testing.T) {
	user := &domain.User{
		Roles: []domain.Role{
			{Name: "user"},
			{Name: "admin"},
		},
	}

	assert.True(t, user.HasRole("user"))
	assert.True(t, user.HasRole("admin"))
	assert.False(t, user.HasRole("superadmin"))
}
