import type { components } from '@grupon/api-client';
import { getTranslations } from 'next-intl/server';
import { redirect } from 'next/navigation';
import type { ReactNode } from 'react';

import { Link } from '@/i18n/navigation';
import { accountGetOrNull, loadForVisitor } from '@/lib/account-api';

type User = components['schemas']['User'];

/** Role guard for the merchant panel. The API enforces the same rule on every merchant operation. */
export default async function PanelLayout({
  children,
  params,
}: {
  children: ReactNode;
  params: Promise<{ locale: string }>;
}) {
  const { locale } = await params;
  const user = await loadForVisitor(locale, `/${locale}/merchant/dashboard`, () =>
    accountGetOrNull<User>('/me'),
  );
  const isMerchant = Boolean(user?.roles.some((role) => role.startsWith('merchant_')));
  if (!isMerchant) redirect(`/${locale}/merchant/register`);

  const t = await getTranslations('Panel');
  return (
    <div className="mx-auto max-w-6xl px-4 py-8 sm:px-6">
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
      </nav>
      <div className="mt-8">{children}</div>
    </div>
  );
}
