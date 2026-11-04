'use client';

import { useTranslations } from 'next-intl';
import { useState } from 'react';

const inputClass = 'rounded-control-ui border border-border bg-surface px-3 py-2 text-ink';

/** Flags a review for moderation. Signed-out visitors are sent to log in by the API response. */
export function ReviewReportButton({ reviewId }: { reviewId: string }) {
  const t = useTranslations('Reviews');
  const [open, setOpen] = useState(false);
  const [reason, setReason] = useState('');
  const [status, setStatus] = useState<'idle' | 'sending' | 'done' | 'error'>('idle');

  async function send() {
    if (reason.trim().length < 10) {
      setStatus('error');
      return;
    }
    setStatus('sending');
    const response = await fetch(`/api/bff/reviews/${reviewId}/report`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ reason: reason.trim() }),
    });
    setStatus(response.ok ? 'done' : 'error');
  }

  if (status === 'done')
    return (
      <p role="status" className="mt-3 text-label text-ink-muted">
        {t('reportDone')}
      </p>
    );
  if (!open) {
    return (
      <button
        className="mt-3 text-label text-ink-muted"
        type="button"
        onClick={() => setOpen(true)}
      >
        {t('report')}
      </button>
    );
  }
  return (
    <div className="mt-3 grid gap-2">
      <label className="grid gap-1 text-sm">
        {t('reportReason')}
        <textarea
          className={inputClass}
          value={reason}
          onChange={(event) => setReason(event.target.value)}
        />
      </label>
      {status === 'error' && (
        <p role="alert" className="text-sm text-error">
          {t('reportError')}
        </p>
      )}
      <div className="flex gap-3">
        <button
          type="button"
          onClick={send}
          disabled={status === 'sending'}
          className="rounded-control-ui bg-ink px-4 py-2 text-sm font-semibold text-surface"
        >
          {t('reportSend')}
        </button>
        <button type="button" onClick={() => setOpen(false)} className="text-sm text-ink-muted">
          {t('cancel')}
        </button>
      </div>
    </div>
  );
}
