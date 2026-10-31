import type { components } from './schema';

export type PublicConfig = components['schemas']['PublicConfig'];
export type Homepage = components['schemas']['Homepage'];
export type Deal = components['schemas']['Deal'];
export type DealSummary = components['schemas']['DealSummary'];
export type DealPage = components['schemas']['DealPage'];
export type MerchantProfile = components['schemas']['MerchantProfile'];
export type ReviewList = components['schemas']['ReviewList'];
export type PromoCodePage = components['schemas']['PromoCodePage'];
export type CollectionList = components['schemas']['CollectionList'];
export type CollectionDetail = components['schemas']['CollectionDetail'];
export type SuggestionList = components['schemas']['SuggestionList'];

declare const process: { env: { NEXT_PUBLIC_API_BASE_URL?: string } };
const apiBaseUrl = process.env.NEXT_PUBLIC_API_BASE_URL ?? 'http://api.mock/api/v1';
type ApiRequestInit = RequestInit & { next?: { revalidate?: number } };

async function request<T>(path: string, init?: ApiRequestInit): Promise<T> {
  const response = await fetch(`${apiBaseUrl}${path}`, {
    ...init,
    headers: { Accept: 'application/json', ...init?.headers },
  });
  if (!response.ok) throw new Error(`API request failed: ${response.status}`);
  return response.json() as Promise<T>;
}

export const publicApi = {
  config: () => request<PublicConfig>('/config', { next: { revalidate: 3600 } }),
  homepage: (city: string) =>
    request<Homepage>(`/homepage?city=${encodeURIComponent(city)}`, { next: { revalidate: 300 } }),
  deals: (query: URLSearchParams) =>
    request<DealPage>(`/deals?${query.toString()}`, { next: { revalidate: 60 } }),
  deal: (slug: string) =>
    request<Deal>(`/deals/${encodeURIComponent(slug)}`, { next: { revalidate: 60 } }),
  reviews: (slug: string) =>
    request<ReviewList>(`/deals/${encodeURIComponent(slug)}/reviews`, { next: { revalidate: 60 } }),
  related: (slug: string) =>
    request<DealPage>(`/deals/${encodeURIComponent(slug)}/related`, { next: { revalidate: 60 } }),
  merchant: (slug: string) =>
    request<MerchantProfile>(`/businesses/${encodeURIComponent(slug)}`, {
      next: { revalidate: 60 },
    }),
  promoCodes: (page = 1) =>
    request<PromoCodePage>(`/promo-codes?page=${page}`, { next: { revalidate: 300 } }),
  collections: () => request<CollectionList>('/collections', { next: { revalidate: 300 } }),
  collection: (slug: string) =>
    request<CollectionDetail>(`/collections/${encodeURIComponent(slug)}`, {
      next: { revalidate: 300 },
    }),
  suggestions: (query: string) =>
    request<SuggestionList>(`/search/suggestions?q=${encodeURIComponent(query)}`),
};
