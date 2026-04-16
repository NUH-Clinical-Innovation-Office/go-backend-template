// Package auth provides authentication handlers.
package auth

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/your-org/go-backend-template/internal/domain"
	http2 "github.com/your-org/go-backend-template/internal/http"
	"github.com/your-org/go-backend-template/internal/middleware"
	"github.com/your-org/go-backend-template/internal/validator"
	"go.uber.org/zap"
)

// RegisterRequest represents a registration request
type RegisterRequest struct {
	Email      string `json:"email"`
	Password   string `json:"password"`
	ApprovedID string `json:"approved_id"`
}

// LoginRequest represents a login request
type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// AuthResponse represents an authentication response
type AuthResponse struct {
	Token     string `json:"token"`
	TokenType string `json:"token_type"`
}

// UserResponse represents a user response
type UserResponse struct {
	ID        string    `json:"id"`
	Email     string    `json:"email"`
	FirstName string    `json:"first_name"`
	IsActive  bool      `json:"is_active"`
	Roles     []string  `json:"roles"`
	CreatedAt time.Time `json:"created_at"`
}

// Handler holds auth dependencies
type Handler struct {
	svc    AuthService
	logger *zap.Logger
}

// NewHandler creates a new auth handler
func NewHandler(svc AuthService, logger *zap.Logger) *Handler {
	return &Handler{
		svc:    svc,
		logger: logger,
	}
}

