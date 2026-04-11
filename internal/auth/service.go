// Package auth provides authentication service.
package auth

import (
	"context"
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	db "github.com/your-org/go-backend-template/internal/db/sqlc"
	"github.com/your-org/go-backend-template/internal/domain"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrUserNotFound       = errors.New("user not found")
)

// Service provides authentication business logic
type Service struct {
	repo      *Repository
	jwtSecret []byte
	jwtExpiry time.Duration
}

// NewService creates a new auth service
func NewService(repo *Repository, jwtSecret string, jwtExpiry time.Duration) *Service {
	return &Service{
		repo:      repo,
		jwtSecret: []byte(jwtSecret),
		jwtExpiry: jwtExpiry,
	}
}

// Register registers a new user
func (s *Service) Register(ctx context.Context, email, password, approvedID string) (string, error) {
	// Verify approved user exists
	approvedUUID, err := uuid.Parse(approvedID)
	if err != nil {
		return "", ErrInvalidCredentials
	}

	approvedUser, err := s.repo.GetApprovedUserByID(ctx, approvedUUID)
	if err != nil {
		return "", ErrUserNotFound
	}
	_ = approvedUser

	// Check if user already exists
	existingUser, err := s.repo.GetUserByEmail(ctx, email)
	if err == nil && existingUser != nil {
		return "", errors.New("user already exists")
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}

	// Create user
	user, err := s.repo.CreateUser(ctx, db.CreateUserParams{
		ApprovedUserID: uuidToPgtype(approvedUUID),
		Email:          email,
		PasswordHash:   string(hashedPassword),
		IsActive:       true,
	})
	if err != nil {
		return "", err
	}

	// Assign default user role
	userRole, err := s.repo.GetRoleByName(ctx, "user")
	if err == nil && userRole != nil {
		_ = s.repo.AssignRoleToUser(ctx, pgtypeToUuid(user.ID), pgtypeToUuid(userRole.ID))
	}

	// Generate JWT token
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": user.ID.Bytes,
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
	// Get user by email
	user, err := s.repo.GetUserByEmail(ctx, email)
	if err != nil {
		return "", ErrInvalidCredentials
	}

	// Check password
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return "", ErrInvalidCredentials
	}

	// Get user roles
	roles, err := s.repo.GetUserRoles(ctx, pgtypeToUuid(user.ID))
	if err != nil {
		roles = []db.Role{}
	}

	// Get approved user
	approvedUser, err := s.repo.GetApprovedUserByID(ctx, pgtypeToUuid(user.ApprovedUserID))
	if err != nil {
		approvedUser = nil
	}

	// Generate JWT token
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": user.ID.Bytes,
		"email":   user.Email,
		"exp":     time.Now().Add(s.jwtExpiry).Unix(),
	})

	tokenString, err := token.SignedString(s.jwtSecret)
	if err != nil {
		return "", err
	}

	_ = approvedUser
	_ = s.repo.ToDomainUser(*user, approvedUser, roles)

	return tokenString, nil
}

// GetUserFromToken validates a JWT token and returns the user
func (s *Service) GetUserFromToken(ctx context.Context, tokenString string) (*domain.User, error) {
	// Parse token
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, ErrInvalidCredentials
		}
		return s.jwtSecret, nil
	})

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

	// Get user from database
	user, err := s.repo.GetUserByID(ctx, id)
	if err != nil {
		return nil, ErrUserNotFound
	}

	// Get user roles
	roles, err := s.repo.GetUserRoles(ctx, pgtypeToUuid(user.ID))
	if err != nil {
		roles = []db.Role{}
	}

	// Get approved user
	approvedUser, err := s.repo.GetApprovedUserByID(ctx, pgtypeToUuid(user.ApprovedUserID))
	if err != nil {
		approvedUser = nil
	}

	return s.repo.ToDomainUser(*user, approvedUser, roles), nil
}
