package merchant

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestMaskNameShortensToFirstNameAndInitial(t *testing.T) {
	t.Parallel()
	cases := map[string]string{
		"Alisher Usmonov": "Alisher U.",
		"Madina":          "Madina",
		"  ":              "",
		"Ali Bek Karimov": "Ali K.",
	}
	for in, want := range cases {
		if got := MaskName(in); got != want {
			t.Fatalf("MaskName(%q) = %q, want %q", in, got, want)
		}
	}
}

// memPanel holds one review and its reply so the reply window can be tested.
type memPanel struct {
	reply *ReviewReply
	now   time.Time
}

func (m *memPanel) Payouts(context.Context, uuid.UUID) ([]PayoutRow, error) { return nil, nil }
func (m *memPanel) Customers(context.Context, uuid.UUID) ([]CustomerRow, error) {
	return nil, nil
}
func (m *memPanel) Analytics(context.Context, uuid.UUID, time.Time, time.Time) (Analytics, error) {
	return Analytics{}, nil
}
func (m *memPanel) Reviews(context.Context, uuid.UUID) ([]ReviewRow, error) { return nil, nil }
func (m *memPanel) ReviewBelongs(context.Context, uuid.UUID, uuid.UUID) (bool, error) {
	return true, nil
}
func (m *memPanel) ReplyFor(context.Context, uuid.UUID) (*ReviewReply, error) { return m.reply, nil }
func (m *memPanel) InsertReply(_ context.Context, _, _ uuid.UUID, text string) (ReviewReply, error) {
	r := ReviewReply{ID: uuid.New(), Text: text, CreatedAt: m.now, UpdatedAt: m.now}
	m.reply = &r
	return r, nil
}
func (m *memPanel) UpdateReply(_ context.Context, _ uuid.UUID, text string) (ReviewReply, error) {
	m.reply.Text = text
	return *m.reply, nil
}
func (m *memPanel) Orders(context.Context, uuid.UUID) ([]OrderRow, error) { return nil, nil }
func (m *memPanel) Settings(context.Context, uuid.UUID) (Settings, error) {
	return Settings{}, nil
}
func (m *memPanel) UpdateSettings(context.Context, uuid.UUID, Settings) error { return nil }

func TestCashierCannotReplyAndReplyLocksAfterWindow(t *testing.T) {
	t.Parallel()
	start := time.Date(2026, 11, 1, 10, 0, 0, 0, time.UTC)
	repo := &memPanel{now: start}
	clock := start
	svc := NewPanelService(repo, func() time.Time { return clock })
	ctx := context.Background()
	merchantID, reviewID := uuid.New(), uuid.New()
	owner := Actor{UserID: uuid.New(), Role: RoleOwner}
	cashier := Actor{UserID: uuid.New(), Role: RoleCashier}

	if _, err := svc.Reply(ctx, cashier, merchantID, reviewID, "Rahmat"); !errors.Is(err, ErrForbidden) {
		t.Fatalf("cashier reply: got %v, want ErrForbidden", err)
	}
	if _, err := svc.Reply(ctx, owner, merchantID, reviewID, "   "); err == nil {
		t.Fatal("empty reply was accepted")
	}
	if _, err := svc.Reply(ctx, owner, merchantID, reviewID, "Rahmat!"); err != nil {
		t.Fatalf("first reply: %v", err)
	}
	clock = start.Add(ReplyWindow + time.Minute)
	if _, err := svc.Reply(ctx, owner, merchantID, reviewID, "Yangi matn"); !errors.Is(err, ErrReplyLocked) {
		t.Fatalf("edit after 24 h: got %v, want ErrReplyLocked", err)
	}
}
