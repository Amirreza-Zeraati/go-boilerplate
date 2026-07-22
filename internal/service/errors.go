package service

import "errors"

// Domain-level errors returned by services. Handlers map these to HTTP status
// codes without importing the repository or infrastructure layers.
var (
	ErrInvalidCredentials = errors.New("invalid email or password")
	ErrEmailTaken         = errors.New("email already registered")
	ErrUserNotFound       = errors.New("user not found")
)
