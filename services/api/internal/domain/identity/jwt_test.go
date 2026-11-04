package identity

import (
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

const testSecret = "unit-test-secret-with-at-least-32-bytes"

func TestAccessTokenRoundTrip(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 10, 31, 12, 0, 0, 0, time.UTC)
	issuer, err := NewTokenIssuer(testSecret, 15*time.Minute, func() time.Time { return now })
	if err != nil {
		t.Fatal(err)
	}
	userID := uuid.Must(uuid.NewV7())
	token, err := issuer.Issue(userID, []Role{RoleUser, RoleMerchantOwner})
	if err != nil {
		t.Fatal(err)
	}
	got, err := issuer.Parse(token)
	if err != nil || got != userID {
		t.Fatalf("Parse() = %v, %v; want %v", got, err, userID)
	}
}

func TestAccessTokenExpires(t *testing.T) {
	t.Parallel()

	issuedAt := time.Date(2026, 10, 31, 12, 0, 0, 0, time.UTC)
	clock := issuedAt
	issuer, err := NewTokenIssuer(testSecret, 15*time.Minute, func() time.Time { return clock })
	if err != nil {
		t.Fatal(err)
	}
	token, err := issuer.Issue(uuid.Must(uuid.NewV7()), nil)
	if err != nil {
		t.Fatal(err)
	}

	clock = issuedAt.Add(14 * time.Minute)
	if _, err := issuer.Parse(token); err != nil {
		t.Fatalf("Parse() before expiry error = %v", err)
	}
	clock = issuedAt.Add(15*time.Minute + time.Second)
	if _, err := issuer.Parse(token); err == nil {
		t.Fatal("Parse() after expiry error = nil, want expired")
	}
}

func TestAccessTokenRejectsForgeryAndAlgorithmSwaps(t *testing.T) {
	t.Parallel()

	now := time.Now()
	issuer, err := NewTokenIssuer(testSecret, time.Minute, func() time.Time { return now })
	if err != nil {
		t.Fatal(err)
	}
	userID := uuid.Must(uuid.NewV7())

	other, err := NewTokenIssuer("another-secret-that-is-long-enough!", time.Minute, func() time.Time { return now })
	if err != nil {
		t.Fatal(err)
	}
	foreign, _ := other.Issue(userID, nil)
	if _, err := issuer.Parse(foreign); err == nil {
		t.Error("token signed with another secret was accepted")
	}

	claims := AccessClaims{RegisteredClaims: jwt.RegisteredClaims{
		Issuer: Issuer, Subject: userID.String(), ExpiresAt: jwt.NewNumericDate(now.Add(time.Minute)),
	}}
	none, err := jwt.NewWithClaims(jwt.SigningMethodNone, claims).SignedString(jwt.UnsafeAllowNoneSignatureType)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := issuer.Parse(none); err == nil {
		t.Error("alg=none token was accepted")
	}

	wrongIssuer := AccessClaims{RegisteredClaims: jwt.RegisteredClaims{
		Issuer: "someone-else", Subject: userID.String(), ExpiresAt: jwt.NewNumericDate(now.Add(time.Minute)),
	}}
	bad, _ := jwt.NewWithClaims(jwt.SigningMethodHS256, wrongIssuer).SignedString([]byte(testSecret))
	if _, err := issuer.Parse(bad); err == nil {
		t.Error("token from another issuer was accepted")
	}
}

func TestNewTokenIssuerRequiresLongSecret(t *testing.T) {
	t.Parallel()

	_, err := NewTokenIssuer(strings.Repeat("x", 31), time.Minute, time.Now)
	if err == nil {
		t.Fatal("NewTokenIssuer() error = nil, want secret length error")
	}
}
