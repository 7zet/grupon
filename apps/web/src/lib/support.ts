import 'server-only';

export type SupportContacts = {
  phone: string | null;
  telegram: { label: string; href: string } | null;
  email: string | null;
};

function optional(value: string | undefined): string | null {
  const trimmed = value?.trim();
  return trimmed ? trimmed : null;
}

export function parseSupportContacts(env: Record<string, string | undefined>): SupportContacts {
  const phone = optional(env.SUPPORT_PHONE);
  const email = optional(env.SUPPORT_EMAIL);
  const telegram = optional(env.SUPPORT_TELEGRAM);

  return {
    phone,
    email,
    telegram: telegram
      ? {
          label: telegram,
          href: /^https?:\/\//i.test(telegram)
            ? telegram
            : `https://t.me/${telegram.replace(/^@/, '')}`,
        }
      : null,
  };
}

// Support contact details are deployment configuration, not product content. Keeping these server-only
// means changing a public phone number never requires bundling it into client-side JavaScript.
export function getSupportContacts(): SupportContacts {
  return parseSupportContacts(process.env);
}
