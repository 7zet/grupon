import type { components } from '@grupon/api-client';
import { getTranslations } from 'next-intl/server';
import { redirect } from 'next/navigation';
import type { ReactNode } from 'react';

import { Link } from '@/i18n/navigation';
import { accountGet, accountGetOrNull, loadForVisitor } from '@/lib/account-api';

type User = components['schemas']['User'];
type Merchant = components['schemas']['Merchant'];

/** Panel sections added after the first release. Labels come from MerchantPanel so each reads in the locale. */
const MERCHANT_LINKS = [
  { href: '/merchant/staff', key: 'navStaff' },
  { href: '/merchant/orders', key: 'navOrders' },
  { href: '/merchant/payouts', key: 'navPayouts' },
  { href: '/merchant/customers', key: 'navCustomers' },
  { href: '/merchant/analytics', key: 'navAnalytics' },
  { href: '/merchant/reviews', key: 'navReviews' },
  { href: '/merchant/calculator', key: 'navCalculator' },
  { href: '/merchant/settings', key: 'navSettings' },
] as const;

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
  return (
    <div className="mx-auto max-w-6xl px-4 py-8 sm:px-6">
      <aside className="mb-5 rounded-card-ui border border-border bg-surface p-4">
        <p className="font-semibold">{user.name}</p>
        <p data-testid="panel-role-badge" className="mt-1 inline-flex rounded-full bg-brand-tint px-2 py-1 text-label font-semibold text-brand">{roleLabel} · {business.display_name}</p>
      </aside>
      <nav aria-label="Merchant" className="flex flex-wrap gap-2 border-b border-border pb-3">
        <Link
          href="/merchant/dashboard"
          className="rounded-control-ui px-4 py-2 text-ink-muted hover:text-ink"
        >
          {t('navOverview')}
        </Link>
        <Link
          href="/merchant/business"
          className="rounded-control-ui px-4 py-2 text-ink-muted hover:text-ink"
        >
          {t('navBusiness')}
        </Link>
        <Link
          href="/merchant/deals"
          className="rounded-control-ui px-4 py-2 text-ink-muted hover:text-ink"
        >
          {t('navDeals')}
        </Link>
        <Link
          href="/merchant/deals/new"
          className="rounded-control-ui px-4 py-2 text-ink-muted hover:text-ink"
        >
          {t('navCreateDeal')}
        </Link>
        <Link
          href="/merchant/media"
          className="rounded-control-ui px-4 py-2 text-ink-muted hover:text-ink"
        >
          {t('navMedia')}
        </Link>
        <Link
          href="/merchant/redeem"
          className="rounded-control-ui px-4 py-2 text-ink-muted hover:text-ink"
        >
          {t('navRedeem')}
        </Link>
        {MERCHANT_LINKS.map(({ href, key }) => (
          <Link key={href} href={href} className="rounded-control-ui px-4 py-2 text-ink-muted hover:text-ink">
            {m(key)}
          </Link>
        ))}
      </nav>
      <div className="mt-8">{children}</div>
    </div>
  );
}
