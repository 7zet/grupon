package merchant

import (
	"context"
	"errors"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/grupon/grupon/services/api/internal/domain/catalog"
)

// ErrNoMerchant reports that the signed-in account holds no merchant role.
// ErrForbidden reports that the role may not perform the action; ErrNotFound a missing branch.
var (
	ErrNoMerchant    = errors.New("merchant: no merchant role")
	ErrForbidden     = errors.New("merchant: forbidden")
	ErrNotFound      = errors.New("merchant: not found")
	ErrLocationInUse = errors.New("merchant: location used by live deals")
)

// Role names mirror the database enum for merchant roles.
const (
	RoleOwner   = "merchant_owner"
	RoleManager = "merchant_manager"
	RoleCashier = "merchant_cashier"
)

// Business is the editable merchant profile.
type Business struct {
	ID uuid.UUID
	// ReviewStatus is draft, submitted, in_review, approved, rejected, suspended or terminated.
	ReviewStatus string
	Slug         string
	LegalName    string
	DisplayName  string
	Description  catalog.Text
	LogoURL      *string
	CoverURL     *string
	Phone        string
	Website      *string
	Instagram    *string
	Telegram     *string
	Status       string
	RatingAvg    float64
	RatingCount  int
}

// BusinessPatch carries the fields a manager or owner may change. Nil means "unchanged".
type BusinessPatch struct {
	DisplayName *string
	Description *catalog.Text
	LogoURL     *string
	CoverURL    *string
	Phone       *string
	Website     *string
	Instagram   *string
	Telegram    *string
}

// Location is a branch with its hours.
type Location struct {
	ID           uuid.UUID
	City         catalog.City
	Address      string
	Lat          float64
	Lng          float64
	OpeningHours map[string]any
	Phone        string
}

// LocationInput is a branch write. Opening hours are an object keyed by weekday.
type LocationInput struct {
	CityID       uuid.UUID
	Address      string
	Lat          float64
	Lng          float64
	OpeningHours map[string]any
	Phone        string
}

// Metric is a headline number with its change against the previous window, in percent.
type Metric struct {
	Value         int64
	ChangePercent float64
}

// ChartPoint is one day of sales in UZS minor units.
type ChartPoint struct {
	Day   time.Time
	Value int64
}

// Dashboard is the merchant overview for the last 30 days.
type Dashboard struct {
	TotalSales       Metric
	VouchersSold     Metric
	VouchersRedeemed Metric
	ActiveDeals      Metric
	SalesChart       []ChartPoint
	PendingDeals     []catalog.DealSummary
}

// WorkspaceRepository is the persistence port for the merchant workspace. Every method is scoped
// to one merchant so a caller can never reach another business's rows.
type WorkspaceRepository interface {
	RoleFor(ctx context.Context, userID uuid.UUID) (merchantID uuid.UUID, role string, err error)
	Business(ctx context.Context, merchantID uuid.UUID) (Business, error)
	UpdateBusiness(ctx context.Context, merchantID uuid.UUID, business Business) (Business, error)
	Locations(ctx context.Context, merchantID uuid.UUID) ([]Location, error)
	CreateLocation(ctx context.Context, merchantID uuid.UUID, input LocationInput) (Location, error)
	UpdateLocation(ctx context.Context, merchantID, locationID uuid.UUID, input LocationInput) (bool, error)
	DeleteLocation(ctx context.Context, merchantID, locationID uuid.UUID) (deleted bool, err error)
	Sales(ctx context.Context, merchantID uuid.UUID, from, to time.Time) (int64, error)
	VouchersSold(ctx context.Context, merchantID uuid.UUID, from, to time.Time) (int64, error)
	VouchersRedeemed(ctx context.Context, merchantID uuid.UUID, from, to time.Time) (int64, error)
	ActiveDeals(ctx context.Context, merchantID uuid.UUID) (int64, error)
	SalesByDay(ctx context.Context, merchantID uuid.UUID, from, to time.Time) ([]ChartPoint, error)
	PendingDealIDs(ctx context.Context, merchantID uuid.UUID, limit int) ([]uuid.UUID, error)
	DealSummaries(ctx context.Context, ids []uuid.UUID) ([]catalog.DealSummary, error)
}

