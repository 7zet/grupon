import { publicApi } from '@grupon/api-client';
import { localized, type Locale } from '@grupon/config';
import { Check, Clock3, MapPin, Star } from 'lucide-react';
import type { Metadata } from 'next';
import { getTranslations, setRequestLocale } from 'next-intl/server';
import { notFound } from 'next/navigation';
import { DealGallery } from '@/components/deal-gallery';
import { DealGrid } from '@/components/deal-grid';
import { MapView } from '@/components/map-view';
import { PurchasePanel } from '@/components/purchase-panel';
import { SectionHeading } from '@/components/section-heading';
import { StickyBuy } from '@/components/sticky-buy';
import { Link } from '@/i18n/navigation';
import { getCurrency } from '@/lib/preferences';

type Props = { params: Promise<{ locale: Locale; slug: string }> };

export async function generateMetadata({ params }: Props): Promise<Metadata> {
  const { locale, slug } = await params;
  try {
    const deal = await publicApi.deal(slug);
    const title = localized(deal.title, locale);
    const description = localized(deal.short_description, locale);
    return {
      title,
      description,
      alternates: {
        canonical: `/${locale}/deal/${slug}`,
        languages: { uz: `/uz/deal/${slug}`, ru: `/ru/deal/${slug}`, en: `/en/deal/${slug}` },
      },
      openGraph: { title, description, images: deal.media[0] ? [deal.media[0].url] : [] },
    };
  } catch {
    return {};
  }
}

