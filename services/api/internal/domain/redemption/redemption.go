// Package redemption lets merchant staff check a voucher and mark it used at one of their branches.
package redemption

import (
	"context"
	"errors"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/grupon/grupon/services/api/internal/domain/commerce"
	"github.com/grupon/grupon/services/api/internal/domain/merchant"
)

// Voucher states shown to staff. Stored statuses are active, redeemed, refunded and cancelled; expired is derived from time.
const (
	StatusActive   = "active"
	StatusRedeemed = "redeemed"
	StatusExpired  = "expired"
	StatusRefunded = "refunded"
	StatusVoided   = "voided"
)

// Errors returned by lookup and redemption. The HTTP layer maps them to problem codes.
var (
	ErrNotFound        = errors.New("redemption: voucher not found")
	ErrInvalidToken    = errors.New("redemption: qr token is not valid")
	ErrWrongMerchant   = errors.New("redemption: voucher belongs to another merchant")
	ErrAlreadyRedeemed = errors.New("redemption: voucher already redeemed")
	ErrExpired         = errors.New("redemption: voucher expired")
	ErrRefunded        = errors.New("redemption: voucher refunded")
	ErrVoided          = errors.New("redemption: voucher voided")
	// ErrOnHold means fraud review or an open dispute holds the voucher.
	ErrOnHold = errors.New("redemption: voucher is on hold for review")
)

// ReversalWindow is how long the cashier may undo a redemption; after it, only support can.
const ReversalWindow = 10 * time.Minute

// ErrReversalWindowClosed reports a reversal attempted after the cashier window.
var ErrReversalWindowClosed = errors.New("redemption: reversal window has closed")

// ErrNoRedemption reports a voucher with nothing to reverse.
var ErrNoRedemption = errors.New("redemption: no redemption to reverse")

// Redemption is one use of a voucher, as needed to reverse it.
type Redemption struct {
	ID         uuid.UUID
	VoucherID  uuid.UUID
	MerchantID uuid.UUID
	StaffID    uuid.UUID
	Amount     int64
	Commission int64
	CreatedAt  time.Time
}

var codePattern = regexp.MustCompile(`^([A-Z0-9]{4}-[A-Z0-9]{4}|[0-9A-HJKMNP-TV-Z]{5}-[0-9A-HJKMNP-TV-Z]{5})$`)

// SupportReverse undoes a redemption after the merchant window closed (PRD FR-05.5). Only support and
// superadmin may do it, and the reason is kept in the audit trail.
func (s *Service) SupportReverse(ctx context.Context, actorID uuid.UUID, role string, voucherID uuid.UUID, reason string) error {
	if role != "support" && role != "superadmin" {
		return merchant.ErrForbidden
	}
	reason = strings.TrimSpace(reason)
	if n := len([]rune(reason)); n < 3 || n > 500 {
		return &merchant.InvalidInputError{Field: "reason", Code: "length"}
	}
	found, err := s.repo.LatestOpenRedemptionForVoucher(ctx, voucherID)
	if err != nil {
		return err
	}
	return s.repo.ReverseUse(ctx, found.MerchantID, found, reason, AuditEntry{
		ActorID: actorID,
		Role:    role,
		Action:  "voucher_redemption_reversed_by_support",
		After:   map[string]any{"reason": reason},
	})
}

// Verifier resolves a signed QR token to a voucher id.
type Verifier func(token string) (uuid.UUID, error)

// AuditEntry records a redemption for the audit trail.
type AuditEntry struct {
	ActorID   uuid.UUID
	Role      string
	VoucherID uuid.UUID
	Action    string
	After     map[string]any
}

// AttemptCounts is what the scanner limits need to know about a staff member.
type AttemptCounts struct {
	LastMinute   int
	FailedRecent int
}

// Repository reads vouchers and performs the conditional redemption write.
type Repository interface {
	Find(ctx context.Context, voucherID uuid.UUID) (commerce.Voucher, error)
	FindByCode(ctx context.Context, code string) (commerce.Voucher, error)
	// FindByBackupCode looks a voucher up by the stored HMAC of its backup code.
	FindByBackupCode(ctx context.Context, codeHMAC string) (commerce.Voucher, error)
	// AttemptStats counts a staff member's lookups: in the last minute, and failed ones since blockSince.
	AttemptStats(ctx context.Context, staffID uuid.UUID, minuteSince, blockSince time.Time) (AttemptCounts, error)
	RecordAttempt(ctx context.Context, staffID uuid.UUID, failed bool) error
	// RecordDenied writes the audit row for a scan by a staff member of another business (AT-04).
	RecordDenied(ctx context.Context, audit AuditEntry) error
	LocationOwned(ctx context.Context, merchantID, locationID uuid.UUID) (bool, error)
	// Redeem marks the voucher redeemed only while it is still active and inside its validity window.
	Redeem(ctx context.Context, merchantID, voucherID, locationID uuid.UUID, at time.Time, audit AuditEntry, geo *Geo) (bool, error)
	View(ctx context.Context, voucher commerce.Voucher) (commerce.VoucherView, error)
	// ReversalCandidate returns the latest unreversed redemption of the voucher, or ErrNotFound.
	ReversalCandidate(ctx context.Context, merchantID, voucherID uuid.UUID) (Redemption, error)
	// LatestOpenRedemptionForVoucher returns the newest unreversed redemption of a voucher, any merchant.
	LatestOpenRedemptionForVoucher(ctx context.Context, voucherID uuid.UUID) (Redemption, error)
	// ReverseUse marks the redemption reversed and rolls the voucher count back, with a reversing entry.
	ReverseUse(ctx context.Context, merchantID uuid.UUID, found Redemption, reason string, audit AuditEntry) error
}

