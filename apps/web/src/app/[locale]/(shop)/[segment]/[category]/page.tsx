import { isCategorySlug, isCitySlug, type Locale } from '@dealo/config';
import { notFound } from 'next/navigation';
import { ListingPage } from '@/components/listing-page';
export default async function CategoryPage({
  params,
  searchParams,
}: {
  params: Promise<{ locale: Locale; segment: string; category: string }>;
  searchParams: Promise<Record<string, string | string[] | undefined>>;
}) {
  const { locale, segment, category } = await params;
  if (!isCitySlug(segment) || !isCategorySlug(category)) notFound();
  return (
    <ListingPage
      locale={locale}
      citySlug={segment}
      categorySlug={category}
      searchParams={await searchParams}
    />
  );
}
