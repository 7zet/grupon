// Package apiapp owns the API process lifecycle.
package apiapp

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"

	"github.com/google/uuid"
	"github.com/grupon/grupon/services/api/internal/app/commerceapp"
	workerapp "github.com/grupon/grupon/services/api/internal/app/worker"
	"github.com/grupon/grupon/services/api/internal/domain/catalog"
	"github.com/grupon/grupon/services/api/internal/domain/commerce"
	"github.com/grupon/grupon/services/api/internal/domain/finance"
	"github.com/grupon/grupon/services/api/internal/domain/identity"
	"github.com/grupon/grupon/services/api/internal/domain/merchant"
	"github.com/grupon/grupon/services/api/internal/domain/moderation"
	"github.com/grupon/grupon/services/api/internal/domain/redemption"
	"github.com/grupon/grupon/services/api/internal/domain/reviews"
	httpapi "github.com/grupon/grupon/services/api/internal/http"
	"github.com/grupon/grupon/services/api/internal/platform/cache"
	"github.com/grupon/grupon/services/api/internal/platform/config"
	"github.com/grupon/grupon/services/api/internal/platform/database"
	"github.com/grupon/grupon/services/api/internal/platform/mail"
	"github.com/grupon/grupon/services/api/internal/platform/ratelimit"
	"github.com/grupon/grupon/services/api/internal/platform/revalidate"
	"github.com/grupon/grupon/services/api/internal/platform/sms"
	"github.com/grupon/grupon/services/api/internal/platform/storage"
	"github.com/grupon/grupon/services/api/internal/store"
)