// Service implements lookup and redemption rules.
type Service struct {
	repo       Repository
	verify     Verifier
	now        func() time.Time
	backupHash func(code string) string
}

// WithBackupCodes lets staff redeem with a voucher backup code as well as the human code.
func (s *Service) WithBackupCodes(hash func(code string) string) *Service {
	s.backupHash = hash
	return s
}

// NewService binds the redemption rules to a repository and a QR verifier.
func NewService(repo Repository, verify Verifier, now func() time.Time) *Service {
	if now == nil {
		now = time.Now
	}
	return &Service{repo: repo, verify: verify, now: now}
}

// Scanner limits (PRD §12.1): thirty lookups a minute per staff member, and a device is blocked for
// fifteen minutes after ten failed lookups.
const (
	LookupsPerMinute  = 30
	FailedLookupLimit = 10
	BlockWindow       = 15 * time.Minute
)

// ErrTooManyAttempts reports a staff member who is over a scanner limit.
var ErrTooManyAttempts = errors.New("redemption: too many lookups, slow down")

// ErrScannerBlocked reports a staff member blocked after repeated failed code entries.
var ErrScannerBlocked = errors.New("redemption: scanner blocked after repeated failed lookups")

// Lookup resolves a voucher from a QR token or a human code. The result carries the effective status.
func (s *Service) Lookup(ctx context.Context, merchantID, staffID uuid.UUID, token, code string) (commerce.VoucherView, error) {
	now := s.now()
	stats, err := s.repo.AttemptStats(ctx, staffID, now.Add(-time.Minute), now.Add(-BlockWindow))
	if err != nil {
		return commerce.VoucherView{}, err
	}
	if stats.FailedRecent >= FailedLookupLimit {
		return commerce.VoucherView{}, ErrScannerBlocked
	}
	if stats.LastMinute >= LookupsPerMinute {
		return commerce.VoucherView{}, ErrTooManyAttempts
	}
	voucher, err := s.resolve(ctx, token, code)
	if err != nil {
		if errors.Is(err, ErrNotFound) || errors.Is(err, ErrInvalidToken) || errors.As(err, new(*merchant.InvalidInputError)) {
			_ = s.repo.RecordAttempt(ctx, staffID, true)
		}
		return commerce.VoucherView{}, err
	}
	_ = s.repo.RecordAttempt(ctx, staffID, false)
	if voucher.MerchantID != merchantID {
		if err := s.repo.RecordDenied(ctx, deniedScan(staffID, scannerRole, voucher.ID, merchantID, voucher.MerchantID)); err != nil {
			return commerce.VoucherView{}, err
		}
		return commerce.VoucherView{}, ErrWrongMerchant
	}
	return s.view(ctx, voucher)
}

// Redeem marks an active voucher as used at one of the merchant's branches.
// scannerRole is the role recorded for lookups, which carry only the staff id. Scanning is a cashier action.
const scannerRole = "merchant_cashier"

// deniedScan builds the audit row for a scan that was refused because the voucher belongs to another business.
func deniedScan(actorID uuid.UUID, role string, voucherID, merchantID, ownerID uuid.UUID) AuditEntry {
	return AuditEntry{
		ActorID:   actorID,
		Role:      role,
		VoucherID: voucherID,
		Action:    "voucher_scan_denied",
		After:     map[string]any{"scanning_merchant_id": merchantID.String(), "voucher_merchant_id": ownerID.String()},
	}
}

// Geo is the staff device position at redemption. It is optional and only feeds fraud rules.
type Geo struct {
	Lat float64
	Lng float64
}

