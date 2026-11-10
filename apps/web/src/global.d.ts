import type { Locale } from '@dealo/config';
import type { Messages } from '@dealo/i18n';

declare module 'next-intl' {
  interface AppConfig {
    Locale: Locale;
    Messages: Messages;
  }
}
