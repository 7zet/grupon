package commerce

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/google/uuid"
)

// The sweeps below are idempotent: each one re-reads state under a row lock and does nothing when
// the work was already done, so several workers or repeated runs are safe.

// ReleaseExpiredReservations cancels unpaid orders whose reservations have expired, returns their
// stock to sale and fails their open payment attempts. It returns the number of orders cancelled.
func (s *Service) ReleaseExpiredReservations(ctx context.Context, batch int) (int, error) {
	orderIDs, err := s.repo.ExpiredUnpaidOrderIDs(ctx, s.now(), batch)
	if err != nil {
		return 0, fmt.Errorf("list expired orders: %w", err)
	}
	cancelled := 0
	for _, orderID := range orderIDs {
		done, err := s.cancelUnpaid(ctx, orderID)
		if err != nil {
			return cancelled, err
		}
		if done {
			cancelled++
		}
	}
	return cancelled, nil
}

func (s *Service) cancelUnpaid(ctx context.Context, orderID uuid.UUID) (bool, error) {
	done := false
	err := s.repo.InTx(ctx, func(tx Store) error {
		done = false
		order, err := tx.LockOrder(ctx, orderID)
		if errors.Is(err, ErrOrderNotFound) {
			return nil
		}
		if err != nil {
			return err
		}
		if order.Status != OrderAwaitingPayment {
			return nil
		}
		released, err := tx.ReleaseReservations(ctx, orderID)
		if err != nil {
			return err
		}
		for _, reservation := range released {
			if err := tx.ReleaseStock(ctx, reservation.OptionID, reservation.Quantity); err != nil {
				return fmt.Errorf("release stock: %w", err)
			}
		}
		if err := tx.FailOpenPayments(ctx, orderID); err != nil {
			return err
		}
		if err := tx.SetOrderStatus(ctx, orderID, OrderExpired, nil); err != nil {
			return err
		}
		done = true
		return nil
	})
	if err != nil {
		return false, fmt.Errorf("cancel unpaid order %s: %w", orderID, err)
	}
	if done {
		s.logger.InfoContext(ctx, "unpaid order cancelled", slog.String("order_id", orderID.String()))
	}
	return done, nil
}

// ExpireVouchers marks active vouchers past their validity as expired. It returns the number changed.
func (s *Service) ExpireVouchers(ctx context.Context, batch int) (int, error) {
	total := 0
	for {
		changed, err := s.repo.ExpireDueVouchers(ctx, s.now(), batch)
		if err != nil {
			return total, fmt.Errorf("expire vouchers: %w", err)
		}
		total += changed
		if changed < batch {
			return total, nil
		}
	}
}
