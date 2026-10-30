import type { Locale } from '@grupon/config';
import type { Messages } from '@grupon/i18n';

declare module 'next-intl' {
  interface AppConfig {
    Locale: Locale;
    Messages: Messages;
  }
}
