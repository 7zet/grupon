package merchant

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
)

type recordingRepo struct {
	got Registration
}

func (r *recordingRepo) RegisterOwned(_ context.Context, _ uuid.UUID, registration Registration) (Created, error) {
	r.got = registration
	return Created{Slug: registration.Slug, Status: "active"}, nil
}

func validInput() RegisterInput {
	return RegisterInput{
		LegalName:   "Lotus Spa LLC",
		DisplayName: "Lotus Spa",
		CategoryID:  uuid.New(),
		CityID:      uuid.New(),
		Phone:       "+998 90 123 45 67",
		Address:     "Amir Temur 10",
		Lat:         41.31,
		Lng:         69.24,
	}
}

func TestRegisterAcceptsValidBusinessAndBuildsUniqueSlug(t *testing.T) {
	t.Parallel()
	repo := &recordingRepo{}
	service := NewService(repo)

	created, err := service.Register(context.Background(), uuid.New(), validInput())
	if err != nil {
		t.Fatalf("Register: %v", err)
	}
	if created.Slug == "" || repo.got.Slug != created.Slug {
		t.Fatalf("slug not passed through: %q vs %q", created.Slug, repo.got.Slug)
	}
	if got := slugFor("Lotus Spa & Hammam!"); got != "lotus-spa-hammam" {
		t.Fatalf("slugFor = %q", got)
	}
}

func TestRegisterRejectsInvalidInputWithStableCodes(t *testing.T) {
	t.Parallel()
	cases := map[string]struct {
		mutate func(*RegisterInput)
		field  string
		code   string
	}{
		"short legal name": {func(i *RegisterInput) { i.LegalName = "A" }, "legal_name", "length"},
		"bad phone":        {func(i *RegisterInput) { i.Phone = "call me" }, "phone", "invalid_phone"},
		"no address":       {func(i *RegisterInput) { i.Address = "   " }, "address", "required"},
		"lat out of range": {func(i *RegisterInput) { i.Lat = 120 }, "lat", "out_of_range"},
		"no category":      {func(i *RegisterInput) { i.CategoryID = uuid.Nil }, "category_id", "required"},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			input := validInput()
			tc.mutate(&input)
			_, err := NewService(&recordingRepo{}).Register(context.Background(), uuid.New(), input)
			var invalid *InvalidInputError
			if !errors.As(err, &invalid) || invalid.Field != tc.field || invalid.Code != tc.code {
				t.Fatalf("err = %v, want %s/%s", err, tc.field, tc.code)
			}
		})
	}
}
