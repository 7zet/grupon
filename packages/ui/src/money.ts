import type { Currency, Locale } from '@dealo/config';

const localeMap: Record<Locale, string> = { uz: 'uz-UZ', ru: 'ru-RU', en: 'en-US' };

/** Money is stored and sent as tiyin (1 so'm = 100 tiyin), per ADR-0003. */
export const TIYIN_PER_SOM = 100;

/** Converts a so'm figure typed by a person into the tiyin amount the API expects. */
export function tiyinFromSom(som: number): number {
  return Math.round(som * TIYIN_PER_SOM);
}

export function formatMoney(
  tiyin: number,
  locale: Locale,
  currency: Currency,
): string {
  const som = Math.round(tiyin / TIYIN_PER_SOM);
  const value = new Intl.NumberFormat(localeMap[locale])
    .format(som)
    .replace(/[\u00a0\u202f]/g, ' ');
  if (currency !== 'UZS') throw new RangeError(`unsupported currency ${currency}`);
  if (locale === 'ru') return `${value} сум`;
  if (locale === 'en') return `${value} UZS`;
  return `${value} so'm`;
}

export function discountPercent(original: number, deal: number): number {
  return original <= 0 ? 0 : Math.round(((original - deal) * 100) / original);
}
