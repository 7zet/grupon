import type { components } from '@grupon/api-client';
import { getFormatter, getTranslations } from 'next-intl/server';

import { VoucherRedeemPanel } from '@/components/voucher-redeem-panel';
import { accountGet, loadForVisitor } from '@/lib/account-api';
import { pickText } from '@/lib/localized';

type LocationList = components['schemas']['LocationList'];
type RedemptionPage = components['schemas']['MerchantRedemptionPage'];

export default async function RedeemPage({ params }: { params: Promise<{ locale: string }> }) {
  const { locale } = await params;
  const t = await getTranslations('Redeem');
  const format = await getFormatter();
  const [list, history] = await loadForVisitor(locale, `/${locale}/merchant/redeem`, () =>
    Promise.all([
      accountGet<LocationList>('/merchant/locations'),
      accountGet<RedemptionPage>('/merchant/redemptions'),
    ]),
  );
  const locations = list.items.map((location) => ({
    id: location.id,
    label: `${pickText(location.city.name, locale)} — ${location.address}`,
  }));
  return (
    <div className="grid gap-8">
      <h1 className="text-title font-bold">{t('title')}</h1>
      <VoucherRedeemPanel locations={locations} locale={locale} />
      <section className="grid gap-4">
        <h2 className="text-lead font-semibold">{t('historyTitle')}</h2>
        {history.items.length === 0 ? (
          <p className="text-ink-muted">{t('historyEmpty')}</p>
        ) : (
          <div className="overflow-x-auto rounded-card border border-border">
            <table className="w-full text-left text-small">
              <thead className="bg-surface-muted">
                <tr>
                  {(
                    ['time', 'deal', 'option', 'buyer', 'branch', 'cashier', 'reversed'] as const
                  ).map((key) => (
                    <th key={key} className="px-3 py-2 font-semibold">
                      {t(`history.${key}`)}
                    </th>
                  ))}
                </tr>
              </thead>
              <tbody>
                {history.items.map((item) => (
                  <tr key={item.id} className="border-t border-border">
                    <td className="px-3 py-2">
                      {format.dateTime(new Date(item.time), {
                        dateStyle: 'medium',
                        timeStyle: 'short',
                      })}
                    </td>
                    <td className="px-3 py-2">{pickText(item.deal, locale)}</td>
                    <td className="px-3 py-2">{pickText(item.option, locale)}</td>
                    <td className="px-3 py-2">{item.buyer}</td>
                    <td className="px-3 py-2">{item.branch}</td>
                    <td className="px-3 py-2">{item.cashier}</td>
                    <td className="px-3 py-2">{item.reversed ? t('yes') : t('no')}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </section>
    </div>
  );
}
