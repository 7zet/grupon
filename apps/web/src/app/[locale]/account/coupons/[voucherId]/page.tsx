import type { components } from '@grupon/api-client';
import { getTranslations, getFormatter } from 'next-intl/server';
import QRCode from 'qrcode';
import { notFound } from 'next/navigation';

import { AccountShell } from '@/components/account-shell';
import { ReviewForm } from '@/components/review-form';
import { SupportContacts } from '@/components/support-contacts';
import { accountGetOrNull, loadForVisitor } from '@/lib/account-api';
import { pickText } from '@/lib/localized';
import { getSupportContacts } from '@/lib/support';

type Voucher = components['schemas']['Voucher'];

export default async function VoucherPage({
  params,
}: {
  params: Promise<{ locale: string; voucherId: string }>;
}) {
  const { locale, voucherId } = await params;
  const t = await getTranslations('Account');
  const format = await getFormatter();
  const contacts = getSupportContacts();

  const voucher = await loadForVisitor(locale, `/${locale}/account/coupons/${voucherId}`, () =>
    accountGetOrNull<Voucher>(`/me/vouchers/${encodeURIComponent(voucherId)}`),
  );
  if (!voucher) notFound();

  // The QR encodes the server-signed token only, never the raw voucher id.
  const qrDataUrl = await QRCode.toDataURL(voucher.qr_token, { margin: 1, width: 320 });

  return (
    <AccountShell section="coupons">
      <article className="mx-auto grid max-w-md justify-items-center gap-5 text-center">
        <h2 className="text-lead font-semibold text-ink">{pickText(voucher.deal.title, locale)}</h2>
        {/* eslint-disable-next-line @next/next/no-img-element -- generated data URL, not an optimizable asset */}
        <img
          src={qrDataUrl}
          alt={t('qrAlt')}
          width={320}
          height={320}
          className="h-auto max-w-full rounded-card bg-surface p-3"
        />
        <p className="text-sm text-ink-muted">{t('brightnessHint')}</p>
        <p className="text-sm text-ink-muted">
          {t('validUntil', {
            date: format.dateTime(new Date(voucher.valid_until), { dateStyle: 'medium' }),
          })}
        </p>
        <dl className="grid w-full gap-2 text-left">
          <div className="flex justify-between gap-4">
            <dt className="text-ink-muted">{t('codeLabel')}</dt>
            <dd className="font-mono font-semibold text-ink">{voucher.code}</dd>
          </div>
          <div className="flex justify-between gap-4">
            <dt className="text-ink-muted">{t('merchantLabel')}</dt>
            <dd className="text-ink">{voucher.merchant.display_name}</dd>
          </div>
        </dl>
        <section className="w-full rounded-card-ui border bg-surface p-4 text-left" aria-labelledby="voucher-support">
          <h3 id="voucher-support" className="font-semibold text-ink">
            {t('supportTitle')}
          </h3>
          <div className="mt-2">
            <SupportContacts
              contacts={contacts}
              labels={{
                phone: t('supportPhone'),
                telegram: t('supportTelegram'),
                email: t('supportEmail'),
                unavailable: t('supportUnavailable'),
              }}
            />
          </div>
        </section>
      </article>
      {voucher.status === 'redeemed' && (
        <div className="mx-auto mt-8 max-w-md">
          <ReviewForm voucherId={voucher.id} locale={locale} />
        </div>
      )}
    </AccountShell>
  );
}
