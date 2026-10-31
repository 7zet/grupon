import { publicApi } from '@grupon/api-client';
import type { Locale } from '@grupon/config';
import { getTranslations } from 'next-intl/server';
import { DealGrid } from '@/components/deal-grid';
import { PromoCard } from '@/components/promo-card';
import { Link } from '@/i18n/navigation';
import { getCurrency } from '@/lib/preferences';
export default async function CouponsPage({
  params,
  searchParams,
}: {
  params: Promise<{ locale: Locale }>;
  searchParams: Promise<{ tab?: string }>;
}) {
  const { locale } = await params;
  const { tab = 'deals' } = await searchParams;
  const [t, deals, promos, currency] = await Promise.all([
    getTranslations('Coupons'),
    publicApi.deals(new URLSearchParams({ category: 'coupons' })),
    publicApi.promoCodes(),
    getCurrency(),
  ]);
  return (
    <div className="mx-auto max-w-layout px-4 py-10 sm:px-6 lg:px-8">
      <h1 className="text-title font-bold">{t('title')}</h1>
      <p className="mt-2 text-ink-muted">{t('description')}</p>
      <div className="my-8 flex gap-2 border-b">
        <Link
          className={`border-b-2 px-4 py-3 font-semibold ${tab === 'deals' ? 'border-brand text-brand' : 'border-transparent text-ink-muted'}`}
          href="/coupons?tab=deals"
        >
          {t('deals')}
        </Link>
        <Link
          className={`border-b-2 px-4 py-3 font-semibold ${tab === 'promo' ? 'border-brand text-brand' : 'border-transparent text-ink-muted'}`}
          href="/coupons?tab=promo"
        >
          {t('promo')}
        </Link>
      </div>
      {tab === 'promo' ? (
        <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4">
          {promos.items.map((promo) => (
            <PromoCard key={promo.id} promo={promo} locale={locale} />
          ))}
        </div>
      ) : (
        <DealGrid deals={deals.items} locale={locale} currency={currency} />
      )}
    </div>
  );
}
