import type { components } from '@grupon/api-client';
import { getTranslations } from 'next-intl/server';
import { notFound } from 'next/navigation';

import { PaymentPoller } from '@/components/payment-poller';
import { SuccessConfetti } from '@/components/success-confetti';
import { Link } from '@/i18n/navigation';
import { accountGetOrNull, loadForVisitor } from '@/lib/account-api';

type Order = components['schemas']['Order'];

export default async function SuccessPage({
  params,
}: {
  params: Promise<{ locale: string; orderId: string }>;
}) {
  const { locale, orderId } = await params;
  const t = await getTranslations('Checkout');
  const order = await loadForVisitor(locale, `/${locale}/checkout/success/${orderId}`, () =>
    accountGetOrNull<Order>(`/me/orders/${encodeURIComponent(orderId)}`),
  );
  if (!order) notFound();

  // The webhook may still be on its way; keep refreshing until the order is marked paid.
  if (order.status === 'pending_payment') {
    return (
      <div className="mx-auto max-w-md px-4 py-16 text-center sm:px-6">
        <PaymentPoller />
        <p className="text-ink-muted">{t('mockBusy')}</p>
      </div>
    );
  }

  return (
    <div className="mx-auto max-w-md px-4 py-16 text-center sm:px-6">
      <SuccessConfetti />
      <h1 className="text-title font-bold">{t('successTitle')}</h1>
      <p className="mt-4 text-ink-muted">{t('successText')}</p>
      <p className="mt-6 text-sm text-ink-muted">
        {t('orderNumber')}: <span className="font-mono text-ink">{order.number}</span>
      </p>
      <Link href="/account/coupons" className="mt-8 inline-block rounded-control-ui bg-brand px-5 py-3 font-semibold text-surface">
        {t('viewCoupons')}
      </Link>
    </div>
  );
}
