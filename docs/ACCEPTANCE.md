# Acceptance tests (Dealo PRD 16) — tracker

> Stage 1 finish line: every AT below is either green (a test proves it) or marked "PSP kerak" / "SMS kerak" / "3-bosqich".
> Sources: `dealo/` PRD moved out of the repo to `../dealo-docs/docs/prd/16-acceptance-tests.md`.
> Status: ✅ test exists and passes · 🟡 code exists, no AT-level test · ❔ not verified · ⏳ waiting for an external dependency

| AT | Scenario | Status | Evidence / what is missing |
|---|---|---|---|
| AT-01 | Webhook duplicate | ✅ | `store/at_payments_test.go` (TestAT01…): 5 deliveries, 1 voucher, order paid. Ledger lines are not checked yet |
| AT-02 | Parallel redeem | ✅ | `store/at_redeem_test.go` (TestAT02…): 100 concurrent scans, one redemption |
| AT-03 | Inventory oversell | ✅ | `store/commerce_concurrency_test.go` (TestOverSellingIsImpossible) |
| AT-04 | Foreign merchant scans voucher | ✅ | `domain/redemption` TestForeignMerchantScanIsRefusedAndAudited: refused, voucher unchanged, `voucher_scan_denied` audit row |
| AT-05 | Expired / refunded voucher scan | ✅ | Domain test refuses it without changes; `http/merchant_redemption_handlers_test.go` (`TestAT05…`) proves both errors map to 422 with the exact reason |
| AT-06 | Price version lock | 🟡 | `store/at_gates_test.go` (TestAT06…): order total and item price keep the agreed price. Terms are not snapshotted yet: deal terms live on the deal row, and the order and voucher do not copy them |
| AT-07 | Failed payment, hold expires | ✅ | `store/at_payments_test.go` (TestAT07…): stock returned, no voucher, order `expired` |
| AT-08 | Reconciliation alert | ⏳ | PSP kerak (3-bosqich) |
| AT-09 | Cashier cannot approve payout | ✅ | `store/at_gates_test.go` (TestAT09…): ErrForbidden, `payout_prepare_denied` audit row |
| AT-10 | Closed category gate | ✅ | `store/at_gates_test.go` (TestAT10…): `category_closed` on `category_id`; open category passes |
| AT-11 | Refund within 14 days | ✅ | `store/finance_test.go` (TestUnusedVoucherRefundsAutomaticallyWithinFourteenDays) |
| AT-12 | Voucher expiry at Tashkent 23:59 | ✅ | `domain/commerce/voucher_window_test.go` |
| AT-13 | Late payment auto-refund | ✅ | `store/at_payments_test.go` (TestAT13…): refund row on the payment (migration 00021), no voucher, order stays expired |
| AT-14 | Collusion fraud signal | ✅ | `store/at_fraud_test.go`: burst signal, payout hold, vouchers `fraud_hold`, held voucher refused (`voucher_on_hold`). Gap: the hold stops the whole merchant's payouts, not only the burst amount |
| AT-15 | Multi-use, parallel last use | ✅ | `store/at_redeem_test.go`: two racing scans, one succeeds; second visit also tested. Fixed a bug: partially used vouchers could not be used again |
| AT-16 | Dispute holds payout | ✅ | `store/at_gates_test.go` (TestAT16…): disputed redemption leaves the payable set, the other stays |
| AT-17 | Bank account change hold (72 h) | ⏳ | 3-bosqich (not started) |
| AT-18 | 4-eyes on payouts | ✅ | `store/finance_test.go` (TestPayoutNeedsADifferentApprover) |

Counts: ✅ 15 · 🟡 1 · ❔ 0 · ⏳ 2 (AT-08 and AT-17 wait for PSP / 3-bosqich).