// RegisterHandler handles user registration
func (h *Handler) RegisterHandler(w http.ResponseWriter, r *http.Request) {
	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http2.RespondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	v := &validator.RegisterRequestValidator{
		Email:      req.Email,
		Password:   req.Password,
		ApprovedID: req.ApprovedID,
	}
	if err := v.Validate(); err != nil {
		http2.RespondError(w, http.StatusBadRequest, err.Error())
		return
	}

	token, err := h.svc.Register(r.Context(), req.Email, req.Password, req.ApprovedID)
	if err != nil {
		h.logger.Error("register failed", zap.Error(err))
		if errors.Is(err, ErrUserNotFound) {
			http2.RespondError(w, http.StatusNotFound, "approved user not found")
			return
		}
		if errors.Is(err, ErrInvalidCredentials) {
			http2.RespondError(w, http.StatusBadRequest, "invalid approved_id format")
			return
		}
		if err.Error() == "user already exists" {
			http2.RespondError(w, http.StatusConflict, "user already exists")
			return
		}
		http2.RespondError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	http2.RespondJSON(w, http.StatusCreated, AuthResponse{
		Token:     token,
		TokenType: "bearer",
	})
}

// LoginHandler handles user login
func (h *Handler) LoginHandler(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http2.RespondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	v := &validator.LoginRequestValidator{
		Email:    req.Email,
		Password: req.Password,
	}
	if err := v.Validate(); err != nil {
		http2.RespondError(w, http.StatusBadRequest, err.Error())
		return
	}

	token, err := h.svc.Login(r.Context(), req.Email, req.Password)
	if err != nil {
		h.logger.Error("login failed", zap.Error(err))
		if errors.Is(err, ErrInvalidCredentials) {
			http2.RespondError(w, http.StatusUnauthorized, "invalid credentials")
			return
		}
		http2.RespondError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	http2.RespondJSON(w, http.StatusOK, AuthResponse{
		Token:     token,
		TokenType: "bearer",
	})
}

// GetMeHandler handles getting current user info
func (h *Handler) GetMeHandler(w http.ResponseWriter, r *http.Request) {
	user := middleware.UserFromContext(r.Context())
	if user == nil {
		http2.RespondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	roles := make([]string, len(user.Roles))
	for i, role := range user.Roles {
		roles[i] = role.Name
	}

	var email string
	var firstName string
	if user.ApprovedUser != nil {
		email = user.ApprovedUser.Email
		firstName = user.ApprovedUser.FirstName
	}

	http2.RespondJSON(w, http.StatusOK, UserResponse{
		ID:        user.ID.String(),
		Email:     email,
		FirstName: firstName,
		IsActive:  user.IsActive,
		Roles:     roles,
		CreatedAt: user.CreatedAt,
	})
}

// ApprovedUserRequest represents an approved user creation request
type ApprovedUserRequest struct {
	Email     string `json:"email"`
	FirstName string `json:"first_name"`
}

// ApprovedUserResponse represents an approved user response
type ApprovedUserResponse struct {
	ID        string    `json:"id"`
	Email     string    `json:"email"`
	FirstName string    `json:"first_name"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// BulkApprovedUserRequest represents a bulk approved user creation request
type BulkApprovedUserRequest struct {
	Users []ApprovedUserRequest `json:"users"`
}

func toApprovedUserResponse(au *domain.ApprovedUser) *ApprovedUserResponse {
	if au == nil {
		return nil
	}
	return &ApprovedUserResponse{
		ID:        au.ID.String(),
		Email:     au.Email,
		FirstName: au.FirstName,
		CreatedAt: au.CreatedAt,
		UpdatedAt: au.UpdatedAt,
	}
}

func toApprovedUserResponses(aus []*domain.ApprovedUser) []*ApprovedUserResponse {
	result := make([]*ApprovedUserResponse, 0, len(aus))
	for _, au := range aus {
		result = append(result, toApprovedUserResponse(au))
	}
	return result
}

// ListApprovedUsersHandler handles GET /admin/approved-users
func (h *Handler) ListApprovedUsersHandler(w http.ResponseWriter, r *http.Request) {
	users, err := h.svc.ListApprovedUsers(r.Context())
	if err != nil {
		h.logger.Error("list approved users failed", zap.Error(err))
		http2.RespondError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	http2.RespondJSON(w, http.StatusOK, toApprovedUserResponses(users))
}

// CreateApprovedUserHandler handles POST /admin/approved-users
func (h *Handler) CreateApprovedUserHandler(w http.ResponseWriter, r *http.Request) {
	var req ApprovedUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http2.RespondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	v := &validator.ApprovedUserRequestValidator{
		Email:     req.Email,
		FirstName: req.FirstName,
	}
	if err := v.Validate(); err != nil {
		http2.RespondError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Get creator from context
	creator := middleware.UserFromContext(r.Context())
	if creator == nil {
		http2.RespondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	approvedUser, err := h.svc.CreateApprovedUser(r.Context(), req.Email, req.FirstName, creator.ApprovedUserID)
	if err != nil {
		h.logger.Error("create approved user failed", zap.Error(err))
		if err.Error() == "email already in approved list" {
			http2.RespondError(w, http.StatusConflict, "Email already in approved list")
			return
		}
		http2.RespondError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	http2.RespondJSON(w, http.StatusCreated, toApprovedUserResponse(approvedUser))
}

// BulkCreateApprovedUsersHandler handles POST /admin/approved-users/bulk
func (h *Handler) BulkCreateApprovedUsersHandler(w http.ResponseWriter, r *http.Request) {
	var req BulkApprovedUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http2.RespondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if len(req.Users) == 0 {
		http2.RespondError(w, http.StatusBadRequest, "users array is required")
		return
	}

	// Get creator from context
	creator := middleware.UserFromContext(r.Context())
	if creator == nil {
		http2.RespondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	emails := make([]string, len(req.Users))
	firstNames := make([]string, len(req.Users))
	for i, u := range req.Users {
		emails[i] = u.Email
		firstNames[i] = u.FirstName
	}

	users, err := h.svc.BulkCreateApprovedUsers(r.Context(), emails, firstNames, creator.ID)
	if err != nil {
		h.logger.Error("bulk create approved users failed", zap.Error(err))
		http2.RespondError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	http2.RespondJSON(w, http.StatusCreated, toApprovedUserResponses(users))
}

// DeleteApprovedUserHandler handles DELETE /admin/approved-users/{id}
func (h *Handler) DeleteApprovedUserHandler(w http.ResponseWriter, r *http.Request) {
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

	if err := h.svc.DeleteApprovedUser(r.Context(), id); err != nil {
		h.logger.Error("delete approved user failed", zap.Error(err))
		http2.RespondError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
