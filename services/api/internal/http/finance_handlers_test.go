package httpapi

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/dealo/dealo/services/api/internal/domain/finance"
)

func TestDisputeErrorsMapToStableHTTPProblems(t *testing.T) {
	t.Parallel()
	api := &FinanceAPI{logger: slog.New(slog.NewTextHandler(io.Discard, nil))}
	tests := []struct {
		name   string
		err    error
		status int
		code   string
	}{
		{name: "forbidden transition", err: finance.ErrInvalidTransition, status: http.StatusUnprocessableEntity, code: "invalid_dispute_transition"},
		{name: "wrong role", err: finance.ErrForbidden, status: http.StatusForbidden, code: "forbidden"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			request := httptest.NewRequest(http.MethodPost, "/api/v1/admin/disputes/test", nil)
			api.fail(recorder, request, test.err)
			if recorder.Code != test.status || problemCode(t, recorder) != test.code {
				t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
			}
		})
	}
}
