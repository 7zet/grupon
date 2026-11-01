import { describe, expect, it } from 'vitest';
import { discountPercent, formatMoney } from './money';

describe('money helpers', () => {
  it('formats UZS by locale', () => {
    expect(formatMoney(15000000, "uz", "UZS")).toBe("150 000 so'm");
    expect(formatMoney(15000000, "en", "UZS")).toBe("150,000 UZS");
  });
  it('rounds discounts', () => expect(discountPercent(250000, 150000)).toBe(40));
});
