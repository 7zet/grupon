import type { components } from '@dealo/api-client';
import { formatMoney } from '@dealo/ui';
import { getFormatter, getTranslations } from 'next-intl/server';

import { AccountShell } from '@/components/account-shell';
import { BuyerNotifications } from '@/components/buyer-notifications';
import { loadForVisitor, accountGet } from '@/lib/account-api';
import { pickText } from '@/lib/localized';
import { Link } from '@/i18n/navigation';

type VoucherPage = components['schemas']['VoucherPage'];
type NotificationList = components['schemas']['NotificationList'];

const TABS = ['active', 'used', 'expired'] as const;
type Tab = (typeof TABS)[number];

/** The UI tab "used" maps to the contract status "redeemed". */
const API_STATUS: Record<Tab, 'active' | 'redeemed' | 'expired'> = {
  active: 'active',
  used: 'redeemed',
  expired: 'expired',
};

export default async function CouponsPage({
  params,
  searchParams,
}: {
  params: Promise<{ locale: string }>;
  searchParams: Promise<{ status?: string }>;
}) {
  const { locale } = await params;
  const { status } = await searchParams;
  const tab: Tab = (TABS as readonly string[]).includes(status ?? '') ? (status as Tab) : 'active';
  const t = await getTranslations('Account');
  const format = await getFormatter();

  const page = await loadForVisitor(locale, `/${locale}/account/coupons?status=${tab}`, () =>
    accountGet<VoucherPage>(`/me/vouchers?status=${API_STATUS[tab]}`),
  );

  const notifications = await accountGet<NotificationList>('/me/notifications');

  return (
    <AccountShell section="coupons">
      <BuyerNotifications items={notifications.items} />
      <div role="tablist" className="flex gap-2">
        {TABS.map((item) => (
          <Link
            key={item}
            role="tab"
            aria-selected={item === tab}
            href={{ pathname: '/account/coupons', query: { status: item } }}
            className={
              item === tab
                ? 'rounded-control-ui bg-brand-tint px-4 py-2 font-semibold text-brand'
                : 'rounded-control-ui px-4 py-2 text-ink-muted'
            }
          >
            {t(`tab${item[0]!.toUpperCase()}${item.slice(1)}` as 'tabActive')}
          </Link>
        ))}
      </div>

      {page.items.length === 0 ? (
        <p className="mt-8 text-ink-muted">{t('emptyCoupons')}</p>
      ) : (
        <ul className="mt-6 grid gap-4">
          {page.items.map((voucher) => (
            <li key={voucher.id}>
              <Link
                href={`/account/coupons/${voucher.id}`}
                className="grid gap-1 rounded-card border border-border bg-surface p-4 hover:border-brand"
              >
                <span className="font-semibold text-ink">{pickText(voucher.deal.title, locale)}</span>
                <span className="text-sm text-ink-muted">
                  {voucher.merchant.display_name}
                </span>
                <span className="font-mono text-sm text-ink">{voucher.code}</span>
                <span className="text-sm text-ink-muted">
                  {t('validUntil', { date: format.dateTime(new Date(voucher.valid_until), { dateStyle: 'medium' }) })}
                </span>
                <span className="text-sm font-semibold text-ink">
                  {formatMoney(voucher.option.deal_price.amount, locale as Parameters<typeof formatMoney>[1], 'UZS')}
                </span>
              </Link>
            </li>
          ))}
        </ul>
      )}
    </AccountShell>
  );
}
