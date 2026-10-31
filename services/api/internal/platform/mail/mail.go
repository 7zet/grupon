// Package mail sends transactional email. Callers depend only on Sender.
package mail

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"mime"
	"net/smtp"
	"strings"
	"time"
)

// Message is a plain-text email.
type Message struct {
	To      string
	Subject string
	Body    string
}

// Sender delivers a message.
type Sender interface {
	Send(ctx context.Context, message Message) error
}

// SMTPSender delivers mail through an SMTP relay without authentication. It is meant for the local
// mailpit container and for relays that accept unauthenticated submission inside the network.
type SMTPSender struct {
	addr string
	from string
}

// NewSMTPSender returns a sender for host:port with the given envelope and header sender.
func NewSMTPSender(host string, port int, from string) *SMTPSender {
	return &SMTPSender{addr: fmt.Sprintf("%s:%d", host, port), from: from}
}

// Send renders the message as UTF-8 text and submits it to the relay.
func (s *SMTPSender) Send(ctx context.Context, message Message) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	raw, err := render(s.from, message)
	if err != nil {
		return err
	}
	// net/smtp has no context support, so the dial is bounded by a deadline of its own.
	done := make(chan error, 1)
	go func() {
		done <- smtp.SendMail(s.addr, nil, s.from, []string{message.To}, raw)
	}()
	select {
	case err := <-done:
		if err != nil {
			return fmt.Errorf("smtp send: %w", err)
		}
		return nil
	case <-time.After(30 * time.Second):
		return fmt.Errorf("smtp send: timed out")
	case <-ctx.Done():
		return ctx.Err()
	}
}

// LogSender is used when no SMTP relay is configured. It records that a message was produced but
// never logs the body, because bodies carry single-use tokens.
type LogSender struct {
	logger *slog.Logger
}

// NewLogSender returns a sender that only logs metadata.
func NewLogSender(logger *slog.Logger) *LogSender {
	return &LogSender{logger: logger}
}

// Send logs the subject and recipient.
func (s *LogSender) Send(ctx context.Context, message Message) error {
	s.logger.InfoContext(ctx, "email not delivered: SMTP is not configured",
		slog.String("subject", message.Subject),
		slog.String("recipient", maskAddress(message.To)),
	)
	return nil
}

func render(from string, message Message) ([]byte, error) {
	if strings.ContainsAny(message.To, "\r\n") || strings.ContainsAny(from, "\r\n") {
		return nil, fmt.Errorf("mail address contains line breaks")
	}
	var buf bytes.Buffer
	fmt.Fprintf(&buf, "From: %s\r\n", from)
	fmt.Fprintf(&buf, "To: %s\r\n", message.To)
	fmt.Fprintf(&buf, "Subject: %s\r\n", mime.QEncoding.Encode("utf-8", message.Subject))
	fmt.Fprintf(&buf, "Date: %s\r\n", time.Now().UTC().Format(time.RFC1123Z))
	buf.WriteString("MIME-Version: 1.0\r\n")
	buf.WriteString("Content-Type: text/plain; charset=utf-8\r\n")
	buf.WriteString("Content-Transfer-Encoding: 8bit\r\n\r\n")
	buf.WriteString(strings.ReplaceAll(message.Body, "\n", "\r\n"))
	return buf.Bytes(), nil
}

func maskAddress(address string) string {
	at := strings.LastIndex(address, "@")
	if at <= 0 {
		return "***"
	}
	return address[:1] + "***" + address[at:]
}
