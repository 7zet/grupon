import type { SupportContacts as Contacts } from '@/lib/support';

type Labels = { phone: string; telegram: string; email: string; unavailable: string };

export function SupportContacts({ contacts, labels }: { contacts: Contacts; labels: Labels }) {
  if (!contacts.phone && !contacts.telegram && !contacts.email) {
    return <p className="text-small text-ink-muted">{labels.unavailable}</p>;
  }

  return (
    <ul className="space-y-2 text-small">
      {contacts.phone && (
        <li>
          <span className="font-semibold">{labels.phone}: </span>
          <a className="underline-offset-2 opacity-90 hover:underline" href={`tel:${contacts.phone}`}>
            {contacts.phone}
          </a>
        </li>
      )}
      {contacts.telegram && (
        <li>
          <span className="font-semibold">{labels.telegram}: </span>
          <a className="underline-offset-2 opacity-90 hover:underline" href={contacts.telegram.href} rel="noreferrer" target="_blank">
            {contacts.telegram.label}
          </a>
        </li>
      )}
      {contacts.email && (
        <li>
          <span className="font-semibold">{labels.email}: </span>
          <a className="underline-offset-2 opacity-90 hover:underline" href={`mailto:${contacts.email}`}>
            {contacts.email}
          </a>
        </li>
      )}
    </ul>
  );
}
