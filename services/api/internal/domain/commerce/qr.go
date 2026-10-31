package commerce

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"

	"github.com/google/uuid"
)

// qrPrefix versions the token layout itself. qrPayloadVersion is the voucher version that is
// signed; there is no version column yet, so every voucher is version 1. Bumping it later
// invalidates older codes without a database migration.
const (
	qrPrefix         = "gv1"
	qrPayloadVersion = 1
	minQRSecretBytes = 32
)

// QRClaims is the verified content of a voucher QR token.
type QRClaims struct {
	VoucherID uuid.UUID
	Version   int
}

// QRSigner issues and verifies voucher QR tokens. The token is base64url(payload) "." base64url(HMAC),
// with payload "gv1:<voucher id>:<version>". The raw voucher id is never shown to the scanner
// without a valid signature.
type QRSigner struct {
	key []byte
}

// NewQRSigner returns a signer for secret. The secret must hold at least 32 bytes.
func NewQRSigner(secret string) (*QRSigner, error) {
	if len(secret) < minQRSecretBytes {
		return nil, fmt.Errorf("commerce: VOUCHER_QR_SECRET must be at least %d bytes", minQRSecretBytes)
	}
	return &QRSigner{key: []byte(secret)}, nil
}

// Sign returns the token for a voucher.
func (s *QRSigner) Sign(voucherID uuid.UUID) string {
	payload := []byte(fmt.Sprintf("%s:%s:%d", qrPrefix, voucherID.String(), qrPayloadVersion))
	return base64.RawURLEncoding.EncodeToString(payload) + "." + base64.RawURLEncoding.EncodeToString(s.mac(payload))
}

// Verify checks the signature and returns the claims. Any malformed or tampered token yields
// ErrInvalidQRToken.
func (s *QRSigner) Verify(token string) (QRClaims, error) {
	encodedPayload, encodedMAC, found := strings.Cut(token, ".")
	if !found {
		return QRClaims{}, ErrInvalidQRToken
	}
	payload, err := base64.RawURLEncoding.DecodeString(encodedPayload)
	if err != nil {
		return QRClaims{}, ErrInvalidQRToken
	}
	mac, err := base64.RawURLEncoding.DecodeString(encodedMAC)
	if err != nil || !hmac.Equal(mac, s.mac(payload)) {
		return QRClaims{}, ErrInvalidQRToken
	}
	parts := strings.Split(string(payload), ":")
	if len(parts) != 3 || parts[0] != qrPrefix {
		return QRClaims{}, ErrInvalidQRToken
	}
	id, err := uuid.Parse(parts[1])
	if err != nil {
		return QRClaims{}, ErrInvalidQRToken
	}
	version, err := strconv.Atoi(parts[2])
	if err != nil || version != qrPayloadVersion {
		return QRClaims{}, ErrInvalidQRToken
	}
	return QRClaims{VoucherID: id, Version: version}, nil
}

func (s *QRSigner) mac(payload []byte) []byte {
	h := hmac.New(sha256.New, s.key)
	h.Write(payload)
	return h.Sum(nil)
}
