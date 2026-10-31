import type { components } from '@grupon/api-client';
import { formatMoney } from '@grupon/ui';
import { getFormatter, getTranslations } from 'next-intl/server';
import { notFound } from 'next/navigation';

import { AccountShell } from '@/components/account-shell';
import { accountGetOrNull, loadForVisitor } from '@/lib/account-api';
import { pickText } from '@/lib/localized';

type Order = components['schemas']['Order'];

export default async function OrderPage({
  params,
}: {
  params: Promise<{ locale: string; id: string }>;
}) {
  const { locale, id } = await params;
  const t = await getTranslations('Account');
  const format = await getFormatter();
  const order = await loadForVisitor(locale, `/${locale}/account/orders/${id}`, () =>
    accountGetOrNull<Order>(`/me/orders/${encodeURIComponent(id)}`),
  );
  if (!order) notFound();
  const money = (amount: number) => formatMoney(amount, locale as Parameters<typeof formatMoney>[1], 'UZS');

  return (
    <AccountShell section="orders">
      <h2 className="text-lead font-semibold text-ink">
        {t('orderTitle')} {order.number}
      </h2>
      <dl className="mt-4 grid gap-2 text-sm">
        <div className="flex justify-between">
          <dt className="text-ink-muted">{t('placedAt')}</dt>
          <dd>{format.dateTime(new Date(order.created_at), { dateStyle: 'medium' })}</dd>
        </div>
        <div className="flex justify-between">
          <dt className="text-ink-muted">{t('status')}</dt>
          <dd>{order.status}</dd>
        </div>
        <div className="flex justify-between">
          <dt className="text-ink-muted">{t('total')}</dt>
          <dd className="font-semibold text-ink">{money(order.total.amount)}</dd>
        </div>
      </dl>
      <ul className="mt-6 grid gap-3">
        {order.items.map((item) => (
          <li key={item.id} className="rounded-card border border-border bg-surface p-4">
            <span className="block font-semibold text-ink">{pickText(item.title_snapshot, locale)}</span>
            <span className="text-sm text-ink-muted">× {item.quantity}</span>
          </li>
        ))}
      </ul>
    </AccountShell>
  );
}
