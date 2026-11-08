import { publicApi } from '@grupon/api-client';
import { brandConfig, isLocale, type Currency } from '@grupon/config';
import { BaseLayout, appFont } from '@grupon/ui';
import type { Metadata } from 'next';
import { NextIntlClientProvider } from 'next-intl';
import { getMessages, getTranslations, setRequestLocale } from 'next-intl/server';
import { notFound } from 'next/navigation';
import type { PropsWithChildren } from 'react';

import '../globals.css';
import { Providers } from '../providers';
import { SiteShell } from '@/components/site-shell';

type LocaleLayoutProps = PropsWithChildren<{
  params: Promise<{ locale: string }>;
}>;

/** Nothing is built ahead of time, so `next build` does not need the API. Pages are rendered on first visit and cached. */
export function generateStaticParams() {
  return [];
}

export async function generateMetadata({ params }: LocaleLayoutProps): Promise<Metadata> {
  const { locale } = await params;

  if (!isLocale(locale)) {
    return {};
  }

  const t = await getTranslations({ locale, namespace: 'Metadata' });

  return {
    description: t('description'),
    metadataBase: new URL(process.env.SITE_URL ?? process.env.NEXT_PUBLIC_SITE_URL ?? 'http://localhost:3000'),
    title: t('title', { brand: brandConfig.name }),
  };
}

export default async function LocaleLayout({ children, params }: LocaleLayoutProps) {
  const { locale } = await params;

  if (!isLocale(locale)) {
    notFound();
  }

  setRequestLocale(locale);
  const messages = await getMessages();
  const currency: Currency = 'UZS';
  const config = await publicApi.config();

  return (
    <html lang={locale}>
      <body className={appFont.variable}>
        <NextIntlClientProvider messages={messages}>
          <Providers>
            <BaseLayout>
              <SiteShell config={config} locale={locale} currency={currency}>
                {children}
              </SiteShell>
            </BaseLayout>
          </Providers>
        </NextIntlClientProvider>
      </body>
    </html>
  );
}
