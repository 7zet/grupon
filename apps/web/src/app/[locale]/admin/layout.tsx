import type { components } from '@grupon/api-client';
import { getTranslations } from 'next-intl/server';
import type { ReactNode } from 'react';

import { accountGetOrNull, loadForVisitor } from '@/lib/account-api';

type User = components['schemas']['User'];

export default async function AdminLayout({ children, params }: { children: ReactNode; params: Promise<{ locale: string }> }) {
  const { locale } = await params;
  const user = await loadForVisitor(locale, `/${locale}/admin/moderation`, () => accountGetOrNull<User>('/me'));
  const t = await getTranslations('Identity');
  const role = user?.roles.find((item) => item !== 'user');
  const label = role ? t(role as 'admin' | 'moderator' | 'support' | 'finance' | 'superadmin') : t('buyer');
  return <div className="mx-auto max-w-6xl px-4 py-8 sm:px-6"><aside className="mb-6 rounded-card-ui border border-border bg-surface p-4"><p className="font-semibold">{user?.name}</p><p data-testid="panel-role-badge" className="mt-1 inline-flex rounded-full bg-brand-tint px-2 py-1 text-label font-semibold text-brand">{t('admin')} · {label}</p></aside>{children}</div>;
}
