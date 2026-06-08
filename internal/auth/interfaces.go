// Package auth provides authentication interfaces for dependency injection.
package auth

import (
	"context"

	"github.com/google/uuid"
	db "github.com/your-org/go-backend-template/internal/db/sqlc"
	"github.com/your-org/go-backend-template/internal/domain"
)

// UserRepository defines the interface for user data access.
type UserRepository interface {
	GetUserByEmail(ctx context.Context, email string) (*db.User, error)
	GetUserByID(ctx context.Context, id uuid.UUID) (*db.User, error)
	CreateUser(ctx context.Context, params db.CreateUserParams) (*db.User, error)
	GetApprovedUserByID(ctx context.Context, id uuid.UUID) (*db.ApprovedUser, error)
	GetUserRoles(ctx context.Context, userID uuid.UUID) ([]db.Role, error)
	GetRoleByName(ctx context.Context, name string) (*db.Role, error)
	AssignRoleToUser(ctx context.Context, userID, roleID uuid.UUID) error
	ToDomainUser(user *db.User, approvedUser *db.ApprovedUser, roles []db.Role) *domain.User
	ListApprovedUsers(ctx context.Context) ([]*domain.ApprovedUser, error)
	CreateApprovedUser(ctx context.Context, email, firstName string, createdBy uuid.UUID) (*domain.ApprovedUser, error)
	BulkCreateApprovedUsers(ctx context.Context, emails, firstNames []string, createdBy uuid.UUID) ([]*domain.ApprovedUser, error)
	DeleteApprovedUser(ctx context.Context, id uuid.UUID) error
	GetApprovedUserByEmail(ctx context.Context, email string) (*domain.ApprovedUser, error)
}

// AuthService is the authentication-focused surface. Middleware that only
// needs to validate tokens depends on this and nothing else.
type AuthService interface {
	Register(ctx context.Context, email, password, approvedID string) (string, error)
	Login(ctx context.Context, email, password string) (string, error)
	GetUserFromToken(ctx context.Context, token string) (*domain.User, error)
}

// ApprovedUserAdminService is the admin surface for managing the approved
// users whitelist. Kept separate from AuthService so handlers and the
// dependency graph don't pull in admin operations they don't need.
type ApprovedUserAdminService interface {
	ListApprovedUsers(ctx context.Context) ([]*domain.ApprovedUser, error)
	CreateApprovedUser(ctx context.Context, email, firstName string, createdBy uuid.UUID) (*domain.ApprovedUser, error)
	BulkCreateApprovedUsers(ctx context.Context, emails, firstNames []string, createdBy uuid.UUID) ([]*domain.ApprovedUser, error)
	DeleteApprovedUser(ctx context.Context, id uuid.UUID) error
}
