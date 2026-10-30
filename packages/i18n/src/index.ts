import type { Locale } from '@grupon/config';

import en from '../messages/en.json';
import ru from '../messages/ru.json';
import uz from '../messages/uz.json';

export type Messages = typeof uz;

const messages = {
  en,
  ru,
  uz,
} satisfies Record<Locale, Messages>;

export function getMessages(locale: Locale): Messages {
  return messages[locale];
}

export { messages };
