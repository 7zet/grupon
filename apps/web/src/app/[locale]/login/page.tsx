import { getTranslations } from 'next-intl/server';
import { publicApi } from '@grupon/api-client';

import { AuthForm } from '@/components/auth-form';
import { safeNextPath } from '@/lib/safe-next';

export default async function LoginPage({
  searchParams,
}: {
  searchParams: Promise<{ next?: string }>;
}) {
  const [{ next }, t, config] = await Promise.all([searchParams, getTranslations('Auth'), publicApi.config()]);
  return (
    <div className="mx-auto max-w-3xl px-4 py-16 sm:px-6">
      <h1 className="text-title font-bold">{t('loginTitle')}</h1>
      <p className="mt-4 text-lead text-ink-muted">{t('loginSubtitle')}</p>
      <AuthForm mode="login" next={safeNextPath(next)} demoMode={config.demo_mode} />
    </div>
  );
}
