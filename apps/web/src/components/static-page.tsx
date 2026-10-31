import { brandConfig } from '@grupon/config';
import { getTranslations } from 'next-intl/server';
export async function StaticPage({
  titleKey,
  textKey,
}: {
  titleKey: 'helpTitle' | 'termsTitle' | 'privacyTitle';
  textKey: 'helpText' | 'termsText' | 'privacyText';
}) {
  const t = await getTranslations('Static');
  return (
    <div className="mx-auto max-w-3xl px-4 py-16 sm:px-6">
      <h1 className="text-title font-bold">{t(titleKey)}</h1>
      <p className="mt-5 text-lead leading-8 text-ink-muted">{t(textKey)}</p>
      {titleKey === 'helpTitle' && (
        <a
          className="mt-8 inline-block rounded-control-ui bg-brand px-5 py-3 font-semibold text-surface"
          href={`mailto:${brandConfig.supportEmail}`}
        >
          {t('contact')}
        </a>
      )}
    </div>
  );
}
