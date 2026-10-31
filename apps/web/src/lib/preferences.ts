import { isCurrency, type Currency } from '@grupon/config';
import { cookies } from 'next/headers';
export async function getCurrency(): Promise<Currency> {
  const value = (await cookies()).get('currency')?.value;
  return isCurrency(value) ? value : 'UZS';
}
