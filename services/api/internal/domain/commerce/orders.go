package commerce

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"sort"

	"github.com/google/uuid"
)

// CreateOrder turns checkout input into a pending order and ten-minute stock reservations in one
// transaction. The boolean reports an idempotent replay of an earlier order with the same key.
func (s *Service) CreateOrder(ctx context.Context, userID uuid.UUID, input OrderInput) (Order, bool, error) {
	if err := ValidateIdempotencyKey(input.IdempotencyKey); err != nil {
		return Order{}, false, err
	}
	if len(input.Items) == 0 {
		return Order{}, false, invalid("items", "required")
	}
	if len(input.Items) > MaxOrderItems {
		return Order{}, false, invalid("items", "too_many")
	}
	for i, item := range input.Items {
		field := fmt.Sprintf("items[%d]", i)
		if err := ValidateQuantity(field+".quantity", item.Quantity); err != nil {
			return Order{}, false, err
		}
	}
	fingerprint, err := RequestFingerprint(input)
	if err != nil {
		return Order{}, false, err
	}
	// The key is scoped to the user so that two users never collide on the same header value.
	scopedKey := userID.String() + ":" + input.IdempotencyKey

	var (
		order    Order
		replayed bool
	)
	err = s.repo.InTx(ctx, func(tx Store) error {
		if err := tx.LockUser(ctx, userID); err != nil {
			return err
		}
		existingID, err := tx.OrderIDByIdempotencyKey(ctx, scopedKey)
		switch {
		case err == nil:
			existing, err := tx.OrderByID(ctx, existingID)
			if err != nil {
				return err
			}
			if existing.RequestHash != fingerprint {
				return ErrIdempotencyConflict
			}
			order, replayed = existing, true
			return nil
		case !errors.Is(err, ErrNotFound):
			return err
		}
		created, err := s.placeOrder(ctx, tx, userID, input, fingerprint, scopedKey)
		if err != nil {
			return err
		}
		order = created
		return nil
	})
	if err != nil {
		return Order{}, false, err
	}
	return order, replayed, nil
}

// placeOrder runs inside the create-order transaction. The user row is already locked by the caller,
// which serializes checkouts of one buyer and makes the per-deal limit check race-free.
func (s *Service) placeOrder(ctx context.Context, tx Store, userID uuid.UUID, input OrderInput, fingerprint, scopedKey string) (Order, error) {
	now := s.now()
	ids := itemOptionIDs(input.Items)
	records, err := tx.OptionRecords(ctx, ids)
	if err != nil {
		return Order{}, fmt.Errorf("load options: %w", err)
	}
	byID := indexRecords(records)

	quantityByOption := make(map[uuid.UUID]int, len(ids))
	unitsByDeal := make(map[uuid.UUID]int)
	maxByDeal := make(map[uuid.UUID]int)
	for _, item := range input.Items {
		record, ok := byID[item.OptionID]
		if !ok || !record.SellableAt(now) {
			return Order{}, ErrOptionUnavailable
		}
		if _, _, valid := record.VoucherWindow(now); !valid {
			return Order{}, ErrOptionUnavailable
		}
		quantityByOption[item.OptionID] += item.Quantity
		unitsByDeal[record.DealID] += item.Quantity
		maxByDeal[record.DealID] = record.MaxPerUser
	}
	for dealID, units := range unitsByDeal {
		purchased, err := tx.UserUnitsForDeal(ctx, userID, dealID)
		if err != nil {
			return Order{}, fmt.Errorf("count purchased units: %w", err)
		}
		if purchased+units > maxByDeal[dealID] {
			return Order{}, ErrMaxPerUserExceeded
		}
	}

	var subtotal, total int64
	for _, item := range input.Items {
		record := byID[item.OptionID]
		original, err := LineTotal(record.OriginalPrice, item.Quantity)
		if err != nil {
			return Order{}, err
		}
		paid, err := LineTotal(record.DealPrice, item.Quantity)
		if err != nil {
			return Order{}, err
		}
		subtotal += original
		total += paid
	}

	sequence, err := tx.NextOrderNumber(ctx)
	if err != nil {
		return Order{}, fmt.Errorf("next order number: %w", err)
	}
	expiresAt := now.Add(ReservationTTL)

	// Reserve in a fixed order so that two checkouts touching the same options cannot deadlock.
	for _, optionID := range sortedOptionIDs(quantityByOption) {
		reserved, err := tx.ReserveStock(ctx, optionID, quantityByOption[optionID])
		if err != nil {
			return Order{}, fmt.Errorf("reserve stock: %w", err)
		}
		if !reserved {
			return Order{}, ErrInsufficientStock
		}
	}

	order, err := tx.InsertOrder(ctx, Order{
		Number:      FormatOrderNumber(now.Year(), sequence),
		UserID:      userID,
		Status:      OrderAwaitingPayment,
		Subtotal:    subtotal,
		Total:       total,
		RequestHash: fingerprint,
	}, scopedKey)
	if err != nil {
		return Order{}, fmt.Errorf("insert order: %w", err)
	}

	for _, item := range input.Items {
		record := byID[item.OptionID]
		line := OrderItem{
			OrderID:       order.ID,
			DealID:        record.DealID,
			OptionID:      record.ID,
			Title:         record.Title,
			UnitPrice:     record.DealPrice,
			OriginalPrice: record.OriginalPrice,
			Terms:         record.Terms,
			Quantity:      item.Quantity,
		}
		line.ID, err = tx.InsertOrderItem(ctx, order.ID, line)
		if err != nil {
			return Order{}, fmt.Errorf("insert order item: %w", err)
		}
		order.Items = append(order.Items, line)
	}
	for _, optionID := range sortedOptionIDs(quantityByOption) {
		reservation := Reservation{OptionID: optionID, Quantity: quantityByOption[optionID], ExpiresAt: expiresAt}
		if err := tx.InsertReservation(ctx, order.ID, reservation); err != nil {
			return Order{}, fmt.Errorf("insert reservation: %w", err)
		}
	}
	order.ReservationExpiresAt = &expiresAt
	order.RequestHash = fingerprint
	return order, nil
}

// GetOrder returns one order of the user. Other users' orders are reported as not found.
func (s *Service) GetOrder(ctx context.Context, userID, orderID uuid.UUID) (Order, error) {
	order, err := s.repo.OrderByID(ctx, orderID)
	if err != nil {
		return Order{}, err
	}
	if order.UserID != userID {
		return Order{}, ErrOrderNotFound
	}
	return order, nil
}

// ListOrders returns one page of the user's orders, newest first.
func (s *Service) ListOrders(ctx context.Context, userID uuid.UUID, page, pageSize int) ([]Order, int64, error) {
	if page < 1 || pageSize < 1 || pageSize > 100 {
		return nil, 0, invalid("page", "out_of_range")
	}
	return s.repo.ListOrders(ctx, userID, pageSize, (page-1)*pageSize)
}

func itemOptionIDs(items []OrderItemInput) []uuid.UUID {
	ids := make([]uuid.UUID, 0, len(items))
	seen := make(map[uuid.UUID]bool, len(items))
	for _, item := range items {
		if !seen[item.OptionID] {
			seen[item.OptionID] = true
			ids = append(ids, item.OptionID)
		}
	}
	return ids
}

func sortedOptionIDs(quantities map[uuid.UUID]int) []uuid.UUID {
	ids := make([]uuid.UUID, 0, len(quantities))
	for id := range quantities {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return bytes.Compare(ids[i][:], ids[j][:]) < 0 })
	return ids
}
