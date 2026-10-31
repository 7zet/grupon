'use client';

import { useTranslations } from 'next-intl';
import { useRouter } from '@/i18n/navigation';
import { useState } from 'react';

const inputClass = 'rounded-control-ui border border-border bg-surface px-4 py-3 text-ink';

/** Approve or reject one deal waiting for review. Rejection needs a reason the merchant can read. */
export function ModerationActions({ dealId }: { dealId: string }) {
  const t = useTranslations('Moderation');
  const router = useRouter();
  const [rejecting, setRejecting] = useState(false);
  const [reason, setReason] = useState('');
  const [message, setMessage] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);

  async function decide(action: 'approve' | 'reject') {
    setMessage(null);
    if (action === 'reject' && reason.trim().length < 3) {
      setMessage(t('reasonError'));
      return;
    }
    setBusy(true);
    const response = await fetch(`/api/bff/admin/deals/${dealId}/${action}`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: action === 'reject' ? JSON.stringify({ reason: reason.trim() }) : null,
    });
    setBusy(false);
    if (!response.ok) {
      setMessage(t('errorGeneric'));
      return;
    }
    router.refresh();
  }

  return (
    <div className="grid gap-3">
      {rejecting ? (
        <div className="grid max-w-xl gap-2">
          <label className="grid gap-1 text-sm">
            {t('reasonLabel')}
            <textarea
              className={inputClass}
              value={reason}
              onChange={(event) => setReason(event.target.value)}
            />
          </label>
          <div className="flex gap-3">
            <button
              type="button"
              disabled={busy}
              onClick={() => decide('reject')}
              className="rounded-control-ui bg-ink px-5 py-3 font-semibold text-surface"
            >
              {t('sendReject')}
            </button>
            <button
              type="button"
              onClick={() => setRejecting(false)}
              className="rounded-control-ui border border-border px-5 py-3"
            >
              {t('cancel')}
            </button>
          </div>
        </div>
      ) : (
        <div className="flex gap-3">
          <button
            type="button"
            disabled={busy}
            onClick={() => decide('approve')}
            className="rounded-control-ui bg-brand px-5 py-3 font-semibold text-surface"
          >
            {t('approve')}
          </button>
          <button
            type="button"
            disabled={busy}
            onClick={() => setRejecting(true)}
            className="rounded-control-ui border border-border px-5 py-3"
          >
            {t('reject')}
          </button>
        </div>
      )}
      {message && (
        <p
          role="alert"
          className="flex items-start gap-2 rounded-control-ui bg-error-tint p-3 text-error"
        >
          <span aria-hidden="true">!</span>
          <span>{message}</span>
        </p>
      )}
    </div>
  );
}
