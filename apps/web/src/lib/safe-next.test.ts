import { describe, expect, it } from 'vitest';

import { safeNextPath } from './safe-next';

describe('safeNextPath', () => {
  it('keeps same-site relative paths', () => {
    expect(safeNextPath('/uz/deal/spa-day')).toBe('/uz/deal/spa-day');
  });

  it('rejects absolute, protocol-relative and backslash targets', () => {
    expect(safeNextPath('https://evil.example')).toBe('/');
    expect(safeNextPath('//evil.example')).toBe('/');
    expect(safeNextPath('/\\evil.example')).toBe('/');
    expect(safeNextPath(undefined)).toBe('/');
  });
});
