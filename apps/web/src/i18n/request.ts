import { isLocale } from '@grupon/config';
import { getMessages } from '@grupon/i18n';
import { getRequestConfig } from 'next-intl/server';

import { routing } from './routing';

export default getRequestConfig(async ({ requestLocale }) => {
  const requestedLocale = await requestLocale;
  const locale =
    requestedLocale && isLocale(requestedLocale) ? requestedLocale : routing.defaultLocale;

  return {
    locale,
    messages: getMessages(locale),
  };
});
