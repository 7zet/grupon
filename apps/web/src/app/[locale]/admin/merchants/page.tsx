import type { components } from '@grupon/api-client';
import { getTranslations } from 'next-intl/server';
import { notFound } from 'next/navigation';

import { MerchantReviewActions } from '@/components/merchant-review-actions';
import { accountGet, accountGetOrNull, loadForVisitor } from '@/lib/account-api';

type User = components['schemas']['User'];
type ReviewList = components['schemas']['MerchantReviewList'];

const REVIEWERS = ['superadmin', 'admin', 'moderator'];

/** Queue of businesses waiting for review: identity and tax data, uploaded documents, decisions. */
export default async function MerchantReviewQueuePage({ params }: { params: Promise<{ locale: string }> }) {
  const { locale } = await params;
  const t = await getTranslations('AdminMerchants');
  const user = await loadForVisitor(locale, `/${locale}/admin/merchants`, () =>
    accountGetOrNull<User>('/me'),
  );
  if (!user?.roles.some((role) => REVIEWERS.includes(role))) notFound();
  const queue = await accountGet<ReviewList>('/admin/merchants');

  return (
    <div className="grid gap-8">
      <h1 className="text-title font-bold">{t('title')}</h1>
      {queue.items.length === 0 ? (
        <p className="text-ink-muted">{t('empty')}</p>
      ) : (
        <ul className="grid gap-4">
          {queue.items.map((item) => (
            <li key={item.id} className="grid gap-3 rounded-card border border-border bg-surface p-4">
              <div className="grid gap-1 text-small">
                <p className="font-semibold text-body">
                  {item.legal_name} · {item.display_name}
                </p>
                <p>
                  {t('stir')}: {item.stir} · {t(`orgForm.${item.org_form}`)} · {t(`tax.${item.tax_regime}`)}
                </p>
                <p>
                  MFO {item.bank_mfo} · {item.bank_account} · {item.responsible_name}
                </p>
                <p className="text-ink-muted">
                  {t(`status.${item.review_status}`)} · {new Date(item.submitted_at).toLocaleString(locale)}
                </p>
                <p className="flex flex-wrap gap-3">
                  {(['certificate', 'bank_letter', 'price_proof'] as const).map((kind) =>
                    item.documents?.includes(kind) ? (
                      <a
                        key={kind}
                        href={`/api/bff/admin/merchants/${item.id}/documents/${kind}`}
                        className="font-semibold text-brand"
                      >
                        {t(`document.${kind}`)}
                      </a>
                    ) : (
                      <span key={kind} className="text-ink-muted">
                        {t(`document.${kind}`)}: {t('missing')}
                      </span>
                    ),
                  )}
                </p>
              </div>
              <MerchantReviewActions merchantId={item.id} reviewStatus={item.review_status} />
            </li>
          ))}
        </ul>
      )}
    </div>
  );
}
