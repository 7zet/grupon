package identity

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// Issuer identifies tokens minted by this API.
const Issuer = "dealo-api"

// MinSecretBytes is the smallest accepted HS256 signing secret.
const MinSecretBytes = 32

// ErrInvalidAccessToken reports an access token that is malformed, forged, or expired.
var ErrInvalidAccessToken = errors.New("invalid access token")

// AccessClaims are the JWT claims carried by an access token.
type AccessClaims struct {
	Roles []string `json:"roles"`
	jwt.RegisteredClaims
}

// TokenIssuer signs and verifies HS256 access tokens.
type TokenIssuer struct {
	secret []byte
	ttl    time.Duration
	now    func() time.Time
}

// NewTokenIssuer validates the signing secret and returns an issuer.
func NewTokenIssuer(secret string, ttl time.Duration, now func() time.Time) (*TokenIssuer, error) {
	if len(secret) < MinSecretBytes {
		return nil, fmt.Errorf("JWT secret must be at least %d bytes", MinSecretBytes)
	}
	if ttl <= 0 {
		return nil, errors.New("access token TTL must be positive")
	}
	return &TokenIssuer{secret: []byte(secret), ttl: ttl, now: now}, nil
}

// TTL returns the lifetime of issued access tokens.
func (t *TokenIssuer) TTL() time.Duration { return t.ttl }

// Issue signs an access token for the user with the roles known at issue time.
func (t *TokenIssuer) Issue(userID uuid.UUID, roles []Role) (string, error) {
	issuedAt := t.now()
	names := make([]string, 0, len(roles))
	for _, role := range roles {
		names = append(names, string(role))
	}
	claims := AccessClaims{
		Roles: names,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    Issuer,
			Subject:   userID.String(),
			IssuedAt:  jwt.NewNumericDate(issuedAt),
			ExpiresAt: jwt.NewNumericDate(issuedAt.Add(t.ttl)),
			ID:        uuid.NewString(),
		},
	}
	signed, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(t.secret)
	if err != nil {
		return "", fmt.Errorf("sign access token: %w", err)
	}
	return signed, nil
}

// Parse verifies the signature, algorithm, issuer and expiry, and returns the subject user id.
func (t *TokenIssuer) Parse(raw string) (uuid.UUID, error) {
	var claims AccessClaims
	_, err := jwt.ParseWithClaims(raw, &claims, func(*jwt.Token) (any, error) {
		return t.secret, nil
	},
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
		jwt.WithIssuer(Issuer),
		jwt.WithExpirationRequired(),
		jwt.WithIssuedAt(),
		jwt.WithTimeFunc(t.now),
	)
	if err != nil {
		return uuid.Nil, ErrInvalidAccessToken
	}
	id, err := uuid.Parse(claims.Subject)
	if err != nil {
		return uuid.Nil, ErrInvalidAccessToken
	}
	return id, nil
}
