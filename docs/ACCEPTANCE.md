# Acceptance tests (Dealo PRD 16) — tracker

> Stage 1 finish line: every AT below is either green (a test proves it) or marked "PSP kerak" / "SMS kerak" / "3-bosqich".
> Sources: `dealo/` PRD moved out of the repo to `../dealo-docs/docs/prd/16-acceptance-tests.md`.
> Status: ✅ test exists and passes · 🟡 code exists, no AT-level test · ❔ not verified · ⏳ waiting for an external dependency

| AT | Scenario | Status | Evidence / what is missing |
|---|---|---|---|
| AT-01 | Webhook duplicate | ❔ | Mock payment only; webhook idempotency not tested. PSP kerak for real webhook |
| AT-02 | Parallel redeem | 🟡 | Atomic redeem in one transaction; no 100-parallel test |
| AT-03 | Inventory oversell | ✅ | `store/commerce_concurrency_test.go` (TestOverSellingIsImpossible) |
| AT-04 | Foreign merchant scans voucher | ❔ | Scope check in redemption; no test |
| AT-05 | Expired / refunded voucher scan | 🟡 | Domain tests in `domain/redemption`; no AT-level check of "no ledger row" |
| AT-06 | Price version lock | ⏳ | Plan: snapshot the terms in the order and test it (no deal versioning). Test first |
| AT-07 | Failed payment, hold expires | ❔ | Hold expiry job exists; no test of inventory return + `expired` order |
| AT-08 | Reconciliation alert | ⏳ | PSP kerak (3-bosqich) |
| AT-09 | Cashier cannot approve payout | 🟡 | Moderator/finance role tests exist; cashier case not tested |
| AT-10 | Closed category gate | 🟡 | `category_closed` check exists; no test |
| AT-11 | Refund within 14 days | ✅ | `store/finance_test.go` (TestUnusedVoucherRefundsAutomaticallyWithinFourteenDays) |
| AT-12 | Voucher expiry at Tashkent 23:59 | ✅ | `domain/commerce/voucher_window_test.go` |
| AT-13 | Late payment auto-refund | 🟡 | Mock late payment is refunded at provider; no `refunds` row yet (Stage 1) |
| AT-14 | Collusion fraud signal | ❔ | `fraud_signals` table; rule not verified against the 8-scans case |
| AT-15 | Multi-use, parallel last use | 🟡 | `partially_redeemed`/`redeemed` transition in SQL; no parallel test |
| AT-16 | Dispute holds payout | 🟡 | Dispute holds payout in code; no AT test |
| AT-17 | Bank account change hold (72 h) | ⏳ | 3-bosqich (not started) |
| AT-18 | 4-eyes on payouts | ✅ | `store/finance_test.go` (TestPayoutNeedsADifferentApprover) |

Counts: ✅ 4 · 🟡 7 · ❔ 4 · ⏳ 3. Stage 1 turns every 🟡 and ❔ into ✅ or into a documented ⏳.
