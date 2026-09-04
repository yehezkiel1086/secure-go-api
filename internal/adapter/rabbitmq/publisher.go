package rabbitmq

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/yehezkiel1086/secure-go-api/internal/core/domain"
	"github.com/yehezkiel1086/secure-go-api/internal/core/port"
)

var _ port.EmailPublisher = (*Publisher)(nil)

// Publisher implements port.EmailPublisher using RabbitMQ AMQP.
type Publisher struct {
	client  *Client
	channel *amqp.Channel
	mu      sync.Mutex
}

// NewPublisher creates an EmailPublisher using a dedicated AMQP channel.
func NewPublisher(client *Client) (*Publisher, error) {
	ch, err := client.Channel()
	if err != nil {
		return nil, fmt.Errorf("opening publisher channel: %w", err)
	}

	return &Publisher{
		client:  client,
		channel: ch,
	}, nil
}

// PublishVerificationEmail serializes the email payload and dispatches a persistent message to the email exchange.
func (p *Publisher) PublishVerificationEmail(ctx context.Context, payload *domain.EmailPayload) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	// Ensure channel is active
	if p.channel == nil || p.channel.IsClosed() {
		ch, err := p.client.Channel()
		if err != nil {
			return fmt.Errorf("reopening publisher channel: %w", err)
		}
		p.channel = ch
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshaling email payload: %w", err)
	}

	msg := amqp.Publishing{
		ContentType:  "application/json",
		DeliveryMode: amqp.Persistent, // message durability
		Timestamp:    time.Now().UTC(),
		Body:         data,
	}

	err = p.channel.PublishWithContext(
		ctx,
		p.client.Exchange,
		p.client.RoutingKey,
		false, // mandatory
		false, // immediate
		msg,
	)
	if err != nil {
		return fmt.Errorf("publishing message to rabbitmq: %w", err)
	}

	slog.Info("email verification event published to rabbitmq",
		"recipient", payload.To,
		"subject", payload.Subject,
		"exchange", p.client.Exchange,
		"routingKey", p.client.RoutingKey,
	)

	return nil
}

// Close closes the publisher's dedicated AMQP channel.
func (p *Publisher) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.channel != nil && !p.channel.IsClosed() {
		return p.channel.Close()
	}
	return nil
}
