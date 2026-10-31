import { describe, expect, it } from 'vitest';
import { convertUzsToUsdCents, discountPercent, formatMoney } from './money';

describe('money helpers', () => {
  it('formats UZS by locale', () => {
    expect(formatMoney(150000, 'uz', 'UZS')).toBe("150 000 so'm");
    expect(formatMoney(150000, 'en', 'UZS')).toBe('150,000 UZS');
  });
  it('converts using integer cents and rounds once', () =>
    expect(convertUzsToUsdCents(150000, 12600)).toBe(1190));
  it('rounds discounts', () => expect(discountPercent(250000, 150000)).toBe(40));
});
