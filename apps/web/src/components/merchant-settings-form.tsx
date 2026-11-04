'use client';

import { useTranslations } from 'next-intl';
import { useRouter } from '@/i18n/navigation';
import { useState } from 'react';

type LocationOption = { id: string; label: string };

/** Notification switches and the default branch. Saved as one PATCH to the merchant settings. */
export function MerchantSettingsForm({
  initial,
  locations,
}: {
  initial: { order_emails: boolean; review_emails: boolean; payout_emails: boolean; default_location_id: string | null };
  locations: LocationOption[];
}) {
  const t = useTranslations('MerchantPanel');
  const router = useRouter();
  const [values, setValues] = useState(initial);
  const [message, setMessage] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);

  async function save(event: React.FormEvent) {
    event.preventDefault();
    setMessage(null);
    setBusy(true);
    const response = await fetch('/api/bff/merchant/settings', {
      method: 'PATCH',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(values),
    });
    setBusy(false);
    setMessage(response.ok ? t('settingsSaved') : t('errorGeneric'));
    if (response.ok) router.refresh();
  }

  const switches = [
    { key: 'order_emails', label: t('orderEmails') },
    { key: 'review_emails', label: t('reviewEmails') },
    { key: 'payout_emails', label: t('payoutEmails') },
  ] as const;

  return (
    <form onSubmit={save} className="grid max-w-xl gap-5">
      {switches.map(({ key, label }) => (
        <label key={key} className="flex items-center gap-3 text-sm">
          <input
            type="checkbox"
            className="h-5 w-5 accent-[var(--color-brand)]"
            checked={values[key]}
            onChange={(e) => setValues({ ...values, [key]: e.target.checked })}
          />
          {label}
        </label>
      ))}
      <label className="grid gap-2 text-sm font-semibold">
        {t('defaultLocation')}
        <select
          className="rounded-control-ui border border-border bg-surface px-4 py-3 text-ink"
          value={values.default_location_id ?? ''}
          onChange={(e) => setValues({ ...values, default_location_id: e.target.value || null })}
        >
          <option value="">{t('noLocation')}</option>
          {locations.map((location) => (
            <option key={location.id} value={location.id}>
              {location.label}
            </option>
          ))}
        </select>
      </label>
      <div className="flex flex-wrap items-center gap-3">
        <button type="submit" disabled={busy} className="rounded-control-ui bg-brand px-5 py-3 font-semibold text-surface">
          {t('saveSettings')}
        </button>
        {message && <span role="status" className="text-sm text-ink-muted">{message}</span>}
      </div>
    </form>
  );
}
