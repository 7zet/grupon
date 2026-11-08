'use client';

import type { DealSummary } from '@grupon/api-client';
import { localized, type Currency, type Locale } from '@grupon/config';
import { discountPercent, formatMoney } from '@grupon/ui';
import { Clock3, Heart, Star } from 'lucide-react';
import { motion, useReducedMotion } from 'motion/react';
import Image from 'next/image';
import { useTranslations } from 'next-intl';
import { Link } from '@/i18n/navigation';

const cardRenderTime = Date.now();

export function DealCard({
  deal,
  locale,
  currency,
}: {
  deal: DealSummary;
  locale: Locale;
  currency: Currency;
}) {
  const t = useTranslations('Common');
  const reduced = useReducedMotion();
  const option = deal.options[0];
  const image = deal.media[0];
  if (!option || !image) return null;
  const days = Math.ceil((new Date(deal.ends_at).getTime() - cardRenderTime) / 86400000);
  return (
    <motion.article
      {...(!reduced ? { whileHover: { y: -2 } } : {})}
      transition={{ duration: 0.18 }}
      className="group relative h-full overflow-hidden rounded-card-ui border bg-surface hover:shadow-floating-ui"
    >
      <Link className="flex h-full flex-col" href={`/deal/${deal.slug}`}>
        <div className="relative aspect-[4/3] overflow-hidden bg-brand-tint">
          <Image
            className="object-cover transition duration-300 group-hover:scale-[1.03] motion-reduce:transition-none"
            src={image.url}
            alt={localized(image.alt, locale)}
            fill
            sizes="(max-width: 640px) 50vw, (max-width: 1024px) 33vw, 25vw"
          />
          <span className="absolute left-3 top-3 rounded-control-ui bg-deal px-2 py-1 text-small font-semibold text-surface">
            -{discountPercent(option.original_price.amount, option.deal_price.amount)}%
          </span>
        </div>
        <div className="flex flex-1 flex-col p-4">
          <p className="line-clamp-2 min-h-12 font-semibold leading-6 text-ink">
            {localized(deal.title, locale)}
          </p>
          <p className="mt-2 line-clamp-1 text-small text-ink-muted">
            {deal.merchant.display_name} ·{' '}
            {localized(deal.merchant.locations?.[0]?.city.name ?? deal.category.name, locale)}
            {deal.rating_count > 0 && (
              <>
                {' · '}
                <Star className="inline size-4 fill-rating text-rating" aria-hidden />{' '}
                {deal.rating_avg.toFixed(1)}
              </>
            )}
          </p>
          <div className="mt-4 tabular-nums text-subtitle font-bold text-ink">
            {formatMoney(option.deal_price.amount, locale, currency)}
          </div>
          <div className="tabular-nums text-small text-ink-muted line-through">
            {formatMoney(option.original_price.amount, locale, currency)}
          </div>
          <div className="mt-3 min-h-5 text-label text-ink-muted">
            {days > 0 && days <= 3 && (
              <span>
                <Clock3 className="mr-1 inline size-3" />
                {t('daysLeft', { count: days })}
              </span>
            )}
            {days > 0 && days <= 3 && deal.sold_count >= 50 && ' · '}
            {deal.sold_count >= 50 && <span>{t('sold', { count: deal.sold_count })}</span>}
          </div>
        </div>
      </Link>
      <button
        className="absolute right-3 top-3 rounded-round-ui bg-surface p-2 text-ink-muted shadow-floating-ui hover:text-deal"
        aria-label={t('save')}
        type="button"
      >
        <Heart className="size-5" />
      </button>
    </motion.article>
  );
}

export function DealCardSkeleton() {
  return (
    <div className="animate-pulse overflow-hidden rounded-card-ui border bg-surface">
      <div className="aspect-[4/3] bg-brand-tint" />
      <div className="space-y-3 p-4">
        <div className="h-5 rounded-control-ui bg-brand-tint" />
        <div className="h-4 w-3/4 rounded-control-ui bg-page" />
        <div className="h-7 w-1/2 rounded-control-ui bg-brand-tint" />
      </div>
    </div>
  );
}
