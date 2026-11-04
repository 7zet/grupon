
/** Sort values allowed in the listing URL; they are exactly the OpenAPI `sort` enum. */
export const DEAL_SORTS = [
  'recommended',
  'popular',
  'newest',
  'price_asc',
  'price_desc',
  'biggest_discount',
  'ending_soon',
] as const;

export type DealListingSort = (typeof DEAL_SORTS)[number];

const MIN_RATINGS = ['4', '4.5'] as const;

type Search = Record<string, string | string[] | undefined>;

function first(value: string | string[] | undefined): string | undefined {
  const raw = Array.isArray(value) ? value[0] : value;
  return raw?.trim() || undefined;
}

function positiveInt(value: string | undefined): string | undefined {
  return value && /^\d+$/.test(value) && Number(value) >= 1 ? value : undefined;
}

/**
 * Maps friendly listing URL params (`?sort=price_asc&rating=4&endingSoon=true`) to the
 * exact query parameters of GET /api/v1/deals. Unknown or invalid values are dropped
 * instead of being forwarded, so a hand-edited URL cannot trigger a 400 from the API.
 */
export function buildDealQuery({
  citySlug,
  categorySlug,
  searchQuery,
  searchParams,
}: {
  citySlug?: string | undefined;
  categorySlug?: string | undefined;
  searchQuery?: string | undefined;
  searchParams: Search;
}): URLSearchParams {
  const query = new URLSearchParams();
  if (citySlug) query.set('city', citySlug);
  if (categorySlug) query.set('category', categorySlug);
  if (searchQuery?.trim()) query.set('q', searchQuery.trim().slice(0, 200));

  const subcategory = first(searchParams.subcategory);
  if (subcategory) query.set('subcategory', subcategory);

  const rating = first(searchParams.rating);
  if (rating && (MIN_RATINGS as readonly string[]).includes(rating)) query.set('min_rating', rating);

  if (first(searchParams.endingSoon) === 'true') query.set('ending_soon', 'true');

  const sort = first(searchParams.sort);
  if (sort && (DEAL_SORTS as readonly string[]).includes(sort)) query.set('sort', sort);

  const page = positiveInt(first(searchParams.page));
  if (page) query.set('page', page);

  return query;
}
