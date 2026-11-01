package merchant

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/grupon/grupon/services/api/internal/domain/catalog"
)

// Deal statuses that a merchant may see in the owned list. They mirror the deal_status enum.
const (
	DealDraft         = "draft"
	DealPendingReview = "pending_review"
	DealRejected      = "rejected"
	DealScheduled     = "scheduled"
	DealActive        = "active"
	DealPaused        = "paused"
	DealSoldOut       = "sold_out"
	DealExpired       = "expired"
)

// Errors returned by the deal workflow. The HTTP layer maps them to problem codes.
var (
	ErrDealNotFound     = errors.New("merchant: deal not found")
	ErrDealStateInvalid = errors.New("merchant: deal state does not allow this action")
)

var (
	slugPattern  = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)
	dealStatuses = map[string]bool{
		DealDraft: true, DealPendingReview: true, DealRejected: true, DealScheduled: true,
		DealActive: true, DealPaused: true, DealSoldOut: true, DealExpired: true,
	}
	// reviewedStatuses are live deals whose price, terms or options edits send them back to review.
	reviewedStatuses = map[string]bool{DealScheduled: true, DealActive: true, DealPaused: true, DealSoldOut: true}
)

// MaxRedemptionsPerVoucher caps how many times one voucher may be used (multi-use options).
const MaxRedemptionsPerVoucher = 50

// DealOptionData is one purchasable variant with its own price and stock.
type DealOptionData struct {
	ID            *uuid.UUID
	Title         catalog.Text
	OriginalPrice int64
	DealPrice     int64
	StockTotal    int
	SortOrder     int
	IsActive      bool
	// RedemptionsAllowed is how many times the voucher of this option can be used. 0 means 1.
	RedemptionsAllowed int
}

// DealMediaData is one image that has been uploaded through the media library.
type DealMediaData struct {
	URL       string
	Alt       catalog.Text
	SortOrder int
	Width     int
	Height    int
	Blurhash  string
}

// DealInput is the complete editable draft of a deal.
type DealInput struct {
	Slug                   string
	CategoryID             uuid.UUID
	SubcategoryID          uuid.UUID
	Title                  catalog.Text
	ShortDescription       catalog.Text
	Description            catalog.Text
	Highlights             []catalog.Text
	WhatsIncluded          []catalog.Text
	Terms                  catalog.Text
	FAQ                    []catalog.FAQItem
	RedemptionMethod       string
	RedemptionInstructions catalog.Text
	BookingRequired        bool
	StartsAt               time.Time
	EndsAt                 time.Time
	VoucherValidDays       *int
	VoucherValidUntil      *time.Time
	MaxPerUser             int
	LocationIDs            []uuid.UUID
	Options                []DealOptionData
	Media                  []DealMediaData
}

// Validate checks the rules the contract cannot express: ranges, cross-field order and required text.
func (in DealInput) Validate() error {
	if !slugPattern.MatchString(in.Slug) {
		return &InvalidInputError{Field: "slug", Code: "invalid_slug"}
	}
	switch in.RedemptionMethod {
	case "qr", "code", "qr_or_code":
	default:
		return &InvalidInputError{Field: "redemption_method", Code: "invalid_redemption_method"}
	}
	for field, text := range map[string]catalog.Text{
		"title": in.Title, "short_description": in.ShortDescription, "description": in.Description,
		"terms": in.Terms, "redemption_instructions": in.RedemptionInstructions,
	} {
		if !fullyLocalized(text) {
			return &InvalidInputError{Field: field, Code: "required"}
		}
	}
	if len(in.Highlights) < 3 || len(in.Highlights) > 5 {
		return &InvalidInputError{Field: "highlights", Code: "highlights_count"}
	}
	for _, highlight := range in.Highlights {
		if !fullyLocalized(highlight) {
			return &InvalidInputError{Field: "highlights", Code: "required"}
		}
	}
	for _, item := range in.WhatsIncluded {
		if !fullyLocalized(item) {
			return &InvalidInputError{Field: "whats_included", Code: "required"}
		}
	}
	for _, item := range in.FAQ {
		if !fullyLocalized(item.Q) || !fullyLocalized(item.A) {
			return &InvalidInputError{Field: "faq", Code: "required"}
		}
	}
	if !in.EndsAt.After(in.StartsAt) {
		return &InvalidInputError{Field: "ends_at", Code: "order"}
	}
	if (in.VoucherValidDays == nil) == (in.VoucherValidUntil == nil) {
		return &InvalidInputError{Field: "voucher_valid_days", Code: "validity_required"}
	}
	if in.VoucherValidDays != nil && *in.VoucherValidDays < 1 {
		return &InvalidInputError{Field: "voucher_valid_days", Code: "range"}
	}
	if in.MaxPerUser < 1 {
		return &InvalidInputError{Field: "max_per_user", Code: "range"}
	}
	if len(in.LocationIDs) == 0 {
		return &InvalidInputError{Field: "location_ids", Code: "required"}
	}
	if len(in.Options) == 0 {
		return &InvalidInputError{Field: "options", Code: "required"}
	}
	for _, option := range in.Options {
		if !fullyLocalized(option.Title) {
			return &InvalidInputError{Field: "options", Code: "required"}
		}
		if option.OriginalPrice < 1 || option.DealPrice < 1 || option.DealPrice > option.OriginalPrice {
			return &InvalidInputError{Field: "options", Code: "price"}
		}
		if option.RedemptionsAllowed < 0 || option.RedemptionsAllowed > MaxRedemptionsPerVoucher {
			return &InvalidInputError{Field: "options", Code: "range"}
		}
		if option.StockTotal < 0 {
			return &InvalidInputError{Field: "options", Code: "range"}
		}
	}
	if len(in.Media) < 3 || len(in.Media) > 8 {
		return &InvalidInputError{Field: "media", Code: "media_count"}
	}
	for _, item := range in.Media {
		if item.URL == "" || item.Width < 1 || item.Height < 1 || !fullyLocalized(item.Alt) {
			return &InvalidInputError{Field: "media", Code: "invalid_media"}
		}
	}
	return nil
}

