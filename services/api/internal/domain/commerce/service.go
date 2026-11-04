package commerce

import (
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/grupon/grupon/services/api/internal/payments"
	"github.com/grupon/grupon/services/api/internal/platform/mail"
)

// Dependencies are the collaborators of the commerce service.
type Dependencies struct {
	Repository Repository
	Deals      DealCatalog
	// Providers maps provider names to implementations. Unknown names are ErrProviderUnavailable.
	Providers []payments.Provider
	// Mock is the mock provider when mock payments are enabled, so the complete endpoint can
	// sign webhooks. Nil disables the mock completion path.
	Mock       *payments.Mock
	Manual     *payments.Manual
	ManualHold time.Duration
	Mailer     mail.Sender
	Signer     *QRSigner
	Logger     *slog.Logger
	Now        func() time.Time
	// BrandName and WebBaseURL are used in notification emails.
	BrandName  string
	WebBaseURL string
}

// Service implements the commerce use cases.
type Service struct {
	repo       Repository
	deals      DealCatalog
	providers  map[string]payments.Provider
	mock       *payments.Mock
	manual     *payments.Manual
	manualHold time.Duration
	mailer     mail.Sender
	signer     *QRSigner
	logger     *slog.Logger
	now        func() time.Time
	brandName  string
	webBaseURL string
}

// NewService validates the dependencies and builds the service.
func NewService(deps Dependencies) (*Service, error) {
	if deps.Repository == nil || deps.Deals == nil || deps.Signer == nil {
		return nil, errors.New("commerce: repository, deal catalog and qr signer are required")
	}
	providers := make(map[string]payments.Provider, len(deps.Providers))
	for _, provider := range deps.Providers {
		providers[provider.Name()] = provider
	}
	if deps.Mock != nil {
		providers[deps.Mock.Name()] = deps.Mock
	}
	if deps.Manual != nil {
		providers[deps.Manual.Name()] = deps.Manual
	}
	logger := deps.Logger
	if logger == nil {
		logger = slog.Default()
	}
	now := deps.Now
	if now == nil {
		now = time.Now
	}
	mailer := deps.Mailer
	if mailer == nil {
		return nil, fmt.Errorf("commerce: mailer is required")
	}
	return &Service{
		repo:       deps.Repository,
		deals:      deps.Deals,
		providers:  providers,
		mock:       deps.Mock,
		manual:     deps.Manual,
		manualHold: deps.ManualHold,
		mailer:     mailer,
		signer:     deps.Signer,
		logger:     logger,
		now:        now,
		brandName:  deps.BrandName,
		webBaseURL: deps.WebBaseURL,
	}, nil
}

// Signer exposes the QR signer for the redemption side (not built yet) and for tests.
func (s *Service) Signer() *QRSigner { return s.signer }

// Now returns the service clock.
func (s *Service) Now() time.Time { return s.now() }
