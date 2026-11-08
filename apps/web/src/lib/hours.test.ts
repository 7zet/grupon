import { describe, expect, it } from 'vitest';

import { groupedHours, hoursLines } from './hours';

describe('hoursLines', () => {
  it('joins start and end with an en dash and marks closed days', () => {
    const lines = hoursLines({ mon: ['09:00', '20:00'], tue: '10:00,18:00', sun: null }, 'en');
    expect(lines[0]).toMatchObject({ day: 'mon', open: '09:00–20:00' });
    expect(lines[1]?.open).toBe('10:00–18:00');
    expect(lines[6]?.open).toBeNull();
    expect(lines).toHaveLength(7);
  });

  it('groups consecutive days that share hours', () => {
    const week = { mon: ['09:00', '20:00'], tue: ['09:00', '20:00'], wed: ['09:00', '20:00'], thu: ['09:00', '20:00'], fri: ['09:00', '20:00'], sat: ['10:00', '18:00'], sun: null };
    const groups = groupedHours(hoursLines(week, 'en'));
    expect(groups.map((group) => group.open)).toEqual(['09:00–20:00', '10:00–18:00', null]);
    expect(groups[0]?.label).toBe('Mon–Fri');
  });
});
