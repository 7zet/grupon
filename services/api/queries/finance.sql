-- name: PayableRedemptions :many
-- Redemptions ready for payout: not reversed, not already paid, older than the dispute window, with no
-- open dispute (BR-15), for one merchant.
SELECT r.id, r.amount, r.commission
FROM redemptions r
WHERE r.merchant_id = sqlc.arg(merchant_id)
  AND r.reversed_at IS NULL
  AND r.payout_run_id IS NULL
  AND r.created_at <= sqlc.arg(eligible_before)
  AND NOT EXISTS (SELECT 1 FROM disputes d WHERE d.redemption_id = r.id AND d.status IN ('open', 'awaiting_evidence', 'under_review'))
ORDER BY r.created_at, r.id;

-- name: MerchantOnHold :one
SELECT EXISTS (
  SELECT 1 FROM merchant_payout_holds h
  WHERE h.merchant_id = sqlc.arg(merchant_id) AND (h.hold_until IS NULL OR h.hold_until > now())
);

-- name: InsertPayoutRun :one
INSERT INTO payout_runs (merchant_id, amount, status, prepared_by)
VALUES (sqlc.arg(merchant_id), sqlc.arg(amount), 'pending_approval', sqlc.arg(prepared_by))
RETURNING id;

-- name: AttachRedemptionToRun :execrows
UPDATE redemptions SET payout_run_id = sqlc.arg(run_id)
WHERE id = sqlc.arg(id) AND payout_run_id IS NULL;

-- name: LockPayoutRun :one
SELECT id, merchant_id, amount, status, prepared_by FROM payout_runs WHERE id = sqlc.arg(id) FOR UPDATE;

-- name: ApprovePayoutRun :execrows
UPDATE payout_runs SET status = 'sent', approved_by = sqlc.arg(approved_by), bank_doc_ref = sqlc.arg(bank_doc_ref), updated_at = now()
WHERE id = sqlc.arg(id) AND status = 'pending_approval';

-- name: InsertRefund :one
INSERT INTO refunds (voucher_id, amount, reason, status, requested_by, approved_by)
VALUES (sqlc.arg(voucher_id), sqlc.arg(amount), sqlc.arg(reason), sqlc.arg(status), sqlc.arg(requested_by), sqlc.narg(approved_by))
RETURNING id;

-- name: VoucherForRefund :one
SELECT v.id, v.user_id, v.merchant_id, v.status, v.redemptions_used, v.created_at, oi.unit_price_snapshot, o.paid_at
FROM vouchers v
JOIN order_items oi ON oi.id = v.order_item_id
JOIN orders o ON o.id = oi.order_id
WHERE v.id = sqlc.arg(id)
FOR UPDATE OF v;

-- name: MarkVoucherRefunded :execrows
UPDATE vouchers SET status = 'refunded', updated_at = now() WHERE id = sqlc.arg(id) AND status = 'active' AND redemptions_used = 0;

-- name: SetRefundStatus :execrows
UPDATE refunds SET status = sqlc.arg(status), updated_at = now() WHERE id = sqlc.arg(id);

-- name: InsertDispute :one
INSERT INTO disputes (redemption_id, opened_by, reason)
VALUES (sqlc.arg(redemption_id), sqlc.arg(opened_by), sqlc.arg(reason))
ON CONFLICT (redemption_id) DO NOTHING
RETURNING id;

-- name: LatestRedemptionForVoucher :one
SELECT id, merchant_id, amount, commission, created_at FROM redemptions
WHERE voucher_id = sqlc.arg(voucher_id) AND reversed_at IS NULL
ORDER BY sequence DESC LIMIT 1;

-- name: ExpiredVouchersForBreakage :many
-- Vouchers whose window closed more than 90 days ago and still hold unused value (PRD §8.5).
SELECT v.id, v.merchant_id, v.redemptions_used, v.redemptions_allowed, oi.unit_price_snapshot
FROM vouchers v
JOIN order_items oi ON oi.id = v.order_item_id
WHERE v.status = 'active' AND v.valid_until < sqlc.arg(cutoff) AND v.redemptions_used < v.redemptions_allowed
ORDER BY v.valid_until
LIMIT sqlc.arg(batch_size);

-- name: ExpireVoucherRow :execrows
UPDATE vouchers SET status = 'expired', updated_at = now() WHERE id = sqlc.arg(id) AND status = 'active';

-- name: RecentStaffRedemptions :one
SELECT count(*) FROM redemptions WHERE staff_user_id = sqlc.arg(staff_user_id) AND created_at > sqlc.arg(since);

-- name: InsertFraudSignal :exec
INSERT INTO fraud_signals (merchant_id, subject_user_id, rule, detail)
VALUES (sqlc.arg(merchant_id), sqlc.arg(subject_user_id), sqlc.arg(rule), sqlc.arg(detail));

-- name: PlaceMerchantHold :exec
INSERT INTO merchant_payout_holds (merchant_id, reason, hold_until)
VALUES (sqlc.arg(merchant_id), sqlc.arg(reason), sqlc.narg(hold_until))
ON CONFLICT (merchant_id) DO UPDATE SET reason = EXCLUDED.reason, hold_until = EXCLUDED.hold_until;

-- name: CustomerPrepaidTotal :one
SELECT COALESCE(SUM(CASE WHEN jl.direction = 'credit' THEN jl.amount ELSE -jl.amount END), 0)::bigint AS balance
FROM journal_lines jl JOIN ledger_accounts la ON la.id = jl.account_id
WHERE la.code LIKE 'customer_prepaid:%';

