package merchant

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
)

type recordingRepo struct {
	got       Registration
	submitted bool
}

func (r *recordingRepo) RegisterOwned(_ context.Context, _ uuid.UUID, registration Registration) (Created, error) {
	r.got = registration
	return Created{Slug: registration.Slug, Status: "active", ReviewStatus: "draft"}, nil
}

func (r *recordingRepo) SubmitOwned(_ context.Context, _ uuid.UUID, _ time.Time) (Created, error) {
	if r.submitted {
		return Created{}, ErrNotDraft
	}
	r.submitted = true
	return Created{ReviewStatus: "submitted"}, nil
}

var fixedNow = func() time.Time { return time.Date(2026, 11, 2, 9, 0, 0, 0, time.UTC) }

func validInput() RegisterInput {
	return RegisterInput{
		LegalName:        "Lotus Spa LLC",
		DisplayName:      "Lotus Spa",
		CategoryID:       uuid.New(),
		CityID:           uuid.New(),
		Phone:            "+998 90 123 45 67",
		Address:          "Amir Temur 10",
		Lat:              41.31,
		Lng:              69.24,
		STIR:             "305123456",
		OrgForm:          "mchj",
		TaxRegime:        "vat_payer",
		BankMFO:          "00014",
		BankAccount:      "20208000900100100001",
		ResponsibleName:  "Ulugbek Karimov",
		ContractVersion:  "v1.1",
		ContractAccepted: true,
		ContractIP:       "198.51.100.7",
	}
}

func TestRegisterAcceptsValidBusinessAndRecordsOfferAcceptance(t *testing.T) {
	t.Parallel()
	repo := &recordingRepo{}
	service := NewService(repo, fixedNow)

	created, err := service.Register(context.Background(), uuid.New(), validInput())
	if err != nil {
		t.Fatalf("Register: %v", err)
	}
	if created.Slug == "" || repo.got.Slug != created.Slug {
		t.Fatalf("slug not passed through: %q vs %q", created.Slug, repo.got.Slug)
	}
	if !repo.got.ContractAcceptedAt.Equal(fixedNow()) {
		t.Fatalf("contract acceptance time = %v", repo.got.ContractAcceptedAt)
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
		"short legal name":   {func(i *RegisterInput) { i.LegalName = "A" }, "legal_name", "length"},
		"bad phone":          {func(i *RegisterInput) { i.Phone = "call me" }, "phone", "invalid_phone"},
		"no address":         {func(i *RegisterInput) { i.Address = "   " }, "address", "required"},
		"lat out of range":   {func(i *RegisterInput) { i.Lat = 120 }, "lat", "out_of_range"},
		"no category":        {func(i *RegisterInput) { i.CategoryID = uuid.Nil }, "category_id", "required"},
		"short STIR":         {func(i *RegisterInput) { i.STIR = "12345" }, "stir", "invalid_stir"},
		"letters in STIR":    {func(i *RegisterInput) { i.STIR = "30512345A" }, "stir", "invalid_stir"},
		"unknown org form":   {func(i *RegisterInput) { i.OrgForm = "llc" }, "org_form", "invalid_org_form"},
		"unknown tax":        {func(i *RegisterInput) { i.TaxRegime = "patent" }, "tax_regime", "invalid_tax_regime"},
		"bad MFO":            {func(i *RegisterInput) { i.BankMFO = "1234" }, "bank_mfo", "invalid_mfo"},
		"short account":      {func(i *RegisterInput) { i.BankAccount = "2020800090010010000" }, "bank_account", "invalid_bank_account"},
		"no responsible":     {func(i *RegisterInput) { i.ResponsibleName = " " }, "responsible_name", "length"},
		"no contract ver":    {func(i *RegisterInput) { i.ContractVersion = "" }, "contract_version", "required"},
		"offer not accepted": {func(i *RegisterInput) { i.ContractAccepted = false }, "contract_accepted", "contract_not_accepted"},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			input := validInput()
			tc.mutate(&input)
			_, err := NewService(&recordingRepo{}, fixedNow).Register(context.Background(), uuid.New(), input)
			var invalid *InvalidInputError
			if !errors.As(err, &invalid) || invalid.Field != tc.field || invalid.Code != tc.code {
				t.Fatalf("err = %v, want %s/%s", err, tc.field, tc.code)
			}
		})
	}
}

func TestSubmitMovesDraftOnceAndRejectsSecondSubmit(t *testing.T) {
	t.Parallel()
	service := NewService(&recordingRepo{}, fixedNow)
	if _, err := service.Submit(context.Background(), uuid.New()); err != nil {
		t.Fatalf("first submit: %v", err)
	}
	if _, err := service.Submit(context.Background(), uuid.New()); !errors.Is(err, ErrNotDraft) {
		t.Fatalf("second submit: got %v, want ErrNotDraft", err)
	}
}
