'use client';

import type { components } from '@grupon/api-client';
import { useTranslations } from 'next-intl';
import { useState, useSyncExternalStore } from 'react';

import { useRouter } from '@/i18n/navigation';
import { pickText } from '@/lib/localized';

type Redemption = components['schemas']['MerchantRedemption'];

/** The server allows a cashier to undo their own redemption for 10 minutes; the screen mirrors that. */
export const UNDO_WINDOW_MS = 10 * 60 * 1000;

// One shared clock for the screen. It changes only on each 15 s tick, so the snapshot React reads is stable
// between ticks (a fresh Date.now() on every read would make React re-render forever).
let clock = 0;
const clockListeners = new Set<() => void>();
let clockTimer: number | undefined;

function subscribeClock(onChange: () => void) {
  clockListeners.add(onChange);
  if (clockListeners.size === 1) {
    clock = Date.now();
    clockTimer = window.setInterval(() => {
      clock = Date.now();
      clockListeners.forEach((listener) => listener());
    }, 15_000);
  }
  return () => {
    clockListeners.delete(onChange);
    if (clockListeners.size === 0) window.clearInterval(clockTimer);
  };
}

function readClock() {
  return clock || null;
}

/** Today's redemptions made by this cashier, each with a cancel button inside the 10-minute window. */
export function KassaHistory({ rows, locale }: { rows: Redemption[]; locale: string }) {
  const t = useTranslations('Redeem');
  const w = useTranslations('Workspace');
  const router = useRouter();
  const [message, setMessage] = useState<string | null>(null);
  const [busy, setBusy] = useState<string | null>(null);

  // The clock ticks every 15 s. The server renders nothing here, so the first client render matches it.
  const now = useSyncExternalStore(subscribeClock, readClock, () => null);

  if (now === null) return null;
  const today = new Date(now).toDateString();
  const todays = rows.filter((row) => new Date(row.time).toDateString() === today);

  async function cancel(row: Redemption) {
    setBusy(row.id);
    setMessage(null);
    const response = await fetch(`/api/bff/merchant/vouchers/${row.voucher_id}/reverse`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ reason: w('cancelReason') }),
    });
    setBusy(null);
    if (!response.ok) {
      setMessage(t('errorGeneric'));
      return;
    }
    router.refresh();
  }

  return (
    <section aria-labelledby="kassa-today" className="grid gap-3">
      <h2 id="kassa-today" className="text-lead font-semibold">
        {w('todayRedemptions')}
      </h2>
      {message && (
        <p role="alert" className="rounded-control-ui bg-error-tint p-3 text-error">
          {message}
        </p>
      )}
      {todays.length === 0 ? (
        <p className="text-ink-muted">{t('historyEmpty')}</p>
      ) : (
        <ul className="grid gap-3">
          {todays.map((row) => {
            const age = now - new Date(row.time).getTime();
            const cancellable = !row.reversed && age < UNDO_WINDOW_MS;
            return (
              <li key={row.id} data-redemption-id={row.id} className="grid gap-2 rounded-card border border-border bg-surface p-4">
                <p className="font-semibold">{pickText(row.deal, locale)}</p>
                <p className="text-small text-ink-muted">
                  {pickText(row.option, locale)} · {new Date(row.time).toLocaleTimeString(locale, { hour: '2-digit', minute: '2-digit' })}
                </p>
                <p className="text-small">
                  {w('buyer')}: <span className="font-semibold">{row.buyer.charAt(0).toUpperCase()}***</span>
                  {row.reversed ? <span className="ml-2 text-error">{t('history.reversed')}</span> : null}
                </p>
                {cancellable && (
                  <button
                    type="button"
                    disabled={busy === row.id}
                    onClick={() => cancel(row)}
                    className="min-h-11 rounded-control-ui border border-border px-4 py-2 font-semibold text-ink disabled:opacity-60"
                  >
                    {w('cancelRedemption')}
                  </button>
                )}
                {cancellable && <p className="text-label text-ink-muted">{w('cancelWindow')}</p>}
              </li>
            );
          })}
        </ul>
      )}
    </section>
  );
}
