package commerce

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/dealo/dealo/services/api/internal/platform/mail"
)

// emailCopy holds translated order confirmation text. Templates live in Go because the API sends
// the mail, the same as the identity emails.
type emailCopy struct {
	subject string
	body    string
}

var orderEmailCopies = map[string]emailCopy{
	"uz": {
		subject: "%s: buyurtma %s tasdiqlandi",
		body: "Assalomu alaykum, %s!\n\nBuyurtma %s uchun to'lov qabul qilindi (%d UZS).\n" +
			"Kuponlaringizni hisob sahifasida ko'rishingiz mumkin:\n%s\n",
	},
	"ru": {
		subject: "%s: заказ %s подтверждён",
		body: "Здравствуйте, %s!\n\nОплата заказа %s получена (%d сум).\n" +
			"Ваши купоны доступны в личном кабинете:\n%s\n",
	},
	"en": {
		subject: "%s: order %s confirmed",
		body: "Hello %s!\n\nWe received your payment for order %s (%d UZS).\n" +
			"Your coupons are available in your account:\n%s\n",
	},
}

func orderEmailCopyFor(locale string) emailCopy {
	if copy, ok := orderEmailCopies[locale]; ok {
		return copy
	}
	return orderEmailCopies["uz"]
}

// sendOrderConfirmation mails the buyer after a paid order. Failures are logged; the order is
// already paid, so a missing email must not fail the webhook.
func (s *Service) sendOrderConfirmation(ctx context.Context, settled settledOrder) {
	contact, err := s.repo.UserContact(ctx, settled.UserID)
	if err != nil {
		s.logger.WarnContext(ctx, "order confirmation skipped", slog.String("error", err.Error()))
		return
	}
	copy := orderEmailCopyFor(contact.Locale)
	link := s.webBaseURL + "/" + contact.Locale + "/account/coupons"
	message := mail.Message{
		To:      contact.Email,
		Subject: fmt.Sprintf(copy.subject, s.brandName, settled.Number),
		Body:    fmt.Sprintf(copy.body, contact.Name, settled.Number, settled.Total, link),
	}
	if err := s.mailer.Send(ctx, message); err != nil {
		s.logger.WarnContext(ctx, "order confirmation not sent", slog.String("error", err.Error()))
	}
}
