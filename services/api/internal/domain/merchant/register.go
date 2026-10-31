// Package merchant holds the business rules for merchants: registration, roles and, later, deals
// and staff. Persistence is behind the Repository port.
package merchant

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/google/uuid"
)

// ErrCityNotFound and ErrCategoryNotFound report references to inactive or unknown taxonomy.
var (
	ErrCityNotFound     = errors.New("merchant: city not found")
	ErrCategoryNotFound = errors.New("merchant: category not found")
)

// InvalidInputError reports a client error with a stable, translatable code.
type InvalidInputError struct {
	Field string
	Code  string
}

func (e *InvalidInputError) Error() string {
	return fmt.Sprintf("merchant: invalid %s: %s", e.Field, e.Code)
}

// RegisterInput is one business sign-up. CategoryID is validated but not stored: merchants have no
// category column, because categories belong to deals.
type RegisterInput struct {
	LegalName   string
	DisplayName string
	CategoryID  uuid.UUID
	CityID      uuid.UUID
	Phone       string
	Address     string
	Lat         float64
	Lng         float64
}

// Registration is the normalized write set handed to the repository.
type Registration struct {
	Slug        string
	LegalName   string
	DisplayName string
	Phone       string
	CityID      uuid.UUID
	CategoryID  uuid.UUID
	Address     string
	Lat         float64
	Lng         float64
}

// Created is what the caller sees after a successful registration.
type Created struct {
	ID          uuid.UUID
	Slug        string
	DisplayName string
	LegalName   string
	Phone       string
	Status      string
}

// Repository persists a registration and its owner role in one transaction. It returns
// ErrCityNotFound or ErrCategoryNotFound when a reference is inactive or missing.
type Repository interface {
	RegisterOwned(ctx context.Context, ownerID uuid.UUID, registration Registration) (Created, error)
}

var (
	phonePattern = regexp.MustCompile(`^\+?[0-9][0-9 ()-]{6,19}$`)
	slugUnsafe   = regexp.MustCompile(`[^a-z0-9]+`)
)

// Service implements merchant registration.
type Service struct {
	repo Repository
}

// NewService binds the registration rules to a repository.
func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

// Register validates the sign-up, creates the business and assigns the owner role.
func (s *Service) Register(ctx context.Context, ownerID uuid.UUID, input RegisterInput) (Created, error) {
	registration, err := normalize(input)
	if err != nil {
		return Created{}, err
	}
	return s.repo.RegisterOwned(ctx, ownerID, registration)
}

func normalize(input RegisterInput) (Registration, error) {
	legal := strings.TrimSpace(input.LegalName)
	display := strings.TrimSpace(input.DisplayName)
	phone := strings.TrimSpace(input.Phone)
	address := strings.TrimSpace(input.Address)

	switch {
	case len([]rune(legal)) < 2 || len([]rune(legal)) > 300:
		return Registration{}, &InvalidInputError{Field: "legal_name", Code: "length"}
	case len([]rune(display)) < 2 || len([]rune(display)) > 200:
		return Registration{}, &InvalidInputError{Field: "display_name", Code: "length"}
	case !phonePattern.MatchString(phone):
		return Registration{}, &InvalidInputError{Field: "phone", Code: "invalid_phone"}
	case address == "":
		return Registration{}, &InvalidInputError{Field: "address", Code: "required"}
	case input.Lat < -90 || input.Lat > 90 || input.Lng < -180 || input.Lng > 180:
		return Registration{}, &InvalidInputError{Field: "lat", Code: "out_of_range"}
	case input.CategoryID == uuid.Nil:
		return Registration{}, &InvalidInputError{Field: "category_id", Code: "required"}
	case input.CityID == uuid.Nil:
		return Registration{}, &InvalidInputError{Field: "city_id", Code: "required"}
	}

	suffix, err := randomSuffix()
	if err != nil {
		return Registration{}, fmt.Errorf("generate slug suffix: %w", err)
	}
	return Registration{
		Slug:        slugFor(display) + "-" + suffix,
		LegalName:   legal,
		DisplayName: display,
		Phone:       phone,
		CityID:      input.CityID,
		CategoryID:  input.CategoryID,
		Address:     address,
		Lat:         input.Lat,
		Lng:         input.Lng,
	}, nil
}

// slugFor turns a display name into a URL segment. The random suffix that follows makes it unique.
func slugFor(name string) string {
	slug := slugUnsafe.ReplaceAllString(strings.ToLower(name), "-")
	slug = strings.Trim(slug, "-")
	if slug == "" {
		return "business"
	}
	if len(slug) > 60 {
		slug = strings.Trim(slug[:60], "-")
	}
	return slug
}

func randomSuffix() (string, error) {
	buffer := make([]byte, 4)
	if _, err := rand.Read(buffer); err != nil {
		return "", err
	}
	return hex.EncodeToString(buffer), nil
}
