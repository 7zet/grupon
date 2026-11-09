import type { components } from '@grupon/api-client';
import type { ReactNode } from 'react';

import { BusinessShell } from '@/components/workspace/business-shell';
import { KassaShell } from '@/components/workspace/kassa-shell';
import { NoAccess } from '@/components/workspace/no-access';
import { accountGet, currentUser, loadForVisitor } from '@/lib/account-api';

type Merchant = components['schemas']['Merchant'];
type LocationList = components['schemas']['LocationList'];

/**
 * Merchant workspace. Owners and managers get "Grupon Business"; a cashier gets "Grupon Kassa". Anyone
 * else sees the "No access" page. Each page also checks its own section (see businessGate).
 */
export default async function MerchantWorkspaceLayout({
  children,
  params,
}: {
  children: ReactNode;
  params: Promise<{ locale: string }>;
}) {
  const { locale } = await params;
  const user = await loadForVisitor(locale, `/${locale}/merchant/dashboard`, () => currentUser());
  if (!user) return null;
  const merchant = user.roles.some((role) => role.startsWith('merchant_'));
  if (!merchant) return <NoAccess roles={user.roles} />;

  const business = await loadForVisitor(locale, `/${locale}/merchant/dashboard`, () =>
    accountGet<Merchant>('/merchant/business'),
  );
  const cashierOnly =
    user.roles.includes('merchant_cashier') && !user.roles.some((role) => role === 'merchant_owner' || role === 'merchant_manager');

  if (cashierOnly) {
    const locations = await loadForVisitor(locale, `/${locale}/merchant/redeem`, () =>
      accountGet<LocationList>('/merchant/locations'),
    );
    const branch = locations.items[0]?.address ?? '';
    return (
      <KassaShell roles={user.roles} name={user.name} businessName={business.display_name} branch={branch}>
        {children}
      </KassaShell>
    );
  }

  return (
    <BusinessShell roles={user.roles} name={user.name} businessName={business.display_name}>
      {children}
    </BusinessShell>
  );
}
