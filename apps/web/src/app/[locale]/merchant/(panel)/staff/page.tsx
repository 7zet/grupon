import type { components } from '@grupon/api-client';
import { getTranslations } from 'next-intl/server';

import { MerchantStaffInvite, MerchantStaffRow } from '@/components/merchant-staff-actions';
import { accountGet, loadForVisitor } from '@/lib/account-api';
import { maskPhone } from '@/lib/phone';

type StaffList = components['schemas']['StaffList'];

const ROLE_KEYS = {
  merchant_owner: 'roleOwner',
  merchant_manager: 'roleManager',
  merchant_cashier: 'roleCashier',
} as const;

/** Staff of the business. The owner sees invite and role controls; others see the list only. */
export default async function StaffPage({ params }: { params: Promise<{ locale: string }> }) {
  const { locale } = await params;
  const t = await getTranslations('MerchantPanel');
  const list = await loadForVisitor(locale, `/${locale}/merchant/staff`, () =>
    accountGet<StaffList>('/merchant/staff'),
  );

  return (
    <div className="grid grid-cols-[minmax(0,1fr)] gap-8">
      <h1 className="text-title font-bold">{t('staffTitle')}</h1>
      <MerchantStaffInvite />
      {list.items.length === 0 ? (
        <p className="text-ink-muted">{t('staffEmpty')}</p>
      ) : (
        <div className="overflow-x-auto">
          <table className="w-full min-w-[560px] text-left text-sm">
            <thead className="text-ink-muted">
              <tr>
                <th className="py-2 pr-4 font-semibold">{t('colName')}</th>
                <th className="py-2 pr-4 font-semibold">{t('colPhone')}</th>
                <th className="py-2 pr-4 font-semibold">{t('staffRole')}</th>
              </tr>
            </thead>
            <tbody>
              {list.items.map((member) => (
                <tr key={member.id} className="border-t border-border">
                  <td className="py-2 pr-4">{member.user.name}</td>
                  <td className="py-2 pr-4">{maskPhone(member.user.phone)}</td>
                  <td className="py-2 pr-4">
                    {member.role === 'merchant_owner' ? (
                      t(ROLE_KEYS[member.role])
                    ) : (
                      <MerchantStaffRow
                        assignmentId={member.id}
                        phone={member.user.phone ?? ''}
                        role={member.role}
                      />
                    )}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
      <p className="text-sm text-ink-muted">{t('staffOwnerOnly')}</p>
    </div>
  );
}
