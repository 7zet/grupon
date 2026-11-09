import { publicApi } from '@grupon/api-client';
import { isLocale, type Currency } from '@grupon/config';
import { notFound } from 'next/navigation';
import type { PropsWithChildren } from 'react';

import { SiteShell } from '@/components/site-shell';

/** Buyer workspace: the marketplace header, category menu and the bottom tab bar on phones. */
export default async function ShopLayout({ children, params }: PropsWithChildren<{ params: Promise<{ locale: string }> }>) {
  const { locale } = await params;
  if (!isLocale(locale)) notFound();
  const currency: Currency = 'UZS';
  const config = await publicApi.config();
  return (
    <SiteShell config={config} locale={locale} currency={currency}>
      {children}
    </SiteShell>
  );
}
