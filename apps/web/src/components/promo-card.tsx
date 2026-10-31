'use client';
import type { components } from '@grupon/api-client';
import { localized, type Locale } from '@grupon/config';
import { CheckCircle2, Copy, ExternalLink } from 'lucide-react';
import Image from 'next/image';
import { useTranslations } from 'next-intl';
import { useState } from 'react';
type Promo = components['schemas']['PromoCode'];
export function PromoCard({ promo, locale }: { promo: Promo; locale: Locale }) {
  const t = useTranslations('Coupons');
  const common = useTranslations('Common');
  const [shown, setShown] = useState(false);
  const [copied, setCopied] = useState(false);
  return (
    <article className="rounded-card-ui border bg-surface p-5">
      <div className="flex items-center gap-3">
        {promo.brand_logo_url && (
          <Image
            src={promo.brand_logo_url}
            alt=""
            width={52}
            height={52}
            className="size-13 rounded-control-ui object-cover"
          />
        )}
        <div>
          <h2 className="font-semibold">{promo.brand_name}</h2>
          {promo.is_verified && (
            <span className="flex items-center gap-1 text-label text-brand">
              <CheckCircle2 className="size-3" />
              {common('verified')}
            </span>
          )}
        </div>
      </div>
      <h3 className="mt-5 text-lead font-bold">{localized(promo.title, locale)}</h3>
      <p className="mt-2 text-subtitle font-bold text-deal">
        {localized(promo.discount_label, locale)}
      </p>
      <p className="mt-3 text-label text-ink-muted">
        {t('expires', { date: new Intl.DateTimeFormat(locale).format(new Date(promo.ends_at)) })}
      </p>
      {promo.code ? (
        shown ? (
          <div className="mt-5 flex gap-2">
            <code className="flex-1 rounded-control-ui border border-dashed border-brand bg-brand-tint p-3 text-center font-bold text-brand">
              {promo.code}
            </code>
            <button
              className="rounded-control-ui border p-3"
              type="button"
              aria-label={t('copy')}
              onClick={() => {
                void navigator.clipboard.writeText(promo.code ?? '');
                setCopied(true);
              }}
            >
              <Copy className="size-5" />
            </button>
            {copied && (
              <span className="sr-only" role="status">
                {t('copied')}
              </span>
            )}
          </div>
        ) : (
          <button
            className="mt-5 w-full rounded-control-ui bg-brand px-4 py-3 font-semibold text-surface"
            type="button"
            onClick={() => setShown(true)}
          >
            {t('showCode')}
          </button>
        )
      ) : (
        <a
          className="mt-5 flex w-full items-center justify-center gap-2 rounded-control-ui bg-brand px-4 py-3 font-semibold text-surface"
          target="_blank"
          rel="noreferrer"
          href={promo.url}
        >
          {t('getDeal')}
          <ExternalLink className="size-4" />
        </a>
      )}
    </article>
  );
}
