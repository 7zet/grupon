'use client';

import {
  currencies,
  currencyCookieName,
  locales,
  type Currency,
  type Locale,
} from '@dealo/config';
import { useTranslations } from 'next-intl';
import { usePathname, useRouter } from '@/i18n/navigation';

export function CurrencySelect({ value }: { value: Currency }) {
  const t = useTranslations('Navigation');
  const router = useRouter();
  return (
    <select
      aria-label={t('currency')}
      className="rounded-control-ui border bg-surface px-2 py-2 text-small"
      value={value}
      onChange={(event) => {
        document.cookie = `${currencyCookieName}=${event.target.value};path=/;max-age=31536000;samesite=lax`;
        router.refresh();
      }}
    >
      {currencies.map((currency) => (
        <option key={currency}>{currency}</option>
      ))}
    </select>
  );
}

export function LanguageSelect({ value }: { value: Locale }) {
  const t = useTranslations('Navigation');
  const names = useTranslations('Locale');
  const router = useRouter();
  const pathname = usePathname();
  return (
    <select
      aria-label={t('language')}
      className="rounded-control-ui border bg-surface px-2 py-2 text-small"
      value={value}
      onChange={(event) => router.replace(pathname, { locale: event.target.value as Locale })}
    >
      {locales.map((locale) => (
        <option key={locale} value={locale}>
          {names(locale)}
        </option>
      ))}
    </select>
  );
}
