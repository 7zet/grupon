import { getTranslations } from 'next-intl/server';

import { MockPaymentActions } from '@/components/mock-payment-actions';

export default async function MockPaymentPage({
  params,
  searchParams,
}: {
  params: Promise<{ locale: string; paymentId: string }>;
  searchParams: Promise<{ order?: string }>;
}) {
  const [{ paymentId }, { order }, t] = await Promise.all([
    params,
    searchParams,
    getTranslations('Checkout'),
  ]);
  return (
    <div className="mx-auto max-w-md px-4 py-16 sm:px-6">
      <h1 className="text-title font-bold">{t('mockTitle')}</h1>
      <div className="mt-8">
        <MockPaymentActions paymentId={paymentId} orderId={order ?? ''} />
      </div>
    </div>
  );
}
