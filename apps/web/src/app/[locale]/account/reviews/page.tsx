import type { components } from '@grupon/api-client';
import { getFormatter, getTranslations } from 'next-intl/server';

import { AccountShell } from '@/components/account-shell';
import { accountGet, loadForVisitor } from '@/lib/account-api';

type ReviewList = components['schemas']['ReviewList'];

export default async function ReviewsPage({ params }: { params: Promise<{ locale: string }> }) {
  const { locale } = await params;
  const t = await getTranslations('Account');
  const format = await getFormatter();
  const list = await loadForVisitor(locale, `/${locale}/account/reviews`, () =>
    accountGet<ReviewList>('/me/reviews'),
  );

  return (
    <AccountShell section="reviews">
      {list.items.length === 0 ? (
        <p className="text-ink-muted">{t('emptyReviews')}</p>
      ) : (
        <ul className="grid gap-3">
          {list.items.map((review) => (
            <li key={review.id} className="rounded-card border border-border bg-surface p-4">
              <span className="font-semibold text-ink" aria-label={`${review.rating}/5`}>
                {'★'.repeat(review.rating)}
              </span>
              <p className="mt-2 text-ink">{review.text}</p>
              <span className="text-sm text-ink-muted">
                {format.dateTime(new Date(review.created_at), { dateStyle: 'medium' })}
              </span>
            </li>
          ))}
        </ul>
      )}
    </AccountShell>
  );
}