var (
	hoursKeyPattern = regexp.MustCompile(`^(mon|tue|wed|thu|fri|sat|sun)$`)
	websitePattern  = regexp.MustCompile(`^https?://[^\s]+$`)
)

// Authorize resolves the merchant for the signed-in account. Managers and owners edit the
// business; cashiers only redeem, so they are refused here as well as at the HTTP layer.
func (s *WorkspaceService) Authorize(ctx context.Context, userID uuid.UUID, mayEdit bool) (uuid.UUID, string, error) {
	merchantID, role, err := s.repo.RoleFor(ctx, userID)
	if err != nil {
		return uuid.Nil, "", err
	}
	if mayEdit && role == RoleCashier {
		return uuid.Nil, "", ErrForbidden
	}
	return merchantID, role, nil
}

// WorkspaceService implements the merchant overview, business profile and branches.
type WorkspaceService struct {
	repo WorkspaceRepository
	now  func() time.Time
}

// NewWorkspaceService binds the workspace rules to a repository.
func NewWorkspaceService(repo WorkspaceRepository, now func() time.Time) *WorkspaceService {
	if now == nil {
		now = time.Now
	}
	return &WorkspaceService{repo: repo, now: now}
}

// UpdateBusiness applies a patch after validating each changed field.
func (s *WorkspaceService) UpdateBusiness(ctx context.Context, merchantID uuid.UUID, patch BusinessPatch) (Business, error) {
	current, err := s.repo.Business(ctx, merchantID)
	if err != nil {
		return Business{}, err
	}
	if patch.DisplayName != nil {
		name := strings.TrimSpace(*patch.DisplayName)
		if len([]rune(name)) < 2 || len([]rune(name)) > 200 {
			return Business{}, &InvalidInputError{Field: "display_name", Code: "length"}
		}
		current.DisplayName = name
	}
	if patch.Description != nil {
		current.Description = *patch.Description
	}
	if patch.LogoURL != nil {
		current.LogoURL = validURL(patch.LogoURL)
	}
	if patch.CoverURL != nil {
		current.CoverURL = validURL(patch.CoverURL)
	}
	if patch.Phone != nil {
		phone := strings.TrimSpace(*patch.Phone)
		if !phonePattern.MatchString(phone) {
			return Business{}, &InvalidInputError{Field: "phone", Code: "invalid_phone"}
		}
		current.Phone = phone
	}
	if patch.Website != nil {
		if *patch.Website != "" && !websitePattern.MatchString(*patch.Website) {
			return Business{}, &InvalidInputError{Field: "website", Code: "invalid_url"}
		}
		current.Website = nilIfEmpty(patch.Website)
	}
	if patch.Instagram != nil {
		current.Instagram = nilIfEmpty(patch.Instagram)
	}
	if patch.Telegram != nil {
		current.Telegram = nilIfEmpty(patch.Telegram)
	}
	return s.repo.UpdateBusiness(ctx, merchantID, current)
}

// Business returns the current profile.
func (s *WorkspaceService) Business(ctx context.Context, merchantID uuid.UUID) (Business, error) {
	return s.repo.Business(ctx, merchantID)
}

// Locations lists every branch of the merchant.
func (s *WorkspaceService) Locations(ctx context.Context, merchantID uuid.UUID) ([]Location, error) {
	return s.repo.Locations(ctx, merchantID)
}

// CreateLocation adds a branch after validating its coordinates, address and hours.
func (s *WorkspaceService) CreateLocation(ctx context.Context, merchantID uuid.UUID, input LocationInput) (Location, error) {
	if err := validateLocation(input); err != nil {
		return Location{}, err
	}
	return s.repo.CreateLocation(ctx, merchantID, input)
}

// UpdateLocation changes a branch the merchant owns.
func (s *WorkspaceService) UpdateLocation(ctx context.Context, merchantID, locationID uuid.UUID, input LocationInput) error {
	if err := validateLocation(input); err != nil {
		return err
	}
	updated, err := s.repo.UpdateLocation(ctx, merchantID, locationID, input)
	if err != nil {
		return err
	}
	if !updated {
		return ErrNotFound
	}
	return nil
}

