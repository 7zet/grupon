import { adminGate } from '@/components/workspace/gate';
import type { components } from '@dealo/api-client';
import { formatMoney } from '@dealo/ui';
import { getTranslations } from 'next-intl/server';
import { notFound } from 'next/navigation';

import { FinanceActions } from '@/components/finance-actions';
import { accountGet, accountGetOrNull, loadForVisitor } from '@/lib/account-api';

type User = components['schemas']['User'];
type Queue = components['schemas']['FinanceQueue'];

const FINANCE_ROLES = ['superadmin', 'finance', 'support'];

/** Translation keys for each status, written out so the message types stay exact. */
const STATUS_KEYS = {
  pending_approval: 'status.pending_approval',
  approved: 'status.approved',
  sent: 'status.sent',
  requested: 'status.requested',
  proposed: 'status.proposed',
  processing: 'status.processing',
  open: 'status.open',
  awaiting_evidence: 'status.awaiting_evidence',
  under_review: 'status.under_review',
  resolved_customer: 'status.resolved_customer',
  resolved_merchant: 'status.resolved_merchant',
} as const;

/** Finance desk: payouts to approve or reconcile, refunds to approve or mark paid, disputes to decide. */
export default async function FinancePage({ params }: { params: Promise<{ locale: string }> }) {
  const denied = await adminGate('finance');
  if (denied) return denied;
  const { locale } = await params;
  const t = await getTranslations('AdminFinance');
  const user = await loadForVisitor(locale, `/${locale}/admin/finance`, () =>
    accountGetOrNull<User>('/me'),
  );
  if (!user?.roles.some((role) => FINANCE_ROLES.includes(role))) notFound();
  const queue = await accountGet<Queue>('/admin/finance/queue');
  const canSupport = user.roles.includes('support') || user.roles.includes('superadmin');
  const canResolve = user.roles.includes('finance') || user.roles.includes('superadmin');
  // Support sees the refund and dispute queue only; payouts are for finance.
  const payoutsVisible = user.roles.some((role) => ['finance', 'superadmin', 'admin'].includes(role));
  const money = (amount: number) =>
    formatMoney(amount, locale as Parameters<typeof formatMoney>[1], 'UZS');
  const empty = queue.payout_runs.length + queue.refunds.length + queue.disputes.length === 0;

  return (
    <div className="grid grid-cols-[minmax(0,1fr)] gap-8">
      <h1 className="text-title font-bold">{t('title')}</h1>
      {empty && <p className="text-ink-muted">{t('empty')}</p>}

      {payoutsVisible && queue.payout_runs.length > 0 && (
        <section className="grid gap-3">
          <h2 className="text-lead font-semibold">{t('payouts')}</h2>
          <ul className="grid gap-3">
            {queue.payout_runs.map((item) => (
              <li
                key={item.id}
                className="grid gap-2 rounded-card border border-border bg-surface p-4"
              >
                <p className="text-small">
                  {item.merchant_name} · {money(item.amount)} ·{' '}
                  {t(STATUS_KEYS[item.status as keyof typeof STATUS_KEYS] ?? 'status.requested')}
                </p>
                <FinanceActions kind="payout" id={item.id} status={item.status} />
              </li>
            ))}
          </ul>
        </section>
      )}

      {queue.refunds.length > 0 && (
        <section className="grid gap-3">
          <h2 className="text-lead font-semibold">{t('refunds')}</h2>
          <ul className="grid gap-3">
            {queue.refunds.map((item) => (
              <li
                key={item.id}
                className="grid gap-2 rounded-card border border-border bg-surface p-4"
              >
                <p className="text-small">
                  {item.voucher_code} · {money(item.amount)} ·{' '}
                  {t(STATUS_KEYS[item.status as keyof typeof STATUS_KEYS] ?? 'status.requested')}
                </p>
                <p className="text-small text-ink-muted">{item.reason}</p>
                <FinanceActions
                  kind="refund"
                  id={item.id}
                  status={item.status}
                  canSupport={canSupport}
                  canResolve={canResolve}
                />
              </li>
            ))}
          </ul>
        </section>
      )}

      {queue.disputes.length > 0 && (
        <section className="grid gap-3">
          <h2 className="text-lead font-semibold">{t('disputes')}</h2>
          <ul className="grid gap-3">
            {queue.disputes.map((item) => (
              <li
                key={item.id}
                className="grid gap-2 rounded-card border border-border bg-surface p-4"
              >
                <p className="text-small">
                  {item.voucher_code} · {item.merchant_name} · {money(item.amount)} ·{' '}
                  {t(STATUS_KEYS[item.status as keyof typeof STATUS_KEYS] ?? 'status.open')}
                </p>
                <p className="text-small text-ink-muted">{item.reason}</p>
                <FinanceActions
                  kind="dispute"
                  id={item.id}
                  status={item.status}
                  canSupport={canSupport}
                  canResolve={canResolve}
                />
              </li>
            ))}
          </ul>
        </section>
      )}
    </div>
  );
}
