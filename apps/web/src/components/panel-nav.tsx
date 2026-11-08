'use client';

import { Link, usePathname } from '@/i18n/navigation';

export type PanelLink = { href: string; label: string };

/** Merchant panel links; the current section is marked so people know where they are. */
export function PanelNav({ links, label }: { links: PanelLink[]; label: string }) {
  const pathname = usePathname();
  return (
    <nav aria-label={label} className="flex flex-wrap gap-2 border-b border-border pb-3">
      {links.map(({ href, label: text }) => {
        const current = pathname === href || (href !== '/merchant/dashboard' && pathname.startsWith(`${href}/`));
        return (
          <Link
            key={href}
            href={href}
            aria-current={current ? 'page' : undefined}
            className={`rounded-control-ui px-4 py-2 ${current ? 'bg-brand-tint font-semibold text-brand' : 'text-ink-muted hover:text-ink'}`}
          >
            {text}
          </Link>
        );
      })}
    </nav>
  );
}
