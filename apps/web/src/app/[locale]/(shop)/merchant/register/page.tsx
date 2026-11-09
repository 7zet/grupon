import { cookies } from 'next/headers';
import { getTranslations } from 'next-intl/server';
import { redirect } from 'next/navigation';

import { publicApi } from '@grupon/api-client';
import { MerchantRegisterForm } from '@/components/merchant-register-form';
import { pickText } from '@/lib/localized';

export default async function MerchantRegisterPage({ params }: { params: Promise<{ locale: string }> }) {
  const { locale } = await params;
  if (!(await cookies()).get('access_token')) redirect(`/${locale}/login?next=/merchant/register`);

  const [t, config] = await Promise.all([getTranslations('MerchantRegister'), publicApi.config()]);
  const cities = config.cities.map((city) => ({
    id: city.id,
    label: pickText(city.name, locale),
    lat: city.lat,
    lng: city.lng,
  }));
  const categories = config.categories.map((category) => ({
    id: category.id,
    label: pickText(category.name, locale),
  }));

  return (
    <div className="mx-auto max-w-3xl px-4 py-12 sm:px-6">
      <h1 className="text-title font-bold">{t('title')}</h1>
      <p className="mt-4 text-lead text-ink-muted">{t('subtitle')}</p>
      <MerchantRegisterForm categories={categories} cities={cities} />
    </div>
  );
}
