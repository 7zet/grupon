package config

import (
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	defaultEnvironment       = "development"
	productionEnvironment    = "production"
	defaultAPIAddress        = ":8080"
	defaultReadHeaderTimeout = 5 * time.Second
	defaultReadTimeout       = 15 * time.Second
	defaultWriteTimeout      = 30 * time.Second
	defaultIdleTimeout       = 60 * time.Second
	defaultShutdownTimeout   = 10 * time.Second
	// developmentDatabaseURL matches the docker-compose PostgreSQL service. Production must set
	// DATABASE_URL explicitly.
	developmentDatabaseURL = "postgres://app:app_dev_password@localhost:5432/app?sslmode=disable"
	// developmentJWTSecret signs tokens in development only. Other environments must set JWT_SECRET.
	developmentJWTSecret = "dev-only-insecure-jwt-secret-0123456789"
	// Development-only secrets for vouchers and the mock payment provider. Other environments must
	// set VOUCHER_QR_SECRET and PAYMENT_MOCK_WEBHOOK_SECRET when they are used.
	developmentQRSecret          = "dev-only-insecure-qr-secret-0123456789"
	developmentMockWebhookSecret = "dev-only-insecure-mock-webhook-0123456789"
	defaultWebPublicURL          = "http://localhost:3000"
	defaultBrandName             = "Grupon"
	defaultSMTPPort              = 1025
	defaultSMTPFrom              = "Grupon <no-reply@grupon.local>"
	defaultAccessTokenTTL        = 15 * time.Minute
	defaultRefreshTokenTTL       = 30 * 24 * time.Hour
)

// Config contains process configuration shared by the API and worker binaries.
type Config struct {
	Environment string
	// DemoMode exposes demo-only login and mock-payment affordances.
	DemoMode        bool
	RunWorkerInAPI  bool
	LogLevel        slog.Level
	HTTP            HTTPConfig
	ShutdownTimeout time.Duration
	// DatabaseURL is the PostgreSQL connection string. It is required outside development.
	DatabaseURL string
	// RedisURL enables the Redis catalog cache when set. Without it the API reads through to PostgreSQL.
	RedisURL string
	// InternalToken guards operational endpoints. Those endpoints stay disabled when it is empty.
	InternalToken string
	// WebRevalidateURL is the web endpoint that revalidates Next.js cache tags after content changes.
	WebRevalidateURL string
	// WebRevalidateSecret authenticates calls to WebRevalidateURL.
	WebRevalidateSecret string
	// MediaDir is the local storage root for uploaded images (development adapter).
	MediaDir string
	// DocumentsDir is the private storage root for business documents. It is never served publicly.
	DocumentsDir string
	// MediaPublicBase is the URL prefix browsers use for stored images.
	MediaPublicBase string
	// Auth configures sessions, credentials recovery and outbound identity providers.
	Auth AuthConfig
	// SMTP configures outbound email. Email is logged instead of sent when SMTPHost is empty.
	SMTP SMTPConfig
	// SMS selects the phone-code delivery adapter. Only log is implemented for the pilot.
	SMS SMSConfig
	// Commerce configures vouchers and payment providers.
	Commerce CommerceConfig
	// TrustProxyHeaders makes rate limiting and audit use X-Forwarded-For or X-Real-IP. Enable only
	// behind a proxy that overwrites those headers.
	TrustProxyHeaders bool
}

// AuthConfig holds identity settings.
type AuthConfig struct {
	// JWTSecret signs HS256 access tokens. It must be at least 32 bytes.
	JWTSecret       string
	AccessTokenTTL  time.Duration
	RefreshTokenTTL time.Duration
	// WebPublicURL is the public web origin used in links sent by email.
	WebPublicURL string
	// BrandName appears in email subjects.
	BrandName string
}

// SMTPConfig holds the relay used for transactional email.
type SMTPConfig struct {
	Host string
	Port int
	From string
}

type SMSConfig struct {
	Provider       string
	EskizEmail     string
	EskizPassword  string
	EskizSender    string
	EskizBaseURL   string
	DailyCapPerIP  int
	DailyCapGlobal int
}

// HTTPConfig contains the public HTTP server settings.
type HTTPConfig struct {
	Address           string
	ReadHeaderTimeout time.Duration
	ReadTimeout       time.Duration
	WriteTimeout      time.Duration
	IdleTimeout       time.Duration
}

// Load reads configuration from the environment and validates every value.
func Load() (Config, error) {
	return load(os.LookupEnv)
}

