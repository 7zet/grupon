package commerce

import (
	"time"

	"github.com/google/uuid"

	"github.com/grupon/grupon/services/api/internal/domain/catalog"
)

// Business rules and limits.
const (
	Currency          = catalog.Currency
	ReservationTTL    = 10 * time.Minute
	MaxLineQuantity   = 100
	MaxOrderItems     = 100
	MaxCartItems      = 100
	MaxMoneyAmount    = int64(1_000_000_000_000_000) // 1e15 minor units; far above any real price
	MinIdempotencyKey = 16
	MaxIdempotencyKey = 200
	MaxGiftNameBytes  = 200
	MaxGiftMsgBytes   = 500
	recentWindowSize  = 30
)

// Order statuses.
const (
	OrderPendingPayment = "pending_payment"
	OrderPaid           = "paid"
	OrderCancelled      = "cancelled"
	OrderRefunded       = "refunded"
)

// Payment statuses.
const (
	PaymentCreated   = "created"
	PaymentPending   = "pending"
	PaymentSucceeded = "succeeded"
	PaymentFailed    = "failed"
	PaymentRefunded  = "refunded"
)

// Voucher statuses.
const (
	VoucherActive   = "active"
	VoucherRedeemed = "redeemed"
	VoucherExpired  = "expired"
)

// Deal kinds that change checkout rules.
const (
	KindVoucher = "voucher"
	KindProduct = "product"
)

// Fulfillment statuses for order items.
const (
	FulfillmentNotApplicable = "not_applicable"
	FulfillmentProcessing    = "processing"
)

// OptionRecord is the purchase-relevant state of one deal option, read in one query.
type OptionRecord struct {
	ID                uuid.UUID
	DealID            uuid.UUID
	Title             catalog.Text
	OriginalPrice     int64
	DealPrice         int64
	StockTotal        int
	StockSold         int
	StockReserved     int
	IsActive          bool
	DealStatus        string
	DealKind          string
	StartsAt          time.Time
	EndsAt            time.Time
	MaxPerUser        int
	IsGiftable        bool
	VoucherValidDays  *int
	VoucherValidUntil *time.Time
	MerchantID        uuid.UUID
	MerchantActive    bool
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
func (o OptionRecord) VoucherWindow(now time.Time) (time.Time, time.Time, bool) {
	validUntil := now
	switch {
	case o.VoucherValidDays != nil:
		validUntil = now.Add(time.Duration(*o.VoucherValidDays) * 24 * time.Hour)
	case o.VoucherValidUntil != nil:
		validUntil = *o.VoucherValidUntil
	default:
		return now, now, false
	}
	return now, validUntil, validUntil.After(now)
}

// CartLine is one stored cart row.
type CartLine struct {
	OptionID uuid.UUID
	Quantity int
	IsGift   bool
}

// CartItem is a cart line with its display data.
type CartItem struct {
	Line      CartLine
	Option    OptionRecord
	Deal      catalog.DealSummary
	LineTotal int64
}

// Cart is the server-side cart of one user.
type Cart struct {
	Items    []CartItem
	Subtotal int64
}

// OrderItemInput is one requested purchase line.
type OrderItemInput struct {
	OptionID           uuid.UUID
	Quantity           int
	IsGift             bool
	GiftRecipientName  *string
	GiftRecipientEmail *string
	GiftMessage        *string
}

// OrderInput is a validated-at-the-edge checkout request.
type OrderInput struct {
	IdempotencyKey  string
	Items           []OrderItemInput
	ShippingAddress map[string]any
}

// Order is a purchase with its items.
type Order struct {
	ID                   uuid.UUID
	Number               string
	UserID               uuid.UUID
	Status               string
	Subtotal             int64
	Total                int64
	ShippingAddress      map[string]any
	PaidAt               *time.Time
	CreatedAt            time.Time
	ReservationExpiresAt *time.Time
	RequestHash          string
	Items                []OrderItem
}

// OrderItem is one purchased line with price snapshots.
type OrderItem struct {
	ID                 uuid.UUID
	OrderID            uuid.UUID
	DealID             uuid.UUID
	OptionID           uuid.UUID
	Title              catalog.Text
	UnitPrice          int64
	OriginalPrice      int64
	Quantity           int
	IsGift             bool
	GiftRecipientName  *string
	GiftRecipientEmail *string
	GiftMessage        *string
	FulfillmentStatus  string
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
	UserID      uuid.UUID
	DealID      uuid.UUID
	OptionID    uuid.UUID
	MerchantID  uuid.UUID
	Code        string
	QRToken     string
	Status      string
	ValidFrom   time.Time
	ValidUntil  time.Time
	RedeemedAt  *time.Time
}

// PaymentSession is what the buyer receives when starting a payment.
type PaymentSession struct {
	PaymentID   uuid.UUID
	Provider    string
	Status      string
	RedirectURL string
}

// Contact is the user data the notifications need.
type Contact struct {
	Email  string
	Name   string
	Locale string
}