// Run connects to PostgreSQL and Redis, serves the HTTP API, and blocks until the server stops
// or the context is canceled.
func Run(ctx context.Context, cfg config.Config, logger *slog.Logger) error {
	if cfg.RunWorkerInAPI {
		go func() {
			if err := workerapp.Run(ctx, cfg, logger); err != nil && ctx.Err() == nil {
				logger.Error("embedded worker stopped", slog.String("error", err.Error()))
			}
		}()
	}
	pool, err := database.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	}
	defer pool.Close()

	readiness := []httpapi.ReadinessCheck{func(ctx context.Context) error { return pool.Ping(ctx) }}
	var responseCache catalog.Cache
	var limiter identity.RateLimiter = ratelimit.Disabled{}
	if cfg.RedisURL != "" {
		options, err := redis.ParseURL(cfg.RedisURL)
		if err != nil {
			return fmt.Errorf("parse REDIS_URL: %w", err)
		}
		client := redis.NewClient(options)
		defer client.Close()

		redisCache := cache.NewRedis(client)
		readiness = append(readiness, redisCache.Ping)
		responseCache = redisCache
		limiter = ratelimit.NewRedis(client, authRateLimit, authRateWindow)
	} else {
		logger.WarnContext(ctx, "REDIS_URL is not set; auth rate limiting is disabled")
	}

	identityService, err := newIdentityService(cfg, pool, limiter, logger)
	if err != nil {
		return err
	}

	catalogService := catalog.NewService(store.NewCatalogRepository(pool), responseCache, time.Now)
	notifier := revalidate.NewNotifier(cfg.WebRevalidateURL, cfg.WebRevalidateSecret)
	invalidateCatalog := func(ctx context.Context) error {
		if err := catalogService.Invalidate(ctx); err != nil {
			return err
		}
		return notifier.Revalidate(ctx, revalidate.CatalogTag)
	}

	mediaStore, err := storage.NewLocal(cfg.MediaDir, cfg.MediaPublicBase)
	if err != nil {
		return fmt.Errorf("configure media storage: %w", err)
	}
	workspaceService := merchant.NewWorkspaceService(store.NewMerchantWorkspaceRepository(pool), time.Now)
	documentStore, err := storage.NewPrivate(cfg.DocumentsDir)
	if err != nil {
		return fmt.Errorf("configure document storage: %w", err)
	}
	mediaService := merchant.NewMediaService(store.NewMerchantMediaRepository(pool), mediaStore, time.Now)

	listener, err := (&net.ListenConfig{}).Listen(ctx, "tcp", cfg.HTTP.Address)
	if err != nil {
		if ctx.Err() != nil {
			return nil
		}
		return fmt.Errorf("listen on %s: %w", cfg.HTTP.Address, err)
	}

	authAPI := httpapi.NewAuthAPI(identityService, logger, httpapi.AuthOptions{
		SecureCookies:     cfg.Environment != "development",
		TrustProxyHeaders: cfg.TrustProxyHeaders,
		RefreshTokenTTL:   cfg.Auth.RefreshTokenTTL,
		AllowedOrigins:    cfg.Auth.AllowedOrigins,
		AllowLocalOrigins: cfg.Environment == "development",
	})

	commerceService, err := commerceapp.New(cfg, pool, store.NewCatalogRepository(pool), logger)
	if err != nil {
		return fmt.Errorf("configure commerce: %w", err)
	}

	router := httpapi.NewRouter(httpapi.RouterOptions{
		Logger:         logger,
		ReadinessCheck: allReady(readiness),
		Catalog:        httpapi.NewCatalogAPI(catalogService, logger, cfg.DemoMode),
		Internal:       httpapi.NewInternalAPI(cfg.InternalToken, invalidateCatalog),
		Auth:           authAPI,
		Commerce:       httpapi.NewCommerceAPI(commerceService, authAPI, logger),
		Merchant:       httpapi.NewMerchantAPI(merchant.NewService(store.NewMerchantRepository(pool), time.Now), authAPI, logger),
		MerchantWorkspace: httpapi.NewMerchantWorkspaceAPI(
			merchant.NewWorkspaceService(store.NewMerchantWorkspaceRepository(pool), time.Now), authAPI, logger),
		MerchantMedia: httpapi.NewMerchantMediaAPI(mediaService, workspaceService, authAPI, logger),
		MerchantDeals: httpapi.NewMerchantDealAPI(
			merchant.NewDealService(store.NewMerchantDealRepository(pool, store.NewCatalogRepository(pool)), time.Now),
			workspaceService, authAPI, logger),
		MerchantPanel:     httpapi.NewMerchantPanelAPI(merchant.NewPanelService(store.NewMerchantPanelRepository(pool), time.Now), merchant.NewWorkspaceService(store.NewMerchantWorkspaceRepository(pool), time.Now), authAPI, logger),
		MerchantStaff:     httpapi.NewMerchantStaffAPI(merchant.NewStaffService(store.NewMerchantStaffRepository(pool)), merchant.NewWorkspaceService(store.NewMerchantWorkspaceRepository(pool), time.Now), authAPI, logger),
		MerchantDocuments: httpapi.NewMerchantDocumentAPI(merchant.NewDocumentService(store.NewMerchantDocumentRepository(pool), documentStore), merchant.NewReviewService(store.NewMerchantReviewRepository(pool)), authAPI, logger),
		MerchantReview:    httpapi.NewMerchantReviewAPI(merchant.NewReviewService(store.NewMerchantReviewRepository(pool)), authAPI, logger),
		Moderation:        httpapi.NewModerationAPI(moderation.NewService(store.NewModerationRepository(pool, store.NewCatalogRepository(pool), store.NewMerchantDealRepository(pool, store.NewCatalogRepository(pool))), time.Now), authAPI, logger),
		MerchantRedemption: httpapi.NewMerchantRedemptionAPI(
			redemption.NewService(store.NewRedemptionRepository(pool, store.NewCommerceRepository(pool), store.NewCatalogRepository(pool)), verifyQRToken(commerceService), time.Now).WithBackupCodes(commerceService.Signer().BackupCodeHMAC),
			workspaceService, authAPI, logger),
		Reviews:      httpapi.NewReviewAPI(reviews.NewService(store.NewReviewRepository(pool)), authAPI, logger),
		AdminReports: httpapi.NewAdminReportsAPI(moderation.NewReportService(store.NewAdminReportRepository(pool, store.NewReviewRepository(pool))), authAPI, logger),
		Finance: httpapi.NewFinanceAPI(
			finance.NewService(store.NewFinanceRepository(pool), time.Now),
			redemption.NewService(store.NewRedemptionRepository(pool, store.NewCommerceRepository(pool), store.NewCatalogRepository(pool)), verifyQRToken(commerceService), time.Now).WithBackupCodes(commerceService.Signer().BackupCodeHMAC),
			workspaceService, authAPI, logger),
		MediaFiles: http.StripPrefix("/media/", http.FileServer(http.Dir(mediaStore.Root()))),
	})
	server := newServer(cfg.HTTP, router, logger)

	logger.InfoContext(ctx, "api server started", slog.String("address", listener.Addr().String()))
	return serve(ctx, server, listener, cfg.ShutdownTimeout, logger)
}

const (
	// authRateLimit is the number of attempts allowed per key in each authRateWindow.
	authRateLimit  = 10
	authRateWindow = 15 * time.Minute
	verifyEmailTTL = 24 * time.Hour
)

