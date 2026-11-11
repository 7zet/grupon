# Acceptance tests (Dealo PRD 16) — tracker

> Each AT is green, or marked ⏳ with the reason it cannot pass in the demo.
> Sources: `dealo/` PRD moved out of the repo to `../dealo-docs/docs/prd/16-acceptance-tests.md`.
> Status: ✅ test exists and passes · 🟡 code exists, no AT-level test · ❔ not verified · ⏳ waiting for an external dependency

| AT | Scenario | Status | Evidence / what is missing |
|---|---|---|---|
| AT-01 | Webhook duplicate | ✅ | `store/at_payments_test.go` (TestAT01…): 5 deliveries, 1 voucher, order paid, exactly 1 journal entry. |
| AT-02 | Parallel redeem | ✅ | `store/at_redeem_test.go` (TestAT02…): 100 concurrent scans, one redemption |
| AT-03 | Inventory oversell | ✅ | `store/commerce_concurrency_test.go` (TestOverSellingIsImpossible) |
| AT-04 | Foreign merchant scans voucher | ✅ | `domain/redemption` TestForeignMerchantScanIsRefusedAndAudited: refused, voucher unchanged, `voucher_scan_denied` audit row |
| AT-05 | Expired / refunded voucher scan | ✅ | Domain test refuses it without changes; `http/merchant_redemption_handlers_test.go` (`TestAT05…`) proves both errors map to 422 with the exact reason |
| AT-06 | Price version lock | ✅ | `store/at_gates_test.go` (TestAT06…): order total, item price, order-item terms and voucher terms keep the buyer's agreed snapshots after the deal changes |
| AT-07 | Failed payment, hold expires | ✅ | `store/at_payments_test.go` (TestAT07…): stock returned, no voucher, order `expired` |
| AT-08 | Reconciliation alert | ⏳ | Excluded: needs a real PSP. The worker runs a ledger reconciliation check (`financeJobs.Reconcile`), but there is no provider statement to compare with. |
| AT-09 | Cashier cannot approve payout | ✅ | `store/at_gates_test.go` and `store/finance_test.go`: denied prepare and denied approve paths are audit-logged. |
| AT-10 | Closed category gate | ✅ | `store/at_gates_test.go`: create and submit return `category_closed`; separate tests prove search, publish, checkout, and redemption capability gates |
| AT-11 | Refund within 14 days | ✅ | `store/finance_test.go` (TestUnusedVoucherRefundsAutomaticallyWithinFourteenDays) |
| AT-12 | Voucher expiry at Tashkent 23:59 | ✅ | `domain/commerce/voucher_window_test.go` |
| AT-13 | Late payment auto-refund | ✅ | `store/at_payments_test.go` (TestAT13…): refund row on the payment (migration 00021), no voucher, order stays expired |
| AT-14 | Collusion fraud signal | ✅ | `store/at_fraud_test.go`: burst signal, payout hold, vouchers `fraud_hold`, held voucher refused (`voucher_on_hold`). Gap: the hold stops the whole merchant's payouts, not only the burst amount |
| AT-15 | Multi-use, parallel last use | ✅ | `store/at_redeem_test.go`: two racing scans, one succeeds; second visit also tested. Fixed a bug: partially used vouchers could not be used again |
| AT-16 | Dispute holds payout | ✅ | `store/at_gates_test.go` (TestAT16…): disputed redemption leaves the payable set. `store/finance_test.go` (`TestDisputeAllowedTransitionsAuditAndPayoutHold`): hold remains through evidence/review and is released on merchant resolution; all transitions are audited |
| AT-17 | Bank account change hold (72 h) | ⏳ | Not built: bank-account change protection is listed under "Not yet built" in `docs/HANDOVER.md`. |
| AT-18 | 4-eyes on payouts | ✅ | `store/finance_test.go` (TestPayoutNeedsADifferentApprover): self and wrong-role approvals are refused and audit-logged. |

### Re-check after the audit fixes (2026-11-07)

Re-checked against `docs/AUDIT.md`: AT-09 and AT-18 now cover the approve path and self-approval with audit rows (audit B-4, fixed in G1). AT-01 asserts exactly one journal entry. The DB-backed store tests run in CI (Postgres service) and locally with `TEST_DATABASE_URL`; they are not skipped there. Partials that stay partial: AT-14 (hold covers the whole merchant, not only the burst amount). New checks since the audit: QR refresh, one merchant and one variant per order, refund four-eyes, voucher sequence uniqueness, deal-branch check on redeem, reversal offset against the next payout, append-only `audit_logs`, Origin check, price proof on submit.

### v1.0 (2026-11-11)

All 16 ✅ tests still exist under the names above and pass in CI (`go test ./...` with Postgres).

Counts: ✅ 16 · 🟡 0 · ❔ 0 · ⏳ 2 (AT-08: needs a real PSP; AT-17: bank-change protection not built).
