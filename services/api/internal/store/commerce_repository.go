package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/grupon/grupon/services/api/internal/domain/ledger"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/grupon/grupon/services/api/internal/domain/catalog"
	"github.com/grupon/grupon/services/api/internal/domain/commerce"
	sqlc "github.com/grupon/grupon/services/api/internal/store/sqlc"
)

// CommerceRepository implements commerce.Repository on top of sqlc queries.
type CommerceRepository struct {
	commerceStore
	pool *pgxpool.Pool
}

// commerceStore implements commerce.Store over any sqlc DBTX: the pool for single statements, or a
// transaction inside InTx.
type commerceStore struct {
	q *sqlc.Queries
}

// NewCommerceRepository binds the repository to a connection pool.
func NewCommerceRepository(pool *pgxpool.Pool) *CommerceRepository {
	return &CommerceRepository{commerceStore: commerceStore{q: sqlc.New(pool)}, pool: pool}
}

var _ commerce.Repository = (*CommerceRepository)(nil)

// InTx runs fn inside a transaction. The transaction commits only when fn returns nil.
func (r *CommerceRepository) InTx(ctx context.Context, fn func(commerce.Store) error) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if err := fn(&commerceStore{q: sqlc.New(tx)}); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}
	return nil
}

func (s *commerceStore) LockUser(ctx context.Context, userID uuid.UUID) error {
	_, err := s.q.LockUserForUpdate(ctx, toPgUUID(userID))
	if errors.Is(err, pgx.ErrNoRows) {
		return commerce.ErrUserNotFound
	}
	if err != nil {
		return fmt.Errorf("lock user: %w", err)
	}
	return nil
}

func (s *commerceStore) UserContact(ctx context.Context, userID uuid.UUID) (commerce.Contact, error) {
	row, err := s.q.GetUserContact(ctx, toPgUUID(userID))
	if errors.Is(err, pgx.ErrNoRows) {
		return commerce.Contact{}, commerce.ErrUserNotFound
	}
	if err != nil {
		return commerce.Contact{}, fmt.Errorf("load contact: %w", err)
	}
	return commerce.Contact{Email: row.Email, Name: row.Name, Locale: row.Locale}, nil
}

func (s *commerceStore) OptionRecords(ctx context.Context, optionIDs []uuid.UUID) ([]commerce.OptionRecord, error) {
	if len(optionIDs) == 0 {
		return nil, nil
	}
	rows, err := s.q.GetDealOptionRows(ctx, toPgUUIDs(optionIDs))
	if err != nil {
		return nil, fmt.Errorf("load options: %w", err)
	}
	records := make([]commerce.OptionRecord, 0, len(rows))
	for _, row := range rows {
		title, err := catalog.ParseText(row.OptionTitle)
		if err != nil {
			return nil, err
		}
		terms, err := catalog.ParseText(row.Terms)
		if err != nil {
			return nil, err
		}
		records = append(records, commerce.OptionRecord{
			ID:                  toUUID(row.OptionID),
			DealID:              toUUID(row.DealID),
			Title:               title,
			OriginalPrice:       row.OriginalPrice,
			DealPrice:           row.DealPrice,
			StockTotal:          int(row.StockTotal),
			RedemptionsAllowed:  int(row.RedemptionsAllowed),
			StockSold:           int(row.StockSold),
			StockReserved:       int(row.StockReserved),
			IsActive:            row.OptionIsActive,
			DealStatus:          string(row.DealStatus),
			StartsAt:            row.StartsAt.Time,
			EndsAt:              row.EndsAt.Time,
			MaxPerUser:          int(row.MaxPerUser),
			VoucherValidDays:    int32PtrToInt(row.VoucherValidDays),
			VoucherValidUntil:   timePtr(row.VoucherValidUntil),
			MerchantID:          toUUID(row.MerchantID),
			MerchantActive:      row.MerchantActive,
			CategoryPurchasable: row.CategoryPurchasable,
			Terms:               terms,
		})
	}
	return records, nil
}

func (s *commerceStore) UserUnitsForDeal(ctx context.Context, userID, dealID uuid.UUID) (int, error) {
	units, err := s.q.UserUnitsForDeal(ctx, sqlc.UserUnitsForDealParams{UserID: toPgUUID(userID), DealID: toPgUUID(dealID)})
	if err != nil {
		return 0, fmt.Errorf("count purchased units: %w", err)
	}
	return int(units), nil
}

