package commerce

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

func signerAt(t *testing.T, now time.Time) *QRSigner {
	t.Helper()
	signer, err := NewQRSigner(strings.Repeat("s", 32))
	if err != nil {
		t.Fatal(err)
	}
	signer.now = func() time.Time { return now }
	return signer
}

func TestQRTokenRotatesAndKeepsOneWindowOfGrace(t *testing.T) {
	voucherID := uuid.New()
	start := time.Unix(1_800_000_000, 0)
	token := signerAt(t, start).Sign(voucherID)

	if claims, err := signerAt(t, start.Add(10*time.Second)).Verify(token); err != nil || claims.VoucherID != voucherID {
		t.Fatalf("same window: claims=%v err=%v", claims, err)
	}
	if _, err := signerAt(t, start.Add(35*time.Second)).Verify(token); err != nil {
		t.Fatalf("previous window should still verify: %v", err)
	}
	if _, err := signerAt(t, start.Add(70*time.Second)).Verify(token); !errors.Is(err, ErrQRExpired) {
		t.Fatalf("two windows later: got %v, want ErrQRExpired", err)
	}
}

func TestBackupCodeIsStableAndNormalisedForLookup(t *testing.T) {
	signer := signerAt(t, time.Unix(1_800_000_000, 0))
	voucherID := uuid.New()
	code := signer.BackupCode(voucherID)
	if code != signer.BackupCode(voucherID) {
		t.Fatal("backup code changed between calls")
	}
	if len(code) != 11 || code[5] != '-' {
		t.Fatalf("backup code %q is not XXXXX-XXXXX", code)
	}
	if signer.BackupCodeHMAC(code) != signer.BackupCodeHMAC(" "+strings.ToLower(code)+" ") {
		t.Fatal("lookup hash must ignore case and surrounding spaces")
	}
}
