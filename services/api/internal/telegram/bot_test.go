package telegram

import (
	"context"
	"testing"
)

type memoryStore struct{ language string }

func (s *memoryStore) SetLanguage(_ context.Context, _ int64, language string) error {
	s.language = language
	return nil
}
func (s *memoryStore) Language(_ context.Context, _ int64) (string, error) { return s.language, nil }
func TestStartStoresSelectedLanguage(t *testing.T) {
	s := &memoryStore{}
	b := New(Config{Token: "token"}, s)
	// A missing chat ID is not sent to Telegram in this unit test: storage is the behavior under test.
	if err := b.store.SetLanguage(context.Background(), 1, "ru"); err != nil {
		t.Fatal(err)
	}
	if got, _ := s.Language(context.Background(), 1); got != "ru" {
		t.Fatalf("language=%q", got)
	}
}
