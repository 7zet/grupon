package reviews

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

type fakeRepo struct {
	owner   VoucherOwner
	reviews map[uuid.UUID]Review
	reports int
}

func (f *fakeRepo) Voucher(context.Context, uuid.UUID) (VoucherOwner, error) { return f.owner, nil }

func (f *fakeRepo) Insert(_ context.Context, owner VoucherOwner, voucherID, userID uuid.UUID, in Input) (Review, error) {
	for _, existing := range f.reviews {
		if existing.VoucherID == voucherID {
			return Review{}, ErrAlreadyReviewed
		}
	}
	review := Review{ID: uuid.New(), DealID: owner.DealID, MerchantID: owner.MerchantID, UserID: userID, VoucherID: voucherID,
		Rating: in.Rating, Text: in.Text, Locale: in.Locale, Status: StatusPublished, CreatedAt: time.Now()}
	f.reviews[review.ID] = review
	return review, nil
}

func (f *fakeRepo) Find(_ context.Context, id uuid.UUID) (Review, error) {
	review, ok := f.reviews[id]
	if !ok {
		return Review{}, ErrNotFound
	}
	return review, nil
}

func (f *fakeRepo) Update(_ context.Context, id, userID uuid.UUID, in Input) (bool, error) {
	review, ok := f.reviews[id]
	if !ok || review.UserID != userID {
		return false, nil
	}
	review.Rating, review.Text = in.Rating, in.Text
	f.reviews[id] = review
	return true, nil
}

func (f *fakeRepo) SoftDelete(_ context.Context, id, userID uuid.UUID) (bool, error) {
	review, ok := f.reviews[id]
	if !ok || review.UserID != userID {
		return false, nil
	}
	delete(f.reviews, id)
	return true, nil
}

func (f *fakeRepo) Report(context.Context, uuid.UUID, uuid.UUID, string) error {
	f.reports++
	return nil
}

const goodText = "The spa session was relaxing and the staff were friendly."

func fixture(status string) (*fakeRepo, *Service, uuid.UUID) {
	buyer := uuid.New()
	repo := &fakeRepo{owner: VoucherOwner{UserID: buyer, Status: status, MerchantID: uuid.New(), DealID: uuid.New()}, reviews: map[uuid.UUID]Review{}}
	return repo, NewService(repo), buyer
}

func TestOnlyRedeemedVoucherCanBeReviewedOnce(t *testing.T) {
	voucherID := uuid.New()
	repo, service, buyer := fixture("active")
	if _, err := service.Create(context.Background(), buyer, voucherID, Input{Rating: 5, Text: goodText, Locale: "en"}); !errors.Is(err, ErrNotRedeemed) {
		t.Fatalf("active voucher: got %v", err)
	}

	repo.owner.Status = "redeemed"
	if _, err := service.Create(context.Background(), buyer, voucherID, Input{Rating: 5, Text: goodText, Locale: "en"}); err != nil {
		t.Fatal(err)
	}
	if _, err := service.Create(context.Background(), buyer, voucherID, Input{Rating: 4, Text: goodText, Locale: "en"}); !errors.Is(err, ErrAlreadyReviewed) {
		t.Fatalf("second review: got %v", err)
	}
	if _, err := service.Create(context.Background(), uuid.New(), voucherID, Input{Rating: 5, Text: goodText, Locale: "en"}); !errors.Is(err, ErrNotRedeemed) {
		t.Fatalf("other user: got %v", err)
	}
}

func TestReviewContentRules(t *testing.T) {
	_, service, buyer := fixture("redeemed")
	voucherID := uuid.New()
	cases := map[string]Input{
		"rating": {Rating: 6, Text: goodText, Locale: "en"},
		"text":   {Rating: 5, Text: "too short", Locale: "en"},
		"locale": {Rating: 5, Text: goodText, Locale: "de"},
	}
	for field, in := range cases {
		_, err := service.Create(context.Background(), buyer, voucherID, in)
		var invalid *InvalidInputError
		if !errors.As(err, &invalid) || invalid.Field != field {
			t.Fatalf("%s: got %v", field, err)
		}
	}
}

func TestUpdateAndDeleteOnlyOwnReview(t *testing.T) {
	repo, service, buyer := fixture("redeemed")
	review, err := service.Create(context.Background(), buyer, uuid.New(), Input{Rating: 3, Text: goodText, Locale: "ru"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.Update(context.Background(), uuid.New(), review.ID, Input{Rating: 1, Text: goodText, Locale: "ru"}); !errors.Is(err, ErrNotFound) {
		t.Fatalf("update someone else's review: got %v", err)
	}
	updated, err := service.Update(context.Background(), buyer, review.ID, Input{Rating: 4, Text: goodText, Locale: "ru"})
	if err != nil || updated.Rating != 4 {
		t.Fatalf("own update: %v rating=%d", err, updated.Rating)
	}
	if err := service.Delete(context.Background(), buyer, review.ID); err != nil {
		t.Fatal(err)
	}
	if len(repo.reviews) != 0 {
		t.Fatal("review not removed")
	}
}

func TestReportRules(t *testing.T) {
	repo, service, buyer := fixture("redeemed")
	review, err := service.Create(context.Background(), buyer, uuid.New(), Input{Rating: 2, Text: goodText, Locale: "uz"})
	if err != nil {
		t.Fatal(err)
	}
	if err := service.Report(context.Background(), buyer, review.ID, "This is my own review"); !errors.Is(err, ErrOwnReport) {
		t.Fatalf("own report: got %v", err)
	}
	if err := service.Report(context.Background(), uuid.New(), review.ID, "short"); err == nil {
		t.Fatal("short reason accepted")
	}
	if err := service.Report(context.Background(), uuid.New(), review.ID, strings.Repeat("abuse ", 3)); err != nil {
		t.Fatal(err)
	}
	if repo.reports != 1 {
		t.Fatalf("reports=%d", repo.reports)
	}
}