func (s *commerceStore) ReserveStock(ctx context.Context, optionID uuid.UUID, quantity int) (bool, error) {
	_, err := s.q.ReserveOptionStock(ctx, sqlc.ReserveOptionStockParams{Quantity: int32(quantity), OptionID: toPgUUID(optionID)})
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("reserve stock: %w", err)
	}
	return true, nil
}

func (s *commerceStore) ReleaseStock(ctx context.Context, optionID uuid.UUID, quantity int) error {
	return s.q.ReleaseOptionStock(ctx, sqlc.ReleaseOptionStockParams{Quantity: int32(quantity), OptionID: toPgUUID(optionID)})
}

func (s *commerceStore) ConvertStock(ctx context.Context, optionID uuid.UUID, quantity int) error {
	return s.q.ConvertReservedStock(ctx, sqlc.ConvertReservedStockParams{Quantity: int32(quantity), OptionID: toPgUUID(optionID)})
}

func (s *commerceStore) AddDealSoldCount(ctx context.Context, dealID uuid.UUID, quantity int) error {
	return s.q.AddDealSoldCount(ctx, sqlc.AddDealSoldCountParams{Quantity: int32(quantity), DealID: toPgUUID(dealID)})
}

func (s *commerceStore) NextOrderNumber(ctx context.Context) (int64, error) {
	value, err := s.q.NextOrderNumberSequence(ctx)
	if err != nil {
		return 0, fmt.Errorf("next order number: %w", err)
	}
	return value, nil
}

func (s *commerceStore) OrderIDByIdempotencyKey(ctx context.Context, key string) (uuid.UUID, error) {
	id, err := s.q.GetOrderIDByIdempotencyKey(ctx, key)
	if errors.Is(err, pgx.ErrNoRows) {
		return uuid.Nil, commerce.ErrNotFound
	}
	if err != nil {
		return uuid.Nil, fmt.Errorf("find idempotent order: %w", err)
	}
	return toUUID(id), nil
}

func (s *commerceStore) InsertOrder(ctx context.Context, order commerce.Order, idempotencyKey string) (commerce.Order, error) {
	row, err := s.q.InsertOrder(ctx, sqlc.InsertOrderParams{
		Number:         order.Number,
		UserID:         toPgUUID(order.UserID),
		Subtotal:       order.Subtotal,
		Total:          order.Total,
		IdempotencyKey: idempotencyKey,
		RequestHash:    optText(order.RequestHash),
	})
	if err != nil {
		return commerce.Order{}, fmt.Errorf("insert order: %w", err)
	}
	order.ID = toUUID(row.ID)
	order.CreatedAt = row.CreatedAt.Time
	order.Status = commerce.OrderAwaitingPayment
	return order, nil
}

func (s *commerceStore) InsertOrderItem(ctx context.Context, orderID uuid.UUID, item commerce.OrderItem) (uuid.UUID, error) {
	title, err := json.Marshal(item.Title)
	if err != nil {
		return uuid.Nil, fmt.Errorf("encode item title: %w", err)
	}
	terms, err := json.Marshal(item.Terms)
	if err != nil {
		return uuid.Nil, fmt.Errorf("encode item terms: %w", err)
	}
	id, err := s.q.InsertOrderItem(ctx, sqlc.InsertOrderItemParams{
		OrderID:       toPgUUID(orderID),
		DealID:        toPgUUID(item.DealID),
		OptionID:      toPgUUID(item.OptionID),
		TitleSnapshot: title,
		UnitPrice:     item.UnitPrice,
		OriginalPrice: item.OriginalPrice,
		TermsSnapshot: terms,
		Quantity:      int32(item.Quantity),
	})
	if err != nil {
		return uuid.Nil, fmt.Errorf("insert order item: %w", err)
	}
	return toUUID(id), nil
}

func (s *commerceStore) InsertReservation(ctx context.Context, orderID uuid.UUID, reservation commerce.Reservation) error {
	return s.q.InsertStockReservation(ctx, sqlc.InsertStockReservationParams{
		OptionID:  toPgUUID(reservation.OptionID),
		OrderID:   toPgUUID(orderID),
		Quantity:  int32(reservation.Quantity),
		ExpiresAt: pgtype.Timestamptz{Time: reservation.ExpiresAt, Valid: true},
	})
}

