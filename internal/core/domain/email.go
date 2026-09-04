package domain

import "time"

// EmailPayload represents the message dispatched to RabbitMQ for asynchronous email sending.
type EmailPayload struct {
	To        string    `json:"to"`
	Subject   string    `json:"subject"`
	Body      string    `json:"body"`
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expires_at"`
}

// ResendVerificationRequest represents the request body for resending verification emails.
type ResendVerificationRequest struct {
	Email string `json:"email" binding:"required,email"`
}
