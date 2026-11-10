import { redirect } from 'next/navigation';
import { publicApi } from '@dealo/api-client';
import { localized, type Locale } from '@dealo/config';
import {
  ArrowRight,
  BookOpen,
  Car,
  Drama,
  Dumbbell,
  Gift,
  Heart,
  House,
  Plane,
  Search,
  Sparkles,
  Ticket,
  Utensils,
  type LucideIcon,
} from 'lucide-react';
import Image from 'next/image';
import { getTranslations, setRequestLocale } from 'next-intl/server';

import { optionalUser } from '@/lib/account-api';
import { roleHome } from '@/lib/roles';
import { DealGrid } from '@/components/deal-grid';
import { HeroMotion } from '@/components/hero-motion';
import { SectionHeading } from '@/components/section-heading';
import { Link } from '@/i18n/navigation';
import { getCurrency } from '@/lib/preferences';

const homeRenderTime = Date.now();

const CATEGORY_ICONS: Record<string, LucideIcon> = {
  sparkles: Sparkles,
  utensils: Utensils,
  ticket: Ticket,
  dumbbell: Dumbbell,
  car: Car,
  book: BookOpen,
  theatre: Drama,
  home: House,
  plane: Plane,
  gift: Gift,
  heart: Heart,
};

/** Keeps whole rows only (multiples of four), so a grid never ends with a half-empty row. */
function wholeRows<T>(items: T[], max = 8): T[] {
  const count = Math.min(items.length, max);
  return items.slice(0, count - (count % 4));
}

/** Public catalog pages are cached and rebuilt at most once a minute. */
export const revalidate = 60;

