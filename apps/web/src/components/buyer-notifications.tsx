'use client';

import type { components } from '@grupon/api-client';
import { useTranslations } from 'next-intl';
import { useRouter } from '@/i18n/navigation';
import { useState } from 'react';

type Notification = components['schemas']['Notification'];

/** Redemptions the buyer may not recognise. Reporting one opens a dispute, which holds the merchant's payout. */
export function BuyerNotifications({ items }: { items: Notification[] }) {
  const t = useTranslations('Notifications');
  const router = useRouter();
  const [message, setMessage] = useState<string | null>(null);
  const [busy, setBusy] = useState<string | null>(null);

  if (items.length === 0) return null;

  async function report(voucherId: string) {
    setMessage(null);
    setBusy(voucherId);
    const response = await fetch(`/api/bff/me/vouchers/${voucherId}/dispute`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ reason: 'not_me' }),
    });
    setBusy(null);
    if (!response.ok) {
      setMessage(t('errorGeneric'));
      return;
    }
    setMessage(t('reported'));
    router.refresh();
  }

  return (
    <section aria-labelledby="notifications-title" className="grid gap-3 rounded-card border border-border bg-surface p-4">
      <h2 id="notifications-title" className="text-lead font-semibold">
        {t('title')}
      </h2>
      <ul className="grid gap-3">
        {items.map((item) => (
          <li key={item.id} className="flex flex-wrap items-center justify-between gap-3 text-small">
            <span>
              {t('redeemed', { merchant: item.merchant_name })}
              <span className="ml-2 text-ink-muted">{new Date(item.created_at).toLocaleString()}</span>
            </span>
            {item.voucher_id && (
              <button
                type="button"
                disabled={busy !== null}
                onClick={() => report(item.voucher_id as string)}
                className="rounded-control-ui border border-border px-3 py-2 font-semibold"
              >
                {t('notMe')}
              </button>
            )}
          </li>
        ))}
      </ul>
      {message && (
        <p role="status" className="text-small text-ink">
          {message}
        </p>
      )}
    </section>
  );
}
