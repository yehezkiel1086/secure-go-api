package port

import (
	"context"

	"github.com/yehezkiel1086/secure-go-api/internal/core/domain"
)

// EmailPublisher is a driven port for publishing email jobs to an asynchronous message broker (RabbitMQ).
type EmailPublisher interface {
	// PublishVerificationEmail enqueues an email verification message.
	PublishVerificationEmail(ctx context.Context, payload *domain.EmailPayload) error
}

// EmailSender is a driven port for directly delivering emails (via SMTP, API provider, or console worker).
type EmailSender interface {
	// SendEmail sends the email message to the specified recipient.
	SendEmail(ctx context.Context, payload *domain.EmailPayload) error
}
