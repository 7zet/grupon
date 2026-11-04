package commerce

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"github.com/grupon/grupon/services/api/internal/domain/ledger"
	"log/slog"
	"net/http"
	"time"

	"github.com/google/uuid"

	"github.com/grupon/grupon/services/api/internal/payments"
)

// PayOrder starts a payment attempt for an order that still awaits payment and whose reservations
// have not expired. Each call creates a new attempt, so a failed attempt can be retried.
//
// The provider call runs inside the transaction. Mock payments are local, so this is safe today;
// a network provider should move the call out and record the attempt in a second step.
func (s *Service) PayOrder(ctx context.Context, userID, orderID uuid.UUID, providerName string) (PaymentSession, error) {
	provider, ok := s.providers[providerName]
	if !ok {
		return PaymentSession{}, ErrProviderUnavailable
	}
	var session PaymentSession
	err := s.repo.InTx(ctx, func(tx Store) error {
		order, err := tx.LockOrder(ctx, orderID)
		if err != nil {
			return err
		}
		if order.UserID != userID {
			return ErrOrderNotFound
		}
		if order.Status != OrderAwaitingPayment {
			return ErrOrderNotPayable
		}
		now := s.now()
		reservations, err := tx.UnreleasedReservations(ctx, orderID)
		if err != nil {
			return err
		}
		if len(reservations) == 0 {
			return ErrReservationExpired
		}
		for _, reservation := range reservations {
			if !reservation.ExpiresAt.After(now) {
				return ErrReservationExpired
			}
		}

		paymentID, err := uuid.NewV7()
		if err != nil {
			return fmt.Errorf("new payment id: %w", err)
		}
		created, err := provider.CreatePayment(ctx, payments.PaymentRequest{
			PaymentID:   paymentID,
			OrderNumber: order.Number,
			Amount:      order.Total,
			Currency:    Currency,
		})
		if err != nil {
			return fmt.Errorf("create provider payment: %w", err)
		}
		payment := Payment{
			ID:                paymentID,
			OrderID:           orderID,
			Provider:          providerName,
			ProviderPaymentID: created.ProviderPaymentID,
			Status:            PaymentCreated,
			Amount:            order.Total,
		}
		if err := tx.InsertPayment(ctx, payment); err != nil {
			return fmt.Errorf("insert payment: %w", err)
		}
		if providerName == "manual" && s.manualHold > 0 {
			if err := tx.ExtendReservations(ctx, orderID, now.Add(s.manualHold)); err != nil {
				return fmt.Errorf("extend manual reservation: %w", err)
			}
		}
		session = PaymentSession{
			PaymentID:    paymentID,
			Provider:     providerName,
			Status:       PaymentCreated,
			RedirectURL:  created.RedirectURL,
			Instructions: created.Instructions,
		}
		return nil
	})
	if err != nil {
		return PaymentSession{}, err
	}
	return session, nil
}

// refundDue records a payment that succeeded after its order was already cancelled or paid.
type refundDue struct {
	payment Payment
}

// HandleWebhook verifies and applies one provider notification. Verification happens for every
// provider, including the mock. Processing is idempotent: a repeated outcome changes nothing.
func (s *Service) HandleWebhook(ctx context.Context, providerName string, body []byte, signature string) error {
	provider, ok := s.providers[providerName]
	if !ok {
		return ErrProviderUnavailable
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, "/", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("build webhook request: %w", err)
	}
	request.Header.Set(payments.MockSignatureHeader, signature)
	event, err := provider.ParseWebhook(ctx, request)
	if err != nil {
		return err
	}

	return s.applyPaymentEvent(ctx, provider, providerName, event, nil)
}

type manualConfirmation struct {
	actorID                    uuid.UUID
	actorRole, reference, note string
}

// ConfirmManualPayment applies the same event path as a verified provider webhook. The audit row
// is inserted inside the settlement transaction, so a recorded confirmation can never exist without
// its vouchers and ledger effects.
func (s *Service) ConfirmManualPayment(ctx context.Context, paymentID, actorID uuid.UUID, actorRole, reference, note string) error {
	if s.manual == nil {
		return ErrPaymentNotFound
	}
	if reference == "" || note == "" {
		return invalid("reference", "required")
	}
	payment, err := s.repo.PaymentByID(ctx, paymentID)
	if err != nil {
		return err
	}
	if payment.Provider != s.manual.Name() {
		return ErrPaymentNotFound
	}
	return s.applyPaymentEvent(ctx, s.manual, s.manual.Name(), payments.Event{ProviderPaymentID: payment.ProviderPaymentID, Outcome: payments.OutcomeSucceeded}, &manualConfirmation{actorID: actorID, actorRole: actorRole, reference: reference, note: note})
}

