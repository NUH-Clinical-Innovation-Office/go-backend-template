package auth

import (
	"context"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	db "github.com/your-org/go-backend-template/internal/db/sqlc"
	"github.com/your-org/go-backend-template/internal/domain"
	"golang.org/x/crypto/bcrypt"
)

// mockUserRepository is a hand-rolled in-memory test double. It implements
// the UserRepository interface via the same exported method names; only
// the methods exercised by Service tests are populated.
type mockUserRepository struct {
	getUserByEmail func(ctx context.Context, email string) (*db.User, error)
	getUserByID    func(ctx context.Context, id uuid.UUID) (*db.User, error)
	getApprovedByID func(ctx context.Context, id uuid.UUID) (*db.ApprovedUser, error)
	getRolesByName func(ctx context.Context, name string) (*db.Role, error)
	assignRole     func(ctx context.Context, userID, roleID uuid.UUID) error
	createUser     func(ctx context.Context, params db.CreateUserParams) (*db.User, error)
	getApprovedByEmail func(ctx context.Context, email string) (*domain.ApprovedUser, error)
}

func (m *mockUserRepository) GetUserByEmail(ctx context.Context, email string) (*db.User, error) {
	if m.getUserByEmail == nil {
		return nil, assert.AnError
	}
	return m.getUserByEmail(ctx, email)
}
func (m *mockUserRepository) GetUserByID(ctx context.Context, id uuid.UUID) (*db.User, error) {
	if m.getUserByID == nil {
		return nil, assert.AnError
	}
	return m.getUserByID(ctx, id)
}
func (m *mockUserRepository) CreateUser(ctx context.Context, params db.CreateUserParams) (*db.User, error) {
	if m.createUser == nil {
		return nil, assert.AnError
	}
	return m.createUser(ctx, params)
}
func (m *mockUserRepository) GetApprovedUserByID(ctx context.Context, id uuid.UUID) (*db.ApprovedUser, error) {
	if m.getApprovedByID == nil {
		return nil, assert.AnError
	}
	return m.getApprovedByID(ctx, id)
}
func (m *mockUserRepository) GetUserRoles(_ context.Context, _ uuid.UUID) ([]db.Role, error) {
	return nil, nil
}
func (m *mockUserRepository) GetRoleByName(ctx context.Context, name string) (*db.Role, error) {
	if m.getRolesByName == nil {
		return nil, nil
	}
	return m.getRolesByName(ctx, name)
}
func (m *mockUserRepository) AssignRoleToUser(ctx context.Context, userID, roleID uuid.UUID) error {
	if m.assignRole == nil {
		return nil
	}
	return m.assignRole(ctx, userID, roleID)
}
func (m *mockUserRepository) ToDomainUser(user *db.User, _ *db.ApprovedUser, _ []db.Role) *domain.User {
	return &domain.User{
		ID:             pgtypeToUuid(user.ID),
		ApprovedUserID: pgtypeToUuid(user.ApprovedUserID),
		HashedPassword: user.PasswordHash,
		IsActive:       user.IsActive,
	}
}
func (m *mockUserRepository) ListApprovedUsers(_ context.Context) ([]*domain.ApprovedUser, error) {
	return nil, nil
}
func (m *mockUserRepository) CreateApprovedUser(_ context.Context, _, _ string, _ uuid.UUID) (*domain.ApprovedUser, error) {
	return nil, nil
}
func (m *mockUserRepository) BulkCreateApprovedUsers(_ context.Context, _, _ []string, _ uuid.UUID) ([]*domain.ApprovedUser, error) {
	return nil, nil
}
func (m *mockUserRepository) DeleteApprovedUser(_ context.Context, _ uuid.UUID) error { return nil }
func (m *mockUserRepository) GetApprovedUserByEmail(ctx context.Context, email string) (*domain.ApprovedUser, error) {
	if m.getApprovedByEmail == nil {
		return nil, assert.AnError
	}
	return m.getApprovedByEmail(ctx, email)
}

func newTestService(t *testing.T, repo UserRepository) *Service {
	t.Helper()
	return NewService(repo, "test-secret-key", time.Hour, 4)
}

func TestService_Login_JWT_IsStringUserID(t *testing.T) {
	// Regression for the Login-issued-token bug: user_id claim must be
	// a string (matching GetUserFromToken's type assertion).
	userID := uuid.New()
	approvedID := uuid.New()
	hash, err := bcryptHash("Password123")
	require.NoError(t, err)

	repo := &mockUserRepository{
		getUserByEmail: func(_ context.Context, _ string) (*db.User, error) {
			return &db.User{
				ID:             pgtype.UUID{Bytes: userID, Valid: true},
				ApprovedUserID: pgtype.UUID{Bytes: approvedID, Valid: true},
				Email:          "x@example.com",
				PasswordHash:   hash,
				IsActive:       true,
			}, nil
		},
	}
	svc := newTestService(t, repo)

	tokenStr, err := svc.Login(context.Background(), "x@example.com", "Password123")
	require.NoError(t, err)
	require.NotEmpty(t, tokenStr)

	// Parse the token ourselves and inspect the raw claim value.
	parsed, _, err := jwt.NewParser().ParseUnverified(tokenStr, jwt.MapClaims{})
	require.NoError(t, err)
	claims := parsed.Claims.(jwt.MapClaims)
	uid, ok := claims["user_id"].(string)
	require.True(t, ok, "user_id claim must be a string, got %T", claims["user_id"])
	assert.Equal(t, userID.String(), uid)
}

