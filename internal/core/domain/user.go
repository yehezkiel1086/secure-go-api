package domain

import (
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
)

// Role constants
const (
	RoleUser  int32 = 2001
	RoleAdmin int32 = 5150
)

// Passwords and tokens are never stored in plaintext. Tokens stored as SHA-256 hashes.
type User struct {
	ID    pgtype.UUID
	Name  string
	Email string
	// bcrypt with secure cost factor
	PasswordHash string
	// 2001=User, 5150=Admin
	Role            int32
	IsEmailVerified bool
	// SHA-256 hash of one-time token
	EmailVerifyTokenHash      pgtype.Text
	EmailVerifyTokenExpiresAt pgtype.Timestamptz
	// SHA-256 hash of one-time token
	PasswordResetTokenHash      pgtype.Text
	PasswordResetTokenExpiresAt pgtype.Timestamptz
	CreatedAt                   pgtype.Timestamptz
	UpdatedAt                   pgtype.Timestamptz
}

// RegisterUserRequest contains input for user registration.
type RegisterUserRequest struct {
	Name     string `json:"name" binding:"required,min=2,max=100"`
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=8,max=72"`
}

// UpdateUserNameRequest contains input for updating user display name.
type UpdateUserNameRequest struct {
	Name string `json:"name" binding:"required,min=2,max=100"`
}

// UserResponse is the client-facing representation of a user without sensitive hashes.
type UserResponse struct {
	ID              string `json:"id"`
	Name            string `json:"name"`
	Email           string `json:"email"`
	Role            int32  `json:"role"`
	IsEmailVerified bool   `json:"is_email_verified"`
	CreatedAt       string `json:"created_at,omitempty"`
	UpdatedAt       string `json:"updated_at,omitempty"`
}

// PaginatedUsersResponse contains paginated user list and metadata.
type PaginatedUsersResponse struct {
	Users      []*UserResponse `json:"users"`
	Total      int64           `json:"total"`
	Page       int32           `json:"page"`
	PageSize   int32           `json:"page_size"`
	TotalPages int32           `json:"total_pages"`
}

// ToResponse maps the internal User domain entity to a safe UserResponse DTO.
func (u *User) ToResponse() *UserResponse {
	var idStr string
	if u.ID.Valid {
		b := u.ID.Bytes
		idStr = fmt.Sprintf("%02x%02x%02x%02x-%02x%02x-%02x%02x-%02x%02x-%02x%02x%02x%02x%02x%02x",
			b[0], b[1], b[2], b[3],
			b[4], b[5],
			b[6], b[7],
			b[8], b[9],
			b[10], b[11], b[12], b[13], b[14], b[15],
		)
	}

	var createdAtStr, updatedAtStr string
	if u.CreatedAt.Valid {
		createdAtStr = u.CreatedAt.Time.Format(time.RFC3339)
	}
	if u.UpdatedAt.Valid {
		updatedAtStr = u.UpdatedAt.Time.Format(time.RFC3339)
	}

	return &UserResponse{
		ID:              idStr,
		Name:            u.Name,
		Email:           u.Email,
		Role:            u.Role,
		IsEmailVerified: u.IsEmailVerified,
		CreatedAt:       createdAtStr,
		UpdatedAt:       updatedAtStr,
	}
}
