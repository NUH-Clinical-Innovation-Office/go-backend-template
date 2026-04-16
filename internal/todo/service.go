// Package todo provides todo item service.
package todo

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	db "github.com/your-org/go-backend-template/internal/db/sqlc"
	"github.com/your-org/go-backend-template/internal/domain"
)

var (
	ErrTodoNotFound      = errors.New("todo not found")
	ErrTodoNotOwned      = errors.New("todo does not belong to user")
	ErrInvalidTodoParams = errors.New("invalid todo parameters")
)

// Service provides todo business logic
type Service struct {
	repo TodoRepository
}

// NewService creates a new todo service
func NewService(repo TodoRepository) *Service {
	return &Service{
		repo: repo,
	}
}

// ListByUserID lists all todos for a user
func (s *Service) ListByUserID(ctx context.Context, userID uuid.UUID) ([]domain.Todo, error) {
	todos, err := s.repo.ListTodosByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	result := make([]domain.Todo, len(todos))
	for i := range todos {
		result[i] = s.toDomainTodo(&todos[i])
	}
	return result, nil
}

// GetByID gets a todo by ID, ensuring it belongs to the user
func (s *Service) GetByID(ctx context.Context, todoID, userID uuid.UUID) (*domain.Todo, error) {
	pgTodoID := pgtype.UUID{Bytes: todoID, Valid: true}
	todo, err := s.repo.GetTodoByID(ctx, pgTodoID)
	if err != nil {
		return nil, ErrTodoNotFound
	}

	if uuid.UUID(todo.UserID.Bytes) != userID {
		return nil, ErrTodoNotOwned
	}

	result := s.toDomainTodo(todo)
	return &result, nil
}

// Create creates a new todo
func (s *Service) Create(ctx context.Context, userID uuid.UUID, title string, description *string, dueDate *time.Time) (*domain.Todo, error) {
	if title == "" {
		return nil, ErrInvalidTodoParams
	}

	var pgDueDate pgtype.Timestamptz
	if dueDate != nil {
		pgDueDate = pgtype.Timestamptz{Time: *dueDate, Valid: true}
	}

	var desc *string
	if description != nil {
		d := *description
		desc = &d
	}

	pgUserID := pgtype.UUID{Bytes: userID, Valid: true}
	todo, err := s.repo.CreateTodo(ctx, &db.CreateTodoParams{
		UserID:      pgUserID,
		Title:       title,
		Description: desc,
		IsCompleted: false,
		DueDate:     pgDueDate,
	})
	if err != nil {
		return nil, err
	}

	result := s.toDomainTodo(todo)
	return &result, nil
}

// Update updates a todo, ensuring it belongs to the user
func (s *Service) Update(ctx context.Context, todoID, userID uuid.UUID, title string, description *string, isCompleted bool, dueDate *time.Time) (*domain.Todo, error) {
	if title == "" {
		return nil, ErrInvalidTodoParams
	}

	pgTodoID := pgtype.UUID{Bytes: todoID, Valid: true}

	// First verify ownership
	existing, err := s.repo.GetTodoByID(ctx, pgTodoID)
	if err != nil {
		return nil, ErrTodoNotFound
	}

	if uuid.UUID(existing.UserID.Bytes) != userID {
		return nil, ErrTodoNotOwned
	}

	var pgDueDate pgtype.Timestamptz
	if dueDate != nil {
		pgDueDate = pgtype.Timestamptz{Time: *dueDate, Valid: true}
	}

	var desc *string
	if description != nil {
		d := *description
		desc = &d
	}

	updated, err := s.repo.UpdateTodo(ctx, &db.UpdateTodoParams{
		ID:          pgTodoID,
		Title:       title,
		Description: desc,
		IsCompleted: isCompleted,
		DueDate:     pgDueDate,
	})
	if err != nil {
		return nil, err
	}

	result := s.toDomainTodo(&updated)
	return &result, nil
}

// Delete deletes a todo, ensuring it belongs to the user
func (s *Service) Delete(ctx context.Context, todoID, userID uuid.UUID) error {
	pgTodoID := pgtype.UUID{Bytes: todoID, Valid: true}

	// First verify ownership
	existing, err := s.repo.GetTodoByID(ctx, pgTodoID)
	if err != nil {
		return ErrTodoNotFound
	}

	if uuid.UUID(existing.UserID.Bytes) != userID {
		return ErrTodoNotOwned
	}

	return s.repo.DeleteTodo(ctx, pgTodoID)
}

func (s *Service) toDomainTodo(todo *db.Todo) domain.Todo {
	var dueDate *time.Time
	if todo.DueDate.Valid {
		t := todo.DueDate.Time
		dueDate = &t
	}

	var description *string
	if todo.Description != nil {
		description = todo.Description
	}

	return domain.Todo{
		ID:          uuid.UUID(todo.ID.Bytes),
		UserID:      uuid.UUID(todo.UserID.Bytes),
		Title:       todo.Title,
		Description: description,
		IsCompleted: todo.IsCompleted,
		DueDate:     dueDate,
		CreatedAt:   todo.CreatedAt.Time,
		UpdatedAt:   todo.UpdatedAt.Time,
	}
}
