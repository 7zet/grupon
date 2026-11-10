// Package catalog holds the public marketplace read model: cities, categories, deals, merchants,
// reviews, promo codes, collections and homepage sections. It has no HTTP or SQL knowledge.
package catalog

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// Currency is the only charging currency. Money is always integer minor units.
const Currency = "UZS"

// Text is a translated string with one value per supported locale.
type Text struct {
	Uz string `json:"uz"`
	Ru string `json:"ru"`
	En string `json:"en"`
}

// Resolve returns the requested locale, then uz, ru and en as fallbacks.
func (t Text) Resolve(locale string) string {
	values := map[string]string{"uz": t.Uz, "ru": t.Ru, "en": t.En}
	if value := values[locale]; value != "" {
		return value
	}
	for _, fallback := range []string{"uz", "ru", "en"} {
		if values[fallback] != "" {
			return values[fallback]
		}
	}
	return ""
}

// ParseText decodes a localized JSON object stored in the database.
func ParseText(raw []byte) (Text, error) {
	var text Text
	if len(raw) == 0 {
		return text, nil
	}
	if err := json.Unmarshal(raw, &text); err != nil {
		return Text{}, fmt.Errorf("decode localized text: %w", err)
	}
	return text, nil
}

// Money is an amount in minor units of Currency.
type Money struct {
	Amount int64 `json:"amount"`
}

// City is an active charging and browsing city.
type City struct {
	ID   uuid.UUID `json:"id"`
	Slug string    `json:"slug"`
	Name Text      `json:"name"`
	Lat  float64   `json:"lat"`
	Lng  float64   `json:"lng"`
}

// Subcategory is a leaf taxonomy node below a category.
type Subcategory struct {
	ID   uuid.UUID `json:"id"`
	Slug string    `json:"slug"`
	Name Text      `json:"name"`
}

// Category is an active top-level taxonomy node.
type Category struct {
	ID                  uuid.UUID     `json:"id"`
	Slug                string        `json:"slug"`
	Name                Text          `json:"name"`
	Icon                string        `json:"icon"`
	Searchable          bool          `json:"searchable"`
	DealCreationEnabled bool          `json:"deal_creation_enabled"`
	PublishEnabled      bool          `json:"publish_enabled"`
	Purchasable         bool          `json:"purchasable"`
	Redeemable          bool          `json:"redeemable"`
	Subcategories       []Subcategory `json:"subcategories"`
}

// Merchant is a business that publishes deals.
type Merchant struct {
	ID            uuid.UUID  `json:"id"`
	Slug          string     `json:"slug"`
	LegalName     string     `json:"legal_name"`
	DisplayName   string     `json:"display_name"`
	Description   Text       `json:"description"`
	LogoURL       *string    `json:"logo_url,omitempty"`
	CoverURL      *string    `json:"cover_url,omitempty"`
	Phone         string     `json:"phone"`
	Website       *string    `json:"website,omitempty"`
	Instagram     *string    `json:"instagram,omitempty"`
	Telegram      *string    `json:"telegram,omitempty"`
	RatingAvg     float64    `json:"rating_avg"`
	RatingCount   int        `json:"rating_count"`
	CommissionBPS int        `json:"commission_bps"`
	Locations     []Location `json:"locations,omitempty"`
}

// Location is a merchant branch.
type Location struct {
	ID           uuid.UUID      `json:"id"`
	City         City           `json:"city"`
	Address      string         `json:"address"`
	Lat          float64        `json:"lat"`
	Lng          float64        `json:"lng"`
	OpeningHours map[string]any `json:"opening_hours"`
	Phone        string         `json:"phone"`
}

// Option is a purchasable variant with its own price and stock.
type Option struct {
	ID            uuid.UUID `json:"id"`
	Title         Text      `json:"title"`
	OriginalPrice Money     `json:"original_price"`
	DealPrice     Money     `json:"deal_price"`
	StockTotal    int       `json:"stock_total"`
	StockSold     int       `json:"stock_sold"`
	StockReserved int       `json:"stock_reserved"`
	IsActive      bool      `json:"is_active"`
}

// Media is an approved deal image.
type Media struct {
	ID        uuid.UUID `json:"id"`
	URL       string    `json:"url"`
	Alt       Text      `json:"alt"`
	SortOrder int       `json:"sort_order"`
	Width     int       `json:"width"`
	Height    int       `json:"height"`
	Blurhash  string    `json:"blurhash"`
}

