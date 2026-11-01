import type { components } from './schema';

export type PublicConfig = components['schemas']['PublicConfig'];
export type Homepage = components['schemas']['Homepage'];
export type Deal = components['schemas']['Deal'];
export type DealSummary = components['schemas']['DealSummary'];
export type DealPage = components['schemas']['DealPage'];
export type MerchantProfile = components['schemas']['MerchantProfile'];
export type ReviewList = components['schemas']['ReviewList'];
export type SuggestionList = components['schemas']['SuggestionList'];

declare const process: { env: { NEXT_PUBLIC_API_BASE_URL?: string } };
const apiBaseUrl = (process.env.NEXT_PUBLIC_API_BASE_URL ?? 'http://localhost:8080/api/v1').replace(
  /\/+$/,
  '',
);

/** Cache tag shared by every catalog fetch; the API revalidates it via /api/revalidate. */
export const CATALOG_CACHE_TAG = 'catalog';

type ApiRequestInit = RequestInit & { next?: { revalidate?: number; tags?: string[] } };

function cached(revalidate: number): ApiRequestInit {
  return { next: { revalidate, tags: [CATALOG_CACHE_TAG] } };
}

async function request<T>(path: string, init?: ApiRequestInit): Promise<T> {
  const response = await fetch(`${apiBaseUrl}${path}`, {
    ...init,
    headers: { Accept: 'application/json', ...init?.headers },
  });
  if (!response.ok) throw new Error(`API request failed: ${response.status}`);
  return response.json() as Promise<T>;
}

function withQuery(path: string, query?: URLSearchParams): string {
  const qs = query?.toString() ?? '';
  return qs ? `${path}?${qs}` : path;
}

export const publicApi = {
  config: () => request<PublicConfig>('/config', cached(3600)),
  homepage: (city: string) =>
    request<Homepage>(withQuery('/homepage', new URLSearchParams({ city })), cached(300)),
  deals: (query?: URLSearchParams) => request<DealPage>(withQuery('/deals', query), cached(60)),
  deal: (slug: string) => request<Deal>(`/deals/${encodeURIComponent(slug)}`, cached(60)),
  reviews: (slug: string) =>
    request<ReviewList>(`/deals/${encodeURIComponent(slug)}/reviews`, cached(60)),
  related: (slug: string, city?: string) =>
    request<DealPage>(
      withQuery(
        `/deals/${encodeURIComponent(slug)}/related`,
        city ? new URLSearchParams({ city }) : undefined,
      ),
      cached(60),
    ),
  merchant: (slug: string) =>
    request<MerchantProfile>(`/businesses/${encodeURIComponent(slug)}`, cached(60)),
  suggestions: (query: string) =>
    request<SuggestionList>(
      withQuery('/search/suggestions', new URLSearchParams({ q: query })),
    ),
};
