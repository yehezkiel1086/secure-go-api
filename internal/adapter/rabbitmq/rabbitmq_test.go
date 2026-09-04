package rabbitmq_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/yehezkiel1086/secure-go-api/internal/adapter/config"
	"github.com/yehezkiel1086/secure-go-api/internal/adapter/rabbitmq"
	"github.com/yehezkiel1086/secure-go-api/internal/core/domain"
)

type testMockSender struct {
	mu       sync.Mutex
	received []*domain.EmailPayload
	notify   chan struct{}
}

func (s *testMockSender) SendEmail(_ context.Context, payload *domain.EmailPayload) error {
	s.mu.Lock()
	s.received = append(s.received, payload)
	s.mu.Unlock()

	select {
	case s.notify <- struct{}{}:
	default:
	}
	return nil
}

func TestRabbitMQ_PublishAndConsume(t *testing.T) {
	cfg := &config.Rabbitmq{
		Host:     "127.0.0.1",
		Port:     "5672",
		User:     "rabbitmq",
		Password: "admin",
	}

	client, err := rabbitmq.NewClient(cfg)
	if err != nil {
		t.Skipf("skipping live rabbitmq test (cannot connect): %v", err)
	}
	defer client.Close()

	// Create an isolated test queue to avoid competing with running dev servers
	testQueueName := "test.email.verification.queue"
	testRoutingKey := "test.email.verification"

	setupCh, err := client.Channel()
	if err != nil {
		t.Fatalf("failed to open setup channel: %v", err)
	}

	q, err := setupCh.QueueDeclare(
		testQueueName,
		false, // durable
		true,  // auto-delete
		false, // exclusive
		false, // no-wait
		nil,
	)
	if err != nil {
		_ = setupCh.Close()
		t.Fatalf("failed to declare test queue: %v", err)
	}

	err = setupCh.QueueBind(q.Name, testRoutingKey, client.Exchange, false, nil)
	_ = setupCh.Close()
	if err != nil {
		t.Fatalf("failed to bind test queue: %v", err)
	}

	testClient := &rabbitmq.Client{
		Conn:       client.Conn,
		Exchange:   client.Exchange,
		Queue:      q.Name,
		RoutingKey: testRoutingKey,
	}

	publisher, err := rabbitmq.NewPublisher(testClient)
	if err != nil {
		t.Fatalf("failed to create publisher: %v", err)
	}
	defer publisher.Close()

	sender := &testMockSender{notify: make(chan struct{}, 1)}
	consumer := rabbitmq.NewConsumer(testClient, sender)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start consumer in background
	go func() {
		_ = consumer.Start(ctx)
	}()

	// Wait briefly for consumer channel to establish
	time.Sleep(100 * time.Millisecond)

	expectedToken := "test-crypto-random-token-12345"
	payload := &domain.EmailPayload{
		To:        "integration@example.com",
		Subject:   "Test Email Verification",
		Body:      "Please click the verification link",
		Token:     expectedToken,
		ExpiresAt: time.Now().Add(15 * time.Minute),
	}

	err = publisher.PublishVerificationEmail(ctx, payload)
	if err != nil {
		t.Fatalf("failed to publish verification email: %v", err)
	}

	// Wait for consumer to process message
	select {
	case <-sender.notify:
		sender.mu.Lock()
		defer sender.mu.Unlock()
		if len(sender.received) == 0 {
			t.Fatal("expected message to be received by consumer")
		}
		got := sender.received[len(sender.received)-1]
		if got.To != "integration@example.com" {
			t.Errorf("expected to integration@example.com, got %s", got.To)
		}
		if got.Token != expectedToken {
			t.Errorf("expected token %s, got %s", expectedToken, got.Token)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for rabbitmq consumer to receive message")
	}
}
