package commerce

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
)

// LineTotal multiplies a unit price by a quantity in integer minor units. It refuses amounts
// beyond MaxMoneyAmount instead of overflowing.
func LineTotal(unit int64, quantity int) (int64, error) {
	if unit <= 0 || quantity <= 0 {
		return 0, fmt.Errorf("commerce: line total needs positive unit and quantity")
	}
	if unit > MaxMoneyAmount/int64(quantity) {
		return 0, fmt.Errorf("commerce: line total exceeds limit")
	}
	return unit * int64(quantity), nil
}

// DiscountPercent returns the discount as a whole percentage, rounded half up. It never uses floats.
func DiscountPercent(original, deal int64) int {
	if original <= 0 || deal >= original {
		return 0
	}
	saved := new(big.Int).Mul(big.NewInt(original-deal), big.NewInt(100))
	// round(saved / original) = floor((2*saved + original) / (2*original))
	numerator := new(big.Int).Add(new(big.Int).Mul(saved, big.NewInt(2)), big.NewInt(original))
	denominator := big.NewInt(2 * original)
	return int(new(big.Int).Quo(numerator, denominator).Int64())
}

// ValidateQuantity checks a requested quantity against the per-line limit.
func ValidateQuantity(field string, quantity int) error {
	if quantity < 1 || quantity > MaxLineQuantity {
		return invalid(field, "out_of_range")
	}
	return nil
}

// ValidateIdempotencyKey checks the header length rules from the contract.
func ValidateIdempotencyKey(key string) error {
	if len(key) < MinIdempotencyKey || len(key) > MaxIdempotencyKey {
		return invalid("Idempotency-Key", "invalid_length")
	}
	return nil
}

// RequestFingerprint hashes the normalized order request. Replays are compared by fingerprint, so
// formatting differences in the JSON body do not count as a different request.
func RequestFingerprint(input OrderInput) (string, error) {
	type normalizedItem struct {
		OptionID string `json:"option_id"`
		Quantity int    `json:"quantity"`
	}
	items := make([]normalizedItem, 0, len(input.Items))
	for _, item := range input.Items {
		items = append(items, normalizedItem{
			OptionID: item.OptionID.String(),
			Quantity: item.Quantity,
		})
	}
	raw, err := json.Marshal(struct {
		Items []normalizedItem `json:"items"`
	}{Items: items})
	if err != nil {
		return "", fmt.Errorf("encode request fingerprint: %w", err)
	}
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:]), nil
}

// VoucherCodeAlphabet has no 0, 1, I, L or O, so codes read aloud or typed by hand are unambiguous.
const VoucherCodeAlphabet = "23456789ABCDEFGHJKMNPQRSTUVWXYZ"

// backupCodeAlphabet is Crockford base32: no I, L, O or U. Backup codes are 10 characters (PRD FR-04.2).
const backupCodeAlphabet = "0123456789ABCDEFGHJKMNPQRSTVWXYZ"

// NewVoucherCode returns a random code formatted XXXX-XXXX. Bytes outside the largest multiple of
// the alphabet size are rejected so that every character is equally likely.
func NewVoucherCode() (string, error) {
	const want = 8
	limit := byte(256 - 256%len(VoucherCodeAlphabet))
	out := make([]byte, 0, want)
	buf := make([]byte, 1)
	for len(out) < want {
		if _, err := rand.Read(buf); err != nil {
			return "", fmt.Errorf("read random voucher code: %w", err)
		}
		if buf[0] >= limit {
			continue
		}
		out = append(out, VoucherCodeAlphabet[int(buf[0])%len(VoucherCodeAlphabet)])
	}
	return string(out[:4]) + "-" + string(out[4:]), nil
}

// FormatOrderNumber renders the human-readable order number, for example DL-2026-000123 (DL for Dealo).
func FormatOrderNumber(year int, sequence int64) string {
	return fmt.Sprintf("DL-%d-%06d", year, sequence)
}
