'use client';

import type { DealSummary } from '@dealo/api-client';
import { localized, type Currency, type Locale } from '@dealo/config';
import { discountPercent, formatMoney } from '@dealo/ui';
import { Clock3, Heart, Star } from 'lucide-react';
import { motion, useReducedMotion } from 'motion/react';
import { useState } from 'react';
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
  const [saved, setSaved] = useState(false);
  const option = deal.options[0];
  const image = deal.media[0];
  if (!option || !image) return null;
  const days = Math.ceil((new Date(deal.ends_at).getTime() - cardRenderTime) / 86400000);
  return (
    <motion.article
      {...(!reduced ? { whileHover: { y: -4 } } : {})}
      transition={{ duration: 0.2, ease: [0.22, 1, 0.36, 1] }}
      className="group relative h-full overflow-hidden rounded-card-ui border bg-surface shadow-card transition-shadow duration-200 hover:shadow-lift"
    >
      <Link className="flex h-full flex-col" href={`/deal/${deal.slug}`} prefetch={false}>
        <div className="relative aspect-[4/3] overflow-hidden bg-brand-tint">
          <Image
            className="object-cover transition-transform duration-300 ease-[cubic-bezier(0.22,1,0.36,1)] group-hover:scale-[1.04] motion-reduce:transition-none"
            src={image.url}
            alt={localized(image.alt, locale)}
            fill
            sizes="(max-width: 640px) 50vw, (max-width: 1024px) 33vw, 25vw"
          />
          <motion.span
            initial={reduced ? false : { scale: 0.6, opacity: 0 }}
            animate={{ scale: 1, opacity: 1 }}
            transition={{ duration: 0.35, ease: [0.22, 1, 0.36, 1] }}
            className="absolute left-3 top-3 rounded-control-ui bg-deal px-2 py-1 text-small font-semibold tabular-nums text-surface"
          >
            -{discountPercent(option.original_price.amount, option.deal_price.amount)}%
          </motion.span>
        </div>
        <div className="flex flex-1 flex-col p-4">
          <p className="line-clamp-2 min-h-12 font-semibold leading-6 text-ink">
            {localized(deal.title, locale)}
          </p>
          {deal.area && (
            <p className="mt-1 line-clamp-1 text-label text-ink-muted" title={deal.area}>
              {deal.area}
            </p>
          )}
          <p className="mt-2 line-clamp-1 text-small text-ink-muted">
            {deal.merchant.display_name} ·{' '}
            {localized(deal.category.name, locale)}
            {deal.rating_count > 0 && (
              <>
                {' · '}
                <Star className="inline size-4 fill-rating text-rating" aria-hidden />{' '}
                {deal.rating_avg.toFixed(1)}
              </>
            )}
          </p>
          <div className="mt-4 whitespace-nowrap font-display tabular-nums text-body font-extrabold text-ink sm:text-subtitle">
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
      <motion.button
        className={`absolute right-3 top-3 min-h-11 min-w-11 rounded-round-ui bg-surface p-2 shadow-floating-ui ${saved ? 'text-deal' : 'text-ink-muted hover:text-deal'}`}
        aria-label={t('save')}
        aria-pressed={saved}
        type="button"
        onClick={() => setSaved((value) => !value)}
        {...(reduced ? {} : { whileTap: { scale: 0.85 }, animate: { scale: saved ? [1, 1.25, 1] : 1 } })}
        transition={{ duration: 0.3, ease: [0.22, 1, 0.36, 1] }}
      >
        <Heart className={`size-5 ${saved ? 'fill-current' : ''}`} />
      </motion.button>
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
