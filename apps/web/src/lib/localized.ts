type LocalizedText = { uz: string; ru: string; en: string };

/** Picks the requested locale, falling back to uz, ru, en in that order (CLAUDE.md, localization rules). */
export function pickText(text: LocalizedText, locale: string): string {
  const values: Record<string, string> = { uz: text.uz, ru: text.ru, en: text.en };
  return values[locale] || text.uz || text.ru || text.en;
}
