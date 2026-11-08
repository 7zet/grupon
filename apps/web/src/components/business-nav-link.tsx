'use client';

import { useTranslations } from 'next-intl';
import { Briefcase } from 'lucide-react';
import { useEffect, useState } from 'react';

import { Link } from '@/i18n/navigation';
import { fetchMe } from '@/lib/me';

/** Shows the business-owner entry only to accounts with a merchant role. Everyone else sees no trace of it. */
export function BusinessNavLink({ variant = 'icon' }: { variant?: 'icon' | 'text' }) {
  const t = useTranslations('Navigation');
  const [isMerchant, setIsMerchant] = useState(false);

  useEffect(() => {
    let active = true;
    fetchMe()
      .then((user) => {
        if (active && user?.roles.some((role) => role.startsWith('merchant_'))) setIsMerchant(true);
      })
      .catch(() => undefined);
    return () => {
      active = false;
    };
  }, []);

  if (!isMerchant) return null;
  if (variant === 'text') {
    return (
      <Link className="mt-2 block text-small font-semibold text-brand" href="/merchant/dashboard">
        {t('businessSection')}
      </Link>
    );
  }
  return (
    <Link
      className="hidden items-center gap-2 rounded-control-ui px-3 py-2 text-small font-semibold text-brand hover:bg-page sm:flex"
      href="/merchant/dashboard"
    >
      <Briefcase aria-hidden="true" size={18} />
      {t('businessSection')}
    </Link>
  );
}
