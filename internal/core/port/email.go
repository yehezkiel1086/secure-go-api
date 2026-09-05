package port

import (
	"context"

	"github.com/yehezkiel1086/secure-go-api/internal/core/domain"
)

type EmailPublisher interface {
	PublishVerificationEmail(ctx context.Context, payload *domain.EmailPayload) error
}

type EmailSender interface {
	SendEmail(ctx context.Context, payload *domain.EmailPayload) error
}
