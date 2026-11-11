package httpapi

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/dealo/dealo/services/api/internal/domain/redemption"
)

func TestAT05ExpiredAndRefundedVoucherErrorsMapToUnprocessableProblems(t *testing.T) {
	t.Parallel()

	api := &MerchantRedemptionAPI{logger: slog.New(slog.NewTextHandler(io.Discard, nil))}
	tests := []struct {
		name string
		err  error
		code string
	}{
		{name: "expired", err: redemption.ErrExpired, code: "voucher_expired"},
		{name: "refunded", err: redemption.ErrRefunded, code: "voucher_refunded"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			request := httptest.NewRequest(http.MethodPost, "/api/v1/merchant/vouchers/test/redeem", nil)

			api.fail(recorder, request, test.err)

			if recorder.Code != http.StatusUnprocessableEntity || problemCode(t, recorder) != test.code {
				t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
			}
			if got := recorder.Header().Get("Content-Type"); got != "application/problem+json" {
				t.Fatalf("content type = %q, want application/problem+json", got)
			}
		})
	}
}
