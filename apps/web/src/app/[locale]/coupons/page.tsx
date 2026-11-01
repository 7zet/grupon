import { publicApi } from '@grupon/api-client';
import type { Locale } from '@grupon/config';
import { getTranslations } from 'next-intl/server';
import { DealGrid } from '@/components/deal-grid';
import { getCurrency } from '@/lib/preferences';

export default async function CouponsPage({ params }: { params: Promise<{ locale: Locale }> }) {
  const { locale } = await params;
  const [t, deals, currency] = await Promise.all([
    getTranslations('Coupons'),
    publicApi.deals(new URLSearchParams({ category: 'coupons' })),
    getCurrency(),
  ]);

  return (
    <div className="mx-auto max-w-layout px-4 py-10 sm:px-6 lg:px-8">
      <h1 className="text-title font-bold">{t('title')}</h1>
      <p className="mt-2 text-ink-muted">{t('description')}</p>
      <div className="my-8">
        <DealGrid deals={deals.items} locale={locale} currency={currency} />
      </div>
    </div>
  );
}
