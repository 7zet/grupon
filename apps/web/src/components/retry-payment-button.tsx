'use client';

import { useTranslations } from 'next-intl';
import { useState } from 'react';

import { useRouter } from '@/i18n/navigation';

/** Starts a fresh payment attempt for an order that is still awaiting payment. */
export function RetryPaymentButton({ orderId }: { orderId: string }) {
  const t = useTranslations('Checkout');
  const router = useRouter();
  const [busy, setBusy] = useState(false);

  async function retry() {
    setBusy(true);
    const response = await fetch(`/api/bff/orders/${orderId}/pay`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ provider: 'mock', terms_accepted: true }),
    });
    if (!response.ok) {
      setBusy(false);
      return;
    }
    const payment = (await response.json()) as { payment_id: string };
    router.push(`/checkout/mock/${payment.payment_id}?order=${orderId}`);
  }

  return (
    <button
      type="button"
      onClick={retry}
      disabled={busy}
      className="rounded-control-ui bg-brand px-5 py-3 font-semibold text-surface disabled:opacity-60"
    >
      {busy ? t('retrying') : t('retry')}
    </button>
  );
}
