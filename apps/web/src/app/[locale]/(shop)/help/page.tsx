import type { Locale } from '@grupon/config';
import { setRequestLocale } from 'next-intl/server';

import { StaticPage } from '@/components/static-page';

export default async function Page({ params }: { params: Promise<{ locale: Locale }> }) {
  setRequestLocale((await params).locale);
  return <StaticPage titleKey="helpTitle" textKey="helpText" />;
}