// DealSummary is the card-level view of a deal.
type DealSummary struct {
	ID          uuid.UUID  `json:"id"`
	Slug        string     `json:"slug"`
	Area        string     `json:"area"`
	Merchant    Merchant   `json:"merchant"`
	Category    Category   `json:"category"`
	Title       Text       `json:"title"`
	Status      string     `json:"status"`
	Options     []Option   `json:"options"`
	Media       []Media    `json:"media"`
	SoldCount   int        `json:"sold_count"`
	RatingAvg   float64    `json:"rating_avg"`
	RatingCount int        `json:"rating_count"`
	StartsAt    *time.Time `json:"starts_at,omitempty"`
	EndsAt      time.Time  `json:"ends_at"`
}

// FAQItem is one question and answer on a deal page.
type FAQItem struct {
	Q Text `json:"q"`
	A Text `json:"a"`
}

// Deal is the full deal-detail view.
type Deal struct {
	DealSummary
	Subcategory            Subcategory `json:"subcategory"`
	ShortDescription       Text        `json:"short_description"`
	Description            Text        `json:"description"`
	Highlights             []Text      `json:"highlights"`
	WhatsIncluded          []Text      `json:"whats_included"`
	Terms                  Text        `json:"terms"`
	FAQ                    []FAQItem   `json:"faq"`
	RedemptionMethod       string      `json:"redemption_method"`
	RedemptionInstructions *Text       `json:"redemption_instructions,omitempty"`
	BookingRequired        bool        `json:"booking_required"`
	VoucherValidDays       *int        `json:"voucher_valid_days,omitempty"`
	VoucherValidUntil      *time.Time  `json:"voucher_valid_until,omitempty"`
	MaxPerUser             int         `json:"max_per_user"`
	Locations              []Location  `json:"locations"`
	// RejectionReason is shown to the owning merchant only; public queries leave it empty.
	RejectionReason *string `json:"rejection_reason,omitempty"`
}

// Review is a published verified-purchase review.
type Review struct {
	ID         uuid.UUID    `json:"id"`
	DealID     uuid.UUID    `json:"deal_id"`
	MerchantID uuid.UUID    `json:"merchant_id"`
	UserID     uuid.UUID    `json:"user_id"`
	VoucherID  uuid.UUID    `json:"voucher_id"`
	Rating     int          `json:"rating"`
	Text       string       `json:"text"`
	Locale     string       `json:"locale"`
	Status     string       `json:"status"`
	Reply      *ReviewReply `json:"reply,omitempty"`
	CreatedAt  time.Time    `json:"created_at"`
}

// ReviewReply is the merchant's single reply to a review.
type ReviewReply struct {
	ID        uuid.UUID `json:"id"`
	Text      string    `json:"text"`
	CreatedAt time.Time `json:"created_at"`
}

// HomepageSection is one curated homepage block with its resolved content.
type HomepageSection struct {
	ID         uuid.UUID     `json:"id"`
	Key        string        `json:"key"`
	Title      Text          `json:"title"`
	Type       string        `json:"type"`
	SortOrder  int           `json:"sort_order"`
	Deals      []DealSummary `json:"deals,omitempty"`
	Categories []Category    `json:"categories,omitempty"`
}

// PageInfo describes where a page sits within a result set.
type PageInfo struct {
	Page     int   `json:"page"`
	PageSize int   `json:"page_size"`
	Total    int64 `json:"total"`
}

// DealPage is one page of deal summaries.
type DealPage struct {
	PageInfo
	Items []DealSummary `json:"items"`
}

// MerchantProfile is a merchant with its active deals and recent reviews.
type MerchantProfile struct {
	Merchant
	Deals   []DealSummary `json:"deals"`
	Reviews []Review      `json:"reviews"`
}

// Suggestion is one search-bar suggestion.
type Suggestion struct {
	Type  string `json:"type"`
	Label string `json:"label"`
	Href  string `json:"href"`
}

// PublicConfig is the bootstrap payload for every page.
type PublicConfig struct {
	Locales    []string   `json:"locales"`
	Currencies []string   `json:"currencies"`
	Cities     []City     `json:"cities"`
	Categories []Category `json:"categories"`
}
