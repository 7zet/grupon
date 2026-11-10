import { getTranslations } from 'next-intl/server';
import type { ReactNode } from 'react';

import { AdminShell } from '@/components/workspace/admin-shell';
import { NoAccess } from '@/components/workspace/no-access';
import { currentUser, loadForVisitor } from '@/lib/account-api';
import { isStaff } from '@/lib/roles';

/** Platform console ("Dealo Admin") for moderator, support, finance and superadmin. */
export default async function AdminWorkspaceLayout({
  children,
  params,
}: {
  children: ReactNode;
  params: Promise<{ locale: string }>;
}) {
  const { locale } = await params;
  const user = await loadForVisitor(locale, `/${locale}/admin`, () => currentUser());
  if (!user) return null;
  if (!isStaff(user.roles)) return <NoAccess roles={user.roles} />;

  const identity = await getTranslations('Identity');
  const role = user.roles.find((item) => item !== 'user' && !item.startsWith('merchant_')) ?? 'user';
  return (
    <AdminShell roles={user.roles} name={user.name} roleLabel={identity(role as 'admin')}>
      {children}
    </AdminShell>
  );
}
