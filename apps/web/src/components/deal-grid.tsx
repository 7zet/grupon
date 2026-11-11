import type { DealSummary } from '@dealo/api-client';
import type { Currency, Locale } from '@dealo/config';
import { getTranslations } from 'next-intl/server';
import { Link } from '@/i18n/navigation';
import { DealCard } from './deal-card';

export async function DealGrid({
  deals,
  locale,
  currency,
}: {
  deals: DealSummary[];
  locale: Locale;
  currency: Currency;
}) {
  const t = await getTranslations('Common');
  if (!deals.length)
    return (
      <div className="rounded-panel-ui border bg-surface px-6 py-16 text-center">
        <h2 className="text-subtitle font-semibold">{t('emptyTitle')}</h2>
        <p className="mt-2 text-ink-muted">{t('emptyText')}</p>
        <Link
          className="mt-6 inline-block rounded-control-ui bg-brand px-4 py-3 font-semibold text-surface"
          href="/"
        >
          {t('clearFilters')}
        </Link>
      </div>
    );
  return (
    <div className="grid grid-cols-2 gap-3 sm:gap-5 lg:grid-cols-3 xl:grid-cols-4">
      {deals.map((deal) => (
        <DealCard key={deal.id} deal={deal} locale={locale} currency={currency} />
      ))}
    </div>
  );
}
