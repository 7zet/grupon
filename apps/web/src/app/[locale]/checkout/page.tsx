import { cookies } from 'next/headers';
import { publicApi } from '@grupon/api-client';
import { getTranslations } from 'next-intl/server';
import { redirect } from 'next/navigation';

import { CheckoutForm } from '@/components/checkout-form';

export default async function CheckoutPage({ params }: { params: Promise<{ locale: string }> }) {
  const { locale } = await params;
  // Browsing never needs login; checkout does. The API still validates the session on order creation.
  if (!(await cookies()).get('access_token')) redirect(`/${locale}/login?next=/checkout`);
  const t = await getTranslations('Checkout');
  const config = await publicApi.config();
  return (
    <div className="mx-auto max-w-5xl px-4 py-10 sm:px-6">
      <h1 className="text-title font-bold">{t('title')}</h1>
      <div className="mt-8">
        <CheckoutForm demoMode={config.demo_mode} />
      </div>
    </div>
  );
}
