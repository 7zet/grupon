package identity_test

import (
	"context"
	"errors"
	"regexp"
	"testing"
	"time"

	"github.com/grupon/grupon/services/api/internal/domain/identity"
	"github.com/grupon/grupon/services/api/internal/domain/identity/identitytest"
)

const otpSecret = "phone-otp-test-secret-that-is-long-enough"

type capturedSender struct{ code string }

func (c *capturedSender) SendCode(_ context.Context, _, code string) error {
	c.code = code
	return nil
}

type otpClock struct{ now time.Time }

func (c *otpClock) Now() time.Time { return c.now }

func otpService(t *testing.T, clk *otpClock, sender identity.CodeSender, cooldown time.Duration) *identity.Service {
	t.Helper()
	return otpServiceWithConfig(t, clk, sender, identity.Config{
		AccessTokenTTL: 15 * time.Minute, RefreshTokenTTL: 24 * time.Hour, OTPEcho: true, OTPCooldown: cooldown,
	})
}

func otpServiceWithConfig(t *testing.T, clk *otpClock, sender identity.CodeSender, cfg identity.Config) *identity.Service {
	t.Helper()
	issuer, err := identity.NewTokenIssuer(otpSecret, 15*time.Minute, clk.Now)
	if err != nil {
		t.Fatal(err)
	}
	repo := identitytest.NewRepository(clk.Now)
	return identity.NewService(identity.Dependencies{
		Repository: repo,
		Tokens:     issuer,
		CodeSender: sender,
		Now:        clk.Now,
	}, cfg)
}

var codePattern = regexp.MustCompile(`^[0-9]{6}$`)

func TestPhoneSignInCreatesAccountOnFirstUse(t *testing.T) {
	clk := &otpClock{now: time.Date(2026, 6, 1, 9, 0, 0, 0, time.UTC)}
	sender := &capturedSender{}
	svc := otpService(t, clk, sender, time.Minute)
	meta := identity.RequestMeta{IP: "203.0.113.10"}

	requested, err := svc.RequestCode(context.Background(), meta, "+998 90 123 45 67")
	if err != nil {
		t.Fatal(err)
	}
	if !codePattern.MatchString(sender.code) || requested.DevCode != sender.code {
		t.Fatalf("code = %q, dev code = %q", sender.code, requested.DevCode)
	}

	session, err := svc.VerifyCode(context.Background(), meta, "+998901234567", sender.code)
	if err != nil {
		t.Fatal(err)
	}
	if session.Account.Phone == nil || *session.Account.Phone != "+998901234567" {
		t.Fatalf("account phone = %v", session.Account.Phone)
	}
	if session.AccessToken == "" || session.RefreshToken == "" {
		t.Fatal("session tokens missing")
	}

	// The same code is single-use.
	if _, err := svc.VerifyCode(context.Background(), meta, "+998901234567", sender.code); !errors.Is(err, identity.ErrOTPInvalid) {
		t.Fatalf("reused code: got %v", err)
	}
}

func TestWrongCodesUseUpTheCode(t *testing.T) {
	clk := &otpClock{now: time.Date(2026, 6, 1, 9, 0, 0, 0, time.UTC)}
	sender := &capturedSender{}
	svc := otpService(t, clk, sender, time.Minute)
	meta := identity.RequestMeta{IP: "203.0.113.11"}
	if _, err := svc.RequestCode(context.Background(), meta, "+998901111111"); err != nil {
		t.Fatal(err)
	}
	wrong := "000000"
	if wrong == sender.code {
		wrong = "111111"
	}
	for i := 0; i < 5; i++ {
		if _, err := svc.VerifyCode(context.Background(), meta, "+998901111111", wrong); !errors.Is(err, identity.ErrOTPInvalid) {
			t.Fatalf("attempt %d: got %v", i, err)
		}
	}
	if _, err := svc.VerifyCode(context.Background(), meta, "+998901111111", sender.code); !errors.Is(err, identity.ErrOTPTooManyTries) {
		t.Fatalf("right code after five misses: got %v", err)
	}
}

func TestCodesExpireAndResendHasACooldown(t *testing.T) {
	clk := &otpClock{now: time.Date(2026, 6, 1, 9, 0, 0, 0, time.UTC)}
	sender := &capturedSender{}
	svc := otpService(t, clk, sender, time.Minute)
	meta := identity.RequestMeta{IP: "203.0.113.12"}

	if _, err := svc.RequestCode(context.Background(), meta, "+998902222222"); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.RequestCode(context.Background(), meta, "+998902222222"); !errors.Is(err, identity.ErrOTPCooldown) {
		t.Fatalf("resend inside cooldown: got %v", err)
	}

	clk.now = clk.now.Add(6 * time.Minute)
	if _, err := svc.VerifyCode(context.Background(), meta, "+998902222222", sender.code); !errors.Is(err, identity.ErrOTPInvalid) {
		t.Fatalf("expired code: got %v", err)
	}
}

func TestRejectsNonUzbekPhones(t *testing.T) {
	clk := &otpClock{now: time.Date(2026, 6, 1, 9, 0, 0, 0, time.UTC)}
	svc := otpService(t, clk, &capturedSender{}, time.Minute)
	for _, phone := range []string{"+79123456789", "998901234567", "+99890123", "abc"} {
		if _, err := svc.RequestCode(context.Background(), identity.RequestMeta{}, phone); !errors.Is(err, identity.ErrInvalidPhone) {
			t.Fatalf("%s: got %v", phone, err)
		}
	}
}

func TestRealSMSDailyCaps(t *testing.T) {
	clk := &otpClock{now: time.Date(2026, 6, 1, 9, 0, 0, 0, time.UTC)}
	svc := otpServiceWithConfig(t, clk, &capturedSender{}, identity.Config{AccessTokenTTL: 15 * time.Minute, RefreshTokenTTL: time.Hour, RealSMS: true, SMSDailyCapPerIP: 1, SMSDailyCapGlobal: 2})
	if _, err := svc.RequestCode(context.Background(), identity.RequestMeta{IP: "203.0.113.55"}, "+998901111111"); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.RequestCode(context.Background(), identity.RequestMeta{IP: "203.0.113.55"}, "+998902222222"); !errors.Is(err, identity.ErrSMSDailyCap) {
		t.Fatalf("per-IP cap: %v", err)
	}
	if _, err := svc.RequestCode(context.Background(), identity.RequestMeta{IP: "203.0.113.56"}, "+998903333333"); !errors.Is(err, identity.ErrSMSDailyCap) {
		t.Fatalf("global cap: %v", err)
	}
}