func TestService_Login_InactiveUserRejected(t *testing.T) {
	// Regression for the IsActive-in-token bug: a deactivated user must
	// not be able to log in, and Login must surface ErrInvalidCredentials
	// rather than a DB error to avoid leaking account state.
	userID := uuid.New()
	hash, err := bcryptHash("Password123")
	require.NoError(t, err)

	repo := &mockUserRepository{
		getUserByEmail: func(_ context.Context, _ string) (*db.User, error) {
			return &db.User{
				ID:             pgtype.UUID{Bytes: userID, Valid: true},
				ApprovedUserID: pgtype.UUID{Bytes: uuid.New(), Valid: true},
				Email:          "x@example.com",
				PasswordHash:   hash,
				IsActive:       false,
			}, nil
		},
	}
	svc := newTestService(t, repo)

	_, err = svc.Login(context.Background(), "x@example.com", "Password123")
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrInvalidCredentials)
}

func TestService_Login_BadPassword(t *testing.T) {
	hash, err := bcryptHash("Password123")
	require.NoError(t, err)

	repo := &mockUserRepository{
		getUserByEmail: func(_ context.Context, _ string) (*db.User, error) {
			return &db.User{
				ID:             pgtype.UUID{Bytes: uuid.New(), Valid: true},
				ApprovedUserID: pgtype.UUID{Bytes: uuid.New(), Valid: true},
				Email:          "x@example.com",
				PasswordHash:   hash,
				IsActive:       true,
			}, nil
		},
	}
	svc := newTestService(t, repo)

	_, err = svc.Login(context.Background(), "x@example.com", "WrongPassword")
	assert.ErrorIs(t, err, ErrInvalidCredentials)
}

func TestService_GetUserFromToken_InactiveUserRejected(t *testing.T) {
	userID := uuid.New()
	approvedID := uuid.New()

	// Build a token signed with the test secret.
	secret := []byte("test-secret-key")
	claims := jwt.MapClaims{
		"user_id": userID.String(),
		"email":   "x@example.com",
		"exp":     time.Now().Add(time.Hour).Unix(),
	}
	signed, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(secret)
	require.NoError(t, err)

	repo := &mockUserRepository{
		getUserByID: func(_ context.Context, _ uuid.UUID) (*db.User, error) {
			return &db.User{
				ID:             pgtype.UUID{Bytes: userID, Valid: true},
				ApprovedUserID: pgtype.UUID{Bytes: approvedID, Valid: true},
				Email:          "x@example.com",
				IsActive:       false, // user deactivated
			}, nil
		},
	}
	svc := NewService(repo, "test-secret-key", time.Hour, 4)

	_, err = svc.GetUserFromToken(context.Background(), signed)
	assert.ErrorIs(t, err, ErrInvalidCredentials)
}

func TestService_Register_DuplicateUser(t *testing.T) {
	approvedID := uuid.New()
	repo := &mockUserRepository{
		getApprovedByID: func(_ context.Context, _ uuid.UUID) (*db.ApprovedUser, error) {
			return &db.ApprovedUser{ID: pgtype.UUID{Bytes: approvedID, Valid: true}}, nil
		},
		getUserByEmail: func(_ context.Context, _ string) (*db.User, error) {
			return &db.User{ID: pgtype.UUID{Bytes: uuid.New(), Valid: true}, IsActive: true}, nil
		},
	}
	svc := newTestService(t, repo)

	_, err := svc.Register(context.Background(), "dup@example.com", "Password123", approvedID.String())
	assert.ErrorIs(t, err, ErrUserAlreadyExists)
}

func TestService_CreateApprovedUser_DuplicateEmail(t *testing.T) {
	repo := &mockUserRepository{
		getApprovedByEmail: func(_ context.Context, _ string) (*domain.ApprovedUser, error) {
			return &domain.ApprovedUser{ID: uuid.New()}, nil
		},
	}
	svc := newTestService(t, repo)

	_, err := svc.CreateApprovedUser(context.Background(), "dup@example.com", "First", uuid.New())
	assert.ErrorIs(t, err, ErrApprovedEmailExists)
}

// bcryptHash wraps bcrypt.GenerateFromPassword to keep test imports tight.
func bcryptHash(pw string) (string, error) {
	h, err := bcrypt.GenerateFromPassword([]byte(pw), 4)
	return string(h), err
}
