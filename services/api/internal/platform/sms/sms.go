// Package sms delivers sign-in codes to phones.
package sms

import (
	"context"
	"log/slog"
)

// LogSender writes codes to the log instead of sending them. Development only: a real provider
// must replace it before production, and production runs without a sender, so codes cannot be issued.
type LogSender struct {
	logger *slog.Logger
}

// NewLogSender returns the development sender.
func NewLogSender(logger *slog.Logger) *LogSender {
	return &LogSender{logger: logger}
}

// SendCode logs the code for the phone.
func (s *LogSender) SendCode(ctx context.Context, phone, code string) error {
	s.logger.InfoContext(ctx, "sign-in code (development sender)", slog.String("phone", phone), slog.String("code", code))
	return nil
}
