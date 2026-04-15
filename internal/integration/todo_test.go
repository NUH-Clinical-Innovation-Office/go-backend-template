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

func TestTodoCRUD(t *testing.T) {
	pool, _, authService, _, todoService, _, authHandler, todoHandler := setupTestDeps(t)
	defer pool.Close()

	// Setup: create approved user and registered user
	ctx := context.Background()
	_, err := pool.Pool.Exec(ctx,
		"INSERT INTO approved_users (id, email, first_name) VALUES ('00000000-0000-0000-0000-000000000010'::uuid, 'todo@example.com', 'Todo')")
	require.NoError(t, err)

	// Register user
	token, err := authService.Register(ctx, "todo@example.com", "password123", "00000000-0000-0000-0000-000000000010")
	require.NoError(t, err)

	// Setup router
	r := router.New(router.RouterConfig{
		AuthSvc:     authService,
		AuthHandler: authHandler,
		TodoHandler: todoHandler,
		TodoService: todoService,
	})

	t.Run("create todo", func(t *testing.T) {
		body := map[string]string{
			"title": "Test todo",
		}
		req := httptest.NewRequest(http.MethodPost, "/api/v1/todos", bytes.NewReader(mustJSON(body)))
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)
		assert.Contains(t, w.Body.String(), "Test todo")
	})

	t.Run("list todos", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/todos", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), "todos")
	})

	t.Run("get todo by id", func(t *testing.T) {
		// First create a todo to get its ID
		createBody := map[string]string{"title": "Get test"}
		createReq := httptest.NewRequest(http.MethodPost, "/api/v1/todos", bytes.NewReader(mustJSON(createBody)))
		createReq.Header.Set("Authorization", "Bearer "+token)
		createW := httptest.NewRecorder()
		r.ServeHTTP(createW, createReq)

		var created map[string]interface{}
		json.Unmarshal(createW.Body.Bytes(), &created)
		todoID := created["id"].(string)

		// Now get it
		req := httptest.NewRequest(http.MethodGet, "/api/v1/todos/"+todoID, nil)
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), "Get test")
	})

	t.Run("update todo", func(t *testing.T) {
		// Create a todo first
		createBody := map[string]string{"title": "Update test"}
		createReq := httptest.NewRequest(http.MethodPost, "/api/v1/todos", bytes.NewReader(mustJSON(createBody)))
		createReq.Header.Set("Authorization", "Bearer "+token)
		createW := httptest.NewRecorder()
		r.ServeHTTP(createW, createReq)

		var created map[string]interface{}
		json.Unmarshal(createW.Body.Bytes(), &created)
		todoID := created["id"].(string)

		// Update it
		updateBody := map[string]interface{}{
			"title":        "Updated title",
			"is_completed": true,
		}
		updateReq := httptest.NewRequest(http.MethodPut, "/api/v1/todos/"+todoID, bytes.NewReader(mustJSON(updateBody)))
		updateReq.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()

		r.ServeHTTP(w, updateReq)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), "Updated title")
	})

	t.Run("delete todo", func(t *testing.T) {
		// Create a todo first
		createBody := map[string]string{"title": "Delete test"}
		createReq := httptest.NewRequest(http.MethodPost, "/api/v1/todos", bytes.NewReader(mustJSON(createBody)))
		createReq.Header.Set("Authorization", "Bearer "+token)
		createW := httptest.NewRecorder()
		r.ServeHTTP(createW, createReq)

		var created map[string]interface{}
		json.Unmarshal(createW.Body.Bytes(), &created)
		todoID := created["id"].(string)

		// Delete it
		delReq := httptest.NewRequest(http.MethodDelete, "/api/v1/todos/"+todoID, nil)
		delReq.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()

		r.ServeHTTP(w, delReq)

		assert.Equal(t, http.StatusNoContent, w.Code)

		// Verify it's gone
		getReq := httptest.NewRequest(http.MethodGet, "/api/v1/todos/"+todoID, nil)
		getReq.Header.Set("Authorization", "Bearer "+token)
		getW := httptest.NewRecorder()
		r.ServeHTTP(getW, getReq)
		assert.Equal(t, http.StatusNotFound, getW.Code)
	})

	t.Run("unauthorized access", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/todos", nil)
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})
}

func TestAuthGetMe(t *testing.T) {
	pool, _, authService, _, _, _, authHandler, _ := setupTestDeps(t)
	defer pool.Close()

	// Setup user
	ctx := context.Background()
	_, err := pool.Pool.Exec(ctx,
		"INSERT INTO approved_users (id, email, first_name) VALUES ('00000000-0000-0000-0000-000000000011'::uuid, 'me@example.com', 'Me')")
	require.NoError(t, err)

	token, err := authService.Register(ctx, "me@example.com", "password123", "00000000-0000-0000-0000-000000000011")
	require.NoError(t, err)

	r := router.New(router.RouterConfig{
		AuthSvc:     authService,
		AuthHandler: authHandler,
	})

	t.Run("get current user", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/me", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), "me@example.com")
	})
}
