import { publicApi } from '@grupon/api-client';
import { localized, type Locale } from '@grupon/config';
import Image from 'next/image';
import { notFound } from 'next/navigation';
import { DealGrid } from '@/components/deal-grid';
import { getCurrency } from '@/lib/preferences';
export default async function CollectionPage({
  params,
}: {
  params: Promise<{ locale: Locale; slug: string }>;
}) {
  const { locale, slug } = await params;
  let data;
  try {
    data = await publicApi.collection(slug);
  } catch {
    notFound();
  }
  const currency = await getCurrency();
  return (
    <div>
      <section className="relative h-72 overflow-hidden">
        <Image
          src={data.cover_url}
          alt={localized(data.title, locale)}
          fill
          priority
          className="object-cover"
        />
        <div className="absolute inset-0 bg-gradient-to-t from-ink/80 to-transparent" />
        <div className="absolute inset-x-0 bottom-0 mx-auto max-w-layout px-4 py-8 text-surface sm:px-6 lg:px-8">
          <h1 className="text-title font-bold">{localized(data.title, locale)}</h1>
          <p className="mt-2 max-w-2xl text-surface/80">{localized(data.description, locale)}</p>
        </div>
      </section>
      <div className="mx-auto max-w-layout px-4 py-10 sm:px-6 lg:px-8">
        <DealGrid deals={data.deals.items} locale={locale} currency={currency} />
      </div>
    </div>
  );
}
