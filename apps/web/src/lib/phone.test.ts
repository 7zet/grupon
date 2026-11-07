import { describe, expect, it } from 'vitest';

import { maskPhone } from './phone';

describe('maskPhone', () => {
  it('hides the middle digits', () => {
    expect(maskPhone('+998901234567')).toBe('+998 90 ••• •• 67');
  });
  it('handles empty and short values', () => {
    expect(maskPhone(null)).toBe('');
    expect(maskPhone('123')).toBe('123');
  });
});
