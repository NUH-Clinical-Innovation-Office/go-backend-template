package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/your-org/go-backend-template/internal/router"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAuthRegister(t *testing.T) {
	pool, _, _, _, _, _, authHandler, _ := setupTestDeps(t)
	defer pool.Close()

	// First create an approved user
	ctx := context.Background()
	_, err := pool.Pool.Exec(ctx,
		"INSERT INTO approved_users (id, email, first_name) VALUES ('00000000-0000-0000-0000-000000000001'::uuid, 'test@example.com', 'Test')")
	require.NoError(t, err)

	r := router.New(router.RouterConfig{
		AuthHandler: authHandler,
	})

	t.Run("successful registration", func(t *testing.T) {
		body := map[string]string{
			"email":       "test@example.com",
			"password":    "password123",
			"approved_id": "00000000-0000-0000-0000-000000000001",
		}
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewReader(mustJSON(body)))
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)
		assert.Contains(t, w.Body.String(), "token")
	})

	t.Run("missing approved_id", func(t *testing.T) {
		body := map[string]string{
			"email":    "test@example.com",
			"password": "password123",
		}
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewReader(mustJSON(body)))
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("non-existent approved user", func(t *testing.T) {
		body := map[string]string{
			"email":       "test2@example.com",
			"password":    "password123",
			"approved_id": "00000000-0000-0000-0000-000000000999",
		}
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewReader(mustJSON(body)))
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})
}

func TestAuthLogin(t *testing.T) {
	pool, _, authService, _, _, _, authHandler, _ := setupTestDeps(t)
	defer pool.Close()

	// Create approved user and registered user
	ctx := context.Background()
	_, err := pool.Pool.Exec(ctx,
		"INSERT INTO approved_users (id, email, first_name) VALUES ('00000000-0000-0000-0000-000000000002'::uuid, 'login@example.com', 'Login')")
	require.NoError(t, err)

	// Register the user
	_, err = authService.Register(ctx, "login@example.com", "password123", "00000000-0000-0000-0000-000000000002")
	require.NoError(t, err)

	r := router.New(router.RouterConfig{
		AuthHandler: authHandler,
	})

	t.Run("successful login", func(t *testing.T) {
		body := map[string]string{
			"email":    "login@example.com",
			"password": "password123",
		}
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(mustJSON(body)))
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), "token")
	})

	t.Run("invalid password", func(t *testing.T) {
		body := map[string]string{
			"email":    "login@example.com",
			"password": "wrongpassword",
		}
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(mustJSON(body)))
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("non-existent user", func(t *testing.T) {
		body := map[string]string{
			"email":    "notfound@example.com",
			"password": "password123",
		}
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(mustJSON(body)))
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})
}

func mustJSON(v interface{}) []byte {
	b, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return b
}

func TestAdminApprovedUsers(t *testing.T) {
	pool, _, authService, _, _, _, authHandler, _ := setupTestDeps(t)
	defer pool.Close()

	ctx := context.Background()

	// Create admin user
	_, err := pool.Pool.Exec(ctx,
		"INSERT INTO approved_users (id, email, first_name) VALUES ('00000000-0000-0000-0000-000000000020'::uuid, 'admin@example.com', 'Admin')")
	require.NoError(t, err)

	adminToken, err := authService.Register(ctx, "admin@example.com", "password123", "00000000-0000-0000-0000-000000000020")
	require.NoError(t, err)

	// Assign admin role
	_, err = pool.Pool.Exec(ctx,
		"INSERT INTO roles (id, name) VALUES ('00000000-0000-0000-0000-000000000001'::uuid, 'admin') ON CONFLICT (name) DO NOTHING")
	require.NoError(t, err)
	_, err = pool.Pool.Exec(ctx,
		"INSERT INTO user_roles (user_id, role_id) VALUES ((SELECT id FROM users WHERE email = 'admin@example.com'), '00000000-0000-0000-0000-000000000001'::uuid)")
	require.NoError(t, err)

	r := router.New(router.RouterConfig{
		AuthSvc:     authService,
		AuthHandler: authHandler,
	})

	t.Run("create approved user", func(t *testing.T) {
		body := map[string]string{
			"email":      "newuser@example.com",
			"first_name": "New",
		}
		req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/approved-users", bytes.NewReader(mustJSON(body)))
		req.Header.Set("Authorization", "Bearer "+adminToken)
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)
		assert.Contains(t, w.Body.String(), "newuser@example.com")
	})

	t.Run("list approved users", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/approved-users", nil)
		req.Header.Set("Authorization", "Bearer "+adminToken)
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), "email")
	})

	t.Run("unauthorized access", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/approved-users", nil)
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})
}
