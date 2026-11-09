import { getTranslations } from 'next-intl/server';

import { AuthForm } from '@/components/auth-form';
import { safeNextPath } from '@/lib/safe-next';

export default async function RegisterPage({
  searchParams,
}: {
  searchParams: Promise<{ next?: string }>;
}) {
  const [{ next }, t] = await Promise.all([searchParams, getTranslations('Auth')]);
  return (
    <div className="mx-auto max-w-3xl px-4 py-16 sm:px-6">
      <h1 className="text-title font-bold">{t('registerTitle')}</h1>
      <p className="mt-4 text-lead text-ink-muted">{t('registerSubtitle')}</p>
      <AuthForm mode="register" next={safeNextPath(next)} />
    </div>
  );
}
