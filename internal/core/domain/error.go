package domain

import "errors"

var (
	// ErrNotFound indicates the requested entity does not exist.
	ErrNotFound = errors.New("record not found")

	// ErrUserNotFound indicates the requested user was not found.
	ErrUserNotFound = errors.New("user not found")

	// ErrEmailAlreadyExists indicates registration with an already registered email.
	ErrEmailAlreadyExists = errors.New("email already registered")

	// ErrInvalidCredentials indicates authentication failure due to bad email or password.
	ErrInvalidCredentials = errors.New("invalid email or password")

	// ErrTokenReuse indicates a revoked or reused refresh token was detected.
	ErrTokenReuse = errors.New("refresh token reuse detected")

	// ErrTokenExpired indicates a verification or reset token has passed its expiration time.
	ErrTokenExpired = errors.New("token has expired")

	// ErrInvalidToken indicates an invalid or malformed token.
	ErrInvalidToken = errors.New("invalid token")

	// ErrInternal indicates an unexpected internal system or operational error.
	ErrInternal = errors.New("internal server error")
)
