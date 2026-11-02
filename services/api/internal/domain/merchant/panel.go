package merchant

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
)

// ReplyWindow is how long a merchant may edit a reply to a review (PRD §5: editable for 24 hours).
const ReplyWindow = 24 * time.Hour

// Panel errors. The HTTP layer maps them to problem codes.
var (
	ErrReviewNotFound = errors.New("merchant: review not found for this business")
	ErrReplyLocked    = errors.New("merchant: the reply can no longer be edited")
)

// PayoutRow is one payout run as the merchant sees it.
type PayoutRow struct {
	ID          uuid.UUID
	PeriodStart time.Time
	PeriodEnd   time.Time
	Gross       int64
	Commission  int64
	Net         int64
	Status      string
	PaidAt      *time.Time
}

// CustomerRow is one buyer of this business, with name shortened for the merchant.
type CustomerRow struct {
	UserID        uuid.UUID
	Name          string
	OrdersCount   int
	VouchersCount int
	TotalSpent    int64
	LastOrderAt   time.Time
}

// DayPoint is one day of sales.
type DayPoint struct {
	Date   time.Time
	Amount int64
}

// Analytics summarises sales for a period.
type Analytics struct {
	GrossSales     int64
	VouchersSold   int64
	VouchersRedeem int64
	RedemptionRate float64
	Sales          []DayPoint
}

