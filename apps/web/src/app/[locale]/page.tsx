import { brandConfig, locales, type Locale } from '@grupon/config';
import { getTranslations, setRequestLocale } from 'next-intl/server';

import { Link } from '@/i18n/navigation';

type HomePageProps = {
  params: Promise<{ locale: Locale }>;
};

export default async function HomePage({ params }: HomePageProps) {
  const { locale } = await params;
  setRequestLocale(locale);

  const home = await getTranslations('Home');
  const localeName = await getTranslations('Locale');
  const navigation = await getTranslations('Navigation');

  return (
    <>
      <a
        className="sr-only z-50 rounded-control-ui bg-surface px-4 py-3 text-small text-brand focus:not-sr-only focus:fixed focus:left-4 focus:top-4"
        href="#main-content"
      >
        {navigation('skipToContent')}
      </a>

      <header className="border-b bg-surface">
        <div className="mx-auto flex max-w-layout items-center justify-between gap-4 px-4 py-4 sm:px-6 lg:px-8">
          <Link
            className="text-subtitle font-bold tracking-tight text-brand"
            href="/"
            locale={locale}
          >
            {brandConfig.name}
          </Link>

          <nav aria-label={navigation('language')}>
            <ul className="flex items-center gap-1">
              {locales.map((targetLocale) => {
                const isCurrent = targetLocale === locale;

                return (
                  <li key={targetLocale}>
                    <Link
                      aria-current={isCurrent ? 'page' : undefined}
                      className={
                        isCurrent
                          ? 'block rounded-control-ui bg-brand-tint px-3 py-2 text-small font-semibold text-brand'
                          : 'block rounded-control-ui px-3 py-2 text-small text-ink-muted hover:bg-page hover:text-ink'
                      }
                      href="/"
                      locale={targetLocale}
                    >
                      {localeName(targetLocale)}
                    </Link>
                  </li>
                );
              })}
            </ul>
          </nav>
        </div>
      </header>

      <main
        className="mx-auto flex [min-height:calc(100dvh-var(--header-height))] max-w-layout items-center px-4 py-12 sm:px-6 lg:px-8"
        id="main-content"
      >
        <section className="max-w-3xl py-12 sm:py-20">
          <p className="mb-4 text-small font-semibold uppercase tracking-widest text-brand">
            {home('eyebrow')}
          </p>
          <h1 className="max-w-3xl text-title font-bold tracking-tight text-ink sm:text-display">
            {home('title')}
          </h1>
          <p className="mt-6 max-w-2xl text-body leading-7 text-ink-muted sm:text-lead">
            {home('description')}
          </p>
          <p className="mt-8 inline-flex items-center gap-2 rounded-control-ui bg-brand-tint px-4 py-3 text-small font-semibold text-brand">
            <span aria-hidden="true" className="size-2 rounded-round-ui bg-brand" />
            {home('status')}
          </p>
        </section>
      </main>
    </>
  );
}
