package identity

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"sync"
	"unicode/utf8"

	"golang.org/x/crypto/argon2"
)

// Argon2id parameters follow the OWASP minimum recommendation. They are stored inside every hash,
// so they can be raised later without invalidating existing passwords.
const (
	argonMemoryKiB   uint32 = 19 * 1024
	argonIterations  uint32 = 2
	argonParallelism uint8  = 1
	argonKeyLength   uint32 = 32
	argonSaltLength         = 16
	minPasswordRunes        = 10
	maxPasswordRunes        = 128
)

// ErrMalformedHash reports a stored password hash that cannot be parsed.
var ErrMalformedHash = errors.New("malformed password hash")

// ValidatePassword enforces the password policy shared by registration, reset and change.
func ValidatePassword(password string) error {
	length := utf8.RuneCountInString(password)
	switch {
	case length < minPasswordRunes:
		return &ValidationError{Field: "password", Code: "password_too_short"}
	case length > maxPasswordRunes:
		return &ValidationError{Field: "password", Code: "password_too_long"}
	}
	return nil
}

// HashPassword returns an encoded argon2id hash in PHC string format.
func HashPassword(password string) (string, error) {
	salt := make([]byte, argonSaltLength)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("read password salt: %w", err)
	}
	key := argon2.IDKey([]byte(password), salt, argonIterations, argonMemoryKiB, argonParallelism, argonKeyLength)
	return fmt.Sprintf("$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version, argonMemoryKiB, argonIterations, argonParallelism,
		base64.RawStdEncoding.EncodeToString(salt), base64.RawStdEncoding.EncodeToString(key)), nil
}

// VerifyPassword compares a password with an encoded hash in constant time.
func VerifyPassword(password, encoded string) (bool, error) {
	memory, iterations, parallelism, salt, key, err := parseHash(encoded)
	if err != nil {
		return false, err
	}
	candidate := argon2.IDKey([]byte(password), salt, iterations, memory, parallelism, uint32(len(key)))
	return subtle.ConstantTimeCompare(candidate, key) == 1, nil
}

var dummyHash = sync.OnceValue(func() string {
	hash, err := HashPassword("timing-equalizer-password")
	if err != nil {
		panic(fmt.Sprintf("create timing equalizer hash: %v", err))
	}
	return hash
})

// equalizeTiming spends the same hashing time as a real verification. It keeps unknown emails and
// wrong passwords indistinguishable by response time.
func equalizeTiming(password string) {
	_, _ = VerifyPassword(password, dummyHash())
}

func parseHash(encoded string) (memory, iterations uint32, parallelism uint8, salt, key []byte, err error) {
	parts := strings.Split(encoded, "$")
	if len(parts) != 6 || parts[1] != "argon2id" || parts[2] != fmt.Sprintf("v=%d", argon2.Version) {
		return 0, 0, 0, nil, nil, ErrMalformedHash
	}
	var m, t, p uint32
	if _, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &m, &t, &p); err != nil {
		return 0, 0, 0, nil, nil, ErrMalformedHash
	}
	if salt, err = base64.RawStdEncoding.DecodeString(parts[4]); err != nil {
		return 0, 0, 0, nil, nil, ErrMalformedHash
	}
	if key, err = base64.RawStdEncoding.DecodeString(parts[5]); err != nil || len(key) == 0 {
		return 0, 0, 0, nil, nil, ErrMalformedHash
	}
	if p > 255 {
		return 0, 0, 0, nil, nil, ErrMalformedHash
	}
	return m, t, uint8(p), salt, key, nil
}