func fullyLocalized(text catalog.Text) bool {
	return strings.TrimSpace(text.Uz) != "" && strings.TrimSpace(text.Ru) != "" && strings.TrimSpace(text.En) != ""
}

// ReviewFingerprint hashes the fields whose change sends a live deal back to moderation.
// Stock is deliberately excluded: stock-only edits do not need review.
func ReviewFingerprint(in DealInput) string {
	type option struct {
		Title         catalog.Text `json:"title"`
		OriginalPrice int64        `json:"original_price"`
		DealPrice     int64        `json:"deal_price"`
		IsActive      bool         `json:"is_active"`
	}
	options := make([]option, 0, len(in.Options))
	for _, item := range in.Options {
		options = append(options, option{Title: item.Title, OriginalPrice: item.OriginalPrice, DealPrice: item.DealPrice, IsActive: item.IsActive})
	}
	payload, _ := json.Marshal(struct {
		Title                  catalog.Text
		Description            catalog.Text
		Terms                  catalog.Text
		RedemptionMethod       string
		RedemptionInstructions catalog.Text
		Highlights             []catalog.Text
		WhatsIncluded          []catalog.Text
		FAQ                    []catalog.FAQItem
		Options                []option
	}{in.Title, in.Description, in.Terms, in.RedemptionMethod, in.RedemptionInstructions,
		in.Highlights, in.WhatsIncluded, in.FAQ, options})
	sum := sha256.Sum256(payload)
	return hex.EncodeToString(sum[:])
}

// DealRecord is the stored workflow state of a deal the merchant owns.
type DealRecord struct {
	ID          uuid.UUID
	Status      string
	Fingerprint string
}

// DealWrite is what the repository persists for one save.
type DealWrite struct {
	Input       DealInput
	Status      string
	Fingerprint string
}

// AuditEntry records who changed a deal and what it looks like afterwards.
type AuditEntry struct {
	ActorID uuid.UUID
	Role    string
	Action  string
	DealID  uuid.UUID
	After   map[string]any
}

// DealRepository persists owned deals. Every method is scoped to the merchant.
type DealRepository interface {
	// References checks categories, subcategories and locations belong to the expected taxonomy and merchant.
	References(ctx context.Context, merchantID uuid.UUID, in DealInput) error
	Insert(ctx context.Context, merchantID uuid.UUID, write DealWrite, audit AuditEntry) (uuid.UUID, error)
	Update(ctx context.Context, merchantID, dealID uuid.UUID, write DealWrite, audit AuditEntry) error
	Record(ctx context.Context, merchantID, dealID uuid.UUID) (DealRecord, error)
	SetStatus(ctx context.Context, merchantID, dealID uuid.UUID, status string, audit AuditEntry) error
	SoftDeleteDraft(ctx context.Context, merchantID, dealID uuid.UUID, audit AuditEntry) (bool, error)
	Detail(ctx context.Context, merchantID, dealID uuid.UUID) (catalog.Deal, error)
	ListIDs(ctx context.Context, merchantID uuid.UUID, status string, page catalog.Page) ([]uuid.UUID, int64, error)
	Summaries(ctx context.Context, ids []uuid.UUID) ([]catalog.DealSummary, error)
}

// DealService implements the merchant deal workflow: draft, review and the rules for editing live deals.
type DealService struct {
	repo DealRepository
	now  func() time.Time
}

// NewDealService binds the deal workflow to a repository.
func NewDealService(repo DealRepository, now func() time.Time) *DealService {
	if now == nil {
		now = time.Now
	}
	return &DealService{repo: repo, now: now}
}

