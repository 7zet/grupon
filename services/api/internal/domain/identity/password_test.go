package identity

import (
	"strings"
	"testing"
)

func TestHashAndVerifyPassword(t *testing.T) {
	t.Parallel()

	hash, err := HashPassword("correct horse battery")
	if err != nil {
		t.Fatalf("HashPassword() error = %v", err)
	}
	if !strings.HasPrefix(hash, "$argon2id$v=19$m=19456,t=2,p=1$") {
		t.Fatalf("hash = %q, want argon2id PHC format with stored parameters", hash)
	}

	ok, err := VerifyPassword("correct horse battery", hash)
	if err != nil || !ok {
		t.Fatalf("VerifyPassword(correct) = %v, %v; want true, nil", ok, err)
	}
	ok, err = VerifyPassword("wrong horse battery", hash)
	if err != nil || ok {
		t.Fatalf("VerifyPassword(wrong) = %v, %v; want false, nil", ok, err)
	}
}

func TestHashesAreSalted(t *testing.T) {
	t.Parallel()

	first, err := HashPassword("same password here")
	if err != nil {
		t.Fatal(err)
	}
	second, err := HashPassword("same password here")
	if err != nil {
		t.Fatal(err)
	}
	if first == second {
		t.Fatal("two hashes of the same password are identical; salt is missing")
	}
}

func TestVerifyPasswordRejectsMalformedHash(t *testing.T) {
	t.Parallel()

	for _, encoded := range []string{"", "plain", "$bcrypt$v=1$x$y$z", "$argon2id$v=19$m=x$c2FsdA$a2V5"} {
		if _, err := VerifyPassword("anything", encoded); err == nil {
			t.Errorf("VerifyPassword(%q) error = nil, want malformed hash error", encoded)
		}
	}
}

func TestValidatePassword(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		password string
		code     string
	}{
		{name: "too short", password: "123456789", code: "password_too_short"},
		{name: "minimum", password: "1234567890"},
		{name: "maximum", password: strings.Repeat("a", 128)},
		{name: "too long", password: strings.Repeat("a", 129), code: "password_too_long"},
		{name: "counts runes not bytes", password: strings.Repeat("ё", 10)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := ValidatePassword(tt.password)
			if tt.code == "" {
				if err != nil {
					t.Fatalf("ValidatePassword() error = %v, want nil", err)
				}
				return
			}
			validation, ok := err.(*ValidationError)
			if !ok || validation.Code != tt.code {
				t.Fatalf("ValidatePassword() error = %v, want code %s", err, tt.code)
			}
		})
	}
}
