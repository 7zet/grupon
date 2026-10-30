import { locales } from '@grupon/config';
import { describe, expect, it } from 'vitest';

import { messages } from './index';

function sortedKeys(value: object): string[] {
  return Object.keys(value).sort();
}

describe('message catalogs', () => {
  it('contains one catalog for every supported locale', () => {
    expect(sortedKeys(messages)).toEqual([...locales].sort());
  });

  it('keeps the same top-level namespaces in every locale', () => {
    const expectedNamespaces = sortedKeys(messages.uz);

    for (const locale of locales) {
      expect(sortedKeys(messages[locale])).toEqual(expectedNamespaces);
    }
  });
});
