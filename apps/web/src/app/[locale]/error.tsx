'use client';
import { AlertCircle } from 'lucide-react';
import { useTranslations } from 'next-intl';
export default function ErrorPage({ reset }: { reset: () => void }) {
  const t = useTranslations('ErrorPage');
  return (
    <div className="mx-auto max-w-xl px-4 py-20 text-center">
      <AlertCircle className="mx-auto size-14 text-error" />
      <h1 className="mt-6 text-title font-bold">{t('errorTitle')}</h1>
      <p className="mt-3 text-ink-muted">{t('errorText')}</p>
      <button
        className="mt-7 rounded-control-ui bg-brand px-5 py-3 font-semibold text-surface"
        onClick={reset}
        type="button"
      >
        {t('retry')}
      </button>
    </div>
  );
}
