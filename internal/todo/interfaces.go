// Package todo provides todo interfaces for dependency injection.
package todo

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	db "github.com/your-org/go-backend-template/internal/db/sqlc"
	"github.com/your-org/go-backend-template/internal/domain"
)

// TodoRepository defines the interface for todo data access
type TodoRepository interface {
	GetTodoByID(ctx context.Context, id pgtype.UUID) (*db.Todo, error)
	ListTodosByUserID(ctx context.Context, userID uuid.UUID) ([]db.Todo, error)
	CreateTodo(ctx context.Context, params *db.CreateTodoParams) (*db.Todo, error)
	UpdateTodo(ctx context.Context, params *db.UpdateTodoParams) (db.Todo, error)
	DeleteTodo(ctx context.Context, id pgtype.UUID) error
}

// TodoService defines the interface for todo business logic
type TodoService interface {
	ListByUserID(ctx context.Context, userID uuid.UUID) ([]domain.Todo, error)
	GetByID(ctx context.Context, todoID, userID uuid.UUID) (*domain.Todo, error)
	Create(ctx context.Context, userID uuid.UUID, title string, description *string, dueDate *time.Time) (*domain.Todo, error)
	Update(ctx context.Context, todoID, userID uuid.UUID, title string, description *string, isCompleted bool, dueDate *time.Time) (*domain.Todo, error)
	Delete(ctx context.Context, todoID, userID uuid.UUID) error
}
