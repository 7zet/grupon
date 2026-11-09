package httpapi

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"

	"github.com/grupon/grupon/services/api/internal/domain/reviews"
	"github.com/grupon/grupon/services/api/internal/http/gen"
)

// ReviewAPI serves buyer reviews and review reports. Every write is limited to the signed-in user.
type ReviewAPI struct {
	reviews *reviews.Service
	auth    *AuthAPI
	logger  *slog.Logger
}

// NewReviewAPI binds the review rules to the HTTP contract.
func NewReviewAPI(service *reviews.Service, auth *AuthAPI, logger *slog.Logger) *ReviewAPI {
	return &ReviewAPI{reviews: service, auth: auth, logger: logger}
}

// ListMyReviews returns the signed-in user's reviews.
func (a *ReviewAPI) ListMyReviews(w http.ResponseWriter, r *http.Request) {
	account, ok := a.auth.authenticate(w, r)
	if !ok {
		return
	}
	list, err := a.reviews.ListOwn(r.Context(), account.ID)
	if err != nil {
		a.fail(w, r, err)
		return
	}
	items := make([]gen.Review, 0, len(list))
	for _, review := range list {
		items = append(items, mapReview(review))
	}
	writeJSON(w, http.StatusOK, gen.ReviewList{Items: items})
}

func (a *ReviewAPI) CreateReview(w http.ResponseWriter, r *http.Request) {
	account, ok := a.auth.authenticate(w, r)
	if !ok {
		return
	}
	var body gen.CreateReviewRequest
	if _, ok := decodeBody(w, r, &body); !ok {
		return
	}
	review, err := a.reviews.Create(r.Context(), account.ID, uuid.UUID(body.VoucherId), reviews.Input{
		Rating: body.Rating,
		Text:   body.Text,
		Locale: string(body.Locale),
	})
	if err != nil {
		a.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, mapReview(review))
}

func (a *ReviewAPI) UpdateReview(w http.ResponseWriter, r *http.Request, id gen.Id) {
	account, ok := a.auth.authenticate(w, r)
	if !ok {
		return
	}
	var body gen.ReviewInput
	if _, ok := decodeBody(w, r, &body); !ok {
		return
	}
	review, err := a.reviews.Update(r.Context(), account.ID, uuid.UUID(id), reviews.Input{
		Rating: body.Rating,
		Text:   body.Text,
		Locale: string(body.Locale),
	})
	if err != nil {
		a.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, mapReview(review))
}

func (a *ReviewAPI) DeleteReview(w http.ResponseWriter, r *http.Request, id gen.Id) {
	account, ok := a.auth.authenticate(w, r)
	if !ok {
		return
	}
	if err := a.reviews.Delete(r.Context(), account.ID, uuid.UUID(id)); err != nil {
		a.fail(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (a *ReviewAPI) ReportReview(w http.ResponseWriter, r *http.Request, id gen.Id) {
	account, ok := a.auth.authenticate(w, r)
	if !ok {
		return
	}
	var body gen.ReportRequest
	if _, ok := decodeBody(w, r, &body); !ok {
		return
	}
	if err := a.reviews.Report(r.Context(), account.ID, uuid.UUID(id), body.Reason); err != nil {
		a.fail(w, r, err)
		return
	}
	w.WriteHeader(http.StatusAccepted)
}

func (a *ReviewAPI) fail(w http.ResponseWriter, r *http.Request, err error) {
	var invalid *reviews.InvalidInputError
	switch {
	case errors.As(err, &invalid):
		writeProblem(w, r, http.StatusUnprocessableEntity, invalid.Code, nil,
			map[string][]string{invalid.Field: {invalid.Code}})
	case errors.Is(err, reviews.ErrVoucherNotFound), errors.Is(err, reviews.ErrNotFound):
		writeProblem(w, r, http.StatusNotFound, "not_found", nil, nil)
	case errors.Is(err, reviews.ErrNotRedeemed):
		writeProblem(w, r, http.StatusForbidden, "voucher_not_redeemed", nil, nil)
	case errors.Is(err, reviews.ErrAlreadyReviewed):
		writeProblem(w, r, http.StatusConflict, "review_exists", nil, nil)
	case errors.Is(err, reviews.ErrOwnReport):
		writeProblem(w, r, http.StatusUnprocessableEntity, "own_review", nil, nil)
	default:
		a.logger.ErrorContext(r.Context(), "review request failed",
			slog.String("request_id", middleware.GetReqID(r.Context())),
			slog.String("error", err.Error()),
		)
		writeProblem(w, r, http.StatusInternalServerError, "internal_error", nil, nil)
	}
}

func mapReview(review reviews.Review) gen.Review {
	return gen.Review{
		Id:         gen.Uuid(review.ID),
		DealId:     gen.Uuid(review.DealID),
		MerchantId: gen.Uuid(review.MerchantID),
		UserId:     gen.Uuid(review.UserID),
		VoucherId:  gen.Uuid(review.VoucherID),
		Rating:     review.Rating,
		Text:       review.Text,
		Locale:     gen.Locale(review.Locale),
		Status:     gen.ReviewStatus(review.Status),
		CreatedAt:  review.CreatedAt,
	}
}

// Delegation from the router.

func (s apiServer) ListMyReviews(w http.ResponseWriter, r *http.Request) {
	if s.reviewAPI != nil {
		s.reviewAPI.ListMyReviews(w, r)
		return
	}
	s.ServerInterface.ListMyReviews(w, r)
}

func (s apiServer) CreateReview(w http.ResponseWriter, r *http.Request) {
	if s.reviewAPI != nil {
		s.reviewAPI.CreateReview(w, r)
		return
	}
	s.ServerInterface.CreateReview(w, r)
}

func (s apiServer) UpdateReview(w http.ResponseWriter, r *http.Request, id gen.Id) {
	if s.reviewAPI != nil {
		s.reviewAPI.UpdateReview(w, r, id)
		return
	}
	s.ServerInterface.UpdateReview(w, r, id)
}

func (s apiServer) DeleteReview(w http.ResponseWriter, r *http.Request, id gen.Id) {
	if s.reviewAPI != nil {
		s.reviewAPI.DeleteReview(w, r, id)
		return
	}
	s.ServerInterface.DeleteReview(w, r, id)
}

func (s apiServer) ReportReview(w http.ResponseWriter, r *http.Request, id gen.Id) {
	if s.reviewAPI != nil {
		s.reviewAPI.ReportReview(w, r, id)
		return
	}
	s.ServerInterface.ReportReview(w, r, id)
}