func (s *commerceStore) OrderByID(ctx context.Context, orderID uuid.UUID) (commerce.Order, error) {
	row, err := s.q.GetOrderByID(ctx, toPgUUID(orderID))
	if errors.Is(err, pgx.ErrNoRows) {
		return commerce.Order{}, commerce.ErrOrderNotFound
	}
	if err != nil {
		return commerce.Order{}, fmt.Errorf("load order: %w", err)
	}
	order, err := orderFromFields(row.ID, row.Number, row.UserID, row.Status, row.Subtotal, row.Total,
		row.PaidAt, row.CreatedAt, row.RequestHash.String)
	if err != nil {
		return commerce.Order{}, err
	}
	order.ReservationExpiresAt = timePtr(row.ReservationExpiresAt)
	items, err := s.orderItems(ctx, []uuid.UUID{order.ID})
	if err != nil {
		return commerce.Order{}, err
	}
	order.Items = items[order.ID]
	return order, nil
}

func (s *commerceStore) LockOrder(ctx context.Context, orderID uuid.UUID) (commerce.Order, error) {
	row, err := s.q.LockOrderByID(ctx, toPgUUID(orderID))
	if errors.Is(err, pgx.ErrNoRows) {
		return commerce.Order{}, commerce.ErrOrderNotFound
	}
	if err != nil {
		return commerce.Order{}, fmt.Errorf("lock order: %w", err)
	}
	return orderFromFields(row.ID, row.Number, row.UserID, row.Status, row.Subtotal, row.Total,
		row.PaidAt, row.CreatedAt, "")
}

func (s *commerceStore) ListOrders(ctx context.Context, userID uuid.UUID, limit, offset int) ([]commerce.Order, int64, error) {
	total, err := s.q.CountOrdersForUser(ctx, toPgUUID(userID))
	if err != nil {
		return nil, 0, fmt.Errorf("count orders: %w", err)
	}
	ids, err := s.q.ListOrderIDsForUser(ctx, sqlc.ListOrderIDsForUserParams{
		UserID:     toPgUUID(userID),
		PageSize:   int32(limit),
		PageOffset: int32(offset),
	})
	if err != nil {
		return nil, 0, fmt.Errorf("list orders: %w", err)
	}
	orders := make([]commerce.Order, 0, len(ids))
	orderIDs := make([]uuid.UUID, 0, len(ids))
	for _, id := range ids {
		orderIDs = append(orderIDs, toUUID(id))
	}
	if len(orderIDs) == 0 {
		return orders, total, nil
	}
	items, err := s.orderItems(ctx, orderIDs)
	if err != nil {
		return nil, 0, err
	}
	for _, orderID := range orderIDs {
		order, err := s.OrderByID(ctx, orderID)
		if err != nil {
			return nil, 0, err
		}
		order.Items = items[orderID]
		orders = append(orders, order)
	}
	return orders, total, nil
}

// orderItems loads the items of several orders in one query and groups them by order.
func (s *commerceStore) orderItems(ctx context.Context, orderIDs []uuid.UUID) (map[uuid.UUID][]commerce.OrderItem, error) {
	rows, err := s.q.ListOrderItemsForOrders(ctx, toPgUUIDs(orderIDs))
	if err != nil {
		return nil, fmt.Errorf("load order items: %w", err)
	}
	grouped := make(map[uuid.UUID][]commerce.OrderItem, len(orderIDs))
	for _, row := range rows {
		title, err := catalog.ParseText(row.TitleSnapshot)
		if err != nil {
			return nil, err
		}
		terms, err := catalog.ParseText(row.TermsSnapshot)
		if err != nil {
			return nil, err
		}
		orderID := toUUID(row.OrderID)
		grouped[orderID] = append(grouped[orderID], commerce.OrderItem{
			ID:            toUUID(row.ID),
			OrderID:       orderID,
			DealID:        toUUID(row.DealID),
			OptionID:      toUUID(row.DealOptionID),
			Title:         title,
			UnitPrice:     row.UnitPriceSnapshot,
			OriginalPrice: row.OriginalPriceSnapshot,
			Terms:         terms,
			Quantity:      int(row.Quantity),
		})
	}
	return grouped, nil
}

func (s *commerceStore) UnreleasedReservations(ctx context.Context, orderID uuid.UUID) ([]commerce.Reservation, error) {
	rows, err := s.q.ListUnreleasedReservations(ctx, toPgUUID(orderID))
	if err != nil {
		return nil, fmt.Errorf("list reservations: %w", err)
	}
	reservations := make([]commerce.Reservation, 0, len(rows))
	for _, row := range rows {
		reservations = append(reservations, commerce.Reservation{
			OptionID:  toUUID(row.DealOptionID),
			Quantity:  int(row.Quantity),
			ExpiresAt: row.ExpiresAt.Time,
		})
	}
	return reservations, nil
}

