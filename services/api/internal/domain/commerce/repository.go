package commerce

import (
	"context"
	"github.com/grupon/grupon/services/api/internal/domain/ledger"
	"time"

	"github.com/google/uuid"

	"github.com/grupon/grupon/services/api/internal/domain/catalog"
)

// Repository is the persistence port. Store methods used outside InTx run as single statements;
// every multi-step write runs inside InTx. Implementations live in internal/store.
type Repository interface {
	Store
	// InTx runs fn in one database transaction. The Store passed to fn is bound to that transaction.
	InTx(ctx context.Context, fn func(tx Store) error) error
}

// Store is the set of persistence operations commerce needs.
type Store interface {
	// Notifications returns the buyer's newest notifications.
	Notifications(ctx context.Context, userID uuid.UUID) ([]Notification, error)
	// Users and stock.
	LockUser(ctx context.Context, userID uuid.UUID) error
	UserContact(ctx context.Context, userID uuid.UUID) (Contact, error)
	OptionRecords(ctx context.Context, optionIDs []uuid.UUID) ([]OptionRecord, error)
	UserUnitsForDeal(ctx context.Context, userID, dealID uuid.UUID) (int, error)
	// ReserveStock moves quantity from available to reserved. It returns false when the conditional
	// update matched nothing, which means not enough stock or an inactive option.
	ReserveStock(ctx context.Context, optionID uuid.UUID, quantity int) (bool, error)
	ReleaseStock(ctx context.Context, optionID uuid.UUID, quantity int) error
	ConvertStock(ctx context.Context, optionID uuid.UUID, quantity int) error
	AddDealSoldCount(ctx context.Context, dealID uuid.UUID, quantity int) error

	// Orders.
	NextOrderNumber(ctx context.Context) (int64, error)
	OrderIDByIdempotencyKey(ctx context.Context, key string) (uuid.UUID, error)
	InsertOrder(ctx context.Context, order Order, idempotencyKey string) (Order, error)
	InsertOrderItem(ctx context.Context, orderID uuid.UUID, item OrderItem) (uuid.UUID, error)
	InsertReservation(ctx context.Context, orderID uuid.UUID, reservation Reservation) error
	OrderByID(ctx context.Context, orderID uuid.UUID) (Order, error)
	LockOrder(ctx context.Context, orderID uuid.UUID) (Order, error)
	ListOrders(ctx context.Context, userID uuid.UUID, limit, offset int) ([]Order, int64, error)
	UnreleasedReservations(ctx context.Context, orderID uuid.UUID) ([]Reservation, error)
	ExtendReservations(ctx context.Context, orderID uuid.UUID, expiresAt time.Time) error
	ReleaseReservations(ctx context.Context, orderID uuid.UUID) ([]Reservation, error)
	SetOrderStatus(ctx context.Context, orderID uuid.UUID, status string, paidAt *time.Time) error
	ExpiredUnpaidOrderIDs(ctx context.Context, now time.Time, batch int) ([]uuid.UUID, error)
	FailOpenPayments(ctx context.Context, orderID uuid.UUID) error

	// Payments.
	InsertPayment(ctx context.Context, payment Payment) error
	PaymentByID(ctx context.Context, paymentID uuid.UUID) (Payment, error)
	LockPaymentByProviderID(ctx context.Context, provider, providerPaymentID string) (Payment, error)
	SetPaymentStatus(ctx context.Context, paymentID uuid.UUID, status string) error
	RecordManualConfirmation(ctx context.Context, paymentID, actorID uuid.UUID, actorRole, reference, note string) error
	// RecordLateRefund writes the refund of a payment that arrived after its order expired (AT-13).
	RecordLateRefund(ctx context.Context, paymentID uuid.UUID, amount int64, reason string) error

	// Vouchers.
	VoucherCodeTaken(ctx context.Context, code string) (bool, error)
	InsertVoucher(ctx context.Context, voucher Voucher) error
	PostLedger(ctx context.Context, entry ledger.Entry) (bool, error)
	ListVouchers(ctx context.Context, userID uuid.UUID, status string, limit, offset int) ([]Voucher, int64, error)
	VoucherForUser(ctx context.Context, userID, voucherID uuid.UUID) (Voucher, error)
	ExpireDueVouchers(ctx context.Context, now time.Time, batch int) (int, error)

	// Engagement.
	RecordView(ctx context.Context, userID, dealID uuid.UUID) error
	RecentDealIDs(ctx context.Context, userID uuid.UUID) ([]uuid.UUID, error)
}

// DealCatalog loads card-level deal data for responses. The catalog repository satisfies it.
type DealCatalog interface {
	DealSummaries(ctx context.Context, ids []uuid.UUID) ([]catalog.DealSummary, error)
}
