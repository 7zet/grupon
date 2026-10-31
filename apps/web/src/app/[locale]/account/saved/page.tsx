import type { components } from '@grupon/api-client';
import { getTranslations } from 'next-intl/server';

import { AccountShell } from '@/components/account-shell';
import { Link } from '@/i18n/navigation';
import { accountGet, loadForVisitor } from '@/lib/account-api';
import { pickText } from '@/lib/localized';

type DealList = components['schemas']['DealList'];

export default async function SavedPage({ params }: { params: Promise<{ locale: string }> }) {
  const { locale } = await params;
  const t = await getTranslations('Account');
  const list = await loadForVisitor(locale, `/${locale}/account/saved`, () =>
    accountGet<DealList>('/me/saved-deals'),
  );

  return (
    <AccountShell section="saved">
      {list.items.length === 0 ? (
        <p className="text-ink-muted">{t('emptySaved')}</p>
      ) : (
        <ul className="grid gap-3 sm:grid-cols-2">
          {list.items.map((deal) => (
            <li key={deal.id}>
              <Link href={`/deal/${deal.slug}`} className="block rounded-card border border-border bg-surface p-4 hover:border-brand">
                <span className="font-semibold text-ink">{pickText(deal.title, locale)}</span>
                <span className="block text-sm text-ink-muted">{deal.merchant.display_name}</span>
              </Link>
            </li>
          ))}
        </ul>
      )}
    </AccountShell>
  );
}
