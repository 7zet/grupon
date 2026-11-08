import { ArrowRight } from 'lucide-react';
import { getTranslations } from 'next-intl/server';
import { Link } from '@/i18n/navigation';
export async function SectionHeading({ title, href }: { title: string; href?: string }) {
  const t = await getTranslations('Common');
  return (
    <div className="mb-5 flex items-end justify-between gap-4">
      <h2 className="text-title-small font-bold tracking-tight">{title}</h2>
      {href && (
        <Link className="flex items-center gap-1 text-small font-semibold text-brand" href={href} prefetch={false}>
          {t('viewAll')}
          <ArrowRight className="size-4" />
        </Link>
      )}
    </div>
  );
}
