// Package todo provides todo item handlers.
package todo

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
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

// TodoResponse represents a todo response
type TodoResponse struct {
	ID          string     `json:"id"`
	UserID      string     `json:"user_id"`
	Title       string     `json:"title"`
	Description *string    `json:"description"`
	IsCompleted bool       `json:"is_completed"`
	DueDate     *time.Time `json:"due_date"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

// Handler holds todo dependencies
type Handler struct {
	svc *Service
}

// NewHandler creates a new todo handler
func NewHandler(svc *Service) *Handler {
	return &Handler{
		svc: svc,
	}
}

// ListHandler handles listing todos for the current user
func (h *Handler) ListHandler(w http.ResponseWriter, r *http.Request) {
	user := middleware.UserFromContext(r.Context())
	if user == nil {
		http2.RespondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	todos, err := h.svc.ListByUserID(r.Context(), user.ID)
	if err != nil {
		http2.RespondError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	response := make([]TodoResponse, len(todos))
	for i, todo := range todos {
		response[i] = h.toTodoResponse(todo)
	}

	http2.RespondJSON(w, http.StatusOK, response)
}

// CreateHandler handles creating a new todo
func (h *Handler) CreateHandler(w http.ResponseWriter, r *http.Request) {
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

	if req.Title == "" {
		http2.RespondError(w, http.StatusBadRequest, "title is required")
		return
	}

	var dueDate *time.Time
	if req.DueDate != nil && *req.DueDate != "" {
		t, err := time.Parse(time.RFC3339, *req.DueDate)
		if err == nil {
			dueDate = &t
		}
	}

	todo, err := h.svc.Create(r.Context(), user.ID, req.Title, req.Description, dueDate)
	if err != nil {
		http2.RespondError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	http2.RespondJSON(w, http.StatusCreated, h.toTodoResponse(*todo))
}

// GetHandler handles getting a single todo by ID
func (h *Handler) GetHandler(w http.ResponseWriter, r *http.Request) {
	user := middleware.UserFromContext(r.Context())
	if user == nil {
		http2.RespondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	idStr := chi.URLParam(r, "id")
	if idStr == "" {
		http2.RespondError(w, http.StatusBadRequest, "id is required")
		return
	}

	id, err := uuid.Parse(idStr)
	if err != nil {
		http2.RespondError(w, http.StatusBadRequest, "invalid id format")
		return
	}

	todo, err := h.svc.GetByID(r.Context(), id, user.ID)
	if err != nil {
		if err == ErrTodoNotFound || err == ErrTodoNotOwned {
			http2.RespondError(w, http.StatusNotFound, "todo not found")
			return
		}
		http2.RespondError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	http2.RespondJSON(w, http.StatusOK, h.toTodoResponse(*todo))
}

// UpdateHandler handles updating a todo
func (h *Handler) UpdateHandler(w http.ResponseWriter, r *http.Request) {
	user := middleware.UserFromContext(r.Context())
	if user == nil {
		http2.RespondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	idStr := chi.URLParam(r, "id")
	if idStr == "" {
		http2.RespondError(w, http.StatusBadRequest, "id is required")
		return
	}

	id, err := uuid.Parse(idStr)
	if err != nil {
		http2.RespondError(w, http.StatusBadRequest, "invalid id format")
		return
	}

	var req UpdateTodoRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http2.RespondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Title == "" {
		http2.RespondError(w, http.StatusBadRequest, "title is required")
		return
	}

	var dueDate *time.Time
	if req.DueDate != nil && *req.DueDate != "" {
		t, err := time.Parse(time.RFC3339, *req.DueDate)
		if err == nil {
			dueDate = &t
		}
	}

	todo, err := h.svc.Update(r.Context(), id, user.ID, req.Title, req.Description, req.IsCompleted, dueDate)
	if err != nil {
		if err == ErrTodoNotFound || err == ErrTodoNotOwned {
			http2.RespondError(w, http.StatusNotFound, "todo not found")
			return
		}
		http2.RespondError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	http2.RespondJSON(w, http.StatusOK, h.toTodoResponse(*todo))
}

// DeleteHandler handles deleting a todo
func (h *Handler) DeleteHandler(w http.ResponseWriter, r *http.Request) {
	user := middleware.UserFromContext(r.Context())
	if user == nil {
		http2.RespondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	idStr := chi.URLParam(r, "id")
	if idStr == "" {
		http2.RespondError(w, http.StatusBadRequest, "id is required")
		return
	}

	id, err := uuid.Parse(idStr)
	if err != nil {
		http2.RespondError(w, http.StatusBadRequest, "invalid id format")
		return
	}

	err = h.svc.Delete(r.Context(), id, user.ID)
	if err != nil {
		if err == ErrTodoNotFound || err == ErrTodoNotOwned {
			http2.RespondError(w, http.StatusNotFound, "todo not found")
			return
		}
		http2.RespondError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) toTodoResponse(todo interface{}) TodoResponse {
	switch t := todo.(type) {
	case TodoResponse:
		return t
	default:
		_ = t
		return TodoResponse{}
	}
}
