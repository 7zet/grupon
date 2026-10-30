import { defaultLocale, localeCookieName, locales } from '@grupon/config';
import { defineRouting } from 'next-intl/routing';

export const routing = defineRouting({
  defaultLocale,
  localeCookie: {
    name: localeCookieName,
  },
  localeDetection: true,
  localePrefix: 'always',
  locales,
});
