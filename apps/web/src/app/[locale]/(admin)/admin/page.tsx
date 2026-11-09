import { adminGate } from '@/components/workspace/gate';
import type { components } from '@grupon/api-client';
import { getFormatter, getTranslations } from 'next-intl/server';
import { notFound } from 'next/navigation';

import { accountGet, accountGetOrNull, loadForVisitor } from '@/lib/account-api';

type User = components['schemas']['User'];
type Overview = components['schemas']['AdminOverview'];

const STAFF = ['superadmin', 'admin', 'moderator'];

/** Platform totals for staff: users, approved businesses, active deals and today's orders. */
export default async function AdminOverviewPage({ params }: { params: Promise<{ locale: string }> }) {
  const denied = await adminGate('overview');
  if (denied) return denied;
  const { locale } = await params;
  const t = await getTranslations('AdminOverview');
  const format = await getFormatter();
  const user = await loadForVisitor(locale, `/${locale}/admin`, () => accountGetOrNull<User>('/me'));
  if (!user?.roles.some((role) => STAFF.includes(role))) notFound();
  const totals = await accountGet<Overview>('/admin/overview');
  const tiles = [
    ['users', totals.users],
    ['businesses', totals.businesses],
    ['activeDeals', totals.active_deals],
    ['ordersToday', totals.orders_today],
  ] as const;

  return (
    <div className="grid grid-cols-[minmax(0,1fr)] gap-8">
      <h1 className="text-title font-bold">{t('title')}</h1>
      <dl className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
        {tiles.map(([key, value]) => (
          <div key={key} className="rounded-card border border-border bg-surface p-4">
            <dt className="text-small text-ink-muted">{t(key)}</dt>
            <dd data-testid={`overview-${key}`} className="mt-1 text-title font-bold">
              {format.number(value)}
            </dd>
          </div>
        ))}
      </dl>
    </div>
  );
}
