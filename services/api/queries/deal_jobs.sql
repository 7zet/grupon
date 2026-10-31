-- Background sweeps for deal lifecycle and denormalized counters. Each statement is idempotent: a
-- second run changes no rows.

-- name: ActivateScheduledDeals :execrows
UPDATE deals
SET status = 'active', updated_at = now()
WHERE status = 'scheduled' AND deleted_at IS NULL AND starts_at <= now() AND ends_at > now();

-- name: ExpireEndedDeals :execrows
UPDATE deals
SET status = 'expired', updated_at = now()
WHERE status IN ('scheduled', 'active', 'sold_out') AND deleted_at IS NULL AND ends_at <= now();

-- name: MarkSoldOutDeals :execrows
UPDATE deals d
SET status = 'sold_out', updated_at = now()
WHERE d.status = 'active' AND d.deleted_at IS NULL
  AND EXISTS (SELECT 1 FROM deal_options o WHERE o.deal_id = d.id)
  AND NOT EXISTS (
    SELECT 1 FROM deal_options o
    WHERE o.deal_id = d.id AND o.is_active AND o.stock_total - o.stock_sold - o.stock_reserved > 0
  );

-- name: ReactivateRestockedDeals :execrows
UPDATE deals d
SET status = 'active', updated_at = now()
WHERE d.status = 'sold_out' AND d.deleted_at IS NULL AND d.ends_at > now()
  AND EXISTS (
    SELECT 1 FROM deal_options o
    WHERE o.deal_id = d.id AND o.is_active AND o.stock_total - o.stock_sold - o.stock_reserved > 0
  );

-- name: RecomputeDealSoldCounts :execrows
UPDATE deals d
SET sold_count = s.total, updated_at = now()
FROM (
  SELECT deal_id, SUM(stock_sold)::int AS total FROM deal_options GROUP BY deal_id
) s
WHERE s.deal_id = d.id AND d.sold_count <> s.total;

-- name: RecomputeDealRatings :execrows
UPDATE deals d
SET rating_avg = r.avg, rating_count = r.cnt, updated_at = now()
FROM (
  SELECT deal_id, ROUND(AVG(rating), 2)::numeric(3,2) AS avg, COUNT(*)::int AS cnt
  FROM reviews
  WHERE status = 'published' AND deleted_at IS NULL
  GROUP BY deal_id
) r
WHERE r.deal_id = d.id AND (d.rating_count <> r.cnt OR d.rating_avg <> r.avg);

-- name: RecomputeMerchantRatings :execrows
UPDATE merchants m
SET rating_avg = r.avg, rating_count = r.cnt, updated_at = now()
FROM (
  SELECT merchant_id, ROUND(AVG(rating), 2)::numeric(3,2) AS avg, COUNT(*)::int AS cnt
  FROM reviews
  WHERE status = 'published' AND deleted_at IS NULL
  GROUP BY merchant_id
) r
WHERE r.merchant_id = m.id AND (m.rating_count <> r.cnt OR m.rating_avg <> r.avg);
