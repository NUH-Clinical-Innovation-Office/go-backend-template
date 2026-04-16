// Package auth provides authentication interfaces for dependency injection.
package auth

import (
	"context"

	"github.com/google/uuid"
	db "github.com/your-org/go-backend-template/internal/db/sqlc"
	"github.com/your-org/go-backend-template/internal/domain"
)

// UserRepository defines the interface for user data access
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

// AuthService defines the interface for authentication business logic
type AuthService interface {
	Register(ctx context.Context, email, password, approvedID string) (string, error)
	Login(ctx context.Context, email, password string) (string, error)
	GetUserFromToken(ctx context.Context, token string) (*domain.User, error)
	ListApprovedUsers(ctx context.Context) ([]*domain.ApprovedUser, error)
	CreateApprovedUser(ctx context.Context, email, firstName string, createdBy uuid.UUID) (*domain.ApprovedUser, error)
	BulkCreateApprovedUsers(ctx context.Context, emails, firstNames []string, createdBy uuid.UUID) ([]*domain.ApprovedUser, error)
	DeleteApprovedUser(ctx context.Context, id uuid.UUID) error
}
