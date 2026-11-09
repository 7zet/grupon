import { publicApi } from '@grupon/api-client';
import { localized, type Locale } from '@grupon/config';
import { MapPin, Star } from 'lucide-react';
import Image from 'next/image';
import { getTranslations } from 'next-intl/server';
import { notFound } from 'next/navigation';
import { DealGrid } from '@/components/deal-grid';
import { MapView } from '@/components/map-view';
import { getCurrency } from '@/lib/preferences';
/** Public catalog pages are cached and rebuilt at most once a minute. */
export const revalidate = 60;

/** No page is built ahead of time; each one is rendered on first visit, then cached. */
export function generateStaticParams() {
  return [];
}

export default async function BusinessPage({
  params,
}: {
  params: Promise<{ locale: Locale; slug: string }>;
}) {
  const { locale, slug } = await params;
  let merchant;
  try {
    merchant = await publicApi.merchant(slug);
  } catch {
    notFound();
  }
  const [t, currency] = await Promise.all([getTranslations('Merchant'), getCurrency()]);
  const locations = merchant.locations ?? [];
  return (
    <div>
      <section className="relative h-56 overflow-hidden bg-brand-tint sm:h-80">
        {merchant.cover_url && (
          <Image src={merchant.cover_url} alt="" fill className="object-cover" priority />
        )}
      </section>
      <div className="mx-auto max-w-layout px-4 sm:px-6 lg:px-8">
        <div className="-mt-12 relative flex items-end gap-4">
          {merchant.logo_url && (
            <Image
              src={merchant.logo_url}
              alt=""
              width={112}
              height={112}
              className="size-28 rounded-panel-ui border-4 border-surface object-cover"
            />
          )}
          <div className="pb-2">
            <h1 className="text-title-small font-bold">{merchant.display_name}</h1>
            {merchant.rating_count > 0 && (
              <p className="mt-1 flex items-center gap-1 text-small text-ink-muted">
                <Star className="size-4 fill-rating text-rating" />
                {merchant.rating_avg.toFixed(1)} ({merchant.rating_count})
              </p>
            )}
          </div>
        </div>
        <section className="py-10">
          <h2 className="text-subtitle font-bold">{t('about')}</h2>
          <p className="mt-3 max-w-3xl leading-7 text-ink-muted">
            {localized(merchant.description, locale)}
          </p>
        </section>
        <section className="pb-12">
          <h2 className="mb-5 text-subtitle font-bold">
            {t('profileDeals', { name: merchant.display_name })}
          </h2>
          <DealGrid deals={merchant.deals} locale={locale} currency={currency} />
        </section>
        {locations.length > 0 && (
          <section className="pb-12">
            <h2 className="mb-5 text-subtitle font-bold">{t('locations')}</h2>
            <div className="mb-4 grid gap-3 sm:grid-cols-2">
              {locations.map((location) => (
                <div className="rounded-card-ui border bg-surface p-4" key={location.id}>
                  <MapPin className="text-brand" />
                  <p className="mt-2 font-semibold">{location.address}</p>
                </div>
              ))}
            </div>
            <MapView locations={locations} />
          </section>
        )}
        <section className="pb-12">
          <h2 className="text-subtitle font-bold">{t('allReviews')}</h2>
          <div className="mt-4 grid gap-3 sm:grid-cols-2">
            {merchant.reviews.slice(0, 6).map((review) => (
              <article className="rounded-card-ui border bg-surface p-5" key={review.id}>
                <span className="text-rating">{'★'.repeat(review.rating)}</span>
                <p className="mt-3 text-ink-muted">{review.text}</p>
              </article>
            ))}
          </div>
        </section>
      </div>
    </div>
  );
}
