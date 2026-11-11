import { brandConfig } from '@dealo/config';
import { getTranslations } from 'next-intl/server';
import type { ReactNode } from 'react';

import { workspacesFor } from '@/lib/roles';

import { type WorkspaceLink, WorkspaceMenu } from './workspace-menu';

/** "Dealo Kassa": a full-screen cashier workspace. Only the redeem screen lives here. */
export async function KassaShell({
  children,
  roles,
  name,
  businessName,
  branch,
}: {
  children: ReactNode;
  roles: string[];
  name: string;
  businessName: string;
  branch: string;
}) {
  const w = await getTranslations('Workspace');
  const identity = await getTranslations('Identity');
  const navigation = await getTranslations('Navigation');
  const switchTo: WorkspaceLink[] = workspacesFor(roles)
    .filter((key) => key === 'shop')
    .map(() => ({ href: '/tashkent', label: w('viewMarketplace') }));

  return (
    <div className="min-h-dvh bg-page">
      <header className="sticky top-0 z-40 border-b bg-surface">
        <div className="mx-auto flex min-h-16 max-w-3xl items-center gap-3 px-4 py-2 sm:px-6">
          <div className="min-w-0 flex-1">
            <p className="text-label font-semibold text-brand">{w('kassa', { brand: brandConfig.name })}</p>
            <p className="truncate text-small font-semibold">
              {businessName}
              {branch ? <span className="font-normal text-ink-muted"> · {branch}</span> : null}
            </p>
            <p className="truncate text-label text-ink-muted">
              {identity('cashier')} · {name}
            </p>
          </div>
          <WorkspaceMenu
            name={name}
            roleLabel={identity('cashier')}
            switchTo={switchTo}
            labels={{
              account: navigation('account'),
              signOut: navigation('logout'),
              switchWorkspace: w('switchWorkspace'),
              viewMarketplace: w('viewMarketplace'),
            }}
          />
        </div>
      </header>
      <main className="mx-auto max-w-3xl px-4 py-6 sm:px-6">{children}</main>
    </div>
  );
}
