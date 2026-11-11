import { businessGate } from '@/components/workspace/gate';
import type { components } from '@dealo/api-client';
import { getTranslations } from 'next-intl/server';

import { MerchantSettingsForm } from '@/components/merchant-settings-form';
import { accountGet, loadForVisitor } from '@/lib/account-api';
import { pickText } from '@/lib/localized';

type MerchantSettings = components['schemas']['MerchantSettings'];
type LocationList = components['schemas']['LocationList'];

/** Notification choices and the default branch. Only the owner and managers can save; the API enforces it. */
export default async function SettingsPage({ params }: { params: Promise<{ locale: string }> }) {
  const denied = await businessGate('settings');
  if (denied) return denied;
  const { locale } = await params;
  const t = await getTranslations('MerchantPanel');
  const settings = await loadForVisitor(locale, `/${locale}/merchant/settings`, () =>
    accountGet<MerchantSettings>('/merchant/settings'),
  );
  const locations = await loadForVisitor(locale, `/${locale}/merchant/settings`, () =>
    accountGet<LocationList>('/merchant/locations'),
  );

  return (
    <div className="grid grid-cols-[minmax(0,1fr)] gap-8">
      <h1 className="text-title font-bold">{t('settingsTitle')}</h1>
      <MerchantSettingsForm
        initial={settings}
        locations={locations.items.map((location) => ({
          id: location.id,
          label: `${pickText(location.city.name, locale)} — ${location.address}`,
        }))}
      />
    </div>
  );
}