export default async function DealPage({ params }: Props) {
  const { locale, slug } = await params;
  setRequestLocale(locale);
  let deal;
  try {
    deal = await publicApi.deal(slug);
  } catch {
    notFound();
  }
  const [reviews, related, currency, t] = await Promise.all([
    publicApi.reviews(slug),
    publicApi.related(slug),
    getCurrency(),
    getTranslations('Deal'),
  ]);
  const option = deal.options[0]!;
  const structured = {
    '@context': 'https://schema.org',
    '@type': 'Product',
    name: localized(deal.title, locale),
    image: deal.media.map((item) => item.url),
    description: localized(deal.short_description, locale),
    brand: { '@type': 'Brand', name: deal.merchant.display_name },
    offers: {
      '@type': 'Offer',
      priceCurrency: 'UZS',
      price: option.deal_price.amount,
      availability:
        deal.status === 'active' ? 'https://schema.org/InStock' : 'https://schema.org/OutOfStock',
    },
    aggregateRating: {
      '@type': 'AggregateRating',
      ratingValue: deal.rating_avg,
      reviewCount: deal.rating_count,
    },
  };
  return (
    <>
      <script
        type="application/ld+json"
        dangerouslySetInnerHTML={{ __html: JSON.stringify(structured).replaceAll('<', '\\u003c') }}
      />
      <div className="mx-auto max-w-layout px-4 py-4 sm:px-6 lg:px-8 lg:py-6">
        <nav className="mb-4 text-small text-ink-muted">
          <Link href={`/${deal.locations[0]?.city.slug ?? 'tashkent'}/${deal.category.slug}`}>
            {localized(deal.category.name, locale)}
          </Link>{' '}
          · <span>{deal.merchant.display_name}</span>
        </nav>
        <div className="grid items-start gap-6 lg:grid-cols-[minmax(0,3fr)_minmax(22rem,2fr)]">
          <div>
            <DealGallery media={deal.media} locale={locale} />
            <div className="mt-4 lg:mt-6">
              <Link
                className="text-small font-semibold text-brand"
                href={`/business/${deal.merchant.slug}`}
              >
                {deal.merchant.display_name}
              </Link>
              <h1 className="mt-1 text-subtitle font-bold leading-tight sm:text-title-small">
                {localized(deal.title, locale)}
              </h1>
              <a
                className="mt-2 inline-flex items-center gap-1 text-small text-ink-muted"
                href="#reviews"
              >
                <Star className="size-4 fill-rating text-rating" />
                {deal.rating_avg.toFixed(1)} ({deal.rating_count})
              </a>
            </div>
            <div className="mt-4 lg:hidden" id="purchase-panel">
              <PurchasePanel deal={deal} locale={locale} currency={currency} />
            </div>
            <div className="mt-10 space-y-10">
              <section>
                <h2 className="text-subtitle font-bold">{t('description')}</h2>
                <p className="mt-3 leading-7 text-ink-muted">
                  {localized(deal.description, locale)}
                </p>
              </section>
              <section>
                <h2 className="text-subtitle font-bold">{t('highlights')}</h2>
                <ul className="mt-3 grid gap-3">
                  {deal.highlights.map((item, index) => (
                    <li className="flex gap-3" key={index}>
                      <Check className="mt-0.5 size-5 shrink-0 text-save" />
                      {localized(item, locale)}
                    </li>
                  ))}
                </ul>
              </section>
              <section>
                <h2 className="text-subtitle font-bold">{t('included')}</h2>
                <ul className="mt-3 list-inside list-disc text-ink-muted">
                  {deal.whats_included.map((item, index) => (
                    <li key={index}>{localized(item, locale)}</li>
                  ))}
                </ul>
              </section>
              <section className="rounded-card-ui border bg-surface p-5">
                <h2 className="text-subtitle font-bold">{t('terms')}</h2>
                <p className="mt-3 text-ink-muted">{localized(deal.terms, locale)}</p>
                <p className="mt-3 flex items-center gap-2 text-small">
                  <Clock3 className="size-4 text-brand" />
                  {t('validity')}: {deal.voucher_valid_days}
                </p>
                {deal.booking_required && <p className="mt-2 text-small">{t('booking')}</p>}
              </section>
              <section>
                <h2 className="mb-4 text-subtitle font-bold">{t('locations')}</h2>
                <div className="mb-4 grid gap-3 sm:grid-cols-2">
                  {deal.locations.map((location) => (
                    <div className="rounded-card-ui border bg-surface p-4" key={location.id}>
                      <MapPin className="mb-2 text-brand" />
                      <strong>{localized(location.city.name, locale)}</strong>
                      <p className="mt-1 text-small text-ink-muted">{location.address}</p>
                      <p className="mt-2 text-small">{String(location.opening_hours.mon)}</p>
                    </div>
                  ))}
                </div>
                <MapView locations={deal.locations} />
              </section>
              <section id="reviews">
                <h2 className="text-subtitle font-bold">
                  {t('reviews')} · {deal.rating_avg.toFixed(1)}
                </h2>
                <div className="mt-5 grid gap-4">
                  {reviews.items.slice(0, 6).map((review) => (
                    <article className="rounded-card-ui border bg-surface p-5" key={review.id}>
                      <div className="flex justify-between">
                        <span className="font-semibold">{t('verifiedPurchase')}</span>
                        <span className="text-rating">{'★'.repeat(review.rating)}</span>
                      </div>
                      <p className="mt-3 text-ink-muted">{review.text}</p>
                      {review.reply && (
                        <div className="mt-4 rounded-control-ui bg-brand-tint p-3 text-small">
                          <strong>{t('merchantReply')}</strong>
                          <p className="mt-1">{review.reply.text}</p>
                        </div>
                      )}
                      <button className="mt-3 text-label text-ink-muted" type="button">
                        {t('report')}
                      </button>
                    </article>
                  ))}
                </div>
              </section>
              <section>
                <h2 className="text-subtitle font-bold">{t('faq')}</h2>
                <div className="mt-4 divide-y rounded-card-ui border bg-surface px-5">
                  {deal.faq.map((item, index) => (
                    <details className="py-4" key={index}>
                      <summary className="cursor-pointer font-semibold">
                        {localized(item.q, locale)}
                      </summary>
                      <p className="mt-3 text-ink-muted">{localized(item.a, locale)}</p>
                    </details>
                  ))}
                </div>
              </section>
            </div>
          </div>
          <div className="sticky top-24 hidden lg:block" id="purchase-panel-desktop">
            <PurchasePanel deal={deal} locale={locale} currency={currency} />
          </div>
        </div>
        <section className="mt-16">
          <SectionHeading title={t('related')} />
          <DealGrid deals={related.items.slice(0, 4)} locale={locale} currency={currency} />
        </section>
      </div>
      <StickyBuy price={option.deal_price.amount} locale={locale} currency={currency} />
    </>
  );
}
