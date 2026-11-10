import { defaultLocale, locales } from '@dealo/config';
import { describe, expect, it } from 'vitest';

import { routing } from './routing';

describe('localized routing', () => {
  it('uses the configured locales with an always-on URL prefix', () => {
    expect(routing.locales).toEqual(locales);
    expect(routing.defaultLocale).toBe(defaultLocale);
    expect(routing.localePrefix).toBe('always');
  });
});
