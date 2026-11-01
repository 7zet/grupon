-- name: LedgerEntryExists :one
SELECT EXISTS (SELECT 1 FROM journal_entries WHERE idempotency_key = sqlc.arg(idempotency_key));

-- name: InsertJournalEntry :one
INSERT INTO journal_entries (kind, reference, idempotency_key)
VALUES (sqlc.arg(kind), sqlc.arg(reference), sqlc.arg(idempotency_key))
RETURNING id;

-- name: EnsureLedgerAccount :one
INSERT INTO ledger_accounts (code, kind) VALUES (sqlc.arg(code), sqlc.arg(kind))
ON CONFLICT (code) DO UPDATE SET code = EXCLUDED.code
RETURNING id;

-- name: InsertJournalLine :exec
INSERT INTO journal_lines (entry_id, account_id, direction, amount)
VALUES (sqlc.arg(entry_id), sqlc.arg(account_id), sqlc.arg(direction), sqlc.arg(amount));

-- name: LedgerBalance :one
SELECT COALESCE(SUM(CASE WHEN jl.direction = 'credit' THEN jl.amount ELSE -jl.amount END), 0)::bigint AS balance
FROM journal_lines jl
JOIN ledger_accounts la ON la.id = jl.account_id
WHERE la.code = sqlc.arg(code);
