'use client';
import { useQuery } from '@tanstack/react-query';
import { Search } from 'lucide-react';
import { useLocale, useTranslations } from 'next-intl';
import { useDeferredValue, useState } from 'react';
import { Link } from '@/i18n/navigation';
export function SearchBox({ compact = false }: { compact?: boolean }) {
  const t = useTranslations('Navigation');
  const locale = useLocale();
  const [value, setValue] = useState('');
  const query = useDeferredValue(value.trim());
  const result = useQuery({
    queryKey: ['suggestions', query],
    queryFn: async () => {
      // Same-origin route: the Go API sends no CORS headers, so the browser must not call it directly.
      const response = await fetch(`/api/suggest?${new URLSearchParams({ q: query })}`);
      return (await response.json()) as { items: { type: string; label: string; href: string }[] };
    },
    enabled: query.length >= 2,
  });
  return (
    <form
      action={`/${locale}/search`}
      className={`relative ${compact ? '' : 'hidden min-w-0 flex-1 md:block'}`}
    >
      <Search className="pointer-events-none absolute left-3 top-1/2 z-10 size-5 -translate-y-1/2 text-ink-muted" />
      <input
        autoComplete="off"
        className="h-11 w-full rounded-control-ui border bg-page pl-10 pr-4 text-small"
        name="q"
        placeholder={t('searchPlaceholder')}
        value={value}
        onChange={(event) => setValue(event.target.value)}
      />
      {result.data?.items.length ? (
        <div className="absolute inset-x-0 top-full z-50 mt-2 overflow-hidden rounded-card-ui border bg-surface shadow-floating-ui">
          {result.data.items.map((item) => (
            <Link
              className="block border-b px-4 py-3 text-small hover:bg-brand-tint"
              href={item.href}
              key={`${item.type}-${item.href}`}
            >
              {item.label}
            </Link>
          ))}
        </div>
      ) : null}
    </form>
  );
}