// DeleteLocation removes a branch unless live deals still sell vouchers there.
func (s *WorkspaceService) DeleteLocation(ctx context.Context, merchantID, locationID uuid.UUID) error {
	deleted, err := s.repo.DeleteLocation(ctx, merchantID, locationID)
	if err != nil {
		return err
	}
	if !deleted {
		return ErrLocationInUse
	}
	return nil
}

// Dashboard builds the overview: 30-day totals compared with the previous 30 days.
func (s *WorkspaceService) Dashboard(ctx context.Context, merchantID uuid.UUID) (Dashboard, error) {
	now := s.now()
	currentFrom := now.AddDate(0, 0, -30)
	previousFrom := now.AddDate(0, 0, -60)

	sales, err := s.repo.Sales(ctx, merchantID, currentFrom, now)
	if err != nil {
		return Dashboard{}, err
	}
	salesPrevious, err := s.repo.Sales(ctx, merchantID, previousFrom, currentFrom)
	if err != nil {
		return Dashboard{}, err
	}
	sold, err := s.repo.VouchersSold(ctx, merchantID, currentFrom, now)
	if err != nil {
		return Dashboard{}, err
	}
	soldPrevious, err := s.repo.VouchersSold(ctx, merchantID, previousFrom, currentFrom)
	if err != nil {
		return Dashboard{}, err
	}
	redeemed, err := s.repo.VouchersRedeemed(ctx, merchantID, currentFrom, now)
	if err != nil {
		return Dashboard{}, err
	}
	redeemedPrevious, err := s.repo.VouchersRedeemed(ctx, merchantID, previousFrom, currentFrom)
	if err != nil {
		return Dashboard{}, err
	}
	active, err := s.repo.ActiveDeals(ctx, merchantID)
	if err != nil {
		return Dashboard{}, err
	}
	chart, err := s.repo.SalesByDay(ctx, merchantID, currentFrom, now)
	if err != nil {
		return Dashboard{}, err
	}
	pendingIDs, err := s.repo.PendingDealIDs(ctx, merchantID, 10)
	if err != nil {
		return Dashboard{}, err
	}
	pending, err := s.repo.DealSummaries(ctx, pendingIDs)
	if err != nil {
		return Dashboard{}, err
	}
	return Dashboard{
		TotalSales:       Metric{Value: sales, ChangePercent: changePercent(sales, salesPrevious)},
		VouchersSold:     Metric{Value: sold, ChangePercent: changePercent(sold, soldPrevious)},
		VouchersRedeemed: Metric{Value: redeemed, ChangePercent: changePercent(redeemed, redeemedPrevious)},
		ActiveDeals:      Metric{Value: active},
		SalesChart:       chart,
		PendingDeals:     pending,
	}, nil
}

// changePercent is relative growth. A zero base reports zero rather than an undefined percentage.
func changePercent(current, previous int64) float64 {
	if previous == 0 {
		return 0
	}
	return float64(current-previous) * 100 / float64(previous)
}

func validateLocation(input LocationInput) error {
	address := strings.TrimSpace(input.Address)
	switch {
	case address == "" || len([]rune(address)) > 500:
		return &InvalidInputError{Field: "address", Code: "length"}
	case input.Lat < -90 || input.Lat > 90 || input.Lng < -180 || input.Lng > 180:
		return &InvalidInputError{Field: "lat", Code: "out_of_range"}
	case input.CityID == uuid.Nil:
		return &InvalidInputError{Field: "city_id", Code: "required"}
	case !phonePattern.MatchString(strings.TrimSpace(input.Phone)):
		return &InvalidInputError{Field: "phone", Code: "invalid_phone"}
	}
	for key, value := range input.OpeningHours {
		if !hoursKeyPattern.MatchString(key) {
			return &InvalidInputError{Field: "opening_hours", Code: "unknown_day"}
		}
		if _, ok := value.(string); !ok {
			return &InvalidInputError{Field: "opening_hours", Code: "invalid_hours"}
		}
	}
	return nil
}

func validURL(value *string) *string {
	if value == nil || *value == "" {
		return nil
	}
	return value
}

func nilIfEmpty(value *string) *string {
	if value == nil || strings.TrimSpace(*value) == "" {
		return nil
	}
	trimmed := strings.TrimSpace(*value)
	return &trimmed
}
