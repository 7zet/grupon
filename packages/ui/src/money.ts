import type { Currency, Locale } from '@grupon/config';

const localeMap: Record<Locale, string> = { uz: 'uz-UZ', ru: 'ru-RU', en: 'en-US' };

export function convertUzsToUsdCents(amount: number, uzsPerUsd: number): number {
  if (!Number.isSafeInteger(amount) || !Number.isSafeInteger(uzsPerUsd) || uzsPerUsd <= 0) {
    throw new RangeError('Money values must be safe positive integers');
  }
  return Math.round((amount * 100) / uzsPerUsd);
}

export function formatMoney(
  amount: number,
  locale: Locale,
  currency: Currency,
  uzsPerUsd = 12650,
): string {
  if (currency === 'USD') {
    const cents = convertUzsToUsdCents(amount, uzsPerUsd);
    return `≈ ${new Intl.NumberFormat(localeMap[locale], { style: 'currency', currency: 'USD', minimumFractionDigits: 2 }).format(cents / 100)}`;
  }
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
