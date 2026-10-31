import type { components } from '@grupon/api-client';
import { getTranslations } from 'next-intl/server';

import { BusinessForm, LocationsPanel } from '@/components/merchant-business';
import { accountGet, accountGetOrNull, loadForVisitor } from '@/lib/account-api';
import { pickText } from '@/lib/localized';

type Merchant = components['schemas']['Merchant'];
type Location = components['schemas']['MerchantLocation'];
type LocationList = components['schemas']['LocationList'];
type Config = components['schemas']['PublicConfig'];
type User = components['schemas']['User'];

export default async function BusinessPage({ params }: { params: Promise<{ locale: string }> }) {
  const { locale } = await params;
  const t = await getTranslations('Panel');
  const [business, list, config, user] = await loadForVisitor(locale, `/${locale}/merchant/business`, () =>
    Promise.all([
      accountGet<Merchant>('/merchant/business'),
      accountGet<LocationList>('/merchant/locations'),
      accountGet<Config>('/config'),
      accountGetOrNull<User>('/me'),
    ]),
  );
  // The API decides what each role may change; this only hides controls a cashier cannot use.
  const canEdit = Boolean(user?.roles.some((role) => role === 'merchant_owner' || role === 'merchant_manager'));
  const cities = config.cities.map((city) => ({ id: city.id, label: pickText(city.name, locale), lat: city.lat, lng: city.lng }));
  const locations: Location[] = list.items;

  return (
    <div className="grid gap-10">
      <h1 className="text-title font-bold">{t('businessTitle')}</h1>
      <section aria-labelledby="profile-title" className="grid gap-4">
        <h2 id="profile-title" className="font-semibold">{t('profile')}</h2>
        <BusinessForm
          business={{
            display_name: business.display_name,
            phone: business.phone,
            website: business.website ?? null,
            instagram: business.instagram ?? null,
            telegram: business.telegram ?? null,
          }}
          canEdit={canEdit}
        />
      </section>
      <LocationsPanel locations={locations} cities={cities} canEdit={canEdit} locale={locale} />
    </div>
  );
}
