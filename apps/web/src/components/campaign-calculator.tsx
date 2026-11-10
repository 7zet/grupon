'use client';

import { formatMoney } from '@dealo/ui';
import type { Currency, Locale } from '@dealo/config';
import { useTranslations } from 'next-intl';
import { useState } from 'react';

const inputClass = 'rounded-control-ui border border-border bg-surface px-4 py-3 text-ink';

/**
 * Shows what a merchant earns from a deal before it is created. Pure arithmetic in so'm; nothing is sent to the API.
 * Money is shown through formatMoney, which expects tiyin, so every so'm figure is multiplied by 100 first.
 */
export function CampaignCalculator({ locale }: { locale: Locale }) {
  const t = useTranslations('MerchantPanel');
  const currency: Currency = 'UZS';
  const [original, setOriginal] = useState(100000);
  const [deal, setDeal] = useState(50000);
  const [capacity, setCapacity] = useState(100);
  const [commission, setCommission] = useState(15);
  const [cost, setCost] = useState(20000);

  const valid = original > 0 && deal > 0 && deal < original && capacity > 0 && commission >= 0 && commission < 100;
  const discount = valid ? Math.round((1 - deal / original) * 100) : 0;
  const perVoucher = valid ? deal * (1 - commission / 100) : 0;
  const totalReceive = perVoucher * capacity;
  const netPerVoucher = perVoucher - cost;
  const netTotal = netPerVoucher * capacity;
  const customersPay = deal * capacity;
  const show = (som: number) => formatMoney(Math.round(som * 100), locale, currency);

  return (
    <div className="grid gap-6">
      <p className="text-sm text-ink-muted">{t('calcHint')}</p>
      <div className="grid max-w-2xl gap-4 sm:grid-cols-2">
        <NumberField label={t('calcOriginal')} value={original} onChange={setOriginal} className={inputClass} />
        <NumberField label={t('calcDeal')} value={deal} onChange={setDeal} className={inputClass} />
        <NumberField label={t('calcCapacity')} value={capacity} onChange={setCapacity} className={inputClass} />
        <NumberField label={t('calcCommission')} value={commission} onChange={setCommission} className={inputClass} />
        <NumberField label={t('calcCost')} value={cost} onChange={setCost} className={inputClass} />
      </div>
      {valid && (
        <dl className="grid max-w-2xl gap-3 rounded-card border border-border p-5 text-sm sm:grid-cols-2">
          <Row label={t('calcDiscount')} value={`${discount}%`} />
          <Row label={t('calcPerVoucher')} value={show(perVoucher)} />
          <Row label={t('calcCustomerPays')} value={show(customersPay)} />
          <Row label={t('calcTotal')} value={show(totalReceive)} />
          <Row label={t('calcNet')} value={show(netPerVoucher)} />
          <Row label={t('calcNetTotal')} value={show(netTotal)} />
        </dl>
      )}
      {valid && netPerVoucher < 0 && (
        <p role="alert" className="max-w-2xl rounded-control-ui bg-error-tint p-3 text-error">
          {t('calcNegative')}
        </p>
      )}
    </div>
  );
}

function NumberField({
  label,
  value,
  onChange,
  className,
}: {
  label: string;
  value: number;
  onChange: (value: number) => void;
  className: string;
}) {
  return (
    <label className="grid gap-2 text-sm font-semibold">
      {label}
      <input
        type="number"
        min={0}
        className={className}
        value={value}
        onChange={(e) => onChange(Number(e.target.value))}
      />
    </label>
  );
}

function Row({ label, value }: { label: string; value: string }) {
  return (
    <div className="flex justify-between gap-4">
      <dt className="text-ink-muted">{label}</dt>
      <dd className="font-semibold text-ink">{value}</dd>
    </div>
  );
}