func (s *commerceStore) ExtendReservations(ctx context.Context, orderID uuid.UUID, expiresAt time.Time) error {
	return s.q.ExtendReservations(ctx, sqlc.ExtendReservationsParams{OrderID: toPgUUID(orderID), ExpiresAt: pgtype.Timestamptz{Time: expiresAt, Valid: true}})
}

func (s *commerceStore) ReleaseReservations(ctx context.Context, orderID uuid.UUID) ([]commerce.Reservation, error) {
	rows, err := s.q.MarkReservationsReleased(ctx, toPgUUID(orderID))
	if err != nil {
		return nil, fmt.Errorf("release reservations: %w", err)
	}
	reservations := make([]commerce.Reservation, 0, len(rows))
	for _, row := range rows {
		reservations = append(reservations, commerce.Reservation{OptionID: toUUID(row.DealOptionID), Quantity: int(row.Quantity)})
	}
	return reservations, nil
}

func (s *commerceStore) SetOrderStatus(ctx context.Context, orderID uuid.UUID, status string, paidAt *time.Time) error {
	return s.q.SetOrderStatus(ctx, sqlc.SetOrderStatusParams{
		Status:  sqlc.OrderStatus(status),
		PaidAt:  optTimestamp(paidAt),
		OrderID: toPgUUID(orderID),
	})
}

func (s *commerceStore) ExpiredUnpaidOrderIDs(ctx context.Context, now time.Time, batch int) ([]uuid.UUID, error) {
	ids, err := s.q.ExpiredUnpaidOrderIDs(ctx, sqlc.ExpiredUnpaidOrderIDsParams{
		Now:       pgtype.Timestamptz{Time: now, Valid: true},
		BatchSize: int32(batch),
	})
	if err != nil {
		return nil, fmt.Errorf("list expired orders: %w", err)
	}
	return toUUIDs(ids), nil
}

func (s *commerceStore) FailOpenPayments(ctx context.Context, orderID uuid.UUID) error {
	_, err := s.q.FailOpenPayments(ctx, toPgUUID(orderID))
	return err
}

func (s *commerceStore) InsertPayment(ctx context.Context, payment commerce.Payment) error {
	return s.q.InsertPayment(ctx, sqlc.InsertPaymentParams{
		ID:                toPgUUID(payment.ID),
		OrderID:           toPgUUID(payment.OrderID),
		Provider:          sqlc.PaymentProvider(payment.Provider),
		ProviderPaymentID: optText(payment.ProviderPaymentID),
		Status:            sqlc.PaymentStatus(payment.Status),
		Amount:            payment.Amount,
		RawPayload:        []byte("{}"),
	})
}

func (s *commerceStore) PaymentByID(ctx context.Context, paymentID uuid.UUID) (commerce.Payment, error) {
	row, err := s.q.GetPaymentByID(ctx, toPgUUID(paymentID))
	if errors.Is(err, pgx.ErrNoRows) {
		return commerce.Payment{}, commerce.ErrPaymentNotFound
	}
	if err != nil {
		return commerce.Payment{}, fmt.Errorf("load payment: %w", err)
	}
	return commerce.Payment{
		ID:                toUUID(row.ID),
		OrderID:           toUUID(row.OrderID),
		Provider:          row.Provider,
		ProviderPaymentID: row.ProviderPaymentID.String,
		Status:            row.Status,
		Amount:            row.Amount,
	}, nil
}

