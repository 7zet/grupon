'use client';

import { useTranslations } from 'next-intl';
import { useRouter } from '@/i18n/navigation';
import { useState } from 'react';

const inputClass = 'rounded-control-ui border border-border bg-surface px-4 py-3 text-ink';
const PHONE = /^\+998\d{9}$/;

type AssignableRole = 'merchant_manager' | 'merchant_cashier';

/** Invite a manager or cashier by phone, change their role, or remove them. Only the owner sees these controls. */
export function MerchantStaffInvite() {
  const t = useTranslations('MerchantPanel');
  const router = useRouter();
  const [phone, setPhone] = useState('+998');
  const [role, setRole] = useState<AssignableRole>('merchant_cashier');
  const [message, setMessage] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);

  async function invite(event: React.FormEvent) {
    event.preventDefault();
    setMessage(null);
    if (!PHONE.test(phone)) {
      setMessage(t('errorGeneric'));
      return;
    }
    setBusy(true);
    const response = await fetch('/api/bff/merchant/staff', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ phone, role }),
    });
    setBusy(false);
    if (!response.ok) {
      setMessage(t('errorGeneric'));
      return;
    }
    setPhone('+998');
    router.refresh();
  }

  return (
    <form onSubmit={invite} className="grid gap-4 rounded-card border border-border p-5 sm:grid-cols-[1fr_1fr_auto] sm:items-end">
      <div className="sm:col-span-3">
        <h2 className="font-semibold">{t('staffInviteTitle')}</h2>
        <p className="text-sm text-ink-muted">{t('staffInviteHint')}</p>
      </div>
      <label className="grid gap-2 text-sm font-semibold">
        {t('staffPhone')}
        <input className={inputClass} value={phone} onChange={(e) => setPhone(e.target.value)} inputMode="tel" />
      </label>
      <label className="grid gap-2 text-sm font-semibold">
        {t('staffRole')}
        <select className={inputClass} value={role} onChange={(e) => setRole(e.target.value as AssignableRole)}>
          <option value="merchant_cashier">{t('roleCashier')}</option>
          <option value="merchant_manager">{t('roleManager')}</option>
        </select>
      </label>
      <button type="submit" disabled={busy} className="rounded-control-ui bg-brand px-5 py-3 font-semibold text-surface">
        {t('staffInvite')}
      </button>
      {message && <p role="alert" className="text-sm text-danger sm:col-span-3">{message}</p>}
    </form>
  );
}

/** Change a staff member's role or remove the assignment. The phone is sent again because the contract expects both fields. */
export function MerchantStaffRow({
  assignmentId,
  phone,
  role,
}: {
  assignmentId: string;
  phone: string;
  role: AssignableRole;
}) {
  const t = useTranslations('MerchantPanel');
  const router = useRouter();
  const [current, setCurrent] = useState<AssignableRole>(role);
  const [busy, setBusy] = useState(false);
  const [message, setMessage] = useState<string | null>(null);

  async function changeRole(next: AssignableRole) {
    setMessage(null);
    setBusy(true);
    const response = await fetch(`/api/bff/merchant/staff/${assignmentId}`, {
      method: 'PATCH',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ phone, role: next }),
    });
    setBusy(false);
    if (!response.ok) {
      setMessage(t('errorGeneric'));
      return;
    }
    setCurrent(next);
    router.refresh();
  }

  async function remove() {
    setMessage(null);
    setBusy(true);
    const response = await fetch(`/api/bff/merchant/staff/${assignmentId}`, { method: 'DELETE' });
    setBusy(false);
    if (!response.ok) {
      setMessage(t('errorGeneric'));
      return;
    }
    router.refresh();
  }

  return (
    <div className="flex flex-wrap items-center gap-3">
      <select
        aria-label={t('staffRole')}
        className={inputClass}
        disabled={busy}
        value={current}
        onChange={(e) => changeRole(e.target.value as AssignableRole)}
      >
        <option value="merchant_cashier">{t('roleCashier')}</option>
        <option value="merchant_manager">{t('roleManager')}</option>
      </select>
      <button
        type="button"
        disabled={busy}
        onClick={remove}
        className="rounded-control-ui border border-border px-4 py-2 font-semibold"
      >
        {t('staffRemove')}
      </button>
      {message && <span role="alert" className="text-sm text-danger">{message}</span>}
    </div>
  );
}
