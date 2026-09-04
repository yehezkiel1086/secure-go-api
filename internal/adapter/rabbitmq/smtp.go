package rabbitmq

import (
	"context"
	"crypto/tls"
	"fmt"
	"log/slog"
	"net"
	"net/smtp"
	"strings"

	"github.com/yehezkiel1086/secure-go-api/internal/adapter/config"
	"github.com/yehezkiel1086/secure-go-api/internal/core/domain"
	"github.com/yehezkiel1086/secure-go-api/internal/core/port"
)

var _ port.EmailSender = (*SMTPEmailSender)(nil)

// SMTPEmailSender dispatches verification emails using standard SMTP.
type SMTPEmailSender struct {
	cfg       *config.SMTP
	logSender *LogEmailSender
}

// NewSMTPEmailSender creates a new SMTP email sender.
func NewSMTPEmailSender(cfg *config.SMTP) *SMTPEmailSender {
	return &SMTPEmailSender{
		cfg:       cfg,
		logSender: NewLogEmailSender(),
	}
}

// SendEmail delivers an HTML verification email via the configured SMTP server.
func (s *SMTPEmailSender) SendEmail(ctx context.Context, payload *domain.EmailPayload) error {
	// Fallback to console logger if SMTP is not configured
	if s.cfg == nil || strings.TrimSpace(s.cfg.Host) == "" {
		slog.Warn("⚠️ [SMTP_HOST not set] Falling back to stdout email logging. Set SMTP_* in .env to deliver real emails.")
		return s.logSender.SendEmail(ctx, payload)
	}

	addr := net.JoinHostPort(s.cfg.Host, s.cfg.Port)
	from := s.cfg.From
	if from == "" {
		from = s.cfg.User
	}
	if from == "" {
		from = "noreply@securego.com"
	}

	confirmURL := fmt.Sprintf("http://localhost:8080/api/v1/confirm-email?token=%s", payload.Token)

	subject := fmt.Sprintf("Subject: %s\r\n", payload.Subject)
	fromHeader := fmt.Sprintf("From: %s\r\n", from)
	toHeader := fmt.Sprintf("To: %s\r\n", payload.To)
	mime := "MIME-Version: 1.0\r\nContent-Type: text/html; charset=UTF-8\r\n\r\n"

	htmlBody := fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
    <meta charset="utf-8">
    <style>
        body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Helvetica, Arial, sans-serif; background-color: #f4f4f7; color: #51545e; margin: 0; padding: 0; }
        .wrapper { width: 100%%; background-color: #f4f4f7; padding: 40px 0; }
        .content { max-width: 580px; margin: 0 auto; background-color: #ffffff; border-radius: 8px; padding: 40px; box-shadow: 0 2px 4px rgba(0,0,0,0.08); }
        h1 { color: #1e293b; font-size: 22px; font-weight: 700; margin-top: 0; }
        p { font-size: 15px; line-height: 1.6; color: #475569; }
        .btn-container { text-align: center; margin: 30px 0; }
        .button { display: inline-block; background-color: #2563eb; color: #ffffff !important; padding: 14px 28px; border-radius: 6px; text-decoration: none; font-weight: 600; font-size: 16px; }
        .footer { margin-top: 30px; font-size: 12px; color: #94a3b8; border-top: 1px solid #e2e8f0; padding-top: 20px; }
        .link-text { word-break: break-all; color: #2563eb; }
    </style>
</head>
<body>
    <div class="wrapper">
        <div class="content">
            <h1>Verify Your Email Address</h1>
            <p>Thank you for registering. Please confirm your email address to activate your account:</p>
            <div class="btn-container">
                <a href="%s" class="button" target="_blank">Verify Email Address</a>
            </div>
            <p>Or copy and paste this link into your browser:<br/><a href="%s" class="link-text">%s</a></p>
            <p><strong>Note:</strong> This verification link will expire in 15 minutes.</p>
            <div class="footer">
                <p>If you did not create an account, you can safely ignore this email.</p>
            </div>
        </div>
    </div>
</body>
</html>`, confirmURL, confirmURL, confirmURL)

	msg := []byte(fromHeader + toHeader + subject + mime + htmlBody)

	err := s.send(addr, s.cfg.Host, s.cfg.Port, from, payload.To, msg)
	if err != nil {
		slog.Error("failed to deliver email via SMTP", "host", s.cfg.Host, "port", s.cfg.Port, "to", payload.To, "error", err)
		return fmt.Errorf("delivering email via SMTP: %w", err)
	}

	slog.Info("✉️ [EMAIL DELIVERED VIA SMTP]",
		"to", payload.To,
		"subject", payload.Subject,
		"smtp_host", s.cfg.Host,
		"smtp_port", s.cfg.Port,
	)

	return nil
}

func (s *SMTPEmailSender) send(addr, host, port, from, to string, msg []byte) error {
	var auth smtp.Auth
	if s.cfg.User != "" && s.cfg.Password != "" {
		auth = smtp.PlainAuth("", s.cfg.User, s.cfg.Password, host)
	}

	// Direct SSL/TLS on port 465
	if port == "465" {
		tlsConfig := &tls.Config{
			ServerName: host,
		}

		conn, err := tls.Dial("tcp", addr, tlsConfig)
		if err != nil {
			return fmt.Errorf("dialing tls on 465: %w", err)
		}
		defer conn.Close()

		client, err := smtp.NewClient(conn, host)
		if err != nil {
			return fmt.Errorf("creating smtp client: %w", err)
		}
		defer client.Close()

		if auth != nil {
			if ok, _ := client.Extension("AUTH"); ok {
				if err := client.Auth(auth); err != nil {
					return fmt.Errorf("smtp auth: %w", err)
				}
			}
		}

		if err := client.Mail(from); err != nil {
			return fmt.Errorf("smtp mail: %w", err)
		}
		if err := client.Rcpt(to); err != nil {
			return fmt.Errorf("smtp rcpt: %w", err)
		}
		w, err := client.Data()
		if err != nil {
			return fmt.Errorf("smtp data: %w", err)
		}
		if _, err := w.Write(msg); err != nil {
			return fmt.Errorf("writing email content: %w", err)
		}
		if err := w.Close(); err != nil {
			return fmt.Errorf("closing email stream: %w", err)
		}
		return client.Quit()
	}

	// Standard submission (port 587 with STARTTLS, or 1025 / 25)
	return smtp.SendMail(addr, auth, from, []string{to}, msg)
}
