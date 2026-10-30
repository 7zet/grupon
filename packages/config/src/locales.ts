export const locales = ['uz', 'ru', 'en'] as const;

export type Locale = (typeof locales)[number];

export const defaultLocale: Locale = 'uz';
export const localeCookieName = 'marketplace_locale';

export function isLocale(value: string): value is Locale {
  return locales.some((locale) => locale === value);
}
