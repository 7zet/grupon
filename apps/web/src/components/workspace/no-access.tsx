import { getTranslations } from 'next-intl/server';

import { Link } from '@/i18n/navigation';
import { roleHome } from '@/lib/roles';

/** Shown when a signed-in user opens another role's workspace. The server check is still the real guard. */
export async function NoAccess({ roles }: { roles: string[] }) {
  const t = await getTranslations('Workspace');
  return (
    <div className="mx-auto max-w-md px-4 py-16 text-center sm:px-6">
      <h1 className="text-title font-bold">{t('noAccessTitle')}</h1>
      <p className="mt-3 text-ink-muted">{t('noAccessText')}</p>
      <Link
        href={roleHome(roles)}
        className="mt-7 inline-flex min-h-11 items-center rounded-control-ui bg-brand px-5 py-3 font-semibold text-surface"
      >
        {t('backHome')}
      </Link>
    </div>
  );
}
