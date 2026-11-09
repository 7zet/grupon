'use client';

import type { ReactNode } from 'react';

import { Link, usePathname } from '@/i18n/navigation';

export type NavItem = { href: string; label: string; icon?: ReactNode };

/** Sidebar or bottom tab links; the current page is marked for assistive tools and sighted users. */
export function NavLinks({ items, label, variant }: { items: NavItem[]; label: string; variant: 'sidebar' | 'tabs' }) {
  const pathname = usePathname();
  const isSidebar = variant === 'sidebar';
  return (
    <nav aria-label={label} className={isSidebar ? 'grid gap-1' : 'grid grid-cols-4'}>
      {items.map((item) => {
        const current = pathname === item.href || pathname.startsWith(`${item.href}/`);
        return (
          <Link
            key={item.href}
            href={item.href}
            aria-current={current ? 'page' : undefined}
            prefetch={false}
            className={
              isSidebar
                ? `flex min-h-11 items-center gap-3 rounded-control-ui px-3 text-small ${current ? 'bg-brand-tint font-semibold text-brand' : 'text-ink-muted hover:bg-page hover:text-ink'}`
                : `flex min-h-14 flex-col items-center justify-center gap-1 px-1 text-center text-[12px] ${current ? 'font-semibold text-brand' : 'text-ink-muted'}`
            }
          >
            {item.icon}
            <span>{item.label}</span>
          </Link>
        );
      })}
    </nav>
  );
}
