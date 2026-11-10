package mail

import (
	"context"
	"log/slog"
	"strings"
	"testing"
)

func TestRenderEncodesSubjectAndUsesCRLF(t *testing.T) {
	t.Parallel()

	raw, err := render("Dealo <no-reply@dealo.local>", Message{
		To:      "buyer@example.test",
		Subject: "Dealo: восстановление пароля",
		Body:    "line one\nline two",
	})
	if err != nil {
		t.Fatal(err)
	}
	text := string(raw)
	if strings.Contains(text, "Subject: Dealo: восстановление пароля") {
		t.Error("non-ASCII subject was not encoded")
	}
	if !strings.Contains(text, "Subject: =?utf-8?") {
		t.Errorf("subject not RFC 2047 encoded: %q", text)
	}
	if !strings.Contains(text, "line one\r\nline two") {
		t.Error("body line endings are not CRLF")
	}
}

func TestRenderRejectsHeaderInjection(t *testing.T) {
	t.Parallel()

	_, err := render("from@dealo.local", Message{To: "a@example.test\r\nBcc: victim@example.test", Subject: "s"})
	if err == nil {
		t.Fatal("render() accepted a CRLF in the recipient")
	}
}

func TestLogSenderDoesNotLogBody(t *testing.T) {
	t.Parallel()

	var logged strings.Builder
	logger := slog.New(slog.NewTextHandler(&logged, nil))
	sender := NewLogSender(logger)

	if err := sender.Send(context.Background(), Message{To: "buyer@example.test", Subject: "Hi", Body: "secret-token-value"}); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(logged.String(), "secret-token-value") {
		t.Fatal("log sender wrote the message body, which carries a single-use token")
	}
	if strings.Contains(logged.String(), "buyer@example.test") {
		t.Fatal("log sender wrote the full recipient address")
	}
}
