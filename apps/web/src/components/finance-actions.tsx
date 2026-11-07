'use client';

import { useTranslations } from 'next-intl';
import { useRouter } from '@/i18n/navigation';
import { useState } from 'react';

const inputClass =
  'rounded-control-ui border border-border bg-surface px-3 py-2 text-ink text-small';

type Kind = 'payout' | 'refund' | 'dispute';

/** Problem codes from the API and the translation key for each. */
const ERROR_KEYS = {
  self_approval: 'error.self_approval',
  not_pending: 'error.not_pending',
  forbidden: 'error.forbidden',
  invalid_dispute_transition: 'error.invalid_dispute_transition',
} as const;

/** One decision on a finance item. Each action is a separate API call that the server checks again. */
export function FinanceActions({
  kind,
  id,
  status,
  canSupport = false,
  canResolve = false,
}: {
  kind: Kind;
  id: string;
  status: string;
  canSupport?: boolean;
  canResolve?: boolean;
}) {
  const t = useTranslations('AdminFinance');
  const router = useRouter();
  const [docRef, setDocRef] = useState('');
  const [message, setMessage] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);

  async function act(path: string, body?: unknown) {
    setMessage(null);
    setBusy(true);
    const response = await fetch(`/api/bff/admin/${path}`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: body === undefined ? null : JSON.stringify(body),
    });
    setBusy(false);
    if (!response.ok) {
      const problem = (await response.json().catch(() => ({}))) as { code?: string };
      setMessage(t(ERROR_KEYS[problem.code as keyof typeof ERROR_KEYS] ?? 'error.generic'));
      return;
    }
    router.refresh();
  }

  const button =
    'rounded-control-ui border border-border px-3 py-2 text-small font-semibold disabled:opacity-60';

  if (kind === 'payout') {
    return (
      <div className="grid gap-2">
        <div className="flex flex-wrap items-center gap-2">
          {status === 'pending_approval' && (
            <>
              <input
                aria-label={t('bankDocRef')}
                placeholder={t('bankDocRef')}
                className={inputClass}
                value={docRef}
                onChange={(event) => setDocRef(event.target.value)}
              />
              <button
                type="button"
                disabled={busy}
                className={button}
                onClick={() => act(`payouts/${id}/approve`, { bank_doc_ref: docRef })}
              >
                {t('approve')}
              </button>
            </>
          )}
          {status === 'sent' && (
            <button
              type="button"
              disabled={busy}
              className={button}
              onClick={() => act(`payouts/${id}/reconcile`)}
            >
              {t('reconcile')}
            </button>
          )}
        </div>
        {message && (
          <p role="alert" className="text-small text-error">
            {message}
          </p>
        )}
      </div>
    );
  }

  if (kind === 'refund') {
    return (
      <div className="grid gap-2">
        <div className="flex flex-wrap gap-2">
          {status === 'requested' && canSupport && (
            <button
              type="button"
              disabled={busy}
              className={button}
              onClick={() => act(`refunds/${id}/propose`)}
            >
              {t('propose')}
            </button>
          )}
          {status === 'proposed' && canResolve && (
            <button type="button" disabled={busy} className={button} onClick={() => act(`refunds/${id}/approve`)}>
              {t('approve')}
            </button>
          )}
          {status === 'approved' && (
            <button
              type="button"
              disabled={busy}
              className={button}
              onClick={() => act(`refunds/${id}/paid`)}
            >
              {t('markPaid')}
            </button>
          )}
        </div>
        {message && (
          <p role="alert" className="text-small text-error">
            {message}
          </p>
        )}
      </div>
    );
  }

  return (
    <div className="grid gap-2">
      <div className="flex flex-wrap gap-2">
        {status === 'open' && canSupport && (
          <button
            type="button"
            disabled={busy}
            className={button}
            onClick={() => act(`disputes/${id}/request-evidence`)}
          >
            {t('requestEvidence')}
          </button>
        )}
        {status === 'awaiting_evidence' && canSupport && (
          <button
            type="button"
            disabled={busy}
            className={button}
            onClick={() => act(`disputes/${id}/start-review`)}
          >
            {t('startReview')}
          </button>
        )}
        {status === 'under_review' && canResolve && (
          <>
            <button
              type="button"
              disabled={busy}
              className={button}
              onClick={() => act(`disputes/${id}/resolve`, { resolution: 'customer' })}
            >
              {t('resolveCustomer')}
            </button>
            <button
              type="button"
              disabled={busy}
              className={button}
              onClick={() => act(`disputes/${id}/resolve`, { resolution: 'merchant' })}
            >
              {t('resolveMerchant')}
            </button>
          </>
        )}
        {(status === 'resolved_customer' || status === 'resolved_merchant') && canSupport && (
          <button
            type="button"
            disabled={busy}
            className={button}
            onClick={() => act(`disputes/${id}/close`)}
          >
            {t('close')}
          </button>
        )}
      </div>
      {message && (
        <p role="alert" className="text-small text-error">
          {message}
        </p>
      )}
    </div>
  );
}
