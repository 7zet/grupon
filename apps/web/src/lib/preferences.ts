import type { Currency } from '@dealo/config';

/** Prices are UZS only, so the currency is a constant. It is not read from a cookie: reading cookies
 * would make every public page dynamic and defeat caching. */
export async function getCurrency(): Promise<Currency> {
  return 'UZS';
}
