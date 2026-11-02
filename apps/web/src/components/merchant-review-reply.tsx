'use client';

import { useTranslations } from 'next-intl';
import { useRouter } from '@/i18n/navigation';
import { useState } from 'react';

const inputClass = 'w-full rounded-control-ui border border-border bg-surface px-4 py-3 text-ink';

/** The merchant's one reply to a review. The API allows edits for 24 hours after the first reply. */
export function MerchantReviewReply({
  reviewId,
  initialText,
  canEdit,
}: {
  reviewId: string;
  initialText: string | null;
  canEdit: boolean;
}) {
  const t = useTranslations('MerchantPanel');
  const router = useRouter();
  const [text, setText] = useState(initialText ?? '');
  const [editing, setEditing] = useState(initialText === null);
  const [message, setMessage] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);

  async function save(event: React.FormEvent) {
    event.preventDefault();
    setMessage(null);
    setBusy(true);
    const response = await fetch(`/api/bff/merchant/reviews/${reviewId}/reply`, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ text: text.trim() }),
    });
    setBusy(false);
    if (!response.ok) {
      const problem = (await response.json().catch(() => ({}))) as { code?: string };
      setMessage(problem.code === 'reply_locked' ? t('replyLocked') : t('replyError'));
      return;
    }
    setMessage(t('replySaved'));
    setEditing(false);
    router.refresh();
  }

  if (!editing && initialText !== null) {
    return (
      <div className="grid gap-2 rounded-control-ui bg-surface-muted p-4">
        <p className="text-sm font-semibold">{t('yourReply')}</p>
        <p className="text-sm">{initialText}</p>
        {canEdit && (
          <button type="button" onClick={() => setEditing(true)} className="justify-self-start text-sm font-semibold text-brand">
            {t('editReply')}
          </button>
        )}
        {!canEdit && <p className="text-xs text-ink-muted">{t('replyLocked')}</p>}
      </div>
    );
  }

  return (
    <form onSubmit={save} className="grid gap-3">
      <textarea
        className={inputClass}
        rows={3}
        maxLength={2000}
        placeholder={t('replyPlaceholder')}
        value={text}
        onChange={(e) => setText(e.target.value)}
      />
      <div className="flex flex-wrap items-center gap-3">
        <button
          type="submit"
          disabled={busy || text.trim().length === 0}
          className="rounded-control-ui bg-brand px-5 py-3 font-semibold text-surface"
        >
          {t('sendReply')}
        </button>
        {message && <span role="status" className="text-sm text-ink-muted">{message}</span>}
      </div>
    </form>
  );
}
