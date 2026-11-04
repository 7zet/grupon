package merchant

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
)

type fakeWorkspace struct {
	WorkspaceRepository
	role string
}

func (f fakeWorkspace) RoleFor(context.Context, uuid.UUID) (uuid.UUID, string, error) {
	return uuid.New(), f.role, nil
}

func TestCashierCanReadButNotEdit(t *testing.T) {
	t.Parallel()
	service := NewWorkspaceService(fakeWorkspace{role: RoleCashier}, time.Now)
	if _, _, err := service.Authorize(context.Background(), uuid.New(), false); err != nil {
		t.Fatalf("cashier read refused: %v", err)
	}
	if _, _, err := service.Authorize(context.Background(), uuid.New(), true); !errors.Is(err, ErrForbidden) {
		t.Fatalf("cashier edit err = %v, want ErrForbidden", err)
	}
}

func TestChangePercentIsZeroWithoutBaseline(t *testing.T) {
	t.Parallel()
	if got := changePercent(500, 0); got != 0 {
		t.Fatalf("changePercent(500,0) = %v, want 0", got)
	}
	if got := changePercent(150, 100); got != 50 {
		t.Fatalf("changePercent(150,100) = %v, want 50", got)
	}
}

func TestLocationHoursRejectUnknownDays(t *testing.T) {
	t.Parallel()
	err := validateLocation(LocationInput{
		CityID: uuid.New(), Address: "Street 1", Lat: 41, Lng: 69, Phone: "+998901234567",
		OpeningHours: map[string]any{"funday": "09:00-18:00"},
	})
	var invalid *InvalidInputError
	if !errors.As(err, &invalid) || invalid.Code != "unknown_day" {
		t.Fatalf("err = %v, want unknown_day", err)
	}
}
