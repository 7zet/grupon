import type { Locale } from '@grupon/config';
import { ListingPage } from '@/components/listing-page';
export default async function SearchPage({
  params,
  searchParams,
}: {
  params: Promise<{ locale: Locale }>;
  searchParams: Promise<Record<string, string | string[] | undefined>>;
}) {
  const { locale } = await params;
  const query = await searchParams;
  const q = typeof query.q === 'string' ? query.q : '';
  return <ListingPage locale={locale} searchQuery={q} searchParams={query} />;
}
