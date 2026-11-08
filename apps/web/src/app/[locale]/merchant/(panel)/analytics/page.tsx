import type { components } from '@grupon/api-client';
import { formatMoney } from '@grupon/ui';
import type { Currency, Locale } from '@grupon/config';
import { getFormatter, getTranslations } from 'next-intl/server';

import { accountGet, loadForVisitor } from '@/lib/account-api';

type MerchantAnalytics = components['schemas']['MerchantAnalytics'];

const DAY_MS = 24 * 60 * 60 * 1000;
const DATE = /^\d{4}-\d{2}-\d{2}$/;

/** Sales for a period (default: the last 30 days). Shows totals, redemption rate and a day-by-day bar list. */
export default async function AnalyticsPage({
  params,
  searchParams,
}: {
  params: Promise<{ locale: string }>;
  searchParams: Promise<{ from?: string; to?: string }>;
}) {
  const { locale } = await params;
  const query = await searchParams;
  const today = new Date().toISOString().slice(0, 10);
  const to = query.to && DATE.test(query.to) ? query.to : today;
  const from =
    query.from && DATE.test(query.from)
      ? query.from
      : new Date(Date.parse(to) - 29 * DAY_MS).toISOString().slice(0, 10);
  const t = await getTranslations('MerchantPanel');
  const format = await getFormatter();
  const data = await loadForVisitor(locale, `/${locale}/merchant/analytics`, () =>
    accountGet<MerchantAnalytics>(`/merchant/analytics?from=${from}&to=${to}`),
  );
  const currency: Currency = 'UZS';
  const peak = Math.max(1, ...data.sales.map((point) => point.value));
  const percent = Math.round(data.redemption_rate * 100);

  return (
    <div className="grid grid-cols-[minmax(0,1fr)] gap-8">
      <h1 className="text-title font-bold">{t('analyticsTitle')}</h1>
      <form className="flex flex-wrap items-end gap-4 text-sm" method="get">
        <label className="grid gap-2 font-semibold">
          {t('fromDate')}
          <input type="date" name="from" defaultValue={from} className="rounded-control-ui border border-border bg-surface px-4 py-3" />
        </label>
        <label className="grid gap-2 font-semibold">
          {t('toDate')}
          <input type="date" name="to" defaultValue={to} className="rounded-control-ui border border-border bg-surface px-4 py-3" />
        </label>
        <button type="submit" className="rounded-control-ui bg-brand px-5 py-3 font-semibold text-surface">
          {t('apply')}
        </button>
      </form>
      <dl className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
        <Kpi label={t('grossSales')} value={formatMoney(data.gross_sales.amount, locale as Locale, currency)} />
        <Kpi label={t('vouchersSold')} value={String(data.vouchers_sold)} />
        <Kpi label={t('vouchersRedeemed')} value={String(data.vouchers_redeemed)} />
        <Kpi label={t('redemptionRate')} value={`${percent}%`} />
      </dl>
      <section className="grid gap-3">
        <h2 className="font-semibold">{t('salesByDay')}</h2>
        {data.sales.length === 0 ? (
          <p className="text-ink-muted">{t('analyticsEmpty')}</p>
        ) : (
          <ul className="grid gap-2">
            {data.sales.map((point) => (
              <li key={point.date} className="grid grid-cols-[7rem_1fr_auto] items-center gap-3 text-sm">
                <span className="text-ink-muted">{format.dateTime(new Date(point.date), { dateStyle: 'medium' })}</span>
                <span className="h-3 rounded-full bg-brand" style={{ width: `${Math.round((point.value / peak) * 100)}%` }} />
                <span>{formatMoney(point.value, locale as Locale, currency)}</span>
              </li>
            ))}
          </ul>
        )}
      </section>
    </div>
  );
}

function Kpi({ label, value }: { label: string; value: string }) {
  return (
    <div className="rounded-card border border-border p-5">
      <dt className="text-sm text-ink-muted">{label}</dt>
      <dd className="mt-2 text-title font-bold">{value}</dd>
    </div>
  );
}
