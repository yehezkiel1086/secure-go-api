package domain

import "errors"

var (
	ErrNotFound             = errors.New("record not found")
	ErrUserNotFound         = errors.New("user not found")
	ErrEmailAlreadyExists   = errors.New("email already registered")
	ErrInvalidCredentials   = errors.New("invalid email or password")
	ErrTokenReuse           = errors.New("refresh token reuse detected")
	ErrTokenExpired         = errors.New("token has expired")
	ErrInvalidToken         = errors.New("invalid token")
	ErrInternal             = errors.New("internal server error")
	ErrEmailNotVerified     = errors.New("email address is not verified")
	ErrInvalidEmailFormat   = errors.New("invalid email address format")
	ErrEmailAlreadyVerified = errors.New("email is already verified")
	ErrJobNotFound          = errors.New("job listing not found")
	ErrInvalidSalaryRange   = errors.New("minimum salary cannot exceed maximum salary")
)
