'use client';

import { LogOut, UserRound } from 'lucide-react';

import { Link, useRouter } from '@/i18n/navigation';

export type WorkspaceLink = { href: string; label: string };

type Labels = { account: string; signOut: string; switchWorkspace: string; viewMarketplace: string };

/**
 * Account button for every header. It always opens the menu, never the login page: the signed-in user
 * is known before this renders (or the caller shows a placeholder), so the icon never sends a session to sign-in.
 */
export function WorkspaceMenu({
  name,
  roleLabel,
  switchTo,
  links = [],
  marketplaceLink = false,
  placement = 'down',
  showRoleBadge = true,
  labels,
}: {
  name: string;
  roleLabel: string;
  switchTo: WorkspaceLink[];
  links?: WorkspaceLink[];
  marketplaceLink?: boolean;
  placement?: 'down' | 'up';
  /** The role pill next to the icon; the phone bar hides it, so it is rendered only where it is visible. */
  showRoleBadge?: boolean;
  labels: Labels;
}) {
  const router = useRouter();

  async function signOut() {
    await fetch('/api/bff/auth/logout', { method: 'POST' });
    router.replace('/login');
    router.refresh();
  }

  return (
    <details className="relative">
      <summary
        aria-label={labels.account}
        className="flex min-h-11 min-w-11 cursor-pointer list-none items-center justify-center gap-2 rounded-control-ui px-2 text-ink-muted hover:bg-page hover:text-brand"
      >
        <UserRound aria-hidden="true" />
        {showRoleBadge && (
          <span data-testid="role-badge" className="hidden max-w-40 truncate rounded-full bg-brand-tint px-2 py-1 text-label font-semibold text-brand sm:inline">
            {roleLabel}
          </span>
        )}
      </summary>
      <div
        className={`absolute right-0 z-50 grid w-64 gap-2 rounded-card-ui border bg-surface p-3 shadow-floating-ui ${placement === 'up' ? 'bottom-full mb-2' : 'mt-2'}`}
      >
        <p className="font-semibold">{name}</p>
        <p className="rounded-full bg-brand-tint px-2 py-1 text-label font-semibold text-brand">{roleLabel}</p>
        {links.map((item) => (
          <Link key={item.href} href={item.href} className="flex min-h-11 items-center rounded-control-ui px-2 text-small hover:bg-page">
            {item.label}
          </Link>
        ))}
        {switchTo.length > 0 && (
          <div className="grid gap-1 border-t pt-2">
            <p className="text-label text-ink-muted">{labels.switchWorkspace}</p>
            {switchTo.map((item) => (
              <Link key={item.href} href={item.href} className="flex min-h-11 items-center rounded-control-ui px-2 text-small hover:bg-page">
                {item.label}
              </Link>
            ))}
          </div>
        )}
        {marketplaceLink && (
          <Link href="/tashkent" className="flex min-h-11 items-center rounded-control-ui px-2 text-small text-brand hover:bg-page">
            {labels.viewMarketplace}
          </Link>
        )}
        <button type="button" onClick={signOut} className="flex min-h-11 items-center gap-2 border-t px-2 pt-2 text-small text-ink-muted">
          <LogOut size={16} aria-hidden="true" />
          {labels.signOut}
        </button>
      </div>
    </details>
  );
}
