import { businessGate } from '@/components/workspace/gate';
import type { components } from '@dealo/api-client';
import { formatMoney } from '@dealo/ui';
import { getFormatter, getTranslations } from 'next-intl/server';

import { DealCard } from '@/components/deal-card';
import { accountGet, loadForVisitor } from '@/lib/account-api';

type Dashboard = components['schemas']['MerchantDashboard'];

export default async function DashboardPage({ params }: { params: Promise<{ locale: string }> }) {
  const denied = await businessGate('dashboard');
  if (denied) return denied;
  const { locale } = await params;
  const t = await getTranslations('Panel');
  const format = await getFormatter();
  const dashboard = await loadForVisitor(locale, `/${locale}/merchant/dashboard`, () =>
    accountGet<Dashboard>('/merchant/dashboard'),
  );
  const money = (amount: number) => formatMoney(amount, locale as Parameters<typeof formatMoney>[1], 'UZS');
  const peak = Math.max(1, ...dashboard.sales_chart.map((point) => point.value));

  const metrics = [
    { label: t('totalSales'), metric: dashboard.total_sales, format: money },
    { label: t('vouchersSold'), metric: dashboard.vouchers_sold, format: (v: number) => String(v) },
    { label: t('vouchersRedeemed'), metric: dashboard.vouchers_redeemed, format: (v: number) => String(v) },
    { label: t('activeDeals'), metric: dashboard.active_deals, format: (v: number) => String(v) },
  ];

  return (
    <div className="grid grid-cols-[minmax(0,1fr)] gap-8">
      <h1 className="text-title font-bold">{t('overviewTitle')}</h1>
      <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
        {metrics.map(({ label, metric, format: render }) => (
          <div key={label} className="rounded-card border border-border bg-surface p-4">
            <p className="text-sm text-ink-muted">{label}</p>
            <p className="mt-2 whitespace-nowrap font-display text-lead font-extrabold tabular-nums text-ink">{render(metric.value)}</p>
            {/* No comparison is shown when there is nothing to compare with (a zero change). */}
            {metric.change_percent !== 0 && (
              <p className={`mt-1 text-sm ${metric.change_percent > 0 ? 'text-brand' : 'text-ink-muted'}`}>
                {t('vsPrevious', { percent: `${metric.change_percent > 0 ? '+' : ''}${metric.change_percent.toFixed(1)}` })}
              </p>
            )}
          </div>
        ))}
      </div>

      <section aria-labelledby="chart-title" className="rounded-card border border-border bg-surface p-5">
        <h2 id="chart-title" className="font-semibold">{t('salesChart')}</h2>
        {dashboard.sales_chart.length === 0 ? (
          <p className="mt-4 text-ink-muted">{t('noChart')}</p>
        ) : (
          <div className="mt-4 grid grid-cols-[auto_minmax(0,1fr)] gap-x-3">
            {/* Y axis: the top line is the busiest day, the bottom is zero. */}
            <div className="flex h-44 flex-col justify-between text-right text-label tabular-nums text-ink-muted">
              <span>{money(peak)}</span>
              <span>{money(Math.round(peak / 2))}</span>
              <span>0</span>
            </div>
            <div className="relative h-44">
              <div aria-hidden="true" className="absolute inset-0 flex flex-col justify-between">
                <span className="border-t border-dashed border-border" />
                <span className="border-t border-dashed border-border" />
                <span className="border-t border-border" />
              </div>
              <div className="relative flex h-full items-end gap-1" role="img" aria-label={t('salesChart')}>
                {dashboard.sales_chart.map((point) => (
                  <div
                    key={point.date}
                    title={`${format.dateTime(new Date(point.date), { dateStyle: 'medium' })}: ${money(point.value)}`}
                    className={`flex-1 rounded-t ${point.value > 0 ? 'bg-brand hover:bg-brand-hover' : 'bg-border'}`}
                    style={{ height: point.value > 0 ? `${Math.max(3, (point.value / peak) * 100)}%` : '2px' }}
                  />
                ))}
              </div>
            </div>
            <span />
            <div className="mt-2 flex justify-between text-label text-ink-muted">
              <span>{format.dateTime(new Date(dashboard.sales_chart[0]!.date), { day: 'numeric', month: 'short' })}</span>
              <span>
                {format.dateTime(new Date(dashboard.sales_chart[dashboard.sales_chart.length - 1]!.date), {
                  day: 'numeric',
                  month: 'short',
                })}
              </span>
            </div>
          </div>
        )}
      </section>

      <section aria-labelledby="pending-title" className="grid gap-3">
        <h2 id="pending-title" className="font-semibold">{t('pendingDeals')}</h2>
        {dashboard.pending_deals.length === 0 ? (
          <p className="text-ink-muted">{t('noPending')}</p>
        ) : (
          <ul className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
            {dashboard.pending_deals.map((deal) => (
              <li key={deal.id}>
                <DealCard deal={deal} locale={locale as Parameters<typeof formatMoney>[1]} currency="UZS" />
              </li>
            ))}
          </ul>
        )}
      </section>
    </div>
  );
}
