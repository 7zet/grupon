import type { Locale } from '@dealo/config';
import { setRequestLocale } from 'next-intl/server';

import { StaticPage } from '@/components/static-page';

export default async function Page({ params }: { params: Promise<{ locale: Locale }> }) {
  setRequestLocale((await params).locale);
  return <StaticPage titleKey="termsTitle" textKey="termsText" />;
}