type lookupEnv func(string) (string, bool)

func load(lookup lookupEnv) (Config, error) {
	environment, err := stringValue(lookup, "APP_ENV", defaultEnvironment)
	if err != nil {
		return Config{}, err
	}
	demoMode, err := boolValue(lookup, "DEMO_MODE", false)
	if err != nil {
		return Config{}, err
	}
	runWorkerInAPI, err := boolValue(lookup, "RUN_WORKER_IN_API", false)
	if err != nil {
		return Config{}, err
	}

	address, err := stringValue(lookup, "API_ADDR", defaultAPIAddress)
	if err != nil {
		return Config{}, err
	}

	logLevel, err := levelValue(lookup, "LOG_LEVEL", slog.LevelInfo)
	if err != nil {
		return Config{}, err
	}

	readHeaderTimeout, err := durationValue(lookup, "HTTP_READ_HEADER_TIMEOUT", defaultReadHeaderTimeout)
	if err != nil {
		return Config{}, err
	}

	readTimeout, err := durationValue(lookup, "HTTP_READ_TIMEOUT", defaultReadTimeout)
	if err != nil {
		return Config{}, err
	}

	writeTimeout, err := durationValue(lookup, "HTTP_WRITE_TIMEOUT", defaultWriteTimeout)
	if err != nil {
		return Config{}, err
	}

	idleTimeout, err := durationValue(lookup, "HTTP_IDLE_TIMEOUT", defaultIdleTimeout)
	if err != nil {
		return Config{}, err
	}

	shutdownTimeout, err := durationValue(lookup, "SHUTDOWN_TIMEOUT", defaultShutdownTimeout)
	if err != nil {
		return Config{}, err
	}

	databaseURL, err := stringValue(lookup, "DATABASE_URL", developmentDatabaseURL)
	if err != nil {
		return Config{}, err
	}
	if _, set := lookup("DATABASE_URL"); !set && environment == productionEnvironment {
		return Config{}, fmt.Errorf("DATABASE_URL must be set in production")
	}

	authConfig, err := loadAuth(lookup, environment)
	if err != nil {
		return Config{}, err
	}
	smtpConfig, err := loadSMTP(lookup)
	if err != nil {
		return Config{}, err
	}
	smsConfig, err := loadSMS(lookup)
	if err != nil {
		return Config{}, err
	}
	commerceConfig, err := loadCommerce(lookup, environment, demoMode)
	if err != nil {
		return Config{}, err
	}
	trustProxy, err := boolValue(lookup, "TRUST_PROXY_HEADERS", false)
	if err != nil {
		return Config{}, err
	}

	mediaDir := optionalString(lookup, "MEDIA_LOCAL_DIR")
	if mediaDir == "" {
		mediaDir = ".data/media"
	}
	documentsDir := optionalString(lookup, "DOCUMENTS_LOCAL_DIR")
	if documentsDir == "" {
		documentsDir = ".data/documents"
	}
	mediaPublicBase := optionalString(lookup, "MEDIA_PUBLIC_BASE")
	if mediaPublicBase == "" {
		mediaPublicBase = "http://localhost:8080/media"
	}

	redisURL := optionalString(lookup, "REDIS_URL")
	internalToken := optionalString(lookup, "INTERNAL_API_TOKEN")
	webRevalidateURL := optionalString(lookup, "WEB_REVALIDATE_URL")
	webRevalidateSecret := optionalString(lookup, "WEB_REVALIDATE_SECRET")

	return Config{
		Environment:         environment,
		DemoMode:            demoMode,
		RunWorkerInAPI:      runWorkerInAPI,
		LogLevel:            logLevel,
		DatabaseURL:         databaseURL,
		RedisURL:            redisURL,
		InternalToken:       internalToken,
		WebRevalidateURL:    webRevalidateURL,
		WebRevalidateSecret: webRevalidateSecret,
		MediaDir:            mediaDir,
		DocumentsDir:        documentsDir,
		MediaPublicBase:     mediaPublicBase,
		Auth:                authConfig,
		SMTP:                smtpConfig,
		SMS:                 smsConfig,
		Commerce:            commerceConfig,
		TrustProxyHeaders:   trustProxy,
		HTTP: HTTPConfig{
			Address:           address,
			ReadHeaderTimeout: readHeaderTimeout,
			ReadTimeout:       readTimeout,
			WriteTimeout:      writeTimeout,
			IdleTimeout:       idleTimeout,
		},
		ShutdownTimeout: shutdownTimeout,
	}, nil
}