func (s *commerceStore) LockPaymentByProviderID(ctx context.Context, provider, providerPaymentID string) (commerce.Payment, error) {
	row, err := s.q.LockPaymentByProviderID(ctx, sqlc.LockPaymentByProviderIDParams{
		Provider:          sqlc.PaymentProvider(provider),
		ProviderPaymentID: optText(providerPaymentID),
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return commerce.Payment{}, commerce.ErrPaymentNotFound
	}
	if err != nil {
		return commerce.Payment{}, fmt.Errorf("lock payment: %w", err)
	}
	return commerce.Payment{
		ID:                toUUID(row.ID),
		OrderID:           toUUID(row.OrderID),
		Provider:          row.Provider,
		ProviderPaymentID: row.ProviderPaymentID.String,
		Status:            row.Status,
		Amount:            row.Amount,
	}, nil
}

func (s *commerceStore) RecordLateRefund(ctx context.Context, paymentID uuid.UUID, amount int64, reason string) error {
	return s.q.InsertLateRefund(ctx, sqlc.InsertLateRefundParams{PaymentID: toPgUUID(paymentID), Amount: amount, Reason: reason})
}

func (s *commerceStore) SetPaymentStatus(ctx context.Context, paymentID uuid.UUID, status string) error {
	return s.q.SetPaymentStatus(ctx, sqlc.SetPaymentStatusParams{
		Status:     sqlc.PaymentStatus(status),
		RawPayload: nil,
		ID:         toPgUUID(paymentID),
	})
}

func (s *commerceStore) RecordManualConfirmation(ctx context.Context, paymentID, actorID uuid.UUID, actorRole, reference, note string) error {
	return s.q.RecordManualPaymentConfirmation(ctx, sqlc.RecordManualPaymentConfirmationParams{PaymentID: toPgUUID(paymentID), ActorID: toPgUUID(actorID), ActorRole: sqlc.AppRole(actorRole), Reference: reference, Note: note})
}

func (s *commerceStore) VoucherCodeTaken(ctx context.Context, code string) (bool, error) {
	taken, err := s.q.VoucherCodeExists(ctx, code)
	if err != nil {
		return false, fmt.Errorf("check voucher code: %w", err)
	}
	return taken, nil
}

func (s *commerceStore) InsertVoucher(ctx context.Context, voucher commerce.Voucher) error {
	return s.q.InsertVoucher(ctx, sqlc.InsertVoucherParams{
		ID:                 toPgUUID(voucher.ID),
		OrderItemID:        toPgUUID(voucher.OrderItemID),
		Seq:                int32(voucher.Seq),
		UserID:             toPgUUID(voucher.UserID),
		DealID:             toPgUUID(voucher.DealID),
		OptionID:           toPgUUID(voucher.OptionID),
		MerchantID:         toPgUUID(voucher.MerchantID),
		Code:               voucher.Code,
		QrToken:            voucher.QRToken,
		ValidFrom:          pgtype.Timestamptz{Time: voucher.ValidFrom, Valid: true},
		ValidUntil:         pgtype.Timestamptz{Time: voucher.ValidUntil, Valid: true},
		RedemptionsAllowed: int32(max(1, voucher.RedemptionsAllowed)),
		ZaxiraCodeHmac:     pgtype.Text{String: voucher.ZaxiraCodeHMAC, Valid: voucher.ZaxiraCodeHMAC != ""},
	})
}

func (s *commerceStore) Notifications(ctx context.Context, userID uuid.UUID) ([]commerce.Notification, error) {
	rows, err := s.q.ListBuyerNotifications(ctx, toPgUUID(userID))
	if err != nil {
		return nil, fmt.Errorf("list notifications: %w", err)
	}
	out := make([]commerce.Notification, 0, len(rows))
	for _, row := range rows {
		item := commerce.Notification{
			ID:           toUUID(row.ID),
			Kind:         row.Kind,
			MerchantName: row.MerchantName,
			CreatedAt:    row.CreatedAt.Time,
			ReadAt:       timePtr(row.ReadAt),
		}
		if row.VoucherID.Valid {
			id := toUUID(row.VoucherID)
			item.VoucherID = &id
		}
		out = append(out, item)
	}
	return out, nil
}

func (s *commerceStore) ListVouchers(ctx context.Context, userID uuid.UUID, status string, limit, offset int) ([]commerce.Voucher, int64, error) {
	filter := optText(status)
	total, err := s.q.CountVouchersForUser(ctx, sqlc.CountVouchersForUserParams{UserID: toPgUUID(userID), Status: filter})
	if err != nil {
		return nil, 0, fmt.Errorf("count vouchers: %w", err)
	}
	rows, err := s.q.ListVouchersForUser(ctx, sqlc.ListVouchersForUserParams{
		UserID:     toPgUUID(userID),
		Status:     filter,
		PageSize:   int32(limit),
		PageOffset: int32(offset),
	})
	if err != nil {
		return nil, 0, fmt.Errorf("list vouchers: %w", err)
	}
	vouchers := make([]commerce.Voucher, 0, len(rows))
	for _, row := range rows {
		voucher := voucherFromFields(row.ID, row.OrderItemID, row.UserID, row.DealID, row.DealOptionID,
			row.MerchantID, row.Code, row.QrToken, row.Status, row.ValidFrom, row.ValidUntil, row.RedeemedAt)
		voucher.Terms, err = catalog.ParseText(row.TermsSnapshot)
		if err != nil {
			return nil, 0, err
		}
		vouchers = append(vouchers, voucher)
	}
	return vouchers, total, nil
}

func (s *commerceStore) VoucherForUser(ctx context.Context, userID, voucherID uuid.UUID) (commerce.Voucher, error) {
	row, err := s.q.GetVoucherForUser(ctx, sqlc.GetVoucherForUserParams{ID: toPgUUID(voucherID), UserID: toPgUUID(userID)})
	if errors.Is(err, pgx.ErrNoRows) {
		return commerce.Voucher{}, commerce.ErrVoucherNotFound
	}
	if err != nil {
		return commerce.Voucher{}, fmt.Errorf("load voucher: %w", err)
	}
	voucher := voucherFromFields(row.ID, row.OrderItemID, row.UserID, row.DealID, row.DealOptionID,
		row.MerchantID, row.Code, row.QrToken, row.Status, row.ValidFrom, row.ValidUntil, row.RedeemedAt)
	voucher.Terms, err = catalog.ParseText(row.TermsSnapshot)
	return voucher, err
}

func (s *commerceStore) ExpireDueVouchers(ctx context.Context, now time.Time, batch int) (int, error) {
	ids, err := s.q.ExpireDueVouchers(ctx, sqlc.ExpireDueVouchersParams{
		Now:       pgtype.Timestamptz{Time: now, Valid: true},
		BatchSize: int32(batch),
	})
	if err != nil {
		return 0, fmt.Errorf("expire vouchers: %w", err)
	}
	return len(ids), nil
}

func (s *commerceStore) RecordView(ctx context.Context, userID, dealID uuid.UUID) error {
	if err := s.q.UpsertRecentlyViewed(ctx, sqlc.UpsertRecentlyViewedParams{UserID: toPgUUID(userID), DealID: toPgUUID(dealID)}); err != nil {
		return fmt.Errorf("record view: %w", err)
	}
	if err := s.q.TrimRecentlyViewed(ctx, toPgUUID(userID)); err != nil {
		return fmt.Errorf("trim recently viewed: %w", err)
	}
	return nil
}

func (s *commerceStore) RecentDealIDs(ctx context.Context, userID uuid.UUID) ([]uuid.UUID, error) {
	ids, err := s.q.ListRecentlyViewedDealIDs(ctx, toPgUUID(userID))
	if err != nil {
		return nil, fmt.Errorf("list recently viewed: %w", err)
	}
	return toUUIDs(ids), nil
}

func orderFromFields(id pgtype.UUID, number string, userID pgtype.UUID, status string, subtotal, total int64,
	paidAt, createdAt pgtype.Timestamptz, requestHash string) (commerce.Order, error) {
	return commerce.Order{
		ID:          toUUID(id),
		Number:      number,
		UserID:      toUUID(userID),
		Status:      status,
		Subtotal:    subtotal,
		Total:       total,
		PaidAt:      timePtr(paidAt),
		CreatedAt:   createdAt.Time,
		RequestHash: requestHash,
	}, nil
}

func voucherFromFields(id, orderItemID, userID, dealID, optionID, merchantID pgtype.UUID, code, qrToken, status string,
	validFrom, validUntil, redeemedAt pgtype.Timestamptz) commerce.Voucher {
	return commerce.Voucher{
		ID:          toUUID(id),
		OrderItemID: toUUID(orderItemID),
		UserID:      toUUID(userID),
		DealID:      toUUID(dealID),
		OptionID:    toUUID(optionID),
		MerchantID:  toUUID(merchantID),
		Code:        code,
		QRToken:     qrToken,
		Status:      status,
		ValidFrom:   validFrom.Time,
		ValidUntil:  validUntil.Time,
		RedeemedAt:  timePtr(redeemedAt),
	}
}

func optTextPtr(value *string) pgtype.Text {
	if value == nil {
		return pgtype.Text{}
	}
	return pgtype.Text{String: *value, Valid: true}
}

func optTimestamp(value *time.Time) pgtype.Timestamptz {
	if value == nil {
		return pgtype.Timestamptz{}
	}
	return pgtype.Timestamptz{Time: *value, Valid: true}
}

// PostLedger writes a balanced journal entry inside the current transaction.
func (s *commerceStore) PostLedger(ctx context.Context, entry ledger.Entry) (bool, error) {
	return PostEntry(ctx, s.q, entry)
}
