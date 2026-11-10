import type { PublicConfig } from '@dealo/api-client';
import { brandConfig, localized, type Currency, type Locale } from '@dealo/config';
import { Home, Menu, Search, Ticket } from 'lucide-react';
import { getTranslations } from 'next-intl/server';
import type { PropsWithChildren } from 'react';
import { Link } from '@/i18n/navigation';
import { Logo } from '@dealo/ui';
import { BusinessNavLink } from './business-nav-link';
import { LanguageSelect } from './preferences';
import { SearchBox } from './search-box';
import { AccountMenu } from '@/components/account-menu';
import { SupportContacts } from './support-contacts';
import { getSupportContacts } from '@/lib/support';

type Props = PropsWithChildren<{ config: PublicConfig; locale: Locale; currency: Currency }>;

export async function SiteShell({ children, config, locale }: Props) {
  const t = await getTranslations('Navigation');
  const footer = await getTranslations('Footer');
  const identity = await getTranslations('Identity');
  const workspace = await getTranslations('Workspace');
  const accountLabels = {
    account: t('account'),
    coupons: t('coupons'),
    logout: t('logout'),
    switchWorkspace: workspace('switchWorkspace'),
    viewMarketplace: workspace('viewMarketplace'),
    buyer: identity('buyer'),
    owner: identity('owner'),
    manager: identity('manager'),
    cashier: identity('cashier'),
    admin: identity('admin'),
    moderator: identity('moderator'),
    support: identity('support'),
    finance: identity('finance'),
    superadmin: identity('superadmin'),
    workspaces: {
      business: workspace('business', { brand: brandConfig.name }),
      kassa: workspace('kassa', { brand: brandConfig.name }),
      admin: workspace('admin', { brand: brandConfig.name }),
    },
  };
  const contacts = getSupportContacts();
  return (
    <>
      <a
        className="sr-only z-50 rounded-control-ui bg-surface px-4 py-3 text-small text-brand focus:not-sr-only focus:fixed focus:left-4 focus:top-4"
        href="#main-content"
      >
        {t('skipToContent')}
      </a>
      <header className="sticky top-0 z-40 border-b bg-surface/95 backdrop-blur">
        <div className="mx-auto flex h-18 max-w-layout items-center gap-3 px-4 sm:px-6 lg:px-8">
          <details className="group lg:hidden">
            <summary className="list-none rounded-control-ui p-2" aria-label={t('menu')}>
              <Menu />
            </summary>
            <nav className="fixed inset-x-0 top-18 max-h-[calc(100dvh-4.5rem)] overflow-auto border-b bg-surface p-4 shadow-floating-ui">
              {config.categories.map((category) => (
                <Link
                  className="block border-b px-2 py-3 font-semibold"
                  href={`/${category.slug}`}
                  key={category.id}
                  prefetch={false}
                >
                  {localized(category.name, locale)}
                </Link>
              ))}
            </nav>
          </details>
          <Link className="shrink-0 text-subtitle font-bold tracking-tight text-brand" href="/" prefetch={false}>
            <Logo name={brandConfig.name} />
          </Link>
          {config.demo_mode && <span className="rounded-full bg-brand-tint px-2 py-1 text-label font-semibold text-brand">Demo</span>}
          <SearchBox />
          <div className="ml-auto flex items-center gap-1">
            <select
              aria-label={t('city')}
              className="hidden rounded-control-ui border bg-surface px-2 py-2 text-small xl:block"
              defaultValue="tashkent"
            >
              {config.cities.map((city) => (
                <option value={city.slug} key={city.id}>
                  {localized(city.name, locale)}
                </option>
              ))}
            </select>
            <LanguageSelect value={locale} />
            <BusinessNavLink />
            <AccountMenu labels={accountLabels} />
          </div>
        </div>
        <nav className="hidden border-t lg:block">
          <div className="mx-auto flex max-w-layout items-center gap-6 flex-wrap px-8 py-3">
            {config.categories.map((category) => (
              <div className="group relative shrink-0" key={category.id}>
                <Link
                  className="text-small font-semibold text-ink-muted hover:text-brand"
                  href={`/${category.slug}`}
                  prefetch={false}
                >
                  {localized(category.name, locale)}
                </Link>
                {category.subcategories.length > 0 && (
                  <div className="invisible absolute left-0 top-full z-50 pt-2 opacity-0 transition group-hover:visible group-hover:opacity-100 group-focus-within:visible group-focus-within:opacity-100">
                    <div className="grid min-w-64 gap-1 rounded-card-ui border bg-surface p-3 shadow-floating-ui">
                      {category.subcategories.map((sub) => (
                        <Link
                          className="rounded-control-ui px-3 py-2 text-small hover:bg-brand-tint hover:text-brand"
                          href={`/tashkent/${category.slug}?subcategory=${sub.slug}`}
                          key={sub.id}
                          prefetch={false}
                        >
                          {localized(sub.name, locale)}
                        </Link>
                      ))}
                    </div>
                  </div>
                )}
              </div>
            ))}
          </div>
        </nav>
      </header>
      <main id="main-content" className="min-h-[70dvh]">
        {children}
      </main>
      <footer className="mb-16 mt-20 bg-brand-dark text-surface lg:mb-0">
        <div className="mx-auto grid max-w-layout gap-10 px-4 py-12 sm:grid-cols-3 sm:px-6 lg:px-8">
          <div>
            <div className="text-subtitle"><Logo name={brandConfig.name} className="text-surface" /></div>
            <p className="mt-3 text-small text-surface/75">{footer('appSoon')}</p>
          </div>
          <div>
            <h2 className="font-semibold">{footer('discover')}</h2>
            <BusinessNavLink variant="text" />
            <Link className="mt-2 block text-small text-surface/75" href="/merchant">
              {t('sell')}
            </Link>
          </div>
          <div>
            <h2 className="font-semibold">{footer('support')}</h2>
            <Link className="mt-3 block text-small text-surface/75" href="/help">
              {footer('help')}
            </Link>
            <div className="mt-3">
              <SupportContacts
                contacts={contacts}
                labels={{
                  phone: footer('phone'),
                  telegram: footer('telegram'),
                  email: footer('email'),
                  unavailable: footer('contactUnavailable'),
                }}
              />
            </div>
            <Link className="mt-2 block text-small text-surface/75" href="/terms">
              {footer('terms')}
            </Link>
            <Link className="mt-2 block text-small text-surface/75" href="/privacy">
              {footer('privacy')}
            </Link>
          </div>
        </div>
        <div className="border-t border-surface/15 py-5 text-center text-label text-surface/75">
          © {new Date().getFullYear()} {brandConfig.name}. {footer('rights')}
        </div>
      </footer>
      <nav className="fixed inset-x-0 bottom-0 z-40 grid grid-cols-5 border-t bg-surface pb-[env(safe-area-inset-bottom)] lg:hidden">
        <MobileLink href="/" label={t('home')}>
          <Home />
        </MobileLink>
        <MobileLink href="/search" label={t('search')}>
          <Search />
        </MobileLink>
        <MobileLink href="/account/coupons" label={t('coupons')}>
          <Ticket />
        </MobileLink>
        <AccountMenu labels={accountLabels} variant="mobile" />
      </nav>
    </>
  );
}

function MobileLink({ href, label, children }: PropsWithChildren<{ href: string; label: string }>) {
  return (
    <Link
      className="flex min-h-16 flex-col items-center justify-center gap-1 text-[12px] text-ink-muted"
      href={href}
      prefetch={false}
    >
      {children}
      <span>{label}</span>
    </Link>
  );
}