func loadSMS(lookup lookupEnv) (SMSConfig, error) {
	provider, err := stringValue(lookup, "SMS_PROVIDER", "log")
	if err != nil {
		return SMSConfig{}, err
	}
	if provider == "log" {
		return SMSConfig{Provider: provider}, nil
	}
	if provider != "eskiz" {
		return SMSConfig{}, fmt.Errorf("SMS_PROVIDER %q is not configured", provider)
	}
	email, err := stringValue(lookup, "ESKIZ_EMAIL", "")
	if err != nil {
		return SMSConfig{}, err
	}
	password, err := stringValue(lookup, "ESKIZ_PASSWORD", "")
	if err != nil {
		return SMSConfig{}, err
	}
	sender, err := stringValue(lookup, "ESKIZ_SENDER", "")
	if err != nil {
		return SMSConfig{}, err
	}
	perIP, err := intValue(lookup, "SMS_DAILY_CAP_PER_IP", 50)
	if err != nil {
		return SMSConfig{}, err
	}
	global, err := intValue(lookup, "SMS_DAILY_CAP_GLOBAL", 1000)
	if err != nil {
		return SMSConfig{}, err
	}
	if perIP < 1 || global < 1 {
		return SMSConfig{}, fmt.Errorf("SMS daily caps must be positive")
	}
	return SMSConfig{Provider: provider, EskizEmail: email, EskizPassword: password, EskizSender: sender, EskizBaseURL: optionalString(lookup, "ESKIZ_BASE_URL"), DailyCapPerIP: perIP, DailyCapGlobal: global}, nil
}

// optionalString returns the trimmed value of an optional variable, or an empty string.
func optionalString(lookup lookupEnv, name string) string {
	value, _ := lookup(name)
	return strings.TrimSpace(value)
}

func stringValue(lookup lookupEnv, name, fallback string) (string, error) {
	value, ok := lookup(name)
	if !ok {
		return fallback, nil
	}

	value = strings.TrimSpace(value)
	if value == "" {
		return "", fmt.Errorf("%s must not be empty", name)
	}

	return value, nil
}

func intValue(lookup lookupEnv, name string, fallback int) (int, error) {
	value, ok := lookup(name)
	if !ok {
		return fallback, nil
	}
	parsed, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil || parsed < 0 {
		return 0, fmt.Errorf("%s must be a non-negative integer", name)
	}
	return parsed, nil
}

func levelValue(lookup lookupEnv, name string, fallback slog.Level) (slog.Level, error) {
	value, ok := lookup(name)
	if !ok {
		return fallback, nil
	}

	value = strings.TrimSpace(value)
	if value == "" {
		return 0, fmt.Errorf("%s must not be empty", name)
	}

	var level slog.Level
	if err := level.UnmarshalText([]byte(value)); err != nil {
		return 0, fmt.Errorf("parse %s: %w", name, err)
	}

	return level, nil
}

func durationValue(lookup lookupEnv, name string, fallback time.Duration) (time.Duration, error) {
	value, ok := lookup(name)
	if !ok {
		return fallback, nil
	}

	value = strings.TrimSpace(value)
	duration, err := time.ParseDuration(value)
	if err != nil {
		return 0, fmt.Errorf("parse %s: %w", name, err)
	}
	if duration <= 0 {
		return 0, fmt.Errorf("%s must be greater than zero", name)
	}

	return duration, nil
}

func loadAuth(lookup lookupEnv, environment string) (AuthConfig, error) {
	secret := optionalString(lookup, "JWT_SECRET")
	if secret == "" {
		if environment != defaultEnvironment {
			return AuthConfig{}, fmt.Errorf("JWT_SECRET must be set outside development")
		}
		secret = developmentJWTSecret
	}
	if len(secret) < 32 {
		return AuthConfig{}, fmt.Errorf("JWT_SECRET must be at least 32 bytes")
	}

	accessTTL, err := durationValue(lookup, "ACCESS_TOKEN_TTL", defaultAccessTokenTTL)
	if err != nil {
		return AuthConfig{}, err
	}
	refreshTTL, err := durationValue(lookup, "REFRESH_TOKEN_TTL", defaultRefreshTokenTTL)
	if err != nil {
		return AuthConfig{}, err
	}
	webURL, err := stringValue(lookup, "WEB_PUBLIC_URL", defaultWebPublicURL)
	if err != nil {
		return AuthConfig{}, err
	}
	if parsed, err := url.Parse(webURL); err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return AuthConfig{}, fmt.Errorf("WEB_PUBLIC_URL must be an absolute URL")
	}
	brand, err := stringValue(lookup, "BRAND_NAME", defaultBrandName)
	if err != nil {
		return AuthConfig{}, err
	}

	return AuthConfig{
		JWTSecret:       secret,
		AccessTokenTTL:  accessTTL,
		RefreshTokenTTL: refreshTTL,
		WebPublicURL:    strings.TrimSuffix(webURL, "/"),
		BrandName:       brand,
	}, nil
}

