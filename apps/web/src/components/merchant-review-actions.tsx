'use client';

import { useTranslations } from 'next-intl';
import { useRouter } from '@/i18n/navigation';
import { useState } from 'react';

const inputClass = 'rounded-control-ui border border-border bg-surface px-4 py-3 text-ink';

/** Start review, approve, or reject one business. A rejection needs a reason the merchant can read. */
export function MerchantReviewActions({ merchantId, reviewStatus }: { merchantId: string; reviewStatus: string }) {
  const t = useTranslations('AdminMerchants');
  const router = useRouter();
  const [rejecting, setRejecting] = useState(false);
  const [reason, setReason] = useState('');
  const [message, setMessage] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);

  async function act(action: 'start-review' | 'approve' | 'reject') {
    setMessage(null);
    if (action === 'reject' && reason.trim().length < 3) {
      setMessage(t('reasonError'));
      return;
    }
    setBusy(true);
    const response = await fetch(`/api/bff/admin/merchants/${merchantId}/${action}`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: action === 'reject' ? JSON.stringify({ reason: reason.trim() }) : null,
    });
    setBusy(false);
    if (!response.ok) {
      const problem = (await response.json().catch(() => ({}))) as { code?: string };
      setMessage(t(problem.code === 'invalid_transition' ? 'errorTransition' : 'errorGeneric'));
      return;
    }
    router.refresh();
  }

  const canStart = reviewStatus === 'submitted';
  const canDecide = reviewStatus === 'in_review';
  return (
    <div className="grid gap-3">
      <div className="flex flex-wrap gap-3">
        {canStart && (
          <button
            type="button"
            disabled={busy}
            onClick={() => act('start-review')}
            className="rounded-control-ui border border-border px-5 py-3 font-semibold"
          >
            {t('startReview')}
          </button>
        )}
        {canDecide && (
          <button
            type="button"
            disabled={busy}
            onClick={() => act('approve')}
            className="rounded-control-ui bg-brand px-5 py-3 font-semibold text-surface"
          >
            {t('approve')}
          </button>
        )}
        {(canStart || canDecide) && (
          <button
            type="button"
            disabled={busy}
            onClick={() => setRejecting((open) => !open)}
            className="rounded-control-ui border border-border px-5 py-3 font-semibold"
          >
            {t('reject')}
          </button>
        )}
      </div>
      {rejecting && (
        <div className="grid max-w-xl gap-2">
          <label className="grid gap-1 text-sm">
            {t('reasonLabel')}
            <textarea className={inputClass} value={reason} onChange={(event) => setReason(event.target.value)} />
          </label>
          <button
            type="button"
            disabled={busy}
            onClick={() => act('reject')}
            className="w-fit rounded-control-ui bg-ink px-5 py-3 font-semibold text-surface"
          >
            {t('sendReject')}
          </button>
        </div>
      )}
      {message && (
        <p role="alert" className="rounded-control-ui bg-error-tint p-3 text-error">
          {message}
        </p>
      )}
    </div>
  );
}
