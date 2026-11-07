'use client';

import { useTranslations } from 'next-intl';
import { useState } from 'react';

import { useRouter } from '@/i18n/navigation';

type Action = 'submit' | 'pause' | 'resume' | 'archive';

const ACTIONS_BY_STATUS: Record<string, Action[]> = {
  draft: ['submit', 'archive'],
  changes_requested: ['submit', 'archive'],
  active: ['pause'],
  paused: ['resume', 'archive'],
  ended: ['archive'],
};

const LABEL = {
  submit: 'submitForReview',
  pause: 'pause',
  resume: 'resume',
  archive: 'archive',
} as const;

/** Status actions for one owned deal: submit, pause, resume, archive. Only valid moves are shown. */
export function DealActions({ dealId, status }: { dealId: string; status: string }) {
  const t = useTranslations('Deals');
  const router = useRouter();
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const actions = ACTIONS_BY_STATUS[status] ?? [];
  if (actions.length === 0) return null;

  async function run(action: Action) {
    setBusy(true);
    setError(null);
    const response = await fetch(`/api/bff/merchant/deals/${dealId}/${action}`, { method: 'POST' });
    setBusy(false);
    if (!response.ok) {
      const problem = (await response.json().catch(() => ({}))) as { errors?: Record<string, string[]> };
      setError(t(problem.errors?.price_proof ? 'errorPriceProof' : 'actionError'));
      return;
    }
    router.refresh();
  }

  return (
    <div className="grid gap-2">
      <div className="flex flex-wrap gap-2">
        {actions.map((action) => (
          <button
            key={action}
            type="button"
            disabled={busy}
            onClick={() => run(action)}
            className="rounded-control-ui border border-border px-3 py-2 text-small font-semibold"
          >
            {t(LABEL[action])}
          </button>
        ))}
      </div>
      {error && (
        <p role="alert" className="text-small text-error">
          {error}
        </p>
      )}
    </div>
  );
}
