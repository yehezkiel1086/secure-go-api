package util_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/yehezkiel1086/secure-go-api/internal/core/domain"
	"github.com/yehezkiel1086/secure-go-api/internal/core/util"
)

func TestValidateEmail_Valid(t *testing.T) {
	validEmails := []string{
		"user@example.com",
		"alice.smith@sub.domain.org",
		"user+tag@domain.co.uk",
		"first.last@company.io",
		"123456@numbers.com",
		"customer_service@store.tech",
	}

	for _, email := range validEmails {
		t.Run(email, func(t *testing.T) {
			if err := util.ValidateEmail(email); err != nil {
				t.Fatalf("expected email %q to be valid, got %v", email, err)
			}
		})
	}
}

func TestValidateEmail_Invalid(t *testing.T) {
	invalidEmails := []struct {
		name  string
		email string
	}{
		{"empty string", ""},
		{"only whitespace", "   "},
		{"missing at symbol", "userexample.com"},
		{"missing domain", "user@"},
		{"missing local part", "@example.com"},
		{"multiple at symbols", "user@sub@example.com"},
		{"consecutive dots in domain", "user@domain..com"},
		{"single letter tld", "user@domain.c"},
		{"newline header injection", "user@example.com\r\nBcc: victim@example.com"},
		{"tab character", "user\t@example.com"},
		{"display name smuggling", "John Doe <john@example.com>"},
		{"exceeds max length", strings.Repeat("a", 245) + "@example.com"},
		{"local part exceeds 64 chars", strings.Repeat("a", 65) + "@example.com"},
	}

	for _, tt := range invalidEmails {
		t.Run(tt.name, func(t *testing.T) {
			err := util.ValidateEmail(tt.email)
			if !errors.Is(err, domain.ErrInvalidEmailFormat) {
				t.Fatalf("expected ErrInvalidEmailFormat for %q, got %v", tt.email, err)
			}
		})
	}
}
