import type { components } from '@grupon/api-client';
import { getTranslations } from 'next-intl/server';
import { redirect } from 'next/navigation';
import type { ReactNode } from 'react';

import { PanelNav } from '@/components/panel-nav';
import { accountGet, accountGetOrNull, loadForVisitor } from '@/lib/account-api';

type User = components['schemas']['User'];
type Merchant = components['schemas']['Merchant'];

/** Role guard for the merchant panel. The API enforces the same rule on every merchant operation. */
export default async function PanelLayout({
  children,
  params,
}: {
  children: ReactNode;
  params: Promise<{ locale: string }>;
}) {
  const { locale } = await params;
  const [user, business] = await loadForVisitor(locale, `/${locale}/merchant/dashboard`, () =>
    Promise.all([accountGetOrNull<User>('/me'), accountGet<Merchant>('/merchant/business')]),
  );
  const isMerchant = Boolean(user?.roles.some((role) => role.startsWith('merchant_')));
  if (!user || !isMerchant) redirect(`/${locale}/merchant/register`);

  const t = await getTranslations('Panel');
  const m = await getTranslations('MerchantPanel');
  const identity = await getTranslations('Identity');
  const role = user.roles.find((item) => item.startsWith('merchant_'));
  const roleLabel = role === 'merchant_owner' ? identity('owner') : role === 'merchant_manager' ? identity('manager') : identity('cashier');
  const owner = role === 'merchant_owner';
  const cashier = role === 'merchant_cashier';
  const all = [
    { href: '/merchant/dashboard', label: t('navOverview'), who: 'om' },
    { href: '/merchant/business', label: t('navBusiness'), who: 'om' },
    { href: '/merchant/deals', label: t('navDeals'), who: 'om' },
    { href: '/merchant/deals/new', label: t('navCreateDeal'), who: 'om' },
    { href: '/merchant/media', label: t('navMedia'), who: 'om' },
    { href: '/merchant/redeem', label: t('navRedeem'), who: 'omc' },
    { href: '/merchant/staff', label: m('navStaff'), who: 'o' },
    { href: '/merchant/orders', label: m('navOrders'), who: 'om' },
    { href: '/merchant/payouts', label: m('navPayouts'), who: 'o' },
    { href: '/merchant/customers', label: m('navCustomers'), who: 'om' },
    { href: '/merchant/analytics', label: m('navAnalytics'), who: 'om' },
    { href: '/merchant/reviews', label: m('navReviews'), who: 'om' },
    { href: '/merchant/calculator', label: m('navCalculator'), who: 'om' },
    { href: '/merchant/settings', label: m('navSettings'), who: 'o' },
  ];
  const mine = owner ? 'o' : cashier ? 'c' : 'm';
  const links = all.filter((item) => item.who.includes(mine)).map(({ href, label }) => ({ href, label }));
  return (
    <div className="mx-auto max-w-6xl px-4 py-8 sm:px-6">
      <aside className="mb-5 rounded-card-ui border border-border bg-surface p-4">
        <p className="font-semibold">{user.name}</p>
        <p data-testid="panel-role-badge" className="mt-1 inline-flex rounded-full bg-brand-tint px-2 py-1 text-label font-semibold text-brand">{roleLabel} · {business.display_name}</p>
      </aside>
      <PanelNav label={t('navLabel')} links={links} />
      <div className="mt-8">{children}</div>
    </div>
  );
}
