// Package auth provides authentication service.
package auth

import (
	"context"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/your-org/go-backend-template/internal/domain"
	"golang.org/x/crypto/bcrypt"
)

// Service provides authentication business logic
type Service struct {
	repo       UserRepository
	jwtSecret  []byte
	jwtExpiry  time.Duration
	bcryptCost int
}

// NewService creates a new auth service
func NewService(repo UserRepository, jwtSecret string, jwtExpiry time.Duration, bcryptCost int) *Service {
	return &Service{
		repo:       repo,
		jwtSecret:  []byte(jwtSecret),
		jwtExpiry:  jwtExpiry,
		bcryptCost: bcryptCost,
	}
}

// Register registers a new user
func (s *Service) Register(ctx context.Context, email, password, approvedID string) (string, error) {
	approvedUUID, err := uuid.Parse(approvedID)
	if err != nil {
		return "", ErrInvalidCredentials
	}

	exists, err := s.repo.ApprovedUserExists(ctx, approvedUUID)
	if err != nil || !exists {
		return "", ErrUserNotFound
	}

	if existing, err := s.repo.GetUserByEmail(ctx, email); err == nil && existing != nil {
		return "", ErrUserAlreadyExists
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), s.bcryptCost)
	if err != nil {
		return "", err
	}

	user, err := s.repo.CreateUser(ctx, UserCreateInput{
		ApprovedUserID: approvedUUID,
		Email:          email,
		PasswordHash:   string(hashedPassword),
		IsActive:       true,
	})
	if err != nil {
		return "", err
	}

	// Assign default user role. The role is seeded by the migration so a
	// missing role here is treated as a no-op rather than a hard error.
	if userRole, err := s.repo.GetRoleByName(ctx, "user"); err == nil && userRole != nil {
		if assignErr := s.repo.AssignRoleToUser(ctx, user.ID, userRole.ID); assignErr != nil {
			return "", assignErr
		}
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": user.ID.String(),
		"email":   user.Email,
		"exp":     time.Now().Add(s.jwtExpiry).Unix(),
	})

	tokenString, err := token.SignedString(s.jwtSecret)
	if err != nil {
		return "", err
	}

	return tokenString, nil
}

// Login authenticates a user and returns a JWT token
func (s *Service) Login(ctx context.Context, email, password string) (string, error) {
	user, err := s.repo.GetUserByEmail(ctx, email)
	if err != nil {
		return "", ErrInvalidCredentials
	}

	if !user.IsActive {
		return "", ErrInvalidCredentials
	}

	if cmpErr := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); cmpErr != nil {
		return "", ErrInvalidCredentials
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": user.ID.String(),
		"email":   user.Email,
		"exp":     time.Now().Add(s.jwtExpiry).Unix(),
	})

	tokenString, err := token.SignedString(s.jwtSecret)
	if err != nil {
		return "", err
	}

	return tokenString, nil
}

// GetUserFromToken validates a JWT token and returns the user
func (s *Service) GetUserFromToken(ctx context.Context, tokenString string) (*domain.User, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (any, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, ErrInvalidCredentials
		}
		return s.jwtSecret, nil
	},
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
		jwt.WithExpirationRequired(),
	)
	if err != nil {
		return nil, ErrInvalidCredentials
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok || !token.Valid {
		return nil, ErrInvalidCredentials
	}

	userID, ok := claims["user_id"].(string)
	if !ok {
		return nil, ErrInvalidCredentials
	}

	id, err := uuid.Parse(userID)
	if err != nil {
		return nil, ErrInvalidCredentials
	}

	user, err := s.repo.GetUserByID(ctx, id)
	if err != nil {
		return nil, ErrUserNotFound
	}

	if !user.IsActive {
		return nil, ErrInvalidCredentials
	}

	roles, err := s.repo.GetUserRoles(ctx, user.ID)
	if err != nil {
		roles = []RoleRow{}
	}

	approvedUser, err := s.repo.GetApprovedUserByID(ctx, user.ApprovedUserID)
	if err != nil {
		approvedUser = nil
	}

	return ToDomainUser(user, approvedUser, roles), nil
}

// ListApprovedUsers lists all approved users (admin only)
func (s *Service) ListApprovedUsers(ctx context.Context) ([]*domain.ApprovedUser, error) {
	rows, err := s.repo.ListApprovedUsers(ctx)
	if err != nil {
		return nil, err
	}
	return approvedUsersToDomain(rows), nil
}

// CreateApprovedUser creates a new approved user (admin only)
func (s *Service) CreateApprovedUser(ctx context.Context, email, firstName string, createdBy uuid.UUID) (*domain.ApprovedUser, error) {
	if existing, err := s.repo.GetApprovedUserByEmail(ctx, email); err == nil && existing != nil {
		return nil, ErrApprovedEmailExists
	}

	row, err := s.repo.CreateApprovedUser(ctx, ApprovedUserCreateInput{
		Email:     email,
		FirstName: firstName,
		CreatedBy: createdBy,
	})
	if err != nil {
		return nil, err
	}
	return approvedUserToDomain(row), nil
}

// BulkCreateApprovedUsers creates multiple approved users (admin only)
func (s *Service) BulkCreateApprovedUsers(ctx context.Context, emails, firstNames []string, createdBy uuid.UUID) ([]*domain.ApprovedUser, error) {
	if len(emails) != len(firstNames) {
		return nil, ErrInvalidInput
	}
	rows, err := s.repo.BulkCreateApprovedUsers(ctx, BulkApprovedUserInput{
		Emails:     emails,
		FirstNames: firstNames,
		CreatedBy:  createdBy,
	})
	if err != nil {
		return nil, err
	}
	return approvedUsersToDomain(rows), nil
}

// DeleteApprovedUser deletes an approved user (admin only)
func (s *Service) DeleteApprovedUser(ctx context.Context, id uuid.UUID) error {
	return s.repo.DeleteApprovedUser(ctx, id)
}

// approvedUserToDomain wraps a row into the cross-feature domain type.
func approvedUserToDomain(r *ApprovedUserRow) *domain.ApprovedUser {
	if r == nil {
		return nil
	}
	return &domain.ApprovedUser{
		ID:        r.ID,
		Email:     r.Email,
		FirstName: r.FirstName,
		CreatedBy: r.CreatedBy,
		CreatedAt: r.CreatedAt,
		UpdatedAt: r.UpdatedAt,
	}
}

func approvedUsersToDomain(rs []*ApprovedUserRow) []*domain.ApprovedUser {
	out := make([]*domain.ApprovedUser, len(rs))
	for i, r := range rs {
		out[i] = approvedUserToDomain(r)
	}
	return out
}
