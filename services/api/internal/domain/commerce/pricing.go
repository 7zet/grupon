package commerce

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"strings"
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

// ValidateGiftFields checks gift recipient data. Gift fields without is_gift are rejected so that
// stale form state cannot silently attach a recipient to a normal purchase.
func ValidateGiftFields(field string, item OrderItemInput) error {
	name, email, message := item.GiftRecipientName, item.GiftRecipientEmail, item.GiftMessage
	if !item.IsGift {
		if name != nil || email != nil || message != nil {
			return invalid(field, "gift_fields_without_gift")
		}
		return nil
	}
	if name != nil && len(*name) > MaxGiftNameBytes {
		return invalid(field+".gift_recipient_name", "too_long")
	}
	if message != nil && len(*message) > MaxGiftMsgBytes {
		return invalid(field+".gift_message", "too_long")
	}
	if email != nil && (!strings.Contains(*email, "@") || len(*email) > 320) {
		return invalid(field+".gift_recipient_email", "invalid_email")
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
		OptionID string  `json:"option_id"`
		Quantity int     `json:"quantity"`
		IsGift   bool    `json:"is_gift"`
		Name     *string `json:"name,omitempty"`
		Email    *string `json:"email,omitempty"`
		Message  *string `json:"message,omitempty"`
	}
	items := make([]normalizedItem, 0, len(input.Items))
	for _, item := range input.Items {
		items = append(items, normalizedItem{
			OptionID: item.OptionID.String(),
			Quantity: item.Quantity,
			IsGift:   item.IsGift,
			Name:     item.GiftRecipientName,
			Email:    item.GiftRecipientEmail,
			Message:  item.GiftMessage,
		})
	}
	raw, err := json.Marshal(struct {
		Items    []normalizedItem `json:"items"`
		Shipping map[string]any   `json:"shipping_address"`
	}{Items: items, Shipping: input.ShippingAddress})
	if err != nil {
		return "", fmt.Errorf("encode request fingerprint: %w", err)
	}
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:]), nil
}

// VoucherCodeAlphabet has no 0, 1, I, L or O, so codes read aloud or typed by hand are unambiguous.
const VoucherCodeAlphabet = "23456789ABCDEFGHJKMNPQRSTUVWXYZ"

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

// FormatOrderNumber renders the human-readable order number, for example GR-2026-000123.
func FormatOrderNumber(year int, sequence int64) string {
	return fmt.Sprintf("GR-%d-%06d", year, sequence)
}
