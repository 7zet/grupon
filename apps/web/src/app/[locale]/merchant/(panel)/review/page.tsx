import type { components } from '@grupon/api-client';
import { getTranslations } from 'next-intl/server';

import { MerchantReviewForm } from '@/components/merchant-review-form';
import { accountGet } from '@/lib/account-api';

type Merchant = components['schemas']['Merchant'];
type DocumentList = components['schemas']['MerchantDocumentList'];

/** Where the business stands in platform review, and the documents still needed to submit it. */
export default async function MerchantReviewPage() {
  const t = await getTranslations('MerchantReview');
  const [business, documents] = await Promise.all([
    accountGet<Merchant>('/merchant/business'),
    accountGet<DocumentList>('/merchant/documents'),
  ]);
  const uploaded = {
    certificate: documents.items.some((doc) => doc.kind === 'certificate'),
    bank_letter: documents.items.some((doc) => doc.kind === 'bank_letter'),
    price_proof: documents.items.some((doc) => doc.kind === 'price_proof'),
  };
  const status = business.review_status ?? 'draft';

  return (
    <div className="grid gap-6">
      <h1 className="text-title font-bold">{t('title')}</h1>
      <p role="status" className="text-lead text-ink">
        {t(`status.${status}`)}
      </p>
      {status === 'draft' && (
        <>
          <p className="text-ink-muted">{t('draftHint')}</p>
          <MerchantReviewForm uploaded={uploaded} />
        </>
      )}
    </div>
  );
}
