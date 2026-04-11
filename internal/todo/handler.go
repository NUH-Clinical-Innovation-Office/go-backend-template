// Package todo provides todo item handlers.
package todo

import (
	"encoding/json"
	"net/http"

	http2 "github.com/your-org/go-backend-template/internal/http"
	"github.com/your-org/go-backend-template/internal/middleware"
)

// CreateTodoRequest represents a create todo request
type CreateTodoRequest struct {
	Title       string  `json:"title"`
	Description *string `json:"description"`
	DueDate     *string `json:"due_date"`
}

// UpdateTodoRequest represents an update todo request
type UpdateTodoRequest struct {
	Title       string  `json:"title"`
	Description *string `json:"description"`
	IsCompleted bool    `json:"is_completed"`
	DueDate     *string `json:"due_date"`
}

// ListHandler handles listing todos for the current user
func ListHandler(w http.ResponseWriter, r *http.Request) {
	user := middleware.UserFromContext(r.Context())
	if user == nil {
		http2.RespondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	// TODO: Implement
	http2.RespondError(w, http.StatusNotImplemented, "not implemented")
}

// CreateHandler handles creating a new todo
func CreateHandler(w http.ResponseWriter, r *http.Request) {
	user := middleware.UserFromContext(r.Context())
	if user == nil {
		http2.RespondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req CreateTodoRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http2.RespondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// TODO: Implement
	http2.RespondError(w, http.StatusNotImplemented, "not implemented")
}

// GetHandler handles getting a single todo by ID
func GetHandler(w http.ResponseWriter, r *http.Request) {
	user := middleware.UserFromContext(r.Context())
	if user == nil {
		http2.RespondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	// TODO: Implement
	http2.RespondError(w, http.StatusNotImplemented, "not implemented")
}

// UpdateHandler handles updating a todo
func UpdateHandler(w http.ResponseWriter, r *http.Request) {
	user := middleware.UserFromContext(r.Context())
	if user == nil {
		http2.RespondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req UpdateTodoRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http2.RespondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// TODO: Implement
	http2.RespondError(w, http.StatusNotImplemented, "not implemented")
}

// DeleteHandler handles deleting a todo
func DeleteHandler(w http.ResponseWriter, r *http.Request) {
	user := middleware.UserFromContext(r.Context())
	if user == nil {
		http2.RespondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	// TODO: Implement
	http2.RespondError(w, http.StatusNotImplemented, "not implemented")
}