export default async function HomePage({ params }: { params: Promise<{ locale: Locale }> }) {
  const { locale } = await params;
  setRequestLocale(locale);
  // Staff and business users work in their own workspace; "/" opens it. Buyers stay on the marketplace.
  const signedIn = await optionalUser();
  const home = signedIn ? roleHome(signedIn.roles) : '/';
  if (home !== '/') redirect(`/${locale}${home}`);
  const t = await getTranslations('Home');
  const [config, homepage, allDeals, currency] = await Promise.all([
    publicApi.config(),
    publicApi.homepage('tashkent'),
    publicApi.deals(new URLSearchParams({ city: 'tashkent', sort: 'popular', page: '1' })),
    getCurrency(),
  ]);
  const featuredAll =
    homepage.sections.find((section) => section.key === 'featured')?.deals ?? allDeals.items;
  const featured = wholeRows(featuredAll.length >= 4 ? featuredAll : allDeals.items);
  const used = new Set(featured.map((deal) => deal.id));
  const popularAll =
    homepage.sections.find((section) => section.key === 'popular')?.deals ?? allDeals.items;
  const popular = wholeRows(popularAll.filter((deal) => !used.has(deal.id)));
  popular.forEach((deal) => used.add(deal.id));
  const limited = wholeRows(
    allDeals.items
      .filter((deal) => !used.has(deal.id))
      .filter((deal) => new Date(deal.ends_at).getTime() < homeRenderTime + 3 * 86400000),
    4,
  );
  limited.forEach((deal) => used.add(deal.id));
  const recommended = wholeRows(allDeals.items.filter((deal) => !used.has(deal.id)));
  return (
    <>
      <section className="overflow-hidden border-b bg-surface">
        <HeroMotion>
          <div className="mx-auto grid max-w-layout items-center gap-10 px-4 py-14 sm:px-6 md:py-20 lg:grid-cols-[1.05fr_.95fr] lg:px-8">
            <div>
              <p
                data-hero
                className="mb-4 text-small font-semibold uppercase tracking-widest text-brand"
              >
                {t('eyebrow')}
              </p>
              <h1
                data-hero
                className="max-w-3xl text-title font-bold tracking-tight sm:text-display"
              >
                {t('title')}
              </h1>
              <p data-hero className="mt-5 max-w-xl text-lead leading-8 text-ink-muted">
                {t('description')}
              </p>
              <form data-hero action={`/${locale}/search`} className="relative mt-8 max-w-xl">
                <Search className="absolute left-4 top-1/2 size-5 -translate-y-1/2 text-ink-muted" />
                <input
                  name="q"
                  className="h-14 w-full rounded-card-ui border bg-page pl-12 pr-16 shadow-floating-ui sm:pr-40"
                  placeholder={(await getTranslations('Navigation'))('searchPlaceholder')}
                />
                <button
                  className="absolute bottom-1.5 right-1.5 top-1.5 rounded-control-ui bg-brand px-5 text-small font-semibold text-surface"
                  type="submit"
                >
                  <Search className="size-5 sm:hidden" aria-hidden="true" />
                  <span className="sr-only sm:not-sr-only">
                    {(await getTranslations('Navigation'))('search')}
                  </span>
                </button>
              </form>
              <div data-hero className="mt-5 flex flex-wrap items-center gap-2">
                <span className="text-small text-ink-muted">{t('searches')}:</span>
                {config.categories.slice(1, 5).map((category) => (
                  <Link
                    className="rounded-round-ui border bg-surface px-3 py-1.5 text-small hover:bg-brand-tint hover:text-brand"
                    href={`/tashkent/${category.slug}`}
                    key={category.id}
                    prefetch={false}
                  >
                    {localized(category.name, locale)}
                  </Link>
                ))}
              </div>
            </div>
            <div
              data-hero
              className="relative hidden aspect-[4/3] overflow-hidden rounded-panel-ui lg:block"
            >
              <Image
                className="object-cover"
                src={featured[0]!.media[0]!.url}
                alt={localized(featured[0]!.media[0]!.alt, locale)}
                fill
                priority
                sizes="45vw"
              />
              <div className="absolute bottom-5 left-5 flex items-center gap-3 rounded-card-ui bg-surface/95 px-4 py-3 shadow-floating-ui">
                <Sparkles className="text-brand" />
                <span className="font-semibold">{localized(featured[0]!.title, locale)}</span>
              </div>
            </div>
          </div>
        </HeroMotion>
      </section>
      <div className="mx-auto max-w-layout space-y-18 px-4 py-14 sm:px-6 lg:px-8">
        <section>
          <SectionHeading title={t('featured')} href="/tashkent" />
          <DealGrid deals={featured} locale={locale} currency={currency} />
        </section>
        {popular.length > 0 && (
          <section>
            <SectionHeading title={t('popular')} href="/tashkent?sort=popular" />
            <DealGrid deals={popular} locale={locale} currency={currency} />
          </section>
        )}
        <section>
          <SectionHeading title={t('categories')} />
          <div className="grid grid-cols-2 gap-3 sm:grid-cols-3 lg:grid-cols-5">
            {config.categories.map((category) => {
              const Icon = CATEGORY_ICONS[category.icon] ?? Sparkles;
              return (
              <Link
                className="group relative min-h-36 overflow-hidden rounded-card-ui border bg-surface p-5"
                href={`/${category.slug}`}
                key={category.id}
                prefetch={false}
              >
                <div className="absolute inset-0 bg-gradient-to-br from-brand-tint to-surface opacity-70" />
                <div className="relative flex h-full flex-col justify-between">
                  <Icon className="size-8 text-brand" aria-hidden="true" />
                  <span className="font-semibold">{localized(category.name, locale)}</span>
                </div>
              </Link>
              );
            })}
          </div>
        </section>
        {limited.length > 0 && (
          <section>
            <SectionHeading title={t('limited')} href="/tashkent?endingSoon=true" />
            <DealGrid deals={limited} locale={locale} currency={currency} />
          </section>
        )}
        {recommended.length > 0 && (
          <section>
            <SectionHeading title={t('recommended')} />
            <DealGrid deals={recommended} locale={locale} currency={currency} />
          </section>
        )}
        <section className="grid overflow-hidden rounded-panel-ui bg-brand text-surface lg:grid-cols-2">
          <div className="p-8 sm:p-12">
            <p className="text-small font-semibold uppercase tracking-widest text-surface/70">
              {t('merchantTitle')}
            </p>
            <h2 className="mt-3 text-title-small font-bold">{t('merchantText')}</h2>
            <Link
              className="mt-7 inline-flex items-center gap-2 rounded-control-ui bg-surface px-5 py-3 font-semibold text-brand"
              href="/merchant"
            >
              {t('merchantCta')}
              <ArrowRight className="size-4" />
            </Link>
          </div>
        </section>
      </div>
    </>
  );
}
