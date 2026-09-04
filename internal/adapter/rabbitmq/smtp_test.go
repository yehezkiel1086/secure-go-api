package rabbitmq_test

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/yehezkiel1086/secure-go-api/internal/adapter/config"
	"github.com/yehezkiel1086/secure-go-api/internal/adapter/rabbitmq"
	"github.com/yehezkiel1086/secure-go-api/internal/core/domain"
)

func TestSMTPEmailSender_FallbackToLogWhenNoHost(t *testing.T) {
	sender := rabbitmq.NewSMTPEmailSender(&config.SMTP{Host: ""})
	payload := &domain.EmailPayload{
		To:        "fallback@example.com",
		Subject:   "Test Fallback",
		Token:     "dummytoken123",
		ExpiresAt: time.Now().Add(15 * time.Minute),
	}

	err := sender.SendEmail(context.Background(), payload)
	if err != nil {
		t.Fatalf("expected fallback to log sender without error, got %v", err)
	}
}

func TestSMTPEmailSender_MailpitDelivery(t *testing.T) {
	smtpCfg := &config.SMTP{
		Host: "127.0.0.1",
		Port: "1025",
		From: "test@securego.com",
	}

	sender := rabbitmq.NewSMTPEmailSender(smtpCfg)
	uniqueToken := "mailpit-test-token-778899"
	recipient := "verify-test@example.com"

	payload := &domain.EmailPayload{
		To:        recipient,
		Subject:   "Verify Your Email - Mailpit Test",
		Token:     uniqueToken,
		ExpiresAt: time.Now().Add(15 * time.Minute),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := sender.SendEmail(ctx, payload)
	if err != nil {
		t.Skipf("skipping live mailpit test (cannot connect to 127.0.0.1:1025): %v", err)
	}

	// Verify message landed in Mailpit via its HTTP API
	resp, err := http.Get("http://127.0.0.1:8025/api/v1/messages")
	if err != nil {
		t.Skipf("cannot reach mailpit http api: %v", err)
	}
	defer resp.Body.Close()

	var result struct {
		Messages []struct {
			Subject string `json:"Subject"`
			To      []struct {
				Address string `json:"Address"`
			} `json:"To"`
			Snippet string `json:"Snippet"`
		} `json:"messages"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode mailpit messages: %v", err)
	}

	var found bool
	for _, m := range result.Messages {
		if m.Subject == "Verify Your Email - Mailpit Test" {
			for _, to := range m.To {
				if to.Address == recipient {
					found = true
					break
				}
			}
		}
	}

	if !found {
		t.Fatalf("expected message with subject %q to be found in Mailpit", payload.Subject)
	}
}
