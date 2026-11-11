import { businessGate } from '@/components/workspace/gate';
import type { components } from '@dealo/api-client';
import { getFormatter, getTranslations } from 'next-intl/server';

import { KassaHistory } from '@/components/kassa-history';
import { VoucherRedeemPanel } from '@/components/voucher-redeem-panel';
import { accountGet, currentUser, loadForVisitor } from '@/lib/account-api';
import { pickText } from '@/lib/localized';

type LocationList = components['schemas']['LocationList'];
type RedemptionPage = components['schemas']['MerchantRedemptionPage'];

export default async function RedeemPage({ params }: { params: Promise<{ locale: string }> }) {
  const denied = await businessGate('redeem');
  if (denied) return denied;
  const { locale } = await params;
  const t = await getTranslations('Redeem');
  const format = await getFormatter();
  const [list, history] = await loadForVisitor(locale, `/${locale}/merchant/redeem`, () =>
    Promise.all([
      accountGet<LocationList>('/merchant/locations'),
      accountGet<RedemptionPage>('/merchant/redemptions'),
    ]),
  );
  const user = await currentUser();
  const cashierOnly = Boolean(user?.roles.includes('merchant_cashier')) && !user?.roles.some((role) => role === 'merchant_owner' || role === 'merchant_manager');
  const locations = list.items.map((location) => ({
    id: location.id,
    label: `${pickText(location.city.name, locale)} — ${location.address}`,
  }));
  if (cashierOnly) {
    // The cashier screen has nothing but the scanner, the result card and today's own redemptions.
    return (
      <div className="grid grid-cols-[minmax(0,1fr)] gap-6">
        <VoucherRedeemPanel locations={locations} locale={locale} kassa />
        <KassaHistory rows={history.items} locale={locale} />
      </div>
    );
  }
  return (
    <div className="grid grid-cols-[minmax(0,1fr)] gap-8">
      <h1 className="text-title font-bold">{t('title')}</h1>
      <VoucherRedeemPanel locations={locations} locale={locale} />
      <section className="grid grid-cols-[minmax(0,1fr)] gap-4">
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
