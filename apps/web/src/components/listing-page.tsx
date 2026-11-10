import { publicApi } from '@dealo/api-client';
import { localized, type Locale } from '@dealo/config';
import { ChevronRight } from 'lucide-react';
import { getTranslations } from 'next-intl/server';
import { Link } from '@/i18n/navigation';
import { buildDealQuery } from '@/lib/deal-query';
import { getCurrency } from '@/lib/preferences';
import { DealGrid } from './deal-grid';
import { ListingControls } from './listing-controls';

type Search = Record<string, string | string[] | undefined>;
export async function ListingPage({
  locale,
  citySlug,
  categorySlug,
  searchParams,
  searchQuery,
}: {
  locale: Locale;
  citySlug?: string | undefined;
  categorySlug?: string | undefined;
  searchParams: Search;
  searchQuery?: string | undefined;
}) {
  const [config, t, common, currency] = await Promise.all([
    publicApi.config(),
    getTranslations('Listing'),
    getTranslations('Common'),
    getCurrency(),
  ]);
  const city = config.cities.find((item) => item.slug === citySlug) ?? config.cities[0]!;
  const category = config.categories.find((item) => item.slug === categorySlug);
  const query = buildDealQuery({ citySlug, categorySlug, searchQuery, searchParams });
  const result = await publicApi.deals(query);
  // Pagination links keep the friendly URL params (not the API names) so they round-trip.
  const nextPageParams = new URLSearchParams();
  for (const [key, value] of Object.entries(searchParams))
    if (typeof value === 'string' && value && key !== 'page') nextPageParams.set(key, value);
  nextPageParams.set('page', String(result.page + 1));
  const title = searchQuery
    ? t('queryFor', { query: searchQuery })
    : category
      ? t('categoryTitle', {
          category: localized(category.name, locale),
          city: localized(city.name, locale),
        })
      : t('allDeals', { city: localized(city.name, locale) });
  const totalPages = Math.max(1, Math.ceil(result.total / result.page_size));
  return (
    <div className="mx-auto max-w-layout px-4 py-10 sm:px-6 lg:px-8">
      <nav className="mb-4 flex items-center gap-1 text-small text-ink-muted">
        <Link href="/">{(await getTranslations('Navigation'))('home')}</Link>
        <ChevronRight className="size-4" />
        <span>{title}</span>
      </nav>
      <div className="mb-8 flex flex-wrap items-end justify-between gap-4">
        <div>
          <h1 className="text-title font-bold tracking-tight">{title}</h1>
          <p className="mt-2 text-ink-muted">{common('results', { count: result.total })}</p>
        </div>
      </div>
      <ListingControls
        subcategories={(category?.subcategories ?? []).map((item) => ({
          slug: item.slug,
          label: localized(item.name, locale),
        }))}
      />
      <DealGrid deals={result.items} locale={locale} currency={currency} />
      {result.page < totalPages && (
        <div className="mt-10 text-center">
          <Link
            className="inline-block rounded-control-ui border bg-surface px-6 py-3 font-semibold text-brand"
            href={`?${nextPageParams}`}
          >
            {common('loadMore')}
          </Link>
        </div>
      )}
    </div>
  );
}
