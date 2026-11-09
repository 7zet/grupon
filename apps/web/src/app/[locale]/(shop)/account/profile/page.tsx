import type { components } from '@grupon/api-client';
import { getTranslations } from 'next-intl/server';

import { AccountShell } from '@/components/account-shell';
import { accountGet, loadForVisitor } from '@/lib/account-api';
import { maskPhone } from '@/lib/phone';

type User = components['schemas']['User'];

export default async function ProfilePage({ params }: { params: Promise<{ locale: string }> }) {
  const { locale } = await params;
  const t = await getTranslations('Account');
  const user = await loadForVisitor(locale, `/${locale}/account/profile`, () =>
    accountGet<User>('/me'),
  );

  const rows: [string, string | null | undefined][] = [
    [t('name'), user.name],
    [t('email'), user.email],
    [t('phone'), user.phone ? maskPhone(user.phone) : t('notSet')],
    [t('language'), user.locale],
    [t('currency'), user.currency],
  ];

  return (
    <AccountShell section="profile">
      <dl className="grid gap-3 rounded-card border border-border bg-surface p-5">
        {rows.map(([label, value]) => (
          <div key={label} className="flex justify-between gap-4">
            <dt className="text-ink-muted">{label}</dt>
            <dd className="text-ink">{value}</dd>
          </div>
        ))}
      </dl>
    </AccountShell>
  );
}
