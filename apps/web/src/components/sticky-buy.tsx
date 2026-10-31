'use client';
import { formatMoney } from '@grupon/ui';
import type { Currency, Locale } from '@grupon/config';
import { useTranslations } from 'next-intl';
import { useEffect, useState } from 'react';
export function StickyBuy({
  price,
  locale,
  currency,
}: {
  price: number;
  locale: Locale;
  currency: Currency;
}) {
  const [show, setShow] = useState(false);
  const t = useTranslations('Deal');
  useEffect(() => {
    const target = document.querySelector('#purchase-panel [data-buy-now]');
    if (!target) return;
    const observer = new IntersectionObserver(
      ([entry]) => setShow((entry?.intersectionRatio ?? 0) < 1),
      { rootMargin: '0px 0px -64px 0px', threshold: 1 },
    );
    observer.observe(target);
    return () => observer.disconnect();
  }, []);
  if (!show) return null;
  return (
    <div className="fixed inset-x-0 bottom-16 z-30 flex items-center justify-between gap-4 border-t bg-surface p-3 shadow-floating-ui lg:hidden">
      <strong className="tabular-nums text-lead">{formatMoney(price, locale, currency)}</strong>
      <button
        data-testid="sticky-buy"
        className="rounded-control-ui bg-deal px-6 py-3 text-small font-semibold text-surface"
        onClick={() =>
          (document.querySelector('#purchase-panel [data-buy-now]') as HTMLButtonElement)?.click()
        }
        type="button"
      >
        {t('buyNow')}
      </button>
    </div>
  );
}
