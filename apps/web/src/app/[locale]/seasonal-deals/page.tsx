import { publicApi } from '@grupon/api-client';
import { localized, type Locale } from '@grupon/config';
import Image from 'next/image';
import { getTranslations } from 'next-intl/server';
import { Link } from '@/i18n/navigation';
export default async function CollectionsPage({ params }: { params: Promise<{ locale: Locale }> }) {
  const { locale } = await params;
  const [data, t] = await Promise.all([publicApi.collections(), getTranslations('Collections')]);
  return (
    <div className="mx-auto max-w-layout px-4 py-10 sm:px-6 lg:px-8">
      <h1 className="text-title font-bold">{t('title')}</h1>
      <p className="mt-2 text-ink-muted">{t('description')}</p>
      <div className="mt-8 grid gap-5 md:grid-cols-2">
        {data.items.map((collection) => (
          <Link
            className="group relative min-h-80 overflow-hidden rounded-panel-ui"
            href={`/collections/${collection.slug}`}
            key={collection.id}
          >
            <Image
              src={collection.cover_url}
              alt={localized(collection.title, locale)}
              fill
              className="object-cover transition duration-300 group-hover:scale-[1.03]"
            />
            <div className="absolute inset-0 bg-gradient-to-t from-ink/80 to-transparent" />
            <div className="absolute inset-x-0 bottom-0 p-6 text-surface">
              <h2 className="text-title-small font-bold">{localized(collection.title, locale)}</h2>
              <p className="mt-2 text-surface/80">{localized(collection.description, locale)}</p>
            </div>
          </Link>
        ))}
      </div>
    </div>
  );
}
