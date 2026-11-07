'use client';

import { useTranslations } from 'next-intl';

import { Link, usePathname } from '@/i18n/navigation';

const SECTIONS = [
  ['/admin', 'overview'],
  ['/admin/moderation', 'moderation'],
  ['/admin/merchants', 'merchants'],
  ['/admin/finance', 'finance'],
  ['/admin/reports', 'reports'],
  ['/admin/fraud', 'fraud'],
  ['/admin/audit', 'audit'],
] as const;

/** Links between the admin sections; the current one is marked for assistive tools and sighted users. */
export function AdminNav() {
  const t = useTranslations('AdminNav');
  const pathname = usePathname();
  return (
    <nav aria-label={t('label')} className="mb-6 flex flex-wrap gap-2 text-small">
      {SECTIONS.map(([href, key]) => {
        const current = href === '/admin' ? pathname === href : pathname.startsWith(href);
        return (
          <Link
            key={href}
            href={href}
            aria-current={current ? 'page' : undefined}
            className={`rounded-full border px-3 py-2 font-semibold ${current ? 'border-brand bg-brand-tint text-brand' : 'border-border text-ink-muted'}`}
          >
            {t(key)}
          </Link>
        );
      })}
    </nav>
  );
}
