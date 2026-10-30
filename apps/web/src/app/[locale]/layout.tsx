import { brandConfig, isLocale, locales } from '@grupon/config';
import { BaseLayout, appFont } from '@grupon/ui';
import type { Metadata } from 'next';
import { NextIntlClientProvider } from 'next-intl';
import { getMessages, getTranslations, setRequestLocale } from 'next-intl/server';
import { notFound } from 'next/navigation';
import type { PropsWithChildren } from 'react';

import '../globals.css';

type LocaleLayoutProps = PropsWithChildren<{
  params: Promise<{ locale: string }>;
}>;

export function generateStaticParams() {
  return locales.map((locale) => ({ locale }));
}

export async function generateMetadata({ params }: LocaleLayoutProps): Promise<Metadata> {
  const { locale } = await params;

  if (!isLocale(locale)) {
    return {};
  }

  const t = await getTranslations({ locale, namespace: 'Metadata' });

  return {
    description: t('description'),
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

  return (
    <html lang={locale}>
      <body className={appFont.variable}>
        <NextIntlClientProvider messages={messages}>
          <BaseLayout>{children}</BaseLayout>
        </NextIntlClientProvider>
      </body>
    </html>
  );
}