// Create stores a new draft. Drafts must already be complete, so they can be submitted without edits.
func (s *DealService) Create(ctx context.Context, merchantID uuid.UUID, actor Actor, in DealInput) (catalog.Deal, error) {
	if err := s.prepare(ctx, merchantID, in); err != nil {
		return catalog.Deal{}, err
	}
	write := DealWrite{Input: in, Status: DealDraft, Fingerprint: ReviewFingerprint(in)}
	dealID, err := s.repo.Insert(ctx, merchantID, write, actor.audit("deal_created", uuid.Nil, DealDraft))
	if err != nil {
		return catalog.Deal{}, err
	}
	return s.repo.Detail(ctx, merchantID, dealID)
}

// Update replaces the draft. Editing price, terms or options of a live deal sends it back to review.
func (s *DealService) Update(ctx context.Context, merchantID, dealID uuid.UUID, actor Actor, in DealInput) (catalog.Deal, error) {
	current, err := s.repo.Record(ctx, merchantID, dealID)
	if err != nil {
		return catalog.Deal{}, err
	}
	if err := s.prepare(ctx, merchantID, in); err != nil {
		return catalog.Deal{}, err
	}
	fingerprint := ReviewFingerprint(in)
	status := current.Status
	switch {
	case current.Status == DealRejected:
		status = DealDraft
	case reviewedStatuses[current.Status] && fingerprint != current.Fingerprint:
		status = DealPendingReview
	}
	write := DealWrite{Input: in, Status: status, Fingerprint: fingerprint}
	if err := s.repo.Update(ctx, merchantID, dealID, write, actor.audit("deal_updated", dealID, status)); err != nil {
		return catalog.Deal{}, err
	}
	return s.repo.Detail(ctx, merchantID, dealID)
}

// Submit sends a draft or rejected deal to moderation.
func (s *DealService) Submit(ctx context.Context, merchantID, dealID uuid.UUID, actor Actor) (catalog.Deal, error) {
	current, err := s.repo.Record(ctx, merchantID, dealID)
	if err != nil {
		return catalog.Deal{}, err
	}
	if current.Status != DealDraft && current.Status != DealRejected {
		return catalog.Deal{}, ErrDealStateInvalid
	}
	if err := s.repo.SetStatus(ctx, merchantID, dealID, DealPendingReview, actor.audit("deal_submitted", dealID, DealPendingReview)); err != nil {
		return catalog.Deal{}, err
	}
	return s.repo.Detail(ctx, merchantID, dealID)
}

// Delete soft-deletes a draft. Anything already submitted or live is kept for the record.
func (s *DealService) Delete(ctx context.Context, merchantID, dealID uuid.UUID, actor Actor) error {
	if _, err := s.repo.Record(ctx, merchantID, dealID); err != nil {
		return err
	}
	deleted, err := s.repo.SoftDeleteDraft(ctx, merchantID, dealID, actor.audit("deal_deleted", dealID, ""))
	if err != nil {
		return err
	}
	if !deleted {
		return ErrDealStateInvalid
	}
	return nil
}

// Get returns one owned deal.
func (s *DealService) Get(ctx context.Context, merchantID, dealID uuid.UUID) (catalog.Deal, error) {
	return s.repo.Detail(ctx, merchantID, dealID)
}

// List returns one page of owned deals, optionally filtered by status.
func (s *DealService) List(ctx context.Context, merchantID uuid.UUID, status string, page catalog.Page) (DealList, error) {
	if status != "" && !dealStatuses[status] {
		return DealList{}, &InvalidInputError{Field: "status", Code: "invalid_status"}
	}
	ids, total, err := s.repo.ListIDs(ctx, merchantID, status, page)
	if err != nil {
		return DealList{}, err
	}
	items, err := s.repo.Summaries(ctx, ids)
	if err != nil {
		return DealList{}, err
	}
	return DealList{Items: items, Page: page, Total: total}, nil
}

// DealList is one page of owned deal cards.
type DealList struct {
	Items []catalog.DealSummary
	Page  catalog.Page
	Total int64
}

func (s *DealService) prepare(ctx context.Context, merchantID uuid.UUID, in DealInput) error {
	if err := in.Validate(); err != nil {
		return err
	}
	return s.repo.References(ctx, merchantID, in)
}

// Actor identifies the signed-in merchant user for the audit trail.
type Actor struct {
	UserID uuid.UUID
	Role   string
}

func (a Actor) audit(action string, dealID uuid.UUID, status string) AuditEntry {
	after := map[string]any{}
	if status != "" {
		after["status"] = status
	}
	return AuditEntry{ActorID: a.UserID, Role: a.Role, Action: action, DealID: dealID, After: after}
}
