import { adminGate } from '@/components/workspace/gate';
import type { components } from '@grupon/api-client';
import { formatMoney } from '@grupon/ui';
import { getTranslations } from 'next-intl/server';
import { notFound } from 'next/navigation';

import { ModerationActions } from '@/components/moderation-actions';
import { accountGet, accountGetOrNull, loadForVisitor } from '@/lib/account-api';
import { pickText } from '@/lib/localized';

type DealPage = components['schemas']['DealPage'];
type User = components['schemas']['User'];

const REVIEWERS = ['superadmin', 'admin', 'moderator'];

export default async function ModerationPage({ params }: { params: Promise<{ locale: string }> }) {
  const denied = await adminGate('moderation');
  if (denied) return denied;
  const { locale } = await params;
  const t = await getTranslations('Moderation');
  const user = await loadForVisitor(locale, `/${locale}/admin/moderation`, () =>
    accountGetOrNull<User>('/me'),
  );
  if (!user?.roles.some((role) => REVIEWERS.includes(role))) notFound();
  const queue = await accountGet<DealPage>('/admin/moderation/deals');
  const money = (amount: number) =>
    formatMoney(amount, locale as Parameters<typeof formatMoney>[1], 'UZS');

  return (
    <div className="grid grid-cols-[minmax(0,1fr)] gap-8">
      <h1 className="text-title font-bold">{t('title')}</h1>
      {queue.items.length === 0 ? (
        <p className="text-ink-muted">{t('empty')}</p>
      ) : (
        <ul className="grid gap-4">
          {queue.items.map((deal) => (
            <li
              key={deal.id}
              className="grid gap-3 rounded-card border border-border bg-surface p-4"
            >
              <div className="grid gap-1">
                <p className="font-semibold">{pickText(deal.title, locale)}</p>
                <p className="text-sm text-ink-muted">{deal.merchant.display_name}</p>
                <p className="text-sm">
                  {deal.options.map((option) => (
                    <span key={option.id} className="mr-3">
                      {pickText(option.title, locale)}: {money(option.deal_price.amount)}
                    </span>
                  ))}
                </p>
              </div>
              {deal.options.some(
                (option) =>
                  option.original_price.amount > 0 &&
                  (1 - option.deal_price.amount / option.original_price.amount) * 100 > 70,
              ) && (
                <p role="note" className="rounded-control-ui bg-error-tint p-3 text-small text-error">
                  {t('highDiscount')}
                </p>
              )}
              <ModerationActions dealId={deal.id} />
            </li>
          ))}
        </ul>
      )}
    </div>
  );
}
