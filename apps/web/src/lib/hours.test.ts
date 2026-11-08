import { describe, expect, it } from 'vitest';

import { hoursLines } from './hours';

describe('hoursLines', () => {
  it('joins start and end with an en dash and marks closed days', () => {
    const lines = hoursLines({ mon: ['09:00', '20:00'], tue: '10:00,18:00', sun: null }, 'en');
    expect(lines[0]).toMatchObject({ day: 'mon', open: '09:00–20:00' });
    expect(lines[1]?.open).toBe('10:00–18:00');
    expect(lines[6]?.open).toBeNull();
    expect(lines).toHaveLength(7);
  });
});
