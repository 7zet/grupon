import { getTranslations } from 'next-intl/server';
import { Search } from 'lucide-react';
import { Link } from '@/i18n/navigation';
export default async function NotFound() {
  const t = await getTranslations('ErrorPage');
  const nav = await getTranslations('Navigation');
  return (
    <div className="mx-auto max-w-xl px-4 py-20 text-center">
      <div className="mx-auto flex size-16 items-center justify-center rounded-round-ui bg-brand-tint text-brand">
        <Search />
      </div>
      <h1 className="mt-6 text-title font-bold">{t('notFoundTitle')}</h1>
      <p className="mt-3 text-ink-muted">{t('notFoundText')}</p>
      <Link
        className="mt-7 inline-block rounded-control-ui bg-brand px-5 py-3 font-semibold text-surface"
        href="/"
      >
        {t('home')}
      </Link>
      <form className="relative mt-8" action="search">
        <Search className="absolute left-3 top-1/2 size-5 -translate-y-1/2 text-ink-muted" />
        <input
          name="q"
          className="h-12 w-full rounded-control-ui border bg-surface pl-10 pr-4"
          placeholder={nav('searchPlaceholder')}
        />
      </form>
    </div>
  );
}
