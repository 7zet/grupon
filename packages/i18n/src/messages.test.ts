import { locales } from '@dealo/config';
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

function leaves(value: unknown, prefix = ''): Map<string, string> {
  const out = new Map<string, string>();
  if (typeof value === 'string') {
    out.set(prefix, value);
  } else if (value && typeof value === 'object') {
    for (const [key, child] of Object.entries(value)) {
      for (const [path, text] of leaves(child, prefix ? `${prefix}.${key}` : key)) out.set(path, text);
    }
  }
  return out;
}

const placeholders = (text: string) => [...text.matchAll(/\{(\w+)\s*[,}]/g)].map((match) => match[1]).sort();

describe('message keys in every locale', () => {
  const reference = leaves(messages.en);

  for (const locale of locales) {
    const own = leaves(messages[locale]);

    it(`${locale} has exactly the English keys (no missing, no extra)`, () => {
      expect([...own.keys()].sort()).toEqual([...reference.keys()].sort());
    });

    it(`${locale} has no empty strings`, () => {
      const empty = [...own].filter(([, text]) => text.trim() === '').map(([key]) => key);
      expect(empty).toEqual([]);
    });

    it(`${locale} uses the same placeholders as English`, () => {
      const different = [...reference]
        .filter(([key, text]) => own.has(key) && placeholders(text).join() !== placeholders(own.get(key)!).join())
        .map(([key]) => key);
      expect(different).toEqual([]);
    });
  }

  it('does not carry shipping strings (nothing is shipped)', () => {
    expect([...reference.keys()].filter((key) => /shipping/i.test(key))).toEqual([]);
  });

  for (const locale of ['uz', 'ru'] as const) {
    it(`${locale} has no English sentences left untranslated`, () => {
      const own = leaves(messages[locale]);
      const copied = [...reference]
        .filter(([key, text]) => /[a-z]{3,}\s+[a-z]{3,}\s+[a-z]{3,}/i.test(text) && own.get(key) === text)
        .map(([key]) => key);
      expect(copied).toEqual([]);
    });
  }
});
