import { describe, expect, it } from 'vitest';
import { categorySlugs, citySlugs, localized } from './catalog';

describe('catalog configuration', () => {
  it('keeps city and category route namespaces disjoint', () => {
    expect(citySlugs.filter((slug) => categorySlugs.includes(slug as never))).toEqual([]);
  });

  it('falls back in requested, Uzbek, Russian, English order', () => {
    expect(localized({ ru: 'Текст', en: 'Text' }, 'en')).toBe('Text');
    expect(localized({ uz: 'Matn', en: 'Text' }, 'ru')).toBe('Matn');
  });
});
