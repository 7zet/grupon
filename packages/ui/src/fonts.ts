import { Inter, Manrope } from 'next/font/google';

/** Body and UI text. Latin and Cyrillic, so Uzbek (Latin and Cyrillic) and Russian render with one face. */
export const appFont = Inter({
  display: 'swap',
  subsets: ['cyrillic', 'latin'],
  variable: '--font-app',
});

/** Headings, wordmark and big numbers (weights 600 to 800). */
export const displayFont = Manrope({
  display: 'swap',
  subsets: ['cyrillic', 'latin'],
  variable: '--font-manrope',
  weight: ['600', '700', '800'],
});
