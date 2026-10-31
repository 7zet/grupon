// Package commerceapp wires the commerce service to its adapters. The API and the worker share it so
// that both processes apply the same rules and the same secrets.
package commerceapp

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/grupon/grupon/services/api/internal/domain/commerce"
	"github.com/grupon/grupon/services/api/internal/payments"
	"github.com/grupon/grupon/services/api/internal/platform/config"
	"github.com/grupon/grupon/services/api/internal/platform/mail"
	"github.com/grupon/grupon/services/api/internal/store"
)

// New builds the commerce service on the given pool. Mock payments are attached only when enabled.
func New(cfg config.Config, pool *pgxpool.Pool, deals commerce.DealCatalog, logger *slog.Logger) (*commerce.Service, error) {
	signer, err := commerce.NewQRSigner(cfg.Commerce.QRSecret)
	if err != nil {
		return nil, fmt.Errorf("configure voucher signer: %w", err)
	}

	var mock *payments.Mock
	if cfg.Commerce.MockPaymentsEnabled {
		mock, err = payments.NewMock(cfg.Commerce.MockWebhookSecret)
		if err != nil {
			return nil, fmt.Errorf("configure mock payments: %w", err)
		}
	}

	var mailer mail.Sender = mail.NewLogSender(logger)
	if cfg.SMTP.Host != "" {
		mailer = mail.NewSMTPSender(cfg.SMTP.Host, cfg.SMTP.Port, cfg.SMTP.From)
	}

	return commerce.NewService(commerce.Dependencies{
		Repository: store.NewCommerceRepository(pool),
		Deals:      deals,
		Mock:       mock,
		Mailer:     mailer,
		Signer:     signer,
		Logger:     logger,
		Now:        time.Now,
		BrandName:  cfg.Auth.BrandName,
		WebBaseURL: cfg.Auth.WebPublicURL,
	})
}
