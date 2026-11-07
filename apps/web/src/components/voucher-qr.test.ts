import { describe, expect, it } from 'vitest';

import { qrRefreshDelay } from './voucher-qr';

describe('qrRefreshDelay', () => {
  it('refreshes at the signed token expiry rather than a fixed interval', () => {
    const now = Date.parse('2026-11-07T00:00:00.000Z');
    expect(qrRefreshDelay('2026-11-07T00:01:00.000Z', now)).toBe(60_250);
  });

  it('refreshes an already expired token immediately', () => {
    expect(qrRefreshDelay('2026-11-06T23:59:00.000Z', Date.parse('2026-11-07T00:00:00.000Z'))).toBe(0);
  });
});
