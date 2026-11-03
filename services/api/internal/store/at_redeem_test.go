package store_test

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/grupon/grupon/services/api/internal/domain/redemption"
	"github.com/grupon/grupon/services/api/internal/store"
)

// redeemRace makes `attempts` concurrent scans of one voucher and returns how many succeeded.
func redeemRace(t *testing.T, env atEnv, voucherID uuid.UUID, attempts int) (succeeded int64) {
	t.Helper()
	merchantID, locationID := env.merchantAndLocation(t)
	repo := store.NewRedemptionRepository(env.fixture.pool, store.NewCommerceRepository(env.fixture.pool), store.NewCatalogRepository(env.fixture.pool))

	var (
		wg    sync.WaitGroup
		start = make(chan struct{})
		okCnt atomic.Int64
	)
	for i := 0; i < attempts; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			audit := redemption.AuditEntry{ActorID: env.buyer, Role: "merchant_cashier", VoucherID: voucherID, Action: "voucher_redeemed"}
			ok, err := repo.UseVoucher(context.Background(), merchantID, voucherID, locationID, time.Now(), audit, nil)
			if err == nil && ok {
				okCnt.Add(1)
			}
		}()
	}
	close(start)
	wg.Wait()
	return okCnt.Load()
}

// AT-02: 100 staff scan the same single-use voucher at the same moment. Exactly one redemption is recorded.
func TestAT02ParallelScansRedeemASingleUseVoucherOnce(t *testing.T) {
	env := newATEnv(t, 1)
	voucherID := env.sellOneVoucher(t, env.newBuyer(t))

	if got := redeemRace(t, env, voucherID, 100); got != 1 {
		t.Fatalf("successful scans = %d, want 1", got)
	}
	if got := env.count(t, `SELECT count(*) FROM redemptions WHERE voucher_id = $1 AND reversed_at IS NULL`, voucherID); got != 1 {
		t.Fatalf("redemption rows = %d, want 1", got)
	}
	if got := env.scalarText(t, `SELECT status::text FROM vouchers WHERE id = $1`, voucherID); got != "redeemed" {
		t.Fatalf("voucher status = %s, want redeemed", got)
	}
}

// AT-15: a ten-visit voucher has nine visits used. Two scans race for the last visit: one succeeds, one is refused.
func TestAT15MultiUseLastVisitIsTakenOnce(t *testing.T) {
	env := newATEnv(t, 1)
	if _, err := env.fixture.pool.Exec(env.ctx, `UPDATE deal_options SET redemptions_allowed = 10 WHERE id = $1`, env.optionID); err != nil {
		t.Fatal(err)
	}
	voucherID := env.sellOneVoucher(t, env.newBuyer(t))
	if _, err := env.fixture.pool.Exec(env.ctx, `
		UPDATE vouchers SET redemptions_used = 9, status = 'partially_redeemed' WHERE id = $1`, voucherID); err != nil {
		t.Fatal(err)
	}

	if got := redeemRace(t, env, voucherID, 2); got != 1 {
		t.Fatalf("successful scans = %d, want 1", got)
	}
	if got := env.count(t, `SELECT redemptions_used FROM vouchers WHERE id = $1`, voucherID); got != 10 {
		t.Fatalf("redemptions_used = %d, want 10", got)
	}
	if got := env.scalarText(t, `SELECT status::text FROM vouchers WHERE id = $1`, voucherID); got != "redeemed" {
		t.Fatalf("voucher status = %s, want redeemed", got)
	}
}

// AT-15 first part: a ten-visit voucher takes its second visit. A partially used voucher must still be usable.
func TestMultiUseVoucherAcceptsItsSecondVisit(t *testing.T) {
	env := newATEnv(t, 1)
	if _, err := env.fixture.pool.Exec(env.ctx, `UPDATE deal_options SET redemptions_allowed = 10 WHERE id = $1`, env.optionID); err != nil {
		t.Fatal(err)
	}
	voucherID := env.sellOneVoucher(t, env.newBuyer(t))

	if got := redeemRace(t, env, voucherID, 1); got != 1 {
		t.Fatalf("first visit: successful scans = %d, want 1", got)
	}
	if got := redeemRace(t, env, voucherID, 1); got != 1 {
		t.Fatalf("second visit: successful scans = %d, want 1", got)
	}
	if got := env.count(t, `SELECT redemptions_used FROM vouchers WHERE id = $1`, voucherID); got != 2 {
		t.Fatalf("redemptions_used = %d, want 2", got)
	}
}
