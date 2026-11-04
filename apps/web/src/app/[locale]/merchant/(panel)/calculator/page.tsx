import { getTranslations } from 'next-intl/server';

import { CampaignCalculator } from '@/components/campaign-calculator';

/** Deal calculator: shows the merchant's take per voucher and in total before the deal is created. */
export default async function CalculatorPage({ params }: { params: Promise<{ locale: string }> }) {
  const { locale } = await params;
  const t = await getTranslations('MerchantPanel');
  return (
    <div className="grid gap-8">
      <h1 className="text-title font-bold">{t('calcTitle')}</h1>
      <CampaignCalculator locale={locale as 'uz' | 'ru' | 'en'} />
    </div>
  );
}
