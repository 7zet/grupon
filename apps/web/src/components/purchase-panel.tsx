'use client';
import type { components } from '@grupon/api-client';
import { localized, type Currency, type Locale } from '@grupon/config';
import { discountPercent, formatMoney } from '@grupon/ui';
import { Heart, Minus, Plus, ShoppingCart } from 'lucide-react';
import { useTranslations } from 'next-intl';
import { useState } from 'react';
import { useRouter } from '@/i18n/navigation';
import { useCart } from '@/stores/cart';
type Deal = components['schemas']['Deal'];

export function PurchasePanel({
  deal,
  locale,
  currency,
}: {
  deal: Deal;
  locale: Locale;
  currency: Currency;
}) {
  const t = useTranslations('Deal');
  const common = useTranslations('Common');
  const router = useRouter();
  const add = useCart((state) => state.add);
  const [selected, setSelected] = useState(deal.options[0]!.id);
  const [quantity, setQuantity] = useState(1);
  const [gift, setGift] = useState(false);
  const [notice, setNotice] = useState(false);
  const option = deal.options.find((item) => item.id === selected)!;
  const remaining = option.stock_total - option.stock_sold - option.stock_reserved;
  const unavailable = deal.status !== 'active' || remaining < 1;
  const savings = (option.original_price.amount - option.deal_price.amount) * quantity;
  const cartLine = () => ({
    optionId: option.id,
    quantity,
    isGift: gift,
    title: localized(option.title, locale),
    dealSlug: deal.slug,
    kind: deal.kind,
    unitPrice: option.deal_price.amount,
    originalPrice: option.original_price.amount,
  });
  const buy = () => {
    add(cartLine());
    router.push('/checkout');
  };

  return (
    <aside className="rounded-panel-ui border bg-surface p-3 shadow-floating-ui sm:p-6">
      <fieldset>
        <legend className="mb-2 text-small font-semibold">{t('selectOption')}</legend>
        <div className="grid gap-1 sm:gap-2">
          {deal.options.map((entry) => (
            <label
              className={`flex cursor-pointer justify-between rounded-control-ui border p-2 sm:p-3 ${selected === entry.id ? 'border-brand bg-brand-tint' : ''}`}
              key={entry.id}
            >
              <span>
                <input
                  className="mr-2"
                  type="radio"
                  checked={selected === entry.id}
                  onChange={() => setSelected(entry.id)}
                />
                {localized(entry.title, locale)}
              </span>
              <strong className="tabular-nums">
                {formatMoney(entry.deal_price.amount, locale, currency)}
              </strong>
            </label>
          ))}
        </div>
      </fieldset>
      <div className="mt-2 flex items-end justify-between gap-4 sm:mt-4">
        <div>
          <div className="tabular-nums text-subtitle font-bold text-ink sm:text-title-small">
            {formatMoney(option.deal_price.amount * quantity, locale, currency)}
          </div>
          <div className="tabular-nums text-small text-ink-muted line-through">
            {formatMoney(option.original_price.amount * quantity, locale, currency)}
          </div>
        </div>
        <span className="rounded-round-ui bg-deal-tint px-3 py-1 text-small font-semibold text-deal-hover">
          -{discountPercent(option.original_price.amount, option.deal_price.amount)}%
        </span>
      </div>
      <div className="mt-1 rounded-control-ui bg-save-tint px-3 py-1.5 text-small font-semibold text-save sm:mt-2 sm:py-2">
        {t('saveAmount', { amount: formatMoney(savings, locale, currency) })}
      </div>
      {remaining <= 10 && remaining > 0 && (
        <p className="mt-1 text-small font-semibold text-deal">
          {t('onlyLeft', { count: remaining })}
        </p>
      )}
      <button
        data-buy-now
        className="mt-2 h-11 w-full rounded-control-ui bg-deal text-small font-semibold text-surface hover:bg-deal-hover disabled:bg-ink-faint sm:mt-4 sm:h-12"
        disabled={unavailable}
        onClick={buy}
        type="button"
      >
        {unavailable ? (deal.status === 'expired' ? t('ended') : t('soldOut')) : t('buyNow')}
      </button>
      <div className="mt-3 flex items-center justify-between">
        <span className="text-small font-semibold">{t('quantity')}</span>
        <div className="flex items-center rounded-control-ui border">
          <button
            aria-label="-"
            className="p-2"
            disabled={quantity <= 1}
            onClick={() => setQuantity((value) => value - 1)}
            type="button"
          >
            <Minus className="size-4" />
          </button>
          <span className="min-w-8 text-center tabular-nums">{quantity}</span>
          <button
            aria-label="+"
            className="p-2"
            disabled={quantity >= Math.min(deal.max_per_user, remaining)}
            onClick={() => setQuantity((value) => value + 1)}
            type="button"
          >
            <Plus className="size-4" />
          </button>
        </div>
      </div>
      {deal.is_giftable && (
        <label className="mt-3 flex items-center gap-2 text-small">
          <input
            type="checkbox"
            checked={gift}
            onChange={(event) => setGift(event.target.checked)}
          />
          {t('gift')}
        </label>
      )}
      <div className="mt-2 grid grid-cols-[1fr_auto] gap-2">
        <button
          className="h-11 rounded-control-ui border border-brand bg-brand-tint px-3 text-small font-semibold text-brand"
          disabled={unavailable}
          onClick={() => {
            add(cartLine());
            setNotice(true);
          }}
          type="button"
        >
          <ShoppingCart className="mr-2 inline size-4" />
          {t('addToCart')}
        </button>
        <button className="rounded-control-ui border p-3" aria-label={common('save')} type="button">
          <Heart className="size-5" />
        </button>
      </div>
      {notice && (
        <p className="mt-2 text-small text-save" role="status">
          {t('addSuccess')}
        </p>
      )}
    </aside>
  );
}
