import type { components } from '@dealo/api-client';
import { getTranslations } from 'next-intl/server';
import type { ReactNode } from 'react';

import { Link } from '@/i18n/navigation';
import { accountGetOrNull } from '@/lib/account-api';

type User = components['schemas']['User'];

/** Only accounts holding a merchant role see the switch; a failed lookup simply hides it. */
async function isMerchant(): Promise<boolean> {
  try {
    const user = await accountGetOrNull<User>('/me');
    return Boolean(user?.roles.some((role) => role.startsWith('merchant_')));
  } catch {
    return false;
  }
}

const SECTIONS = ['coupons', 'orders', 'reviews', 'profile'] as const;

export async function AccountShell({
  section,
  children,
}: {
  section: (typeof SECTIONS)[number];
  children: ReactNode;
}) {
  const [t, merchant, tm] = await Promise.all([getTranslations('Account'), isMerchant(), getTranslations('MerchantRegister')]);
  return (
    <div className="mx-auto max-w-4xl px-4 py-10 sm:px-6">
      <h1 className="text-title font-bold">{t('title')}</h1>
      <nav aria-label={t('title')} className="mt-6 flex flex-wrap gap-2 border-b border-border pb-3">
        {SECTIONS.map((item) => (
          <Link
            key={item}
            href={`/account/${item}`}
            aria-current={item === section ? 'page' : undefined}
            className={
              item === section
                ? 'rounded-control-ui bg-brand-tint px-4 py-2 font-semibold text-brand'
                : 'rounded-control-ui px-4 py-2 text-ink-muted hover:text-ink'
            }
          >
            {t(`${item}Title` as 'couponsTitle')}
          </Link>
        ))}
      </nav>
      {merchant && (
        <Link href="/merchant" className="mt-4 inline-block text-sm font-semibold text-brand">
          {tm('switchToBusiness')}
        </Link>
      )}
      <div className="mt-8">{children}</div>
    </div>
  );
}
