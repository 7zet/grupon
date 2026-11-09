import { brandConfig, isLocale } from '@grupon/config';
import { BaseLayout, appFont } from '@grupon/ui';
import type { Metadata } from 'next';
import { NextIntlClientProvider } from 'next-intl';
import { getMessages, getTranslations, setRequestLocale } from 'next-intl/server';
import { notFound } from 'next/navigation';
import type { PropsWithChildren } from 'react';

import '../globals.css';
import { Providers } from '../providers';

type LocaleLayoutProps = PropsWithChildren<{
  params: Promise<{ locale: string }>;
}>;

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

/**
 * Root layout: the document, translations and the base theme. Each workspace adds its own shell in a
 * route group: (shop) for buyers, (business) for merchant staff, (admin) for platform staff.
 */
export default async function LocaleLayout({ children, params }: LocaleLayoutProps) {
  const { locale } = await params;

  if (!isLocale(locale)) {
    notFound();
  }

  setRequestLocale(locale);
  const messages = await getMessages();

  return (
    <html lang={locale}>
      <body className={appFont.variable}>
        <NextIntlClientProvider messages={messages}>
          <Providers>
            <BaseLayout>{children}</BaseLayout>
          </Providers>
        </NextIntlClientProvider>
      </body>
    </html>
  );
}