// CommerceConfig holds voucher signing and payment provider settings.
type CommerceConfig struct {
	// QRSecret signs voucher QR tokens. It must be at least 32 bytes.
	QRSecret string
	// MockPaymentsEnabled exposes the mock provider. It defaults to on in development only.
	MockPaymentsEnabled bool
	// MockWebhookSecret signs and verifies mock provider webhooks. It must be at least 32 bytes.
	MockWebhookSecret         string
	ManualPaymentEnabled      bool
	ManualPaymentHold         time.Duration
	ManualPaymentInstructions string
}

func loadCommerce(lookup lookupEnv, environment string, demoMode bool) (CommerceConfig, error) {
	development := environment == defaultEnvironment

	qrSecret := optionalString(lookup, "VOUCHER_QR_SECRET")
	if qrSecret == "" {
		if !development {
			return CommerceConfig{}, fmt.Errorf("VOUCHER_QR_SECRET must be set outside development")
		}
		qrSecret = developmentQRSecret
	}
	if len(qrSecret) < 32 {
		return CommerceConfig{}, fmt.Errorf("VOUCHER_QR_SECRET must be at least 32 bytes")
	}

	mockEnabled, err := boolValue(lookup, "PAYMENT_MOCK_ENABLED", demoMode)
	if err != nil {
		return CommerceConfig{}, err
	}
	if !demoMode {
		mockEnabled = false
	}
	mockSecret := optionalString(lookup, "PAYMENT_MOCK_WEBHOOK_SECRET")
	if mockSecret == "" && mockEnabled {
		if !development {
			return CommerceConfig{}, fmt.Errorf("PAYMENT_MOCK_WEBHOOK_SECRET must be set when mock payments are enabled outside development")
		}
		mockSecret = developmentMockWebhookSecret
	}
	if mockEnabled && len(mockSecret) < 32 {
		return CommerceConfig{}, fmt.Errorf("PAYMENT_MOCK_WEBHOOK_SECRET must be at least 32 bytes")
	}

	manualEnabled, err := boolValue(lookup, "MANUAL_PAYMENT_ENABLED", false)
	if err != nil {
		return CommerceConfig{}, err
	}
	manualHold, err := durationValue(lookup, "MANUAL_PAYMENT_HOLD", 24*time.Hour)
	if err != nil {
		return CommerceConfig{}, err
	}
	manualInstructions, err := stringValue(lookup, "MANUAL_PAYMENT_INSTRUCTIONS", "Bank transfer details will be provided by support. Include your order number.")
	if err != nil {
		return CommerceConfig{}, err
	}
	return CommerceConfig{QRSecret: qrSecret, MockPaymentsEnabled: mockEnabled, MockWebhookSecret: mockSecret, ManualPaymentEnabled: manualEnabled, ManualPaymentHold: manualHold, ManualPaymentInstructions: manualInstructions}, nil
}

func loadSMTP(lookup lookupEnv) (SMTPConfig, error) {
	host := optionalString(lookup, "SMTP_HOST")
	port := defaultSMTPPort
	if value := optionalString(lookup, "SMTP_PORT"); value != "" {
		parsed, err := strconv.Atoi(value)
		if err != nil || parsed < 1 || parsed > 65535 {
			return SMTPConfig{}, fmt.Errorf("SMTP_PORT must be a valid TCP port")
		}
		port = parsed
	}
	from, err := stringValue(lookup, "SMTP_FROM", defaultSMTPFrom)
	if err != nil {
		return SMTPConfig{}, err
	}
	return SMTPConfig{Host: host, Port: port, From: from}, nil
}

func boolValue(lookup lookupEnv, name string, fallback bool) (bool, error) {
	value, ok := lookup(name)
	if !ok {
		return fallback, nil
	}
	parsed, err := strconv.ParseBool(strings.TrimSpace(value))
	if err != nil {
		return false, fmt.Errorf("parse %s: %w", name, err)
	}
	return parsed, nil
}
