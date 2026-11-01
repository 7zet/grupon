import type { Currency, Locale } from '@grupon/config';

const localeMap: Record<Locale, string> = { uz: 'uz-UZ', ru: 'ru-RU', en: 'en-US' };

export function formatMoney(
  amount: number,
  locale: Locale,
  currency: Currency,
): string {
  const value = new Intl.NumberFormat(localeMap[locale])
    .format(amount)
    .replace(/[\u00a0\u202f]/g, ' ');
  if (locale === 'ru') return `${value} сум`;
  if (locale === 'en') return `${value} UZS`;
  return `${value} so'm`;
}

export function discountPercent(original: number, deal: number): number {
  return original <= 0 ? 0 : Math.round(((original - deal) * 100) / original);
}
