package identity

import (
	"testing"
	"time"
)

func TestTOTPUsesRFC6238Window(t *testing.T) {
	// RFC 6238 SHA-1 test secret; its 8-digit value at 59s is 94287082, so six digits are 287082.
	secret := "GEZDGNBVGY3TQOJQGEZDGNBVGY3TQOJQ"
	if !validTOTP(secret, "287082", time.Unix(59, 0)) {
		t.Fatal("valid current TOTP was rejected")
	}
	if validTOTP(secret, "000000", time.Unix(59, 0)) {
		t.Fatal("invalid TOTP was accepted")
	}
}

func TestRecoveryCodesReturnsTenUniqueCodes(t *testing.T) {
	codes, hashes, err := recoveryCodes()
	if err != nil {
		t.Fatal(err)
	}
	if len(codes) != 10 || len(hashes) != 10 {
		t.Fatalf("got %d codes and %d hashes", len(codes), len(hashes))
	}
	seen := map[string]bool{}
	for i, code := range codes {
		if code == "" || hashes[i] != HashToken(code) {
			t.Fatal("recovery code was not hashed correctly")
		}
		if seen[code] {
			t.Fatal("duplicate recovery code")
		}
		seen[code] = true
	}
}

func TestPrivilegedTwoFactorRoles(t *testing.T) {
	for _, role := range []Role{RoleAdmin, RoleModerator, RoleSupport, RoleFinance, RoleSuperadmin} {
		if _, ok := privileged([]Role{role}); !ok {
			t.Fatalf("%s must require 2FA", role)
		}
	}
	if _, ok := privileged([]Role{RoleUser, RoleMerchantOwner}); ok {
		t.Fatal("merchant owner must not require staff 2FA")
	}
}