func (s *Service) Redeem(ctx context.Context, merchantID uuid.UUID, actor merchant.Actor, voucherID, locationID uuid.UUID, geo *Geo) (commerce.VoucherView, error) {
	voucher, err := s.repo.Find(ctx, voucherID)
	if err != nil {
		return commerce.VoucherView{}, err
	}
	if voucher.MerchantID != merchantID {
		if err := s.repo.RecordDenied(ctx, deniedScan(actor.UserID, actor.Role, voucherID, merchantID, voucher.MerchantID)); err != nil {
			return commerce.VoucherView{}, err
		}
		return commerce.VoucherView{}, ErrWrongMerchant
	}
	owned, err := s.repo.LocationOwned(ctx, merchantID, locationID)
	if err != nil {
		return commerce.VoucherView{}, err
	}
	if !owned {
		return commerce.VoucherView{}, &merchant.InvalidInputError{Field: "location_id", Code: "location_not_found"}
	}
	at := s.now()
	audit := AuditEntry{
		ActorID:   actor.UserID,
		Role:      actor.Role,
		VoucherID: voucherID,
		Action:    "voucher_redeemed",
		After:     map[string]any{"location_id": locationID.String()},
	}
	redeemed, err := s.repo.Redeem(ctx, merchantID, voucherID, locationID, at, audit, geo)
	if err != nil {
		return commerce.VoucherView{}, err
	}
	if !redeemed {
		// Someone else changed the voucher between the read and the write; report why it cannot be used now.
		current, err := s.repo.Find(ctx, voucherID)
		if err != nil {
			return commerce.VoucherView{}, err
		}
		return commerce.VoucherView{}, unusable(current, at)
	}
	updated, err := s.repo.Find(ctx, voucherID)
	if err != nil {
		return commerce.VoucherView{}, err
	}
	return s.view(ctx, updated)
}

func (s *Service) resolve(ctx context.Context, token, code string) (commerce.Voucher, error) {
	switch {
	case token != "":
		id, err := s.verify(token)
		if err != nil {
			return commerce.Voucher{}, ErrInvalidToken
		}
		return s.repo.Find(ctx, id)
	case code != "":
		normalized := strings.ToUpper(strings.TrimSpace(code))
		if !codePattern.MatchString(normalized) {
			return commerce.Voucher{}, &merchant.InvalidInputError{Field: "code", Code: "invalid_code"}
		}
		voucher, err := s.repo.FindByCode(ctx, normalized)
		if err == nil || s.backupHash == nil {
			return voucher, err
		}
		if backup, backupErr := s.repo.FindByBackupCode(ctx, s.backupHash(normalized)); backupErr == nil {
			return backup, nil
		}
		return voucher, err
	default:
		return commerce.Voucher{}, &merchant.InvalidInputError{Field: "code", Code: "required"}
	}
}

// view attaches the deal and option and replaces the stored status with the effective one.
func (s *Service) view(ctx context.Context, voucher commerce.Voucher) (commerce.VoucherView, error) {
	view, err := s.repo.View(ctx, voucher)
	if err != nil {
		return commerce.VoucherView{}, err
	}
	view.Voucher.Status = EffectiveStatus(voucher, s.now())
	return view, nil
}

// EffectiveStatus returns the status staff should act on. An active voucher past its end date is expired.
func EffectiveStatus(voucher commerce.Voucher, at time.Time) string {
	if voucher.Status == StatusActive && !at.Before(voucher.ValidUntil) {
		return StatusExpired
	}
	return voucher.Status
}

func unusable(voucher commerce.Voucher, at time.Time) error {
	switch EffectiveStatus(voucher, at) {
	case StatusRedeemed:
		return ErrAlreadyRedeemed
	case StatusExpired:
		return ErrExpired
	case StatusRefunded:
		return ErrRefunded
	case StatusVoided:
		return ErrVoided
	default:
		return ErrNotFound
	}
}

// Reverse undoes the latest redemption of a voucher. The cashier who made it may do so within
// ReversalWindow; merchant owners and managers may do so at any time.
func (s *Service) Reverse(ctx context.Context, merchantID uuid.UUID, actor merchant.Actor, voucherID uuid.UUID, reason string) error {
	if actor.Role != merchant.RoleOwner && actor.Role != merchant.RoleManager && actor.Role != merchant.RoleCashier {
		return merchant.ErrForbidden
	}
	found, err := s.repo.ReversalCandidate(ctx, merchantID, voucherID)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return ErrNoRedemption
		}
		return err
	}
	now := s.now()
	cashierOwnsIt := found.StaffID == actor.UserID
	if actor.Role == merchant.RoleCashier && !cashierOwnsIt {
		return merchant.ErrForbidden
	}
	// PRD FR-05.5: the window is 10 minutes for the cashier; after it only support can reverse.
	if now.Sub(found.CreatedAt) > ReversalWindow {
		return ErrReversalWindowClosed
	}
	return s.repo.ReverseUse(ctx, merchantID, found, reason, AuditEntry{
		ActorID: actor.UserID,
		Role:    actor.Role,
		Action:  "voucher_redemption_reversed",
		After:   map[string]any{"reason": reason},
	})
}
