'use client';

import { useTranslations } from 'next-intl';
import { useRouter } from '@/i18n/navigation';
import { useState } from 'react';

/** Closes a review report without a change, or hides the reviewed text from public pages. */
export function ReportActions({ reportId, reviewId }: { reportId: string; reviewId: string }) {
  const t = useTranslations('Reports');
  const router = useRouter();
  const [message, setMessage] = useState<string | null>(null);

  async function act(path: string) {
    setMessage(null);
    const response = await fetch(`/api/bff/${path}`, { method: 'POST' });
    if (!response.ok) {
      setMessage(t('errorGeneric'));
      return;
    }
    router.refresh();
  }

  return (
    <div className="grid gap-2">
      <div className="flex gap-3">
        <button
          type="button"
          onClick={() => act(`admin/review-reports/${reportId}/dismiss`)}
          className="rounded-control-ui border border-border px-4 py-2 text-sm"
        >
          {t('dismiss')}
        </button>
        <button
          type="button"
          onClick={() => act(`admin/reviews/${reviewId}/hide`)}
          className="rounded-control-ui bg-ink px-4 py-2 text-sm font-semibold text-surface"
        >
          {t('hide')}
        </button>
      </div>
      {message && (
        <p role="alert" className="text-sm text-error">
          {message}
        </p>
      )}
    </div>
  );
}
