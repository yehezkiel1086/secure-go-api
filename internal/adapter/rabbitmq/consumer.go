package rabbitmq

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/yehezkiel1086/secure-go-api/internal/core/domain"
	"github.com/yehezkiel1086/secure-go-api/internal/core/port"
)

// LogEmailSender is a default EmailSender implementation that logs outgoing emails.
type LogEmailSender struct{}

func NewLogEmailSender() *LogEmailSender {
	return &LogEmailSender{}
}

func (s *LogEmailSender) SendEmail(_ context.Context, payload *domain.EmailPayload) error {
	slog.Info("📧 [EMAIL DISPATCHED VIA WORKER]",
		"to", payload.To,
		"subject", payload.Subject,
		"token", payload.Token,
		"expires_at", payload.ExpiresAt,
		"body", payload.Body,
	)
	return nil
}

// Consumer is a background worker that consumes messages from RabbitMQ and sends emails.
type Consumer struct {
	client      *Client
	emailSender port.EmailSender
}

// NewConsumer creates a new email queue consumer.
func NewConsumer(client *Client, sender port.EmailSender) *Consumer {
	if sender == nil {
		sender = NewLogEmailSender()
	}
	return &Consumer{
		client:      client,
		emailSender: sender,
	}
}

// Start runs the message consumer loop on a dedicated channel until the context is canceled.
func (c *Consumer) Start(ctx context.Context) error {
	ch, err := c.client.Channel()
	if err != nil {
		return fmt.Errorf("opening consumer channel: %w", err)
	}
	defer ch.Close()

	// Set prefetch count for fair dispatch
	if err := ch.Qos(10, 0, false); err != nil {
		return fmt.Errorf("setting rabbitmq qos: %w", err)
	}

	msgs, err := ch.Consume(
		c.client.Queue,
		"",    // consumer tag (empty generates unique tag)
		false, // auto-ack (false for manual acknowledgment)
		false, // exclusive
		false, // no-local
		false, // no-wait
		nil,   // args
	)
	if err != nil {
		return fmt.Errorf("initiating rabbitmq consumer: %w", err)
	}

	slog.Info("rabbitmq email worker started, waiting for messages...", "queue", c.client.Queue)

	for {
		select {
		case <-ctx.Done():
			slog.Info("shutting down rabbitmq email worker...")
			return nil
		case msg, ok := <-msgs:
			if !ok {
				slog.Warn("rabbitmq message channel closed")
				return nil
			}

			c.handleMessage(ctx, msg)
		}
	}
}

func (c *Consumer) handleMessage(ctx context.Context, msg amqp.Delivery) {
	var payload domain.EmailPayload
	if err := json.Unmarshal(msg.Body, &payload); err != nil {
		slog.Error("failed to unmarshal email payload, rejecting message", "error", err)
		_ = msg.Nack(false, false) // discard poisoned message
		return
	}

	if err := c.emailSender.SendEmail(ctx, &payload); err != nil {
		slog.Error("failed to send email, requeuing message", "recipient", payload.To, "error", err)
		_ = msg.Nack(false, true) // requeue for retry
		return
	}

	if err := msg.Ack(false); err != nil {
		slog.Error("failed to ack rabbitmq message", "error", err)
	}
}
