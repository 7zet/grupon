'use client';

import { useTranslations } from 'next-intl';
import { useState } from 'react';

import { useRouter } from '@/i18n/navigation';

/** Simulates the outcome a real provider would report through its webhook. */
export function MockPaymentActions({ paymentId, orderId }: { paymentId: string; orderId: string }) {
  const t = useTranslations('Checkout');
  const router = useRouter();
  const [busy, setBusy] = useState(false);

  async function complete(outcome: 'succeeded' | 'failed') {
    setBusy(true);
    const response = await fetch(`/api/bff/payments/mock/${paymentId}/complete`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ outcome }),
    });
    if (!response.ok) {
      setBusy(false);
      return;
    }
    router.push(`/checkout/${outcome === 'succeeded' ? 'success' : 'failed'}/${orderId}`);
  }

  return (
    <div className="grid gap-3">
      <p className="text-sm text-ink-muted">{t('mockHint')}</p>
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
