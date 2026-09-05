package rabbitmq

import (
	"fmt"
	"log/slog"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/yehezkiel1086/secure-go-api/internal/adapter/config"
)

const (
	EmailExchange         = "email.exchange"
	EmailExchangeType     = "direct"
	EmailVerifyQueue      = "email.verification.queue"
	EmailVerifyRoutingKey = "email.verification"
)

// Client manages the underlying RabbitMQ connection and declared topology settings.
type Client struct {
	Conn       *amqp.Connection
	Exchange   string
	Queue      string
	RoutingKey string
}

func NewClient(cfg *config.Rabbitmq) (*Client, error) {
	url := cfg.AMQPURL()
	conn, err := amqp.Dial(url)
	if err != nil {
		return nil, fmt.Errorf("connecting to rabbitmq: %w", err)
	}

	ch, err := conn.Channel()
	if err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("opening setup channel: %w", err)
	}
	defer ch.Close()

	err = ch.ExchangeDeclare(
		EmailExchange,
		EmailExchangeType,
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("declaring exchange %q: %w", EmailExchange, err)
	}

	q, err := ch.QueueDeclare(
		EmailVerifyQueue,
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("declaring queue %q: %w", EmailVerifyQueue, err)
	}

	err = ch.QueueBind(
		q.Name,
		EmailVerifyRoutingKey,
		EmailExchange,
		false,
		nil,
	)
	if err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("binding queue %q to exchange %q: %w", q.Name, EmailExchange, err)
	}

	slog.Info("rabbitmq topology initialized successfully",
		"exchange", EmailExchange,
		"queue", q.Name,
		"routingKey", EmailVerifyRoutingKey,
	)

	return &Client{
		Conn:       conn,
		Exchange:   EmailExchange,
		Queue:      q.Name,
		RoutingKey: EmailVerifyRoutingKey,
	}, nil
}

// Channel opens and returns a new AMQP channel for dedicated use by a worker or publisher.
func (c *Client) Channel() (*amqp.Channel, error) {
	return c.Conn.Channel()
}

// Close terminates the RabbitMQ connection.
func (c *Client) Close() error {
	if c.Conn != nil && !c.Conn.IsClosed() {
		return c.Conn.Close()
	}
	return nil
}
