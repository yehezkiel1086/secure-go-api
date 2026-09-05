package domain

import "time"

type EmailPayload struct {
	To        string    `json:"to"`
	Subject   string    `json:"subject"`
	Body      string    `json:"body"`
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expires_at"`
}

type ResendVerificationRequest struct {
	Email string `json:"email" binding:"required,email"`
}
