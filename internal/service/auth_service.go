// Package service holds business logic. It sits between the HTTP handlers and
// the repositories: handlers translate HTTP <-> service calls, services own the
// rules, repositories own persistence. Services never import gin or net/http.
package service

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"github.com/Amirreza-Zeraati/go-boilerplate/internal/models"
	"github.com/Amirreza-Zeraati/go-boilerplate/internal/repository"
)

// AuthService handles registration and credential verification. It knows
// nothing about sessions or cookies — that's the handler's job.
type AuthService struct {
	users repository.UserRepository
}

// NewAuthService constructs the auth service.
func NewAuthService(users repository.UserRepository) *AuthService {
	return &AuthService{users: users}
}

// Register creates a new user with a bcrypt-hashed password and the default
// "user" role. Returns ErrEmailTaken if the email already exists.
func (s *AuthService) Register(ctx context.Context, email, password string) (*models.User, error) {
	if _, err := s.users.GetByEmail(ctx, email); err == nil {
		return nil, ErrEmailTaken
	} else if !errors.Is(err, repository.ErrNotFound) {
		return nil, err
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	user := &models.User{
		Email:        email,
		PasswordHash: string(hash),
		Role:         "user",
	}
	if err := s.users.Create(ctx, user); err != nil {
		return nil, err
	}
	return user, nil
}

// Authenticate verifies an email/password pair and returns the user on success.
// It always returns ErrInvalidCredentials for both "no such user" and "wrong
// password" so callers can't distinguish the two (avoids user enumeration).
func (s *AuthService) Authenticate(ctx context.Context, email, password string) (*models.User, error) {
	user, err := s.users.GetByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			// Run a dummy compare to keep timing roughly constant.
			_ = bcrypt.CompareHashAndPassword([]byte("$2a$10$invalidinvalidinvalidinvalidinvalidinvalidinvalidin"), []byte(password))
			return nil, ErrInvalidCredentials
		}
		return nil, err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return nil, ErrInvalidCredentials
	}
	return user, nil
}

// GetByID returns a user by ID, mapping the repo's not-found to ErrUserNotFound.
func (s *AuthService) GetByID(ctx context.Context, id uuid.UUID) (*models.User, error) {
	user, err := s.users.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	return user, nil
}
