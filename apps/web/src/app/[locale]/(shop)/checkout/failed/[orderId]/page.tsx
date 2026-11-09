import { getTranslations } from 'next-intl/server';
import { publicApi } from '@grupon/api-client';

import { RetryPaymentButton } from '@/components/retry-payment-button';
import { Link } from '@/i18n/navigation';

export default async function FailedPage({ params }: { params: Promise<{ orderId: string }> }) {
  const { orderId } = await params;
  const t = await getTranslations('Checkout');
  const config = await publicApi.config();
  return (
    <div className="mx-auto max-w-md px-4 py-16 text-center sm:px-6">
      <h1 className="text-title font-bold">{t('failedTitle')}</h1>
      <p className="mt-4 text-ink-muted">{t('failedText')}</p>
      <div className="mt-8 grid gap-3">
        {config.demo_mode && <RetryPaymentButton orderId={orderId} />}
        <Link href="/help" className="font-semibold text-brand">
          {t('support')}
        </Link>
      </div>
    </div>
  );
}
