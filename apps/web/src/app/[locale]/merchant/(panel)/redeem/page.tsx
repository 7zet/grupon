import type { components } from '@grupon/api-client';
import { getTranslations } from 'next-intl/server';

import { VoucherRedeemPanel } from '@/components/voucher-redeem-panel';
import { accountGet, loadForVisitor } from '@/lib/account-api';
import { pickText } from '@/lib/localized';

type LocationList = components['schemas']['LocationList'];

export default async function RedeemPage({ params }: { params: Promise<{ locale: string }> }) {
  const { locale } = await params;
  const t = await getTranslations('Redeem');
  const list = await loadForVisitor(locale, `/${locale}/merchant/redeem`, () =>
    accountGet<LocationList>('/merchant/locations'),
  );
  const locations = list.items.map((location) => ({
    id: location.id,
    label: `${pickText(location.city.name, locale)} — ${location.address}`,
  }));
  return (
    <div className="grid gap-8">
      <h1 className="text-title font-bold">{t('title')}</h1>
      <VoucherRedeemPanel locations={locations} locale={locale} />
    </div>
  );
}
