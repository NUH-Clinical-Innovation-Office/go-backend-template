// Package todo provides todo item repository.
package todo

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	db "github.com/your-org/go-backend-template/internal/db/sqlc"
)

// Repository provides database access for todos
type Repository struct {
	db *db.Queries
}

// NewRepository creates a new todo repository
func NewRepository(q *db.Queries) *Repository {
	return &Repository{
		db: q,
	}
}

// GetTodoByID gets a todo by ID
func (r *Repository) GetTodoByID(ctx context.Context, id pgtype.UUID) (*db.Todo, error) {
	todo, err := r.db.GetTodoByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return &todo, nil
}

// ListTodosByUserID lists all todos for a user
func (r *Repository) ListTodosByUserID(ctx context.Context, userID uuid.UUID) ([]db.Todo, error) {
	pgUserID := pgtype.UUID{Bytes: userID, Valid: true}
	return r.db.ListTodosByUserID(ctx, pgUserID)
}

// CreateTodo creates a new todo
func (r *Repository) CreateTodo(ctx context.Context, params db.CreateTodoParams) (*db.Todo, error) {
	todo, err := r.db.CreateTodo(ctx, params)
	if err != nil {
		return nil, err
	}
	return &todo, nil
}

// UpdateTodo updates a todo
func (r *Repository) UpdateTodo(ctx context.Context, params db.UpdateTodoParams) (db.Todo, error) {
	return r.db.UpdateTodo(ctx, params)
}

// DeleteTodo deletes a todo
func (r *Repository) DeleteTodo(ctx context.Context, id pgtype.UUID) error {
	return r.db.DeleteTodo(ctx, id)
}
