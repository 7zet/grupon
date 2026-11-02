package commerce

import (
	"testing"
	"time"
)

func TestVoucherExpiresAtTheEndOfTheTashkentDayOrAtTheFixedDate(t *testing.T) {
	t.Parallel()
	days := 30
	fixedEnd := time.Date(2026, 12, 1, 0, 0, 0, 0, time.UTC)
	record := OptionRecord{VoucherValidDays: &days, VoucherValidUntil: &fixedEnd}

	// 23:30 in Tashkent on 2 November; 30 days later is 2 December, but the fixed date (1 December
	// 00:00 UTC, i.e. 05:00 Tashkent) is earlier, so it wins.
	purchased := time.Date(2026, 11, 2, 23, 30, 0, 0, TashkentZone)
	_, until, ok := record.VoucherWindow(purchased)
	if !ok || !until.Equal(fixedEnd) {
		t.Fatalf("fixed date should win: until=%v ok=%v", until, ok)
	}

	// Without a fixed date, the day-based end is the last second of that Tashkent day.
	dayOnly := OptionRecord{VoucherValidDays: &days}
	_, until, ok = dayOnly.VoucherWindow(purchased)
	want := time.Date(2026, 12, 2, 23, 59, 59, 999999999, TashkentZone)
	if !ok || !until.Equal(want) {
		t.Fatalf("day-based end: until=%v want %v", until, want)
	}
	if until.In(TashkentZone).Hour() != 23 || until.In(TashkentZone).Minute() != 59 {
		t.Fatalf("expiry is not 23:59 Tashkent time: %v", until.In(TashkentZone))
	}
}
