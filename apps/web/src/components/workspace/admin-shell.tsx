import { brandConfig } from '@grupon/config';
import { getTranslations } from 'next-intl/server';
import type { ReactNode } from 'react';

import { type AdminSection, adminSectionsFor, workspacesFor } from '@/lib/roles';

import { type NavItem, NavLinks } from './nav-links';
import { type WorkspaceLink, WorkspaceMenu } from './workspace-menu';

const HREF: Record<AdminSection, string> = {
  overview: '/admin',
  moderation: '/admin/moderation',
  merchants: '/admin/merchants',
  fraud: '/admin/fraud',
  reports: '/admin/reports',
  finance: '/admin/finance',
  audit: '/admin/audit',
};

/** "Grupon Admin": the platform console. Each staff role sees only its own sections. */
export async function AdminShell({
  children,
  roles,
  name,
  roleLabel,
}: {
  children: ReactNode;
  roles: string[];
  name: string;
  roleLabel: string;
}) {
  const w = await getTranslations('Workspace');
  const nav = await getTranslations('AdminNav');
  const navigation = await getTranslations('Navigation');
  const sections: NavItem[] = adminSectionsFor(roles).map((section) => ({ href: HREF[section], label: nav(section) }));
  const switchTo: WorkspaceLink[] = workspacesFor(roles)
    .filter((key) => key !== 'admin')
    .map((key) => ({
      href: key === 'shop' ? '/tashkent' : key === 'business' ? '/merchant/dashboard' : '/merchant/redeem',
      label: key === 'shop' ? w('viewMarketplace') : key === 'business' ? w('business', { brand: brandConfig.name }) : w('kassa', { brand: brandConfig.name }),
    }));

  return (
    <div className="min-h-dvh bg-page">
      <header className="sticky top-0 z-40 border-b bg-surface">
        <div className="mx-auto flex h-16 max-w-6xl items-center gap-3 px-4 sm:px-6">
          <span className="shrink-0 text-subtitle font-bold text-brand">{w('admin', { brand: brandConfig.name })}</span>
          <span data-testid="panel-role-badge" className="hidden truncate rounded-full bg-brand-tint px-2 py-1 text-label font-semibold text-brand md:inline">
            {roleLabel}
          </span>
          <div className="ml-auto">
            <WorkspaceMenu
              name={name}
              roleLabel={roleLabel}
              switchTo={switchTo}
              labels={{
                account: navigation('account'),
                signOut: navigation('logout'),
                switchWorkspace: w('switchWorkspace'),
                viewMarketplace: w('viewMarketplace'),
              }}
            />
          </div>
        </div>
      </header>
      <div className="mx-auto grid max-w-6xl gap-6 px-4 py-6 sm:px-6 lg:grid-cols-[14rem_minmax(0,1fr)]">
        <aside className="hidden lg:block">
          <NavLinks items={sections} label={w('workspace')} variant="sidebar" />
        </aside>
        <details className="rounded-card-ui border bg-surface p-3 lg:hidden">
          <summary className="flex min-h-11 cursor-pointer list-none items-center font-semibold">{w('menu')}</summary>
          <div className="mt-2">
            <NavLinks items={sections} label={w('workspace')} variant="sidebar" />
          </div>
        </details>
        <main className="min-w-0">{children}</main>
      </div>
    </div>
  );
}
