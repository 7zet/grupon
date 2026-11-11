import { businessGate } from '@/components/workspace/gate';
import { getTranslations } from 'next-intl/server';

import { MerchantMediaLibrary, type MediaAsset } from '@/components/merchant-media-library';
import { accountGet, loadForVisitor } from '@/lib/account-api';

export default async function MediaPage({ params }: { params: Promise<{ locale: string }> }) {
  const denied = await businessGate('media');
  if (denied) return denied;
  const { locale } = await params;
  const t = await getTranslations('Deals');
  const list = await loadForVisitor(locale, `/${locale}/merchant/media`, () =>
    accountGet<{ items: MediaAsset[] }>('/merchant/media'),
  );
  return (
    <div className="grid grid-cols-[minmax(0,1fr)] gap-8">
      <h1 className="text-title font-bold">{t('mediaTitle')}</h1>
      <MerchantMediaLibrary assets={list.items} />
    </div>
  );
}
