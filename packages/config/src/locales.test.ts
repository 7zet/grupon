import { describe, expect, it } from 'vitest';

import { defaultLocale, isLocale, locales } from './locales';

describe('locale configuration', () => {
  it('keeps the required locale order and Uzbek default', () => {
    expect(locales).toEqual(['uz', 'ru', 'en']);
    expect(defaultLocale).toBe('uz');
  });

  it('rejects unsupported locale values', () => {
    expect(isLocale('ru')).toBe(true);
    expect(isLocale('de')).toBe(false);
  });
});
