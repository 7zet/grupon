'use client';

import { useTranslations } from 'next-intl';
import { useRouter } from '@/i18n/navigation';
import { useState } from 'react';

type DocumentKind = 'certificate' | 'bank_letter' | 'price_proof';
type Uploaded = Record<DocumentKind, boolean>;

const MAX_BYTES = 10 * 1024 * 1024;
const ACCEPTED = ['application/pdf', 'image/jpeg', 'image/png'];
const REQUIRED: DocumentKind[] = ['certificate', 'bank_letter'];
const OPTIONAL: DocumentKind[] = ['price_proof'];

/** Uploads the two required documents and sends the business to review. Shown only while it is a draft. */
export function MerchantReviewForm({ uploaded }: { uploaded: Uploaded }) {
  const t = useTranslations('MerchantReview');
  const router = useRouter();
  const [done, setDone] = useState(uploaded);
  const [message, setMessage] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);

  async function upload(kind: DocumentKind, file: File | undefined) {
    setMessage(null);
    if (!file) return;
    if (!ACCEPTED.includes(file.type)) return setMessage(t('errorType'));
    if (file.size > MAX_BYTES) return setMessage(t('errorSize'));
    setBusy(true);
    const response = await fetch(`/api/bff/merchant/documents/${kind}`, {
      method: 'PUT',
      headers: { 'Content-Type': file.type },
      body: file,
    });
    setBusy(false);
    if (!response.ok) return setMessage(t('errorGeneric'));
    setDone((current) => ({ ...current, [kind]: true }));
    router.refresh();
  }

  async function submit() {
    setMessage(null);
    setBusy(true);
    const response = await fetch('/api/bff/merchant/submit', { method: 'POST' });
    setBusy(false);
    if (!response.ok) {
      const problem = (await response.json().catch(() => ({}))) as { code?: string };
      return setMessage(t(problem.code === 'documents_required' ? 'errorMissing' : 'errorGeneric'));
    }
    router.refresh();
  }

  const complete = REQUIRED.every((kind) => done[kind]);
  return (
    <div className="grid max-w-xl gap-6">
      {[...REQUIRED, ...OPTIONAL].map((kind) => (
        <label key={kind} className="grid gap-2">
          <span className="font-semibold">
            {t(`kind.${kind}`)}
            {OPTIONAL.includes(kind) && <span className="font-normal text-ink-muted"> · {t('optional')}</span>}
          </span>
          <span className="text-small text-ink-muted">{done[kind] ? t('uploaded') : t('notUploaded')}</span>
          <input
            type="file"
            accept="application/pdf,image/jpeg,image/png"
            disabled={busy}
            onChange={(event) => upload(kind, event.target.files?.[0])}
            className="text-small"
          />
        </label>
      ))}
      {message && (
        <p role="alert" className="rounded-control-ui bg-error-tint p-3 text-error">
          {message}
        </p>
      )}
      <button
        type="button"
        disabled={!complete || busy}
        onClick={submit}
        className="rounded-control-ui bg-brand px-5 py-3 font-semibold text-surface disabled:opacity-60"
      >
        {t('submit')}
      </button>
    </div>
  );
}