-- name: ActiveVoucherValue :one
-- Unused value still owed to buyers: price minus what the redeemed shares already consumed.
SELECT COALESCE(SUM(oi.unit_price_snapshot - (oi.unit_price_snapshot * v.redemptions_used / v.redemptions_allowed)), 0)::bigint AS value
FROM vouchers v JOIN order_items oi ON oi.id = v.order_item_id
WHERE v.status = 'active';

-- name: InsertReconciliationFinding :exec
INSERT INTO reconciliation_findings (check_name, expected, actual)
VALUES (sqlc.arg(check_name), sqlc.arg(expected), sqlc.arg(actual));

-- name: InsertOutbox :exec
INSERT INTO outbox (topic, payload) VALUES (sqlc.arg(topic), sqlc.arg(payload));

-- name: ActiveVouchersForValue :many
SELECT v.id, v.merchant_id, v.redemptions_used, v.redemptions_allowed, oi.unit_price_snapshot
FROM vouchers v JOIN order_items oi ON oi.id = v.order_item_id
WHERE v.status = 'active';

-- name: PayoutRunDetail :one
SELECT pr.id, pr.merchant_id, pr.amount, pr.status, pr.prepared_by FROM payout_runs pr WHERE pr.id = sqlc.arg(id);

-- name: RefundDetail :one
SELECT r.id, r.voucher_id, r.amount, r.status, r.requested_by FROM refunds r WHERE r.id = sqlc.arg(id) FOR UPDATE;

-- name: MarkRefundSucceeded :execrows
UPDATE refunds SET status = 'succeeded', approved_by = COALESCE(approved_by, sqlc.arg(approver_id)), updated_at = now()
WHERE id = sqlc.arg(id) AND status = 'approved';

-- name: ClaimOutboxBatch :many
SELECT id, topic, payload FROM outbox WHERE published_at IS NULL ORDER BY created_at, id LIMIT sqlc.arg(batch_size) FOR UPDATE SKIP LOCKED;

-- name: MarkOutboxPublished :exec
UPDATE outbox SET published_at = now() WHERE id = sqlc.arg(id);

-- name: MarkPayoutRunReconciled :execrows
UPDATE payout_runs SET status = 'reconciled', updated_at = now() WHERE id = sqlc.arg(id) AND status = 'sent';

-- name: ApproveRequestedRefund :one
UPDATE refunds r SET status = 'approved', approved_by = sqlc.arg(approver_id), updated_at = now()
FROM vouchers v
WHERE r.id = sqlc.arg(id) AND r.status = 'proposed' AND r.proposed_by <> sqlc.arg(approver_id) AND v.id = r.voucher_id
RETURNING r.id, r.amount, r.voucher_id, v.merchant_id;

-- name: ProposeRequestedRefund :execrows
UPDATE refunds SET status = 'proposed', proposed_by = sqlc.arg(proposed_by), updated_at = now()
WHERE id = sqlc.arg(id) AND status = 'requested';

-- name: DisputeForResolution :one
SELECT d.id, d.redemption_id, d.status, r.voucher_id, r.merchant_id, r.amount, r.commission, r.payout_run_id
FROM disputes d JOIN redemptions r ON r.id = d.redemption_id
WHERE d.id = sqlc.arg(id) FOR UPDATE OF d;

-- name: TransitionDispute :execrows
UPDATE disputes
SET status = sqlc.arg(to_status),
    resolved_at = CASE WHEN sqlc.arg(to_status)::text IN ('resolved_customer', 'resolved_merchant') THEN now() ELSE resolved_at END
WHERE id = sqlc.arg(id) AND status = sqlc.arg(from_status);

-- name: ListOpenPayoutRuns :many
SELECT pr.id, m.display_name AS merchant_name, pr.amount, pr.status, pr.created_at
FROM payout_runs pr JOIN merchants m ON m.id = pr.merchant_id
WHERE pr.status IN ('pending_approval', 'approved', 'sent')
ORDER BY pr.created_at;

-- name: ListOpenRefunds :many
SELECT r.id, v.code AS voucher_code, r.amount, r.status, r.reason, r.created_at
FROM refunds r JOIN vouchers v ON v.id = r.voucher_id
WHERE r.status IN ('requested', 'proposed', 'approved', 'processing')
ORDER BY r.created_at;

-- name: ListOpenDisputes :many
SELECT d.id, v.code AS voucher_code, m.display_name AS merchant_name, rd.amount, d.status, d.reason, d.created_at
FROM disputes d
JOIN redemptions rd ON rd.id = d.redemption_id
JOIN vouchers v ON v.id = rd.voucher_id
JOIN merchants m ON m.id = rd.merchant_id
WHERE d.status <> 'closed'
ORDER BY d.created_at;

-- name: SetVoucherDisputeFlag :exec
-- PRD 08: an open dispute sets the voucher flag, which blocks redemption until the dispute closes.
UPDATE vouchers SET dispute_open = sqlc.arg(open)::boolean, updated_at = now() WHERE id = sqlc.arg(id);

-- name: InsertLateRefund :exec
-- AT-13: the system refunds a payment that arrived after its order expired. The provider refund already happened.
INSERT INTO refunds (payment_id, amount, reason, status)
VALUES (sqlc.arg(payment_id), sqlc.arg(amount), sqlc.arg(reason), 'succeeded');
