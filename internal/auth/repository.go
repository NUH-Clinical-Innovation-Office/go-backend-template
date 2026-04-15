// Package auth provides authentication repository.
package auth

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	db "github.com/your-org/go-backend-template/internal/db/sqlc"
	"github.com/your-org/go-backend-template/internal/domain"
)

// Repository provides database access for auth
type Repository struct {
	db *db.Queries
}

// NewRepository creates a new auth repository
func NewRepository(q *db.Queries) *Repository {
	return &Repository{
		db: q,
	}
}

// GetUserByEmail gets a user by email
func (r *Repository) GetUserByEmail(ctx context.Context, email string) (*db.User, error) {
	user, err := r.db.GetUserByEmail(ctx, email)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// GetUserByID gets a user by ID
func (r *Repository) GetUserByID(ctx context.Context, id uuid.UUID) (*db.User, error) {
	pgID := pgtype.UUID{Bytes: id, Valid: true}
	user, err := r.db.GetUserByID(ctx, pgID)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// CreateUser creates a new user
func (r *Repository) CreateUser(ctx context.Context, params db.CreateUserParams) (*db.User, error) {
	user, err := r.db.CreateUser(ctx, params)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// GetApprovedUserByID gets an approved user by ID
func (r *Repository) GetApprovedUserByID(ctx context.Context, id uuid.UUID) (*db.ApprovedUser, error) {
	pgID := pgtype.UUID{Bytes: id, Valid: true}
	approvedUser, err := r.db.GetApprovedUserByID(ctx, pgID)
	if err != nil {
		return nil, err
	}
	return &approvedUser, nil
}

// GetUserRoles gets roles for a user
func (r *Repository) GetUserRoles(ctx context.Context, userID uuid.UUID) ([]db.Role, error) {
	pgUserID := pgtype.UUID{Bytes: userID, Valid: true}
	return r.db.GetUserRoles(ctx, pgUserID)
}

// GetRoleByName gets a role by name
func (r *Repository) GetRoleByName(ctx context.Context, name string) (*db.Role, error) {
	roles, err := r.db.GetRolesByNames(ctx, []string{name})
	if err != nil {
		return nil, err
	}
	if len(roles) == 0 {
		return nil, nil
	}
	return &roles[0], nil
}

// AssignRoleToUser assigns a role to a user
func (r *Repository) AssignRoleToUser(ctx context.Context, userID, roleID uuid.UUID) error {
	return r.db.AssignRole(ctx, db.AssignRoleParams{
		UserID: pgtype.UUID{Bytes: userID, Valid: true},
		RoleID: pgtype.UUID{Bytes: roleID, Valid: true},
	})
}

// uuidToPgtype converts uuid.UUID to pgtype.UUID
func uuidToPgtype(u uuid.UUID) pgtype.UUID {
	return pgtype.UUID{Bytes: u, Valid: true}
}

// pgtypeToUuid converts pgtype.UUID to uuid.UUID
func pgtypeToUuid(p pgtype.UUID) uuid.UUID {
	if !p.Valid {
		return uuid.Nil
	}
	return uuid.UUID(p.Bytes)
}

// pgApprovedUserToUuid converts pgtype.UUID to *uuid.UUID
func pgApprovedUserToUuid(p pgtype.UUID) *uuid.UUID {
	if !p.Valid {
		return nil
	}
	u := uuid.UUID(p.Bytes)
	return &u
}

// ToDomainUser converts db.User to domain.User
func (r *Repository) ToDomainUser(user *db.User, approvedUser *db.ApprovedUser, roles []db.Role) *domain.User {
	domainRoles := make([]domain.Role, len(roles))
	for i, role := range roles {
		domainRoles[i] = domain.Role{
			ID:          pgtypeToUuid(role.ID),
			Name:        role.Name,
			Description: role.Description,
			CreatedAt:   role.CreatedAt.Time,
		}
	}

	var approvedUserDomain *domain.ApprovedUser
	if approvedUser != nil {
		approvedUserDomain = &domain.ApprovedUser{
			ID:        pgtypeToUuid(approvedUser.ID),
			Email:     approvedUser.Email,
			FirstName: approvedUser.FirstName,
			CreatedBy: pgApprovedUserToUuid(approvedUser.CreatedBy),
			CreatedAt: approvedUser.CreatedAt.Time,
			UpdatedAt: approvedUser.UpdatedAt.Time,
		}
	}

	return &domain.User{
		ID:             pgtypeToUuid(user.ID),
		ApprovedUserID: pgtypeToUuid(user.ApprovedUserID),
		HashedPassword: user.PasswordHash,
		IsActive:       user.IsActive,
		CreatedAt:      user.CreatedAt.Time,
		UpdatedAt:      user.UpdatedAt.Time,
		Roles:          domainRoles,
		ApprovedUser:   approvedUserDomain,
	}
}

// ListApprovedUsers lists all approved users
func (r *Repository) ListApprovedUsers(ctx context.Context) ([]*domain.ApprovedUser, error) {
	rows, err := r.db.ListApprovedUsers(ctx)
	if err != nil {
		return nil, err
	}

	result := make([]*domain.ApprovedUser, 0, len(rows))
	for i := range rows {
		row := &rows[i]
		result = append(result, &domain.ApprovedUser{
			ID:        pgtypeToUuid(row.ID),
			Email:     row.Email,
			FirstName: row.FirstName,
			CreatedBy: pgApprovedUserToUuid(row.CreatedBy),
			CreatedAt: row.CreatedAt.Time,
			UpdatedAt: row.UpdatedAt.Time,
		})
	}
	return result, nil
}

// CreateApprovedUser creates a new approved user
func (r *Repository) CreateApprovedUser(ctx context.Context, email, firstName string, createdBy uuid.UUID) (*domain.ApprovedUser, error) {
	row, err := r.db.CreateApprovedUser(ctx, db.CreateApprovedUserParams{
		Email:     email,
		FirstName: firstName,
		CreatedBy: uuidToPgtype(createdBy),
	})
	if err != nil {
		return nil, err
	}

	return &domain.ApprovedUser{
		ID:        pgtypeToUuid(row.ID),
		Email:     row.Email,
		FirstName: row.FirstName,
		CreatedBy: pgApprovedUserToUuid(row.CreatedBy),
		CreatedAt: row.CreatedAt.Time,
		UpdatedAt: row.UpdatedAt.Time,
	}, nil
}

// BulkCreateApprovedUsers creates multiple approved users
func (r *Repository) BulkCreateApprovedUsers(ctx context.Context, emails, firstNames []string, createdBy uuid.UUID) ([]*domain.ApprovedUser, error) {
	pgCreatedBy := uuidToPgtype(createdBy)
	emailsPg := make([]string, len(emails))
	firstNamesPg := make([]string, len(firstNames))
	createdByPg := make([]pgtype.UUID, len(emails))
	copy(emailsPg, emails)
	copy(firstNamesPg, firstNames)
	for i := range createdByPg {
		createdByPg[i] = pgCreatedBy
	}

	rows, err := r.db.CreateApprovedUsersBulk(ctx, db.CreateApprovedUsersBulkParams{
		Column1: emailsPg,
		Column2: firstNamesPg,
		Column3: createdByPg,
	})
	if err != nil {
		return nil, err
	}

	result := make([]*domain.ApprovedUser, 0, len(rows))
	for i := range rows {
		row := &rows[i]
		result = append(result, &domain.ApprovedUser{
			ID:        pgtypeToUuid(row.ID),
			Email:     row.Email,
			FirstName: row.FirstName,
			CreatedBy: pgApprovedUserToUuid(row.CreatedBy),
			CreatedAt: row.CreatedAt.Time,
			UpdatedAt: row.UpdatedAt.Time,
		})
	}
	return result, nil
}

// DeleteApprovedUser deletes an approved user
func (r *Repository) DeleteApprovedUser(ctx context.Context, id uuid.UUID) error {
	return r.db.DeleteApprovedUser(ctx, uuidToPgtype(id))
}

// GetApprovedUserByEmail gets an approved user by email
func (r *Repository) GetApprovedUserByEmail(ctx context.Context, email string) (*domain.ApprovedUser, error) {
	row, err := r.db.GetApprovedUserByEmail(ctx, email)
	if err != nil {
		return nil, err
	}
	return &domain.ApprovedUser{
		ID:        pgtypeToUuid(row.ID),
		Email:     row.Email,
		FirstName: row.FirstName,
		CreatedBy: pgApprovedUserToUuid(row.CreatedBy),
		CreatedAt: row.CreatedAt.Time,
		UpdatedAt: row.UpdatedAt.Time,
	}, nil
}
