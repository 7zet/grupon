import type { components } from '@grupon/api-client';
import { formatMoney } from '@grupon/ui';
import type { Currency, Locale } from '@grupon/config';
import { getFormatter, getTranslations } from 'next-intl/server';

import { accountGet, loadForVisitor } from '@/lib/account-api';

type CustomerPage = components['schemas']['CustomerPage'];

/** Buyers of this business with their order counts and totals. Names are shortened to "Alisher U." by the API. */
export default async function CustomersPage({ params }: { params: Promise<{ locale: string }> }) {
  const { locale } = await params;
  const t = await getTranslations('MerchantPanel');
  const format = await getFormatter();
  const list = await loadForVisitor(locale, `/${locale}/merchant/customers`, () =>
    accountGet<CustomerPage>('/merchant/customers'),
  );
  const currency: Currency = 'UZS';

  return (
    <div className="grid gap-8">
      <h1 className="text-title font-bold">{t('customersTitle')}</h1>
      {list.items.length === 0 ? (
        <p className="text-ink-muted">{t('customersEmpty')}</p>
      ) : (
        <div className="overflow-x-auto">
          <table className="w-full min-w-[640px] text-left text-sm">
            <thead className="text-ink-muted">
              <tr>
                <th className="py-2 pr-4 font-semibold">{t('colCustomer')}</th>
                <th className="py-2 pr-4 font-semibold">{t('colOrders')}</th>
                <th className="py-2 pr-4 font-semibold">{t('colVouchers')}</th>
                <th className="py-2 pr-4 font-semibold">{t('colSpent')}</th>
                <th className="py-2 pr-4 font-semibold">{t('colLast')}</th>
              </tr>
            </thead>
            <tbody>
              {list.items.map((customer) => (
                <tr key={customer.user_id} className="border-t border-border">
                  <td className="py-2 pr-4">{customer.name}</td>
                  <td className="py-2 pr-4">{customer.orders_count}</td>
                  <td className="py-2 pr-4">{customer.vouchers_count}</td>
                  <td className="py-2 pr-4">
                    {formatMoney(customer.total_spent.amount, locale as Locale, currency)}
                  </td>
                  <td className="py-2 pr-4">
                    {format.dateTime(new Date(customer.last_order_at), { dateStyle: 'medium' })}
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
