package commerce

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/dealo/dealo/services/api/internal/domain/catalog"
)

// Voucher status filters accepted by the account listing. Empty means all.
const (
	VoucherFilterAll     = ""
	VoucherFilterActive  = VoucherActive
	VoucherFilterUsed    = VoucherRedeemed
	VoucherFilterExpired = VoucherExpired
)

// VoucherView is a voucher with the option and deal it belongs to.
type VoucherView struct {
	Voucher Voucher
	Option  OptionRecord
	Deal    catalog.DealSummary
	// BackupCode is derived from the voucher and shown next to the QR for offline staff entry.
	BackupCode string
	// QRExpiresAt is when the current QR token rotates.
	QRExpiresAt time.Time
}

// ListVouchers returns one page of the user's vouchers with their deals.
func (s *Service) ListVouchers(ctx context.Context, userID uuid.UUID, status string, page, pageSize int) ([]VoucherView, int64, error) {
	switch status {
	case VoucherFilterAll, VoucherFilterActive, VoucherFilterUsed, VoucherFilterExpired:
	default:
		return nil, 0, invalid("status", "unsupported")
	}
	if page < 1 || pageSize < 1 || pageSize > 100 {
		return nil, 0, invalid("page", "out_of_range")
	}
	vouchers, total, err := s.repo.ListVouchers(ctx, userID, status, pageSize, (page-1)*pageSize)
	if err != nil {
		return nil, 0, err
	}
	views, err := s.voucherViews(ctx, vouchers)
	if err != nil {
		return nil, 0, err
	}
	return views, total, nil
}

// GetVoucher returns one voucher of the user. Other users' vouchers are reported as not found.
func (s *Service) GetVoucher(ctx context.Context, userID, voucherID uuid.UUID) (VoucherView, error) {
	voucher, err := s.repo.VoucherForUser(ctx, userID, voucherID)
	if err != nil {
		return VoucherView{}, err
	}
	views, err := s.voucherViews(ctx, []Voucher{voucher})
	if err != nil {
		return VoucherView{}, err
	}
	if len(views) != 1 {
		return VoucherView{}, ErrVoucherNotFound
	}
	return views[0], nil
}

// VerifyQRToken checks a scanned token. Redemption (not built yet) uses it to find the voucher.
func (s *Service) VerifyQRToken(token string) (QRClaims, error) {
	return s.signer.Verify(token)
}

func (s *Service) voucherViews(ctx context.Context, vouchers []Voucher) ([]VoucherView, error) {
	if len(vouchers) == 0 {
		return []VoucherView{}, nil
	}
	optionIDs := make([]uuid.UUID, 0, len(vouchers))
	dealIDs := make([]uuid.UUID, 0, len(vouchers))
	for _, voucher := range vouchers {
		optionIDs = append(optionIDs, voucher.OptionID)
		dealIDs = append(dealIDs, voucher.DealID)
	}
	records, err := s.repo.OptionRecords(ctx, optionIDs)
	if err != nil {
		return nil, fmt.Errorf("load voucher options: %w", err)
	}
	summaries, err := s.deals.DealSummaries(ctx, dealIDs)
	if err != nil {
		return nil, fmt.Errorf("load voucher deals: %w", err)
	}
	optionByID := indexRecords(records)
	dealByID := make(map[uuid.UUID]catalog.DealSummary, len(summaries))
	for _, summary := range summaries {
		dealByID[summary.ID] = summary
	}
	views := make([]VoucherView, 0, len(vouchers))
	for _, voucher := range vouchers {
		option, ok := optionByID[voucher.OptionID]
		if !ok {
			return nil, fmt.Errorf("voucher %s references a missing option", voucher.ID)
		}
		deal, ok := dealByID[voucher.DealID]
		if !ok {
			return nil, fmt.Errorf("voucher %s references a missing deal", voucher.ID)
		}
		// The stored token is only a seed; the live token rotates with the window.
		voucher.QRToken = s.signer.Sign(voucher.ID)
		views = append(views, VoucherView{
			Voucher:     voucher,
			Option:      option,
			Deal:        deal,
			BackupCode:  s.signer.BackupCode(voucher.ID),
			QRExpiresAt: s.signer.NextRotation(),
		})
	}
	return views, nil
}
