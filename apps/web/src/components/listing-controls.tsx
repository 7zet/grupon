'use client';

import { SlidersHorizontal } from 'lucide-react';
import { useTranslations } from 'next-intl';
import { parseAsBoolean, parseAsString, useQueryStates } from 'nuqs';

export function ListingControls({
  subcategories,
}: {
  subcategories: { slug: string; label: string }[];
}) {
  const t = useTranslations('Listing');
  const [filters, setFilters] = useQueryStates(
    {
      subcategory: parseAsString,
      rating: parseAsString,
      sort: parseAsString.withDefault('recommended'),
      endingSoon: parseAsBoolean.withDefault(false),
    },
    { shallow: false },
  );
  return (
    <div className="mb-6 flex flex-wrap items-center gap-3 rounded-card-ui border bg-surface p-3">
      <span className="flex items-center gap-2 px-2 text-small font-semibold">
        <SlidersHorizontal className="size-4" />
        {t('filters')}
      </span>
      {subcategories.length > 0 && (
        <select
          aria-label={t('subcategory')}
          className="rounded-control-ui border bg-surface px-3 py-2 text-small"
          value={filters.subcategory ?? ''}
          onChange={(event) =>
            void setFilters({ subcategory: event.target.value || null, page: null } as never)
          }
        >
          <option value="">{t('subcategory')}</option>
          {subcategories.map((item) => (
            <option key={item.slug} value={item.slug}>
              {item.label}
            </option>
          ))}
        </select>
      )}
      <select
        aria-label={t('rating')}
        className="rounded-control-ui border bg-surface px-3 py-2 text-small"
        value={filters.rating ?? ''}
        onChange={(event) => void setFilters({ rating: event.target.value || null } as never)}
      >
        <option value="">{t('rating')}</option>
        <option value="4">4.0+</option>
        <option value="4.5">4.5+</option>
      </select>
      <label className="flex items-center gap-2 rounded-control-ui border px-3 py-2 text-small">
        <input
          type="checkbox"
          checked={filters.endingSoon}
          onChange={(event) => void setFilters({ endingSoon: event.target.checked })}
        />
        {t('endingSoon')}
      </label>
      <select
        aria-label={t('sort')}
        className="ml-auto rounded-control-ui border bg-surface px-3 py-2 text-small"
        value={filters.sort}
        onChange={(event) => void setFilters({ sort: event.target.value })}
      >
        <option value="recommended">{t('recommended')}</option>
        <option value="popular">{t('popular')}</option>
        <option value="newest">{t('newest')}</option>
        <option value="price_asc">{t('priceLow')}</option>
        <option value="price_desc">{t('priceHigh')}</option>
        <option value="biggest_discount">{t('biggestDiscount')}</option>
        <option value="ending_soon">{t('soon')}</option>
      </select>
    </div>
  );
}
