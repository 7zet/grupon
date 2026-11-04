package commerce

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
)

// qrPrefix versions the token layout itself. The payload is "gv2:<voucher id>:<window>", where window
// is the number of QRRotation intervals since the Unix epoch. Tokens rotate every QRRotation, so a
// screenshot of a QR goes stale; the previous window is still accepted to absorb clock skew.
const (
	qrPrefix         = "gv2"
	qrRotation       = 30 * time.Second
	minQRSecretBytes = 32
	backupCodeLen    = 10
)

// QRRotation is how often a voucher QR token changes. The voucher page refreshes at this rate.
const QRRotation = qrRotation

// QRClaims is the verified content of a voucher QR token.
type QRClaims struct {
	VoucherID uuid.UUID
	Window    int64
}

// QRSigner issues and verifies voucher QR tokens. The token is base64url(payload) "." base64url(HMAC),
// with payload "gv2:<voucher id>:<window>". The raw voucher id is never shown to the scanner without
// a valid signature. The signer also derives the backup code of a voucher, so no plain code is stored.
type QRSigner struct {
	key []byte
	now func() time.Time
}

// NewQRSigner returns a signer for secret. The secret must hold at least 32 bytes.
func NewQRSigner(secret string) (*QRSigner, error) {
	if len(secret) < minQRSecretBytes {
		return nil, fmt.Errorf("commerce: VOUCHER_QR_SECRET must be at least %d bytes", minQRSecretBytes)
	}
	return &QRSigner{key: []byte(secret), now: time.Now}, nil
}

func window(at time.Time) int64 {
	return at.Unix() / int64(qrRotation/time.Second)
}

// Sign returns the token for a voucher in the current rotation window.
func (s *QRSigner) Sign(voucherID uuid.UUID) string {
	return s.signAt(voucherID, window(s.now()))
}

// NextRotation returns the moment the current QR token stops being the newest one.
func (s *QRSigner) NextRotation() time.Time {
	return time.Unix((window(s.now())+1)*int64(qrRotation/time.Second), 0)
}

func (s *QRSigner) signAt(voucherID uuid.UUID, w int64) string {
	payload := []byte(fmt.Sprintf("%s:%s:%d", qrPrefix, voucherID.String(), w))
	return base64.RawURLEncoding.EncodeToString(payload) + "." + base64.RawURLEncoding.EncodeToString(s.mac(payload))
}

// Verify checks the signature and the window. Tokens from the current or the previous window pass.
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
	w, err := strconv.ParseInt(parts[2], 10, 64)
	if err != nil {
		return QRClaims{}, ErrInvalidQRToken
	}
	if current := window(s.now()); w != current && w != current-1 {
		return QRClaims{}, ErrQRExpired
	}
	return QRClaims{VoucherID: id, Window: w}, nil
}

// BackupCode returns the 10-character backup code of a voucher in the XXXXX-XXXXX form. It is derived
// from the voucher id and the server secret, so it is stable and never stored in plain text.
func (s *QRSigner) BackupCode(voucherID uuid.UUID) string {
	raw := s.mac([]byte("zaxira:" + voucherID.String()))
	out := make([]byte, 0, backupCodeLen)
	for _, b := range raw {
		if len(out) == backupCodeLen {
			break
		}
		out = append(out, backupCodeAlphabet[int(b)%len(backupCodeAlphabet)])
	}
	return string(out[:5]) + "-" + string(out[5:])
}

// BackupCodeHMAC is what the database stores for a backup code. The code is normalised first.
func (s *QRSigner) BackupCodeHMAC(code string) string {
	normalized := strings.ToUpper(strings.TrimSpace(code))
	return base64.RawURLEncoding.EncodeToString(s.mac([]byte("zaxira-code:" + normalized)))
}

func (s *QRSigner) mac(payload []byte) []byte {
	h := hmac.New(sha256.New, s.key)
	h.Write(payload)
	return h.Sum(nil)
}
