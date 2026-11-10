import { businessGate } from '@/components/workspace/gate';
import type { components } from '@dealo/api-client';
import { formatMoney } from '@dealo/ui';
import type { Currency, Locale } from '@dealo/config';
import { getFormatter, getTranslations } from 'next-intl/server';

import { accountGet, loadForVisitor } from '@/lib/account-api';
import { pickText } from '@/lib/localized';

type OrderPage = components['schemas']['OrderPage'];

const STATUS_KEYS = {
  awaiting_payment: 'orderPending',
  created: 'orderPending',
  paid: 'orderPaid',
  completed: 'orderCompleted',
  expired: 'orderExpired',
  failed: 'orderFailed',
  cancelled: 'orderCancelled',
  partially_refunded: 'orderPartiallyRefunded',
  refunded: 'orderRefunded',
} as const;

/** Orders that include this business's deals, newest first. Shows the deal names and what the buyer paid. */
export default async function OrdersPage({ params }: { params: Promise<{ locale: string }> }) {
  const denied = await businessGate('orders');
  if (denied) return denied;
  const { locale } = await params;
  const t = await getTranslations('MerchantPanel');
  const format = await getFormatter();
  const list = await loadForVisitor(locale, `/${locale}/merchant/orders`, () =>
    accountGet<OrderPage>('/merchant/orders'),
  );
  const currency: Currency = 'UZS';

  return (
    <div className="grid grid-cols-[minmax(0,1fr)] gap-8">
      <h1 className="text-title font-bold">{t('ordersTitle')}</h1>
      {list.items.length === 0 ? (
        <p className="text-ink-muted">{t('ordersEmpty')}</p>
      ) : (
        <div className="overflow-x-auto">
          <table className="w-full min-w-[640px] text-left text-sm">
            <thead className="text-ink-muted">
              <tr>
                <th className="py-2 pr-4 font-semibold">{t('colOrder')}</th>
                <th className="py-2 pr-4 font-semibold">{t('colDate')}</th>
                <th className="py-2 pr-4 font-semibold">{t('colTotal')}</th>
                <th className="py-2 pr-4 font-semibold">{t('colStatus')}</th>
              </tr>
            </thead>
            <tbody>
              {list.items.map((order) => (
                <tr key={order.id} className="border-t border-border align-top">
                  <td className="py-2 pr-4">
                    <div className="font-semibold">{order.number}</div>
                    {order.items.map((item) => (
                      <div key={item.id} className="text-ink-muted">
                        {pickText(item.title_snapshot, locale)} × {item.quantity}
                      </div>
                    ))}
                  </td>
                  <td className="py-2 pr-4">{format.dateTime(new Date(order.created_at), { dateStyle: 'medium' })}</td>
                  <td className="py-2 pr-4">{formatMoney(order.total.amount, locale as Locale, currency)}</td>
                  <td className="py-2 pr-4">
                    {order.status in STATUS_KEYS
                      ? t(STATUS_KEYS[order.status as keyof typeof STATUS_KEYS])
                      : order.status}
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
