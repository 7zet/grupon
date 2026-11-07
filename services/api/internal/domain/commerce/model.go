package commerce

import (
	"time"

	"github.com/google/uuid"

	"github.com/grupon/grupon/services/api/internal/domain/catalog"
)

// Business rules and limits.
const (
	Currency          = catalog.Currency
	ReservationTTL    = 15 * time.Minute // PRD FR-03.4
	MaxLineQuantity   = 100
	MaxOrderItems     = 100
	MaxMoneyAmount    = int64(1_000_000_000_000_000) // 1e15 minor units; far above any real price
	MinIdempotencyKey = 16
	MaxIdempotencyKey = 200
	recentWindowSize  = 30
)

// Order statuses.
const (
	OrderCreated         = "created"
	OrderAwaitingPayment = "awaiting_payment"
	OrderPaid            = "paid"
	OrderCompleted       = "completed"
	OrderExpired         = "expired"
	OrderCancelled       = "cancelled"
	OrderRefunded        = "refunded"
)

// Payment statuses.
const (
	PaymentCreated   = "created"
	PaymentPending   = "pending"
	PaymentSucceeded = "succeeded"
	PaymentFailed    = "failed"
)

// Voucher statuses.
const (
	VoucherActive            = "active"
	VoucherPartiallyRedeemed = "partially_redeemed"
	VoucherRedeemed          = "redeemed"
	VoucherExpired           = "expired"
	VoucherRefunded          = "refunded"
	VoucherVoided            = "voided"
)

// OptionRecord is the purchase-relevant state of one deal option, read in one query.
type OptionRecord struct {
	ID                  uuid.UUID
	DealID              uuid.UUID
	Title               catalog.Text
	OriginalPrice       int64
	DealPrice           int64
	StockTotal          int
	StockSold           int
	StockReserved       int
	RedemptionsAllowed  int
	IsActive            bool
	DealStatus          string
	StartsAt            time.Time
	EndsAt              time.Time
	MaxPerUser          int
	VoucherValidDays    *int
	VoucherValidUntil   *time.Time
	MerchantID          uuid.UUID
	MerchantActive      bool
	CategoryPurchasable bool
	Terms               catalog.Text
}

// Available is the number of units that can still be reserved.
func (o OptionRecord) Available() int {
	return o.StockTotal - o.StockSold - o.StockReserved
}

// SellableAt reports whether a buyer may put this option in a cart or order now.
func (o OptionRecord) SellableAt(now time.Time) bool {
	return o.IsActive && o.MerchantActive && o.DealStatus == "active" &&
		!o.StartsAt.After(now) && o.EndsAt.After(now)
}

// VoucherWindow returns the validity of a voucher bought now.
//
// PRD FR-02.1 and GAP-17: a voucher expires at the earlier of the fixed end date and the purchase date
// plus N days, and the day-based end is the last moment of that day in Asia/Tashkent (AT-12).
func (o OptionRecord) VoucherWindow(now time.Time) (time.Time, time.Time, bool) {
	var candidates []time.Time
	if o.VoucherValidDays != nil {
		local := now.In(TashkentZone).AddDate(0, 0, *o.VoucherValidDays)
		candidates = append(candidates, endOfDayTashkent(local))
	}
	if o.VoucherValidUntil != nil {
		candidates = append(candidates, *o.VoucherValidUntil)
	}
	if len(candidates) == 0 {
		return now, now, false
	}
	validUntil := candidates[0]
	for _, c := range candidates[1:] {
		if c.Before(validUntil) {
			validUntil = c
		}
	}
	return now, validUntil, validUntil.After(now)
}

// TashkentZone is Asia/Tashkent. The country has no daylight saving time, so a fixed offset is exact.
var TashkentZone = time.FixedZone("Asia/Tashkent", 5*60*60)

func endOfDayTashkent(t time.Time) time.Time {
	y, m, d := t.Date()
	return time.Date(y, m, d, 23, 59, 59, 999999999, TashkentZone)
}

// OrderItemInput is one requested purchase line.
type OrderItemInput struct {
	OptionID uuid.UUID
	Quantity int
}

// OrderInput is a validated-at-the-edge checkout request.
type OrderInput struct {
	IdempotencyKey string
	Items          []OrderItemInput
}

// Order is a purchase with its items.
type Order struct {
	ID                   uuid.UUID
	Number               string
	UserID               uuid.UUID
	Status               string
	Subtotal             int64
	Total                int64
	PaidAt               *time.Time
	CreatedAt            time.Time
	ReservationExpiresAt *time.Time
	RequestHash          string
	Items                []OrderItem
}

// OrderItem is one purchased line with price snapshots.
type OrderItem struct {
	ID            uuid.UUID
	OrderID       uuid.UUID
	DealID        uuid.UUID
	OptionID      uuid.UUID
	Title         catalog.Text
	UnitPrice     int64
	OriginalPrice int64
	Terms         catalog.Text
	Quantity      int
}

// Reservation is a stock hold on one option for one order.
type Reservation struct {
	OptionID  uuid.UUID
	Quantity  int
	ExpiresAt time.Time
}

// Payment is one payment attempt for an order.
type Payment struct {
	ID                uuid.UUID
	OrderID           uuid.UUID
	Provider          string
	ProviderPaymentID string
	Status            string
	Amount            int64
}

// Voucher is one redeemable unit of a paid order item.
type Voucher struct {
	ID          uuid.UUID
	OrderItemID uuid.UUID
	// Seq is the one-based unit number inside an order item. It is immutable and unique per item.
	Seq        int
	UserID     uuid.UUID
	DealID     uuid.UUID
	OptionID   uuid.UUID
	MerchantID uuid.UUID
	Code       string
	QRToken    string
	Status     string
	ValidFrom  time.Time
	ValidUntil time.Time
	RedeemedAt *time.Time
	Terms      catalog.Text
	// RedemptionsAllowed is the use count copied from the option at purchase time.
	RedemptionsAllowed int
	// ZaxiraCodeHMAC is the stored hash of the backup code; the plain code is derived, not stored.
	ZaxiraCodeHMAC string
}

// Notification is one in-app message for a buyer.
type Notification struct {
	ID           uuid.UUID
	Kind         string
	VoucherID    *uuid.UUID
	MerchantName string
	CreatedAt    time.Time
	ReadAt       *time.Time
}

// PaymentSession is what the buyer receives when starting a payment.
type PaymentSession struct {
	PaymentID    uuid.UUID
	Provider     string
	Status       string
	RedirectURL  string
	Instructions string
}

// Contact is the user data the notifications need.
type Contact struct {
	Email  string
	Name   string
	Locale string
}