// ReviewReply is the merchant's answer to a review.
type ReviewReply struct {
	ID        uuid.UUID
	Text      string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// ReviewRow is a review of this business, with its reply if any.
type ReviewRow struct {
	ID        uuid.UUID
	DealID    uuid.UUID
	UserID    uuid.UUID
	VoucherID uuid.UUID
	Rating    int
	Text      string
	Locale    string
	Status    string
	CreatedAt time.Time
	Reply     *ReviewReply
}

// OrderRow is an order that contains at least one deal of this business.
type OrderRow struct {
	ID        uuid.UUID
	Number    string
	Status    string
	Subtotal  int64
	Total     int64
	CreatedAt time.Time
}

// Settings are the merchant's notification and default-branch choices.
type Settings struct {
	OrderEmails       bool
	ReviewEmails      bool
	PayoutEmails      bool
	DefaultLocationID *uuid.UUID
}

// PanelRepository reads and writes the merchant panel data. Every method is scoped to one business.
type PanelRepository interface {
	Payouts(ctx context.Context, merchantID uuid.UUID) ([]PayoutRow, error)
	Customers(ctx context.Context, merchantID uuid.UUID) ([]CustomerRow, error)
	Analytics(ctx context.Context, merchantID uuid.UUID, from, to time.Time) (Analytics, error)
	Reviews(ctx context.Context, merchantID uuid.UUID) ([]ReviewRow, error)
	ReviewBelongs(ctx context.Context, merchantID, reviewID uuid.UUID) (bool, error)
	ReplyFor(ctx context.Context, reviewID uuid.UUID) (*ReviewReply, error)
	InsertReply(ctx context.Context, reviewID, userID uuid.UUID, text string) (ReviewReply, error)
	UpdateReply(ctx context.Context, reviewID uuid.UUID, text string) (ReviewReply, error)
	Orders(ctx context.Context, merchantID uuid.UUID) ([]OrderRow, error)
	Settings(ctx context.Context, merchantID uuid.UUID) (Settings, error)
	UpdateSettings(ctx context.Context, merchantID uuid.UUID, s Settings) error
}

// PanelService applies the rules of the merchant panel.
type PanelService struct {
	repo PanelRepository
	now  func() time.Time
}

// NewPanelService binds the panel rules to a repository.
func NewPanelService(repo PanelRepository, now func() time.Time) *PanelService {
	if now == nil {
		now = time.Now
	}
	return &PanelService{repo: repo, now: now}
}

// Payouts lists the payout runs of the business, newest first.
func (s *PanelService) Payouts(ctx context.Context, merchantID uuid.UUID) ([]PayoutRow, error) {
	return s.repo.Payouts(ctx, merchantID)
}

// Customers lists the buyers of the business with their totals.
func (s *PanelService) Customers(ctx context.Context, merchantID uuid.UUID) ([]CustomerRow, error) {
	rows, err := s.repo.Customers(ctx, merchantID)
	if err != nil {
		return nil, err
	}
	for i := range rows {
		rows[i].Name = MaskName(rows[i].Name)
	}
	return rows, nil
}

// Analytics summarises the period [from, to). The redemption rate is redeemed / sold, both in the period.
func (s *PanelService) Analytics(ctx context.Context, merchantID uuid.UUID, from, to time.Time) (Analytics, error) {
	if !from.Before(to) {
		return Analytics{}, &InvalidInputError{Field: "to", Code: "range"}
	}
	out, err := s.repo.Analytics(ctx, merchantID, from, to)
	if err != nil {
		return Analytics{}, err
	}
	if out.VouchersSold > 0 {
		out.RedemptionRate = float64(out.VouchersRedeem) / float64(out.VouchersSold)
	}
	return out, nil
}

// Reviews lists the reviews of the business with their replies.
func (s *PanelService) Reviews(ctx context.Context, merchantID uuid.UUID) ([]ReviewRow, error) {
	return s.repo.Reviews(ctx, merchantID)
}

// Reply writes the merchant's answer to a review. One reply per review; it can be edited for 24 hours.
// Cashiers cannot reply.
func (s *PanelService) Reply(ctx context.Context, actor Actor, merchantID, reviewID uuid.UUID, text string) (ReviewReply, error) {
	if actor.Role == RoleCashier {
		return ReviewReply{}, ErrForbidden
	}
	text = strings.TrimSpace(text)
	if n := len([]rune(text)); n < 1 || n > 2000 {
		return ReviewReply{}, &InvalidInputError{Field: "text", Code: "length"}
	}
	belongs, err := s.repo.ReviewBelongs(ctx, merchantID, reviewID)
	if err != nil {
		return ReviewReply{}, err
	}
	if !belongs {
		return ReviewReply{}, ErrReviewNotFound
	}
	existing, err := s.repo.ReplyFor(ctx, reviewID)
	if err != nil {
		return ReviewReply{}, err
	}
	if existing == nil {
		return s.repo.InsertReply(ctx, reviewID, actor.UserID, text)
	}
	if s.now().Sub(existing.CreatedAt) > ReplyWindow {
		return ReviewReply{}, ErrReplyLocked
	}
	return s.repo.UpdateReply(ctx, reviewID, text)
}

// Orders lists orders that include this business's deals, newest first.
func (s *PanelService) Orders(ctx context.Context, merchantID uuid.UUID) ([]OrderRow, error) {
	return s.repo.Orders(ctx, merchantID)
}

// Settings returns the notification and default-branch choices.
func (s *PanelService) Settings(ctx context.Context, merchantID uuid.UUID) (Settings, error) {
	return s.repo.Settings(ctx, merchantID)
}

// UpdateSettings stores the choices. Only the owner or a manager may change them; cashiers cannot.
func (s *PanelService) UpdateSettings(ctx context.Context, actor Actor, merchantID uuid.UUID, settings Settings) (Settings, error) {
	if actor.Role == RoleCashier {
		return Settings{}, ErrForbidden
	}
	if err := s.repo.UpdateSettings(ctx, merchantID, settings); err != nil {
		return Settings{}, err
	}
	return s.repo.Settings(ctx, merchantID)
}

// MaskName shortens a buyer name to "Alisher U." — the merchant sees a first name and an initial (PRD FR-05.3).
func MaskName(full string) string {
	fields := strings.Fields(full)
	switch len(fields) {
	case 0:
		return ""
	case 1:
		return fields[0]
	default:
		last := []rune(fields[len(fields)-1])
		return fields[0] + " " + string(last[0]) + "."
	}
}
