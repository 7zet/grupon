import { defaultLocale, localeCookieName, locales } from '@dealo/config';
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