// newIdentityService wires the identity rules to PostgreSQL, the mailer and the Google client.
func newIdentityService(cfg config.Config, pool *pgxpool.Pool, limiter identity.RateLimiter, logger *slog.Logger) (*identity.Service, error) {
	// Codes are logged only in development. Other environments have no sender until an SMS provider exists.
	// Provider selection is explicit even though the pilot currently ships only the safe LogSender.
	// A real provider can be added behind this switch without changing identity's CodeSender port.
	var codeSender identity.CodeSender
	if cfg.SMS.Provider == "log" {
		codeSender = sms.NewLogSender(logger)
	} else if cfg.SMS.Provider == "eskiz" {
		sender, senderErr := sms.NewEskizSender(cfg.SMS.EskizBaseURL, cfg.SMS.EskizEmail, cfg.SMS.EskizPassword, cfg.SMS.EskizSender)
		if senderErr != nil {
			return nil, fmt.Errorf("configure Eskiz sender: %w", senderErr)
		}
		codeSender = sender
	}
	tokens, err := identity.NewTokenIssuer(cfg.Auth.JWTSecret, cfg.Auth.AccessTokenTTL, time.Now)
	if err != nil {
		return nil, fmt.Errorf("configure access tokens: %w", err)
	}

	var mailer mail.Sender = mail.NewLogSender(logger)
	if cfg.SMTP.Host != "" {
		mailer = mail.NewSMTPSender(cfg.SMTP.Host, cfg.SMTP.Port, cfg.SMTP.From)
	} else {
		logger.Warn("SMTP_HOST is not set; emails are logged without bodies and not delivered")
	}

	return identity.NewService(identity.Dependencies{
		Repository: store.NewIdentityRepository(pool),
		Tokens:     tokens,
		Mailer:     mailer,
		Limiter:    limiter,
		Logger:     logger,
		Now:        time.Now,
		CodeSender: codeSender,
	}, identity.Config{
		OTPEcho:           cfg.DemoMode,
		OTPCooldown:       otpCooldown(cfg.Environment),
		AccessTokenTTL:    cfg.Auth.AccessTokenTTL,
		RefreshTokenTTL:   cfg.Auth.RefreshTokenTTL,
		VerifyEmailTTL:    verifyEmailTTL,
		WebBaseURL:        cfg.Auth.WebPublicURL,
		BrandName:         cfg.Auth.BrandName,
		RealSMS:           cfg.SMS.Provider == "eskiz",
		SMSDailyCapPerIP:  cfg.SMS.DailyCapPerIP,
		SMSDailyCapGlobal: cfg.SMS.DailyCapGlobal,

		RateLimitFailClosed: cfg.Environment == "production",
	}), nil
}

// allReady reports the first failing dependency check.
func allReady(checks []httpapi.ReadinessCheck) httpapi.ReadinessCheck {
	return func(ctx context.Context) error {
		for _, check := range checks {
			if err := check(ctx); err != nil {
				return err
			}
		}
		return nil
	}
}

func newServer(cfg config.HTTPConfig, handler http.Handler, logger *slog.Logger) *http.Server {
	return &http.Server{
		Handler:           handler,
		ReadHeaderTimeout: cfg.ReadHeaderTimeout,
		ReadTimeout:       cfg.ReadTimeout,
		WriteTimeout:      cfg.WriteTimeout,
		IdleTimeout:       cfg.IdleTimeout,
		ErrorLog:          slog.NewLogLogger(logger.Handler(), slog.LevelError),
	}
}

func serve(
	ctx context.Context,
	server *http.Server,
	listener net.Listener,
	shutdownTimeout time.Duration,
	logger *slog.Logger,
) error {
	serveErrors := make(chan error, 1)
	go func() {
		serveErrors <- server.Serve(listener)
	}()

	select {
	case err := <-serveErrors:
		return unexpectedServeError(err)
	case <-ctx.Done():
		logger.Info("api server stopping")

		shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer cancel()

		shutdownErr := server.Shutdown(shutdownCtx)
		if shutdownErr != nil {
			closeErr := server.Close()
			serveErr := unexpectedServeError(<-serveErrors)
			return errors.Join(
				fmt.Errorf("shut down api server: %w", shutdownErr),
				closeErr,
				serveErr,
			)
		}

		if err := unexpectedServeError(<-serveErrors); err != nil {
			return err
		}

		logger.Info("api server stopped")
		return nil
	}
}

func unexpectedServeError(err error) error {
	if err == nil || errors.Is(err, http.ErrServerClosed) {
		return nil
	}

	return fmt.Errorf("serve http: %w", err)
}

// verifyQRToken adapts the commerce QR signer to the redemption rules.
func verifyQRToken(service *commerce.Service) redemption.Verifier {
	return func(token string) (uuid.UUID, error) {
		claims, err := service.VerifyQRToken(token)
		if err != nil {
			return uuid.Nil, err
		}
		return claims.VoucherID, nil
	}
}

// otpCooldown keeps the one-minute wait in every environment except development, where automated
// tests sign in repeatedly.
func otpCooldown(environment string) time.Duration {
	if environment == "development" {
		return 0
	}
	return time.Minute
}
