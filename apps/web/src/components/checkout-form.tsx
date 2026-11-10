'use client';

import { zodResolver } from '@hookform/resolvers/zod';
import { formatMoney } from '@dealo/ui';
import { useLocale, useTranslations } from 'next-intl';
import { useState, useSyncExternalStore } from 'react';
import { useForm } from 'react-hook-form';
import { z } from 'zod';

import { Link, useRouter } from '@/i18n/navigation';
import { useCheckout } from '@/stores/checkout';

type Values = { terms: boolean };

const termsSchema = z.object({
  terms: z.boolean().refine((accepted) => accepted),
});

/** Maps stable API problem codes to checkout messages. */
function problemKey(code: string | undefined) {
  switch (code) {
    case 'option_unavailable':
      return 'errorUnavailable' as const;
    case 'insufficient_stock':
      return 'errorStock' as const;
    case 'max_per_user_exceeded':
      return 'errorLimit' as const;
    case 'unauthenticated':
      return 'errorSession' as const;
    default:
      return 'errorGeneric' as const;
  }
}

const subscribeNothing = () => () => {};

export function CheckoutForm({ demoMode }: { demoMode: boolean }) {
  const t = useTranslations('Checkout');
  const locale = useLocale();
  const router = useRouter();
  const item = useCheckout((state) => state.item);
  const [error, setError] = useState<string | null>(null);
  const [idempotencyKey] = useState(() => crypto.randomUUID());

  // False during server render and first hydration pass; true once the persisted cart is readable.
  const ready = useSyncExternalStore(
    subscribeNothing,
    () => true,
    () => false,
  );

  const form = useForm<Values>({
    resolver: zodResolver(termsSchema),
    defaultValues: { terms: false },
  });

  if (!ready) return <p className="text-ink-muted">…</p>;
  if (!item) {
    return (
      <div className="grid gap-4">
        <p className="text-ink-muted">{t('emptyCart')}</p>
        <Link href="/" className="font-semibold text-brand">
          {t('backToDeals')}
        </Link>
      </div>
    );
  }

  // Capture the narrowed state for submit callbacks, where TypeScript cannot
  // retain the render-time null check.
  const checkoutItem = item;

  const money = (amount: number) =>
    formatMoney(amount, locale as Parameters<typeof formatMoney>[1], 'UZS');
  const total = item.unitPrice * item.quantity;
  const savings = (item.originalPrice - item.unitPrice) * item.quantity;

  async function submit() {
    setError(null);
    const orderBody = {
      items: [{ deal_option_id: checkoutItem.optionId, quantity: checkoutItem.quantity }],
    };

    const orderResponse = await fetch('/api/bff/orders', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json', 'Idempotency-Key': idempotencyKey },
      body: JSON.stringify(orderBody),
    });
    if (!orderResponse.ok) {
      const problem = (await orderResponse.json().catch(() => ({}))) as { code?: string };
      setError(t(problemKey(problem.code)));
      if (problem.code === 'unauthenticated') router.replace('/login?next=/checkout');
      return;
    }
    const order = (await orderResponse.json()) as { id: string };

    const payResponse = await fetch(`/api/bff/orders/${order.id}/pay`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ provider: 'mock', terms_accepted: true }),
    });
    if (!payResponse.ok) {
      setError(t('errorGeneric'));
      return;
    }
    const payment = (await payResponse.json()) as { payment_id: string };
    router.push(`/checkout/mock/${payment.payment_id}?order=${order.id}`);
  }

  return (
    <div className="grid gap-8 lg:grid-cols-[1fr_20rem]">
      <section aria-labelledby="summary-heading" className="grid gap-4">
        <h2 id="summary-heading" className="text-lead font-semibold">
          {t('summary')}
        </h2>
        <ul className="grid gap-3">
          <li className="rounded-card border border-border bg-surface p-4">
            <span className="block font-semibold text-ink">{item.title}</span>
            <span className="text-sm text-ink-muted">
              {t('quantity')}: {item.quantity}
            </span>
            <span className="block font-semibold text-ink tabular-nums">
              {money(item.unitPrice * item.quantity)}
            </span>
          </li>
        </ul>
        {savings > 0 && (
          <p className="rounded-control-ui bg-save-tint px-4 py-2 font-semibold text-save">
            {t('savings', { amount: money(savings) })}
          </p>
        )}
      </section>

      <form className="grid content-start gap-5" noValidate onSubmit={form.handleSubmit(submit)}>
        <div className="text-lead font-bold tabular-nums">
          {t('total')}: {money(total)}
        </div>
        <p className="text-sm text-ink-muted">{t('paymentNote')}</p>

        <label className="flex items-start gap-2 text-sm">
          <input type="checkbox" className="mt-1" {...form.register('terms')} />
          <span>{t('terms')}</span>
        </label>

        {error && (
          <p
            role="alert"
            className="flex items-start gap-2 rounded-control-ui bg-error-tint p-3 text-error"
          >
            <span aria-hidden="true">!</span>
            <span>{error}</span>
          </p>
        )}

        {demoMode && <button
          type="submit"
          disabled={form.formState.isSubmitting}
          className="rounded-control-ui bg-deal px-5 py-3 font-semibold text-surface disabled:opacity-60"
        >
          {form.formState.isSubmitting ? t('placing') : t('placeOrder')}
        </button>}
      </form>
    </div>
  );
}
