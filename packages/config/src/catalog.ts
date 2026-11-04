import type { Locale } from './locales';

export type LocalizedText = Record<Locale, string>;

export const currencies = ['UZS'] as const;
export type Currency = (typeof currencies)[number];
export const currencyCookieName = 'currency';
export const cityCookieName = 'city';
export const defaultCity = 'tashkent';

export const citySlugs = ['tashkent', 'namangan', 'samarkand', 'andijan', 'fergana'] as const;
// PRD 03: four pilot categories are open; the rest exist behind their feature flags.
export const categorySlugs = [
  'beauty-and-spas',
  'food-and-drink',
  'things-to-do',
  'health-and-fitness',
  'automotive',
  'education',
  'events',
  'home-services',
  'travel',
  'pickup-goods',
  'medical',
] as const;

export function isCurrency(value: string | undefined): value is Currency {
  return currencies.includes(value as Currency);
}

export function isCitySlug(value: string): value is (typeof citySlugs)[number] {
  return citySlugs.includes(value as (typeof citySlugs)[number]);
}

export function isCategorySlug(value: string): value is (typeof categorySlugs)[number] {
  return categorySlugs.includes(value as (typeof categorySlugs)[number]);
}

export function localized(text: Partial<LocalizedText>, locale: Locale): string {
  return text[locale] ?? text.uz ?? text.ru ?? text.en ?? '';
}
