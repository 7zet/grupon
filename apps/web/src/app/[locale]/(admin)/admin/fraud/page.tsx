import { adminGate } from '@/components/workspace/gate';
import type { components } from '@dealo/api-client';
import { getFormatter, getTranslations } from 'next-intl/server';
import { notFound } from 'next/navigation';

import { accountGet, accountGetOrNull, loadForVisitor } from '@/lib/account-api';

type User = components['schemas']['User'];
type SignalPage = components['schemas']['FraudSignalPage'];

const READERS = ['superadmin', 'admin'];

/** Fraud signals raised by the rules, newest first. Admins and superadmins only. */
export default async function FraudSignalsPage({ params }: { params: Promise<{ locale: string }> }) {
  const denied = await adminGate('fraud');
  if (denied) return denied;
  const { locale } = await params;
  const t = await getTranslations('AdminFraud');
  const format = await getFormatter();
  const user = await loadForVisitor(locale, `/${locale}/admin/fraud`, () => accountGetOrNull<User>('/me'));
  if (!user?.roles.some((role) => READERS.includes(role))) notFound();
  const signals = await accountGet<SignalPage>('/admin/fraud-signals');

  return (
    <div className="grid grid-cols-[minmax(0,1fr)] gap-8">
      <h1 className="text-title font-bold">{t('title')}</h1>
      {signals.items.length === 0 ? (
        <p className="text-ink-muted">{t('empty')}</p>
      ) : (
        <div className="overflow-x-auto">
          <table className="w-full min-w-[640px] text-left text-sm">
            <thead className="text-ink-muted">
              <tr>
                <th className="py-2 pr-4 font-semibold">{t('rule')}</th>
                <th className="py-2 pr-4 font-semibold">{t('merchant')}</th>
                <th className="py-2 pr-4 font-semibold">{t('status')}</th>
                <th className="py-2 pr-4 font-semibold">{t('detail')}</th>
                <th className="py-2 pr-4 font-semibold">{t('date')}</th>
              </tr>
            </thead>
            <tbody>
              {signals.items.map((signal) => (
                <tr key={signal.id} className="border-t border-border align-top">
                  <td className="py-2 pr-4">
                    {signal.rule === 'redemption_burst' ? t('rules.redemption_burst') : signal.rule}
                  </td>
                  <td className="py-2 pr-4">{signal.merchant_name ?? '—'}</td>
                  <td className="py-2 pr-4">{t(signal.status)}</td>
                  <td className="py-2 pr-4 text-ink-muted">
                    {Object.entries(signal.detail)
                      .map(([key, value]) => `${key}: ${String(value)}`)
                      .join(', ')}
                  </td>
                  <td className="py-2 pr-4">
                    {format.dateTime(new Date(signal.created_at), { dateStyle: 'medium', timeStyle: 'short' })}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  );
}
