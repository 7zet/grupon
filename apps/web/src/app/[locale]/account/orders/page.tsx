import type { components } from '@grupon/api-client';
import { formatMoney } from '@grupon/ui';
import { getFormatter, getTranslations } from 'next-intl/server';

import { AccountShell } from '@/components/account-shell';
import { Link } from '@/i18n/navigation';
import { accountGet, loadForVisitor } from '@/lib/account-api';

type OrderPage = components['schemas']['OrderPage'];

export default async function OrdersPage({ params }: { params: Promise<{ locale: string }> }) {
  const { locale } = await params;
  const t = await getTranslations('Account');
  const format = await getFormatter();
  const page = await loadForVisitor(locale, `/${locale}/account/orders`, () =>
    accountGet<OrderPage>('/me/orders'),
  );

  return (
    <AccountShell section="orders">
      {page.items.length === 0 ? (
        <p className="text-ink-muted">{t('emptyOrders')}</p>
      ) : (
        <ul className="grid gap-4">
          {page.items.map((order) => (
            <li key={order.id}>
              <Link
                href={`/account/orders/${order.id}`}
                className="grid gap-1 rounded-card border border-border bg-surface p-4 hover:border-brand"
              >
                <span className="font-mono font-semibold text-ink">{order.number}</span>
                <span className="text-sm text-ink-muted">
                  {format.dateTime(new Date(order.created_at), { dateStyle: 'medium' })}
                </span>
                <span className="text-sm font-semibold text-ink">
                  {formatMoney(order.total.amount, locale as Parameters<typeof formatMoney>[1], 'UZS')}
                </span>
              </Link>
            </li>
          ))}
        </ul>
      )}
    </AccountShell>
  );
}
