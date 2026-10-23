// Package service holds business logic. It depends on the repository layer
// for persistence and never touches the database directly.
package service

import (
	"context"
	"errors"
	"strings"

	"github.com/KhanbalaRashidov/ai-travel-platform/pkg/auth"
	"github.com/KhanbalaRashidov/ai-travel-platform/pkg/models"
	"github.com/KhanbalaRashidov/ai-travel-platform/services/gateway/repository"
)

// Common auth errors surfaced to handlers.
var (
	ErrInvalidCredentials = errors.New("invalid email or password")
	ErrEmailTaken         = errors.New("email already registered")
	ErrValidation         = errors.New("invalid input")
)

// AuthService implements registration/login and issues JWTs.
type AuthService struct {
	users *repository.UserRepository
	jwt   *auth.JWTManager
}

// NewAuthService wires the auth service.
func NewAuthService(users *repository.UserRepository, jwt *auth.JWTManager) *AuthService {
	return &AuthService{users: users, jwt: jwt}
}

// Register creates a new user in the tenant and returns the user + access token.
func (s *AuthService) Register(ctx context.Context, tenantID, email, password, fullName string) (models.User, string, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	if email == "" || len(password) < 8 {
		return models.User{}, "", ErrValidation
	}
	if _, err := s.users.GetByEmail(ctx, tenantID, email); err == nil {
		return models.User{}, "", ErrEmailTaken
	} else if !errors.Is(err, repository.ErrNotFound) {
		return models.User{}, "", err
	}

	hash, err := auth.HashPassword(password)
	if err != nil {
		return models.User{}, "", err
	}
	user, err := s.users.Create(ctx, models.User{
		TenantID:     tenantID,
		Email:        email,
		PasswordHash: hash,
		FullName:     strings.TrimSpace(fullName),
	})
	if err != nil {
		return models.User{}, "", err
	}
	token, err := s.jwt.Generate(user)
	return user, token, err
}

// Login verifies credentials and returns the user + access token.
func (s *AuthService) Login(ctx context.Context, tenantID, email, password string) (models.User, string, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	user, err := s.users.GetByEmail(ctx, tenantID, email)
	if errors.Is(err, repository.ErrNotFound) {
		return models.User{}, "", ErrInvalidCredentials
	} else if err != nil {
		return models.User{}, "", err
	}
	if !auth.CheckPassword(user.PasswordHash, password) {
		return models.User{}, "", ErrInvalidCredentials
	}
	token, err := s.jwt.Generate(user)
	return user, token, err
}

// Me returns the current user within a tenant.
func (s *AuthService) Me(ctx context.Context, tenantID, userID string) (models.User, error) {
	return s.users.GetByID(ctx, tenantID, userID)
}