func (s *Service) applyPaymentEvent(ctx context.Context, provider payments.Provider, providerName string, event payments.Event, confirmation *manualConfirmation) error {
	var (
		paidOrder *settledOrder
		refund    *refundDue
	)
	err := s.repo.InTx(ctx, func(tx Store) error {
		paidOrder, refund = nil, nil
		payment, err := tx.LockPaymentByProviderID(ctx, providerName, event.ProviderPaymentID)
		if err != nil {
			return err
		}
		switch event.Outcome {
		case payments.OutcomeFailed:
			if payment.Status == PaymentCreated || payment.Status == PaymentPending {
				return tx.SetPaymentStatus(ctx, payment.ID, PaymentFailed)
			}
			return nil
		case payments.OutcomeSucceeded:
			if payment.Status == PaymentSucceeded {
				return nil
			}
			order, err := tx.LockOrder(ctx, payment.OrderID)
			if err != nil {
				return err
			}
			reservations, err := tx.UnreleasedReservations(ctx, order.ID)
			if err != nil {
				return err
			}
			if order.Status == OrderAwaitingPayment && len(reservations) > 0 {
				settled, err := s.settlePaid(ctx, tx, payment, order.ID)
				if err != nil {
					return err
				}
				if confirmation != nil {
					if err := tx.RecordManualConfirmation(ctx, payment.ID, confirmation.actorID, confirmation.actorRole, confirmation.reference, confirmation.note); err != nil {
						return err
					}
				}
				paidOrder = settled
				return nil
			}
			// The money arrived after the stock went back to sale, or the order is already paid by
			// another attempt. Keep the record and refund it outside the transaction.
			if err := tx.SetPaymentStatus(ctx, payment.ID, PaymentSucceeded); err != nil {
				return err
			}
			refund = &refundDue{payment: payment}
			return nil
		default:
			return ErrMalformedWebhook
		}
	})
	if err != nil {
		return err
	}
	if refund != nil {
		s.refundLatePayment(ctx, provider, refund.payment)
	}
	if paidOrder != nil {
		s.sendOrderConfirmation(ctx, *paidOrder)
	}
	return nil
}

// settledOrder is what a successful payment produced; it feeds the confirmation email.
type settledOrder struct {
	UserID       uuid.UUID
	Number       string
	Total        int64
	VoucherCount int
}

// settlePaid converts reservations to sold stock, issues vouchers for voucher items, and marks the
// order and payment paid. It runs inside the webhook transaction.
func (s *Service) settlePaid(ctx context.Context, tx Store, payment Payment, orderID uuid.UUID) (*settledOrder, error) {
	order, err := tx.OrderByID(ctx, orderID)
	if err != nil {
		return nil, err
	}
	released, err := tx.ReleaseReservations(ctx, orderID)
	if err != nil {
		return nil, err
	}
	for _, reservation := range released {
		if err := tx.ConvertStock(ctx, reservation.OptionID, reservation.Quantity); err != nil {
			return nil, fmt.Errorf("convert reserved stock: %w", err)
		}
	}

	now := s.now()
	optionIDs := itemOptionIDs(itemInputs(order.Items))
	records, err := tx.OptionRecords(ctx, optionIDs)
	if err != nil {
		return nil, fmt.Errorf("load options: %w", err)
	}
	byID := indexRecords(records)

	issued := 0
	for _, item := range order.Items {
		if err := tx.AddDealSoldCount(ctx, item.DealID, item.Quantity); err != nil {
			return nil, fmt.Errorf("count sold deal: %w", err)
		}
		record, ok := byID[item.OptionID]
		if !ok {
			return nil, ErrOptionUnavailable
		}
		validFrom, validUntil, valid := record.VoucherWindow(now)
		if !valid {
			return nil, ErrOptionUnavailable
		}
		for unit := 0; unit < item.Quantity; unit++ {
			if err := s.issueVoucher(ctx, tx, order.UserID, item, record, validFrom, validUntil); err != nil {
				return nil, err
			}
			issued++
		}
		paid := ledger.PaymentReceived(item.ID.String(), record.MerchantID.String(), item.UnitPrice*int64(item.Quantity))
		if _, err := tx.PostLedger(ctx, paid); err != nil {
			return nil, fmt.Errorf("post payment to ledger: %w", err)
		}
	}

	if err := tx.SetOrderStatus(ctx, orderID, OrderPaid, &now); err != nil {
		return nil, fmt.Errorf("mark order paid: %w", err)
	}
	if err := tx.SetPaymentStatus(ctx, payment.ID, PaymentSucceeded); err != nil {
		return nil, fmt.Errorf("mark payment succeeded: %w", err)
	}
	return &settledOrder{UserID: order.UserID, Number: order.Number, Total: order.Total, VoucherCount: issued}, nil
}

