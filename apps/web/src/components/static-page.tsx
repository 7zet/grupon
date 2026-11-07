import { getTranslations } from 'next-intl/server';
import { SupportContacts } from './support-contacts';
import { getSupportContacts } from '@/lib/support';
export async function StaticPage({
  titleKey,
  textKey,
}: {
  titleKey: 'helpTitle' | 'termsTitle' | 'privacyTitle';
  textKey: 'helpText' | 'termsText' | 'privacyText';
}) {
  const t = await getTranslations('Static');
  const contacts = getSupportContacts();
  return (
    <div className="mx-auto max-w-3xl px-4 py-16 sm:px-6">
      <h1 className="text-title font-bold">{t(titleKey)}</h1>
      <p className="mt-5 text-lead leading-8 text-ink-muted">{t(textKey)}</p>
      {titleKey === 'helpTitle' && (
        <section className="mt-8 rounded-card-ui border bg-surface p-5" aria-labelledby="support-contacts">
          <h2 id="support-contacts" className="font-semibold text-ink">
            {t('contact')}
          </h2>
          <div className="mt-3">
            <SupportContacts
              contacts={contacts}
              labels={{
                phone: t('phone'),
                telegram: t('telegram'),
                email: t('email'),
                unavailable: t('contactUnavailable'),
              }}
            />
          </div>
        </section>
      )}
    </div>
  );
}
