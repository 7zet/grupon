import { businessGate } from '@/components/workspace/gate';
import type { components } from '@dealo/api-client';
import { getTranslations } from 'next-intl/server';

import { DealWizard } from '@/components/deal-wizard';
import type { MediaAsset } from '@/components/merchant-media-library';
import { accountGet, loadForVisitor } from '@/lib/account-api';
import { pickText } from '@/lib/localized';

type Config = components['schemas']['PublicConfig'];
type LocationList = components['schemas']['LocationList'];

export default async function NewDealPage({ params }: { params: Promise<{ locale: string }> }) {
  const denied = await businessGate('createDeal');
  if (denied) return denied;
  const { locale } = await params;
  const t = await getTranslations('Deals');
  const [config, locations, media] = await loadForVisitor(
    locale,
    `/${locale}/merchant/deals/new`,
    () =>
      Promise.all([
        accountGet<Config>('/config'),
        accountGet<LocationList>('/merchant/locations'),
        accountGet<{ items: MediaAsset[] }>('/merchant/media'),
      ]),
  );
  const categories = config.categories.map((category) => ({
    id: category.id,
    label: pickText(category.name, locale),
    subcategories: category.subcategories.map((sub) => ({
      id: sub.id,
      label: pickText(sub.name, locale),
    })),
  }));
  const options = locations.items.map((location) => ({
    id: location.id,
    label: `${pickText(location.city.name, locale)} — ${location.address}`,
  }));

  return (
    <div className="grid grid-cols-[minmax(0,1fr)] gap-8">
      <h1 className="text-title font-bold">{t('createTitle')}</h1>
      <DealWizard categories={categories} locations={options} media={media.items} />
    </div>
  );
}
