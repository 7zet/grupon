'use client';

import { useTranslations } from 'next-intl';
import { useState } from 'react';

import { useRouter } from '@/i18n/navigation';

/** Simulates the outcome a real provider would report through its webhook. */
export function MockPaymentActions({ paymentId, orderId }: { paymentId: string; orderId: string }) {
  const t = useTranslations('Checkout');
  const router = useRouter();
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState(false);

  async function complete(outcome: 'succeeded' | 'failed') {
    setBusy(true);
    setError(false);
    try {
      const response = await fetch(`/api/bff/payments/mock/${paymentId}/complete`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ outcome }),
      });
      const problem = response.ok
        ? null
        : ((await response.json().catch(() => ({}))) as { code?: string });
      if (problem?.code === 'unauthenticated') {
        router.replace('/login?next=/account/orders');
        return;
      }
      // "Not pending" means the outcome was already applied (for example a retried click).
      if (problem && problem.code !== 'payment_not_pending') {
        setError(true);
        setBusy(false);
        return;
      }
    } catch {
      setError(true);
      setBusy(false);
      return;
    }
    // Without the order id (a reloaded or shared link) the order list shows the result.
    if (!orderId) {
      router.push('/account/orders');
      return;
    }
    router.push(`/checkout/${outcome === 'succeeded' ? 'success' : 'failed'}/${orderId}`);
  }

  return (
    <div className="grid gap-3">
      <p className="text-sm text-ink-muted">{t('mockHint')}</p>
      {error ? (
        <p role="alert" className="text-sm text-danger">
          {t('errorGeneric')}
        </p>
      ) : null}
      <button
        type="button"
        disabled={busy}
        onClick={() => complete('succeeded')}
        className="rounded-control-ui bg-brand px-5 py-3 font-semibold text-surface disabled:opacity-60"
      >
        {busy ? t('mockBusy') : t('mockSuccess')}
      </button>
      <button
        type="button"
        disabled={busy}
        onClick={() => complete('failed')}
        className="rounded-control-ui border border-border px-5 py-3 font-semibold text-ink disabled:opacity-60"
      >
        {t('mockFailure')}
      </button>
    </div>
  );
}
