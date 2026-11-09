import { isCategorySlug, isCitySlug, type Locale } from '@grupon/config';
import { notFound, redirect } from 'next/navigation';
import { ListingPage } from '@/components/listing-page';
export default async function SegmentPage({
  params,
  searchParams,
}: {
  params: Promise<{ locale: Locale; segment: string }>;
  searchParams: Promise<Record<string, string | string[] | undefined>>;
}) {
  const { locale, segment } = await params;
  if (isCategorySlug(segment)) redirect(`/${locale}/tashkent/${segment}`);
  if (!isCitySlug(segment)) notFound();
  return <ListingPage locale={locale} citySlug={segment} searchParams={await searchParams} />;
}