// issueVoucher creates one voucher for one unit with a fresh unambiguous code and a signed QR token.
func (s *Service) issueVoucher(ctx context.Context, tx Store, userID uuid.UUID, item OrderItem, record OptionRecord, validFrom, validUntil time.Time) error {
	voucherID, err := uuid.NewV7()
	if err != nil {
		return fmt.Errorf("new voucher id: %w", err)
	}
	code, err := s.uniqueVoucherCode(ctx, tx)
	if err != nil {
		return err
	}
	return tx.InsertVoucher(ctx, Voucher{
		ID:                 voucherID,
		OrderItemID:        item.ID,
		UserID:             userID,
		DealID:             item.DealID,
		OptionID:           item.OptionID,
		MerchantID:         record.MerchantID,
		Code:               code,
		QRToken:            s.signer.Sign(voucherID),
		Status:             VoucherActive,
		ValidFrom:          validFrom,
		ValidUntil:         validUntil,
		RedemptionsAllowed: max(1, record.RedemptionsAllowed),
		ZaxiraCodeHMAC:     s.signer.BackupCodeHMAC(s.signer.BackupCode(voucherID)),
	})
}

const maxVoucherCodeAttempts = 8

func (s *Service) uniqueVoucherCode(ctx context.Context, tx Store) (string, error) {
	for attempt := 0; attempt < maxVoucherCodeAttempts; attempt++ {
		code, err := NewVoucherCode()
		if err != nil {
			return "", err
		}
		taken, err := tx.VoucherCodeTaken(ctx, code)
		if err != nil {
			return "", err
		}
		if !taken {
			return code, nil
		}
	}
	return "", errors.New("commerce: could not allocate a unique voucher code")
}

func itemInputs(items []OrderItem) []OrderItemInput {
	inputs := make([]OrderItemInput, 0, len(items))
	for _, item := range items {
		inputs = append(inputs, OrderItemInput{OptionID: item.OptionID, Quantity: item.Quantity})
	}
	return inputs
}

// refundLatePayment asks the provider to return money for a payment that could not be applied.
// Failures are logged for manual follow-up; the payment stays succeeded until a refund is confirmed.
func (s *Service) refundLatePayment(ctx context.Context, provider payments.Provider, payment Payment) {
	reference := payments.Reference{PaymentID: payment.ID, ProviderPaymentID: payment.ProviderPaymentID}
	if err := provider.Refund(ctx, reference, payment.Amount); err != nil {
		s.logger.ErrorContext(ctx, "late payment refund failed; manual refund required",
			slog.String("payment_id", payment.ID.String()),
			slog.String("order_id", payment.OrderID.String()),
			slog.String("error", err.Error()),
		)
		return
	}
	if err := s.repo.RecordLateRefund(ctx, payment.ID, payment.Amount, "payment arrived after the order expired"); err != nil {
		s.logger.ErrorContext(ctx, "late payment refunded but refund record failed",
			slog.String("payment_id", payment.ID.String()),
			slog.String("error", err.Error()),
		)
		return
	}
	s.logger.WarnContext(ctx, "late payment refunded",
		slog.String("payment_id", payment.ID.String()),
		slog.String("order_id", payment.OrderID.String()),
	)
}

// CompleteMockPayment is the mock checkout's button. It signs a webhook for the chosen outcome and
// applies it through the same path a real provider's webhook takes.
func (s *Service) CompleteMockPayment(ctx context.Context, paymentID uuid.UUID, outcome payments.Outcome) error {
	if s.mock == nil {
		return ErrPaymentNotFound
	}
	if outcome != payments.OutcomeSucceeded && outcome != payments.OutcomeFailed {
		return invalid("outcome", "unsupported")
	}
	payment, err := s.repo.PaymentByID(ctx, paymentID)
	if err != nil {
		return err
	}
	if payment.Provider != s.mock.Name() {
		return ErrPaymentNotFound
	}
	if payment.Status != PaymentCreated && payment.Status != PaymentPending {
		return ErrPaymentNotPending
	}
	body, signature, err := s.mock.SignedWebhook(payment.ProviderPaymentID, outcome)
	if err != nil {
		return err
	}
	return s.HandleWebhook(ctx, s.mock.Name(), body, signature)
}
