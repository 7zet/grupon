import { businessGate } from '@/components/workspace/gate';
import type { components } from '@dealo/api-client';
import { formatMoney } from '@dealo/ui';
import type { Currency, Locale } from '@dealo/config';
import { getFormatter, getTranslations } from 'next-intl/server';
import { notFound } from 'next/navigation';

import { StatementsDownload, type StatementRow } from '@/components/statements-download';
import { accountGet, accountGetOrNull, loadForVisitor } from '@/lib/account-api';

type PayoutPage = components['schemas']['PayoutPage'];
type User = components['schemas']['User'];

const STATUS_KEYS = {
  pending: 'payoutPending',
  processing: 'payoutProcessing',
  paid: 'payoutPaid',
} as const;

/** Payout runs with gross, commission and net. The statement button downloads the same rows as CSV. */
export default async function PayoutsPage({ params }: { params: Promise<{ locale: string }> }) {
  const denied = await businessGate('payouts');
  if (denied) return denied;
  const { locale } = await params;
  const t = await getTranslations('MerchantPanel');
  const format = await getFormatter();
  const user = await loadForVisitor(locale, `/${locale}/merchant/payouts`, () =>
    accountGetOrNull<User>('/me'),
  );
  if (!user?.roles.some((role) => role === 'merchant_owner' || role === 'merchant_manager')) notFound();
  const list = await loadForVisitor(locale, `/${locale}/merchant/payouts`, () =>
    accountGet<PayoutPage>('/merchant/payouts'),
  );
  const currency: Currency = 'UZS';
  const money = (amount: number) => formatMoney(amount, locale as Locale, currency);
  const day = (value: string) => format.dateTime(new Date(value), { dateStyle: 'medium' });

  const rows: StatementRow[] = list.items.map((payout) => ({
    period: `${payout.period_start} – ${payout.period_end}`,
    gross: String(payout.gross.amount / 100),
    commission: String(payout.commission.amount / 100),
    net: String(payout.net.amount / 100),
    status: payout.status,
    paid_on: payout.paid_at ? payout.paid_at : '',
  }));

  return (
    <div className="grid grid-cols-[minmax(0,1fr)] gap-8">
      <div className="flex flex-wrap items-center justify-between gap-4">
        <h1 className="text-title font-bold">{t('payoutsTitle')}</h1>
        <StatementsDownload rows={rows} filename={`payouts-${new Date().toISOString().slice(0, 10)}.csv`} />
      </div>
      {list.items.length === 0 ? (
        <p className="text-ink-muted">{t('payoutsEmpty')}</p>
      ) : (
        <div className="overflow-x-auto">
          <table className="w-full min-w-[640px] text-left text-sm">
            <thead className="text-ink-muted">
              <tr>
                <th className="py-2 pr-4 font-semibold">{t('colPeriod')}</th>
                <th className="py-2 pr-4 font-semibold">{t('colGross')}</th>
                <th className="py-2 pr-4 font-semibold">{t('colCommission')}</th>
                <th className="py-2 pr-4 font-semibold">{t('colNet')}</th>
                <th className="py-2 pr-4 font-semibold">{t('colPaid')}</th>
              </tr>
            </thead>
            <tbody>
              {list.items.map((payout) => (
                <tr key={payout.id} className="border-t border-border">
                  <td className="py-2 pr-4">
                    {day(payout.period_start)} – {day(payout.period_end)}
                  </td>
                  <td className="py-2 pr-4">{money(payout.gross.amount)}</td>
                  <td className="py-2 pr-4">{money(payout.commission.amount)}</td>
                  <td className="py-2 pr-4 font-semibold">{money(payout.net.amount)}</td>
                  <td className="py-2 pr-4">
                    {payout.status === 'paid' && payout.paid_at ? (
                      day(payout.paid_at)
                    ) : (
                      <span className="text-ink-muted">{t(STATUS_KEYS[payout.status])}</span>
                    )}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  );
}
