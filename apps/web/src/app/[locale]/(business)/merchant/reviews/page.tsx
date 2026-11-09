import { businessGate } from '@/components/workspace/gate';
import type { components } from '@grupon/api-client';
import { getFormatter, getTranslations } from 'next-intl/server';

import { MerchantReviewReply } from '@/components/merchant-review-reply';
import { accountGet, loadForVisitor } from '@/lib/account-api';

type ReviewList = components['schemas']['ReviewList'];

const REPLY_WINDOW_MS = 24 * 60 * 60 * 1000;
const reviewsRenderTime = Date.now();

/** Reviews of this business. One reply per review; the API allows edits for 24 hours after the reply is first written. */
export default async function ReviewsPage({ params }: { params: Promise<{ locale: string }> }) {
  const denied = await businessGate('reviews');
  if (denied) return denied;
  const { locale } = await params;
  const t = await getTranslations('MerchantPanel');
  const format = await getFormatter();
  const list = await loadForVisitor(locale, `/${locale}/merchant/reviews`, () =>
    accountGet<ReviewList>('/merchant/reviews'),
  );
  return (
    <div className="grid grid-cols-[minmax(0,1fr)] gap-8">
      <h1 className="text-title font-bold">{t('reviewsTitle')}</h1>
      {list.items.length === 0 ? (
        <p className="text-ink-muted">{t('reviewsEmpty')}</p>
      ) : (
        <ul className="grid gap-4">
          {list.items.map((review) => (
            <li key={review.id} className="grid gap-3 rounded-card border border-border p-5">
              <div className="flex flex-wrap items-center justify-between gap-2 text-sm">
                <span className="font-semibold">
                  {t('rating')}: {review.rating}/5
                </span>
                <span className="text-ink-muted">
                  {format.dateTime(new Date(review.created_at), { dateStyle: 'medium' })}
                </span>
              </div>
              <p>{review.text}</p>
              <MerchantReviewReply
                reviewId={review.id}
                initialText={review.reply?.text ?? null}
                canEdit={
                  review.reply
                    ? reviewsRenderTime - Date.parse(review.reply.created_at) <= REPLY_WINDOW_MS
                    : true
                }
              />
            </li>
          ))}
        </ul>
      )}
    </div>
  );
}
