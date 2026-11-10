import type { Locale } from '@dealo/config';
import { ArrowRight, BarChart3, QrCode, Store } from 'lucide-react';
import { getTranslations, setRequestLocale } from 'next-intl/server';
import { Link } from '@/i18n/navigation';
export default async function MerchantLanding({ params }: { params: Promise<{ locale: Locale }> }) {
  setRequestLocale((await params).locale);
  const t = await getTranslations('Merchant');
  const steps = [
    { icon: Store, title: 'step1Title', text: 'step1Text' },
    { icon: BarChart3, title: 'step2Title', text: 'step2Text' },
    { icon: QrCode, title: 'step3Title', text: 'step3Text' },
  ] as const;
  return (
    <div>
      <section className="bg-brand text-surface">
        <div className="mx-auto max-w-layout px-4 py-20 sm:px-6 lg:px-8">
          <p className="text-small font-semibold uppercase tracking-widest text-surface/70">
            {t('landingEyebrow')}
          </p>
          <h1 className="mt-4 max-w-3xl text-title font-bold sm:text-display">
            {t('landingTitle')}
          </h1>
          <p className="mt-5 max-w-2xl text-lead leading-8 text-surface/80">{t('landingText')}</p>
          <Link
            className="mt-8 inline-flex items-center gap-2 rounded-control-ui bg-surface px-6 py-3 font-semibold text-brand"
            href="/merchant/register"
          >
            {t('cta')}
            <ArrowRight className="size-4" />
          </Link>
        </div>
      </section>
      <div className="mx-auto max-w-layout px-4 py-16 sm:px-6 lg:px-8">
        <section>
          <h2 className="text-title-small font-bold">{t('how')}</h2>
          <div className="mt-8 grid gap-5 md:grid-cols-3">
            {steps.map(({ icon: Icon, title, text }, index) => (
              <article className="rounded-panel-ui border bg-surface p-6" key={title}>
                <div className="flex size-11 items-center justify-center rounded-control-ui bg-brand-tint text-brand">
                  <Icon />
                </div>
                <span className="mt-5 block text-label font-semibold text-ink-faint">
                  0{index + 1}
                </span>
                <h3 className="mt-2 text-lead font-bold">{t(title)}</h3>
                <p className="mt-2 text-ink-muted">{t(text)}</p>
              </article>
            ))}
          </div>
        </section>
        <section className="mt-16 rounded-panel-ui bg-brand-tint p-8 sm:p-12">
          <h2 className="text-title-small font-bold text-brand">{t('commission')}</h2>
          <p className="mt-3 max-w-2xl text-ink-muted">{t('commissionText')}</p>
        </section>
        <section className="mt-16">
          <h2 className="text-title-small font-bold">{t('faq')}</h2>
          <div className="mt-5 divide-y rounded-card-ui border bg-surface px-5">
            {steps.map(({ title, text }) => (
              <details className="py-5" key={title}>
                <summary className="cursor-pointer font-semibold">{t(title)}</summary>
                <p className="mt-3 text-ink-muted">{t(text)}</p>
              </details>
            ))}
          </div>
        </section>
      </div>
    </div>
  );
}
