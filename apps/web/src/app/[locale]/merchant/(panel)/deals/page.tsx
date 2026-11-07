import type { components } from '@grupon/api-client';
import { pickText } from '@/lib/localized';
import { getFormatter, getTranslations } from 'next-intl/server';

import { DealActions } from '@/components/deal-actions';
import { Link } from '@/i18n/navigation';
import { accountGet, loadForVisitor } from '@/lib/account-api';

type DealPage = components['schemas']['DealPage'];
type Deal = components['schemas']['Deal'];

/** Owned deals, 20 per page, newest first. Next and previous pages keep the list short for large catalogues. */
export default async function MyDealsPage({
  params,
  searchParams,
}: {
  params: Promise<{ locale: string }>;
  searchParams: Promise<{ page?: string }>;
}) {
  const { locale } = await params;
  const { page: rawPage } = await searchParams;
  const page = Math.max(1, Number.parseInt(rawPage ?? '1', 10) || 1);
  const t = await getTranslations('Deals');
  const format = await getFormatter();
  const list = await loadForVisitor(locale, `/${locale}/merchant/deals?page=${page}`, () =>
    accountGet<DealPage>(`/merchant/deals?page=${page}`),
  );
  // A rejected deal shows the moderator's reason; it is only on the detail view.
  const reasons = new Map<string, string>();
  await Promise.all(
    list.items
      .filter((deal) => deal.status === 'changes_requested')
      .map(async (deal) => {
        const detail = await accountGet<Deal>(`/merchant/deals/${deal.id}`).catch(() => null);
        if (detail?.rejection_reason) reasons.set(deal.id, detail.rejection_reason);
      }),
  );
  const pages = Math.max(1, Math.ceil(list.total / list.page_size));

  return (
    <div className="grid gap-8">
      <div className="flex flex-wrap items-center justify-between gap-4">
        <h1 className="text-title font-bold">{t('myDealsTitle')}</h1>
        <Link
          href="/merchant/deals/new"
          className="rounded-control-ui bg-brand px-5 py-3 font-semibold text-surface"
        >
          {t('createTitle')}
        </Link>
      </div>
      {list.items.length === 0 ? (
        <p className="text-ink-muted">{t('dealsEmpty')}</p>
      ) : (
        <div className="overflow-x-auto">
          <table className="w-full min-w-[640px] text-left text-sm">
            <thead className="text-ink-muted">
              <tr>
                <th className="py-2 pr-4 font-semibold">{t('colDeal')}</th>
                <th className="py-2 pr-4 font-semibold">{t('colStatus')}</th>
                <th className="py-2 pr-4 font-semibold">{t('colSold')}</th>
                <th className="py-2 pr-4 font-semibold">{t('colEnds')}</th>
                <th className="py-2 pr-4 font-semibold">{t('colActions')}</th>
              </tr>
            </thead>
            <tbody>
              {list.items.map((deal) => (
                <tr key={deal.id} className="border-t border-border">
                  <td className="py-2 pr-4">
                    {pickText(deal.title, locale)}
                    {reasons.has(deal.id) && (
                      <p className="mt-1 text-small text-error">
                        {t('rejectionReason')}: {reasons.get(deal.id)}
                      </p>
                    )}
                  </td>
                  <td className="py-2 pr-4">{t(`status_${deal.status}` as 'status_active')}</td>
                  <td className="py-2 pr-4">{deal.sold_count}</td>
                  <td className="py-2 pr-4">
                    {format.dateTime(new Date(deal.ends_at), { dateStyle: 'medium' })}
                  </td>
                  <td className="py-2 pr-4">
                    <DealActions dealId={deal.id} status={deal.status} />
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
      {pages > 1 && (
        <nav aria-label={t('pagination')} className="flex items-center gap-4 text-sm">
          {page > 1 && (
            <Link href={`/merchant/deals?page=${page - 1}`} className="font-semibold text-brand">
              {t('prevPage')}
            </Link>
          )}
          <span className="text-ink-muted">{t('pageInfo', { page, pages })}</span>
          {page < pages && (
            <Link href={`/merchant/deals?page=${page + 1}`} className="font-semibold text-brand">
              {t('nextPage')}
            </Link>
          )}
        </nav>
      )}
    </div>
  );
}
