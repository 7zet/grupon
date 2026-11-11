'use client';

import { UserRound } from 'lucide-react';
import { usePathname } from 'next/navigation';
import { useEffect, useState } from 'react';

import { Link } from '@/i18n/navigation';
import { WorkspaceMenu, type WorkspaceLink } from '@/components/workspace/workspace-menu';
import { fetchMe } from '@/lib/me';
import { type WorkspaceKey, roleHome, workspacesFor } from '@/lib/roles';

type User = { name: string; phone?: string | null; roles: string[] };
type Business = { display_name: string };

export type AccountLabels = {
  account: string;
  coupons: string;
  logout: string;
  switchWorkspace: string;
  viewMarketplace: string;
  buyer: string;
  owner: string;
  manager: string;
  cashier: string;
  admin: string;
  moderator: string;
  support: string;
  finance: string;
  superadmin: string;
  /** Names of the other workspaces, for "Switch workspace". */
  workspaces: Record<Exclude<WorkspaceKey, 'shop'>, string>;
};

function identity(user: User, business: Business | null, labels: AccountLabels) {
  const role = user.roles.find((item) => item !== 'user') ?? 'user';
  const key = role.replace(/^merchant_/, '') as keyof AccountLabels;
  const title = typeof labels[key] === 'string' ? (labels[key] as string) : labels.buyer;
  if (role.startsWith('merchant_')) return `${title} · ${business?.display_name ?? ''}`.replace(/ · $/, '');
  return role === 'user' ? labels.buyer : `${labels.admin} · ${title}`;
}

/** Where each workspace is opened from the marketplace header. */
function workspaceHref(key: Exclude<WorkspaceKey, 'shop'>, roles: string[]) {
  if (key === 'business') return '/merchant/dashboard';
  if (key === 'kassa') return '/merchant/redeem';
  return roleHome(roles);
}

/**
 * Account button in the marketplace header (desktop) and the bottom bar (phones). While the session is
 * loading it shows a placeholder, never a sign-in link, so a signed-in visitor is not sent to log in again.
 */
export function AccountMenu({ labels, variant = 'desktop' }: { labels: AccountLabels; variant?: 'desktop' | 'mobile' }) {
  const [user, setUser] = useState<User | null | undefined>(undefined);
  const [business, setBusiness] = useState<Business | null>(null);
  const pathname = usePathname();

  // Re-check on every page change: signing in or out does not remount the header.
  useEffect(() => {
    let active = true;
    fetchMe().then((value) => {
      if (!active) return;
      setUser(value);
      if (value?.roles.some((role) => role.startsWith('merchant_'))) {
        fetch('/api/bff/merchant/business', { cache: 'no-store' })
          .then((response) => (response.ok ? response.json() : null))
          .then((value2) => active && setBusiness(value2))
          .catch(() => undefined);
      }
    });
    return () => {
      active = false;
    };
  }, [pathname]);

  // The header account button shows from 1024 px; below that the bottom bar carries it.
  const wrap = variant === 'desktop' ? 'hidden lg:block' : '';
  const slot = variant === 'mobile' ? 'flex min-h-16 flex-col items-center justify-center gap-1 text-[12px] text-ink-muted' : '';

  if (user === undefined) {
    return (
      <div className={wrap}>
        <span aria-hidden="true" className={`${slot || 'flex min-h-11 min-w-11 items-center justify-center'} text-ink-muted`}>
          <UserRound />
        </span>
      </div>
    );
  }

  if (user === null) {
    return (
      <div className={wrap}>
        <Link
          href="/login"
          prefetch={false}
          aria-label={labels.account}
          className={slot || 'flex min-h-11 min-w-11 items-center justify-center rounded-control-ui text-ink-muted hover:bg-page hover:text-brand'}
        >
          <UserRound />
          {variant === 'mobile' && <span>{labels.account}</span>}
        </Link>
      </div>
    );
  }

  const switchTo: WorkspaceLink[] = workspacesFor(user.roles)
    .filter((key): key is Exclude<WorkspaceKey, 'shop'> => key !== 'shop')
    .map((key) => ({ href: workspaceHref(key, user.roles), label: labels.workspaces[key] }));

  return (
    <div className={`${wrap} ${slot}`.trim()}>
      <WorkspaceMenu
        name={user.name}
        roleLabel={identity(user, business, labels)}
        links={[{ href: '/account/coupons', label: labels.coupons }]}
        switchTo={switchTo}
        placement={variant === 'mobile' ? 'up' : 'down'}
        showRoleBadge={variant === 'desktop'}
        labels={{
          account: labels.account,
          signOut: labels.logout,
          switchWorkspace: labels.switchWorkspace,
          viewMarketplace: labels.viewMarketplace,
        }}
      />
      {variant === 'mobile' && <span className="text-[12px] text-ink-muted">{labels.account}</span>}
    </div>
  );
}
