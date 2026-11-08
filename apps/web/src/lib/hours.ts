const DAYS = ['mon', 'tue', 'wed', 'thu', 'fri', 'sat', 'sun'] as const;

export type HoursLine = { day: (typeof DAYS)[number]; label: string; open: string | null };

/** Reads opening hours stored as `{mon: ["09:00","20:00"], sun: null}` (or "09:00,20:00") per weekday. */
export function hoursLines(hours: Record<string, unknown> | undefined, locale: string): HoursLine[] {
  const names = new Intl.DateTimeFormat(locale, { weekday: 'short', timeZone: 'UTC' });
  return DAYS.map((day, index) => {
    // 2024-01-01 was a Monday.
    const label = names.format(new Date(Date.UTC(2024, 0, 1 + index)));
    const value = hours?.[day];
    const parts = Array.isArray(value)
      ? value.map(String)
      : typeof value === 'string'
        ? value.split(/[,\-–]/).map((part) => part.trim())
        : [];
    return { day, label, open: parts.length === 2 && parts[0] && parts[1] ? `${parts[0]}–${parts[1]}` : null };
  });
}

/** Merges consecutive days with the same hours: "Mon–Fri 09:00–20:00", "Sat 10:00–18:00", "Sun closed". */
export function groupedHours(lines: HoursLine[]): { key: string; label: string; open: string | null }[] {
  const groups: { key: string; label: string; open: string | null; last: string }[] = [];
  for (const line of lines) {
    const previous = groups[groups.length - 1];
    if (previous && previous.open === line.open) {
      previous.last = line.label;
      previous.label = `${previous.label.split('–')[0]}–${line.label}`;
    } else {
      groups.push({ key: line.day, label: line.label, open: line.open, last: line.label });
    }
  }
  return groups.map(({ key, label, open }) => ({ key, label, open }));
}
