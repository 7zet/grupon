'use client';

import { zodResolver } from '@hookform/resolvers/zod';
import { useTranslations } from 'next-intl';
import { useState } from 'react';
import { useForm } from 'react-hook-form';
import { z } from 'zod';

import { useRouter } from '@/i18n/navigation';

type Option = { id: string; label: string };
type City = Option & { lat: number; lng: number };

const schema = z.object({
  legalName: z.string().trim().min(2),
  displayName: z.string().trim().min(2),
  categoryId: z.string().min(1),
  cityId: z.string().min(1),
  phone: z.string().trim().min(7),
  address: z.string().trim().min(3),
  stir: z.string().regex(/^[0-9]{9}$/),
  orgForm: z.enum(['mchj', 'yatt']),
  taxRegime: z.enum(['vat_payer', 'turnover_tax']),
  bankMfo: z.string().regex(/^[0-9]{5}$/),
  bankAccount: z.string().regex(/^[0-9]{20}$/),
  responsibleName: z.string().trim().min(2),
  contractAccepted: z.literal(true),
});
type Values = z.infer<typeof schema>;

/** The offer text version the merchant accepts; it is stored with the acceptance time. */
const CONTRACT_VERSION = 'v1.1';

/** Maps API problem codes to translation keys. */
function problemKey(code: string | undefined) {
  switch (code) {
    case 'city_not_found':
      return 'errorCity' as const;
    case 'category_not_found':
      return 'errorCategory' as const;
    case 'invalid_phone':
      return 'errorPhone' as const;
    case 'invalid_stir':
      return 'errorStir' as const;
    case 'stir_taken':
      return 'errorStirTaken' as const;
    case 'invalid_org_form':
      return 'errorOrgForm' as const;
    case 'invalid_tax_regime':
      return 'errorTaxRegime' as const;
    case 'invalid_mfo':
      return 'errorMfo' as const;
    case 'invalid_bank_account':
      return 'errorAccount' as const;
    case 'contract_not_accepted':
      return 'errorContract' as const;
    case 'already_submitted':
      return 'errorAlreadySubmitted' as const;
    default:
      return 'errorGeneric' as const;
  }
}

const inputClass = 'rounded-control-ui border border-border bg-surface px-4 py-3 text-ink';

export function MerchantRegisterForm({ categories, cities }: { categories: Option[]; cities: City[] }) {
  const t = useTranslations('MerchantRegister');
  const router = useRouter();
  const [error, setError] = useState<string | null>(null);

  const form = useForm<Values>({
    resolver: zodResolver(schema),
    defaultValues: {
      legalName: '',
      displayName: '',
      categoryId: '',
      cityId: '',
      phone: '',
      address: '',
      stir: '',
      orgForm: 'mchj',
      taxRegime: 'vat_payer',
      bankMfo: '',
      bankAccount: '',
      responsibleName: '',
      contractAccepted: undefined as unknown as true,
    },
  });

  // Registration and review submission are one user step: the merchant only sees the review status.
  // Coordinates default to the city centre; the map pin on the business page refines them later.
  async function submit(values: Values) {
    setError(null);
    const city = cities.find((item) => item.id === values.cityId);
    const registered = await fetch('/api/bff/merchants', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        legal_name: values.legalName,
        display_name: values.displayName,
        category_id: values.categoryId,
        city_id: values.cityId,
        phone: values.phone,
        address: values.address,
        lat: city?.lat ?? 0,
        lng: city?.lng ?? 0,
        stir: values.stir,
        org_form: values.orgForm,
        tax_regime: values.taxRegime,
        bank_mfo: values.bankMfo,
        bank_account: values.bankAccount,
        responsible_name: values.responsibleName,
        contract_version: CONTRACT_VERSION,
        contract_accepted: values.contractAccepted,
      }),
    });
    if (!registered.ok) {
      const problem = (await registered.json().catch(() => ({}))) as { code?: string };
      setError(t(problemKey(problem.code)));
      return;
    }
    // Documents and the final submission happen on the review page, which shows the status too.
    router.push('/merchant/review');
  }

  const errors = form.formState.errors;
  return (
    <form className="mt-10 grid max-w-xl gap-5" noValidate onSubmit={form.handleSubmit(submit)}>
      <label className="grid gap-1" htmlFor="legalName">
        {t('legalName')}
        <input id="legalName" className={inputClass} aria-invalid={Boolean(errors.legalName)} {...form.register('legalName')} />
      </label>
      <label className="grid gap-1" htmlFor="displayName">
        {t('displayName')}
        <input id="displayName" className={inputClass} aria-invalid={Boolean(errors.displayName)} {...form.register('displayName')} />
      </label>
      <label className="grid gap-1" htmlFor="stir">
        {t('stir')}
        <input
          id="stir"
          inputMode="numeric"
          maxLength={9}
          className={inputClass}
          aria-invalid={Boolean(errors.stir)}
          {...form.register('stir')}
        />
      </label>
      <label className="grid gap-1" htmlFor="orgForm">
        {t('orgForm')}
        <select id="orgForm" className={inputClass} {...form.register('orgForm')}>
          <option value="mchj">{t('orgFormMchj')}</option>
          <option value="yatt">{t('orgFormYatt')}</option>
        </select>
      </label>
      <label className="grid gap-1" htmlFor="taxRegime">
        {t('taxRegime')}
        <select id="taxRegime" className={inputClass} {...form.register('taxRegime')}>
          <option value="vat_payer">{t('taxVat')}</option>
          <option value="turnover_tax">{t('taxTurnover')}</option>
        </select>
      </label>
      <label className="grid gap-1" htmlFor="bankMfo">
        {t('bankMfo')}
        <input
          id="bankMfo"
          inputMode="numeric"
          maxLength={5}
          className={inputClass}
          aria-invalid={Boolean(errors.bankMfo)}
          {...form.register('bankMfo')}
        />
      </label>
      <label className="grid gap-1" htmlFor="bankAccount">
        {t('bankAccount')}
        <input
          id="bankAccount"
          inputMode="numeric"
          maxLength={20}
          className={inputClass}
          aria-invalid={Boolean(errors.bankAccount)}
          {...form.register('bankAccount')}
        />
      </label>
      <label className="grid gap-1" htmlFor="responsibleName">
        {t('responsibleName')}
        <input
          id="responsibleName"
          className={inputClass}
          aria-invalid={Boolean(errors.responsibleName)}
          {...form.register('responsibleName')}
        />
      </label>
      <label className="grid gap-1" htmlFor="categoryId">
        {t('category')}
        <select id="categoryId" className={inputClass} {...form.register('categoryId')}>
          <option value="">{t('choose')}</option>
          {categories.map((item) => (
            <option key={item.id} value={item.id}>
              {item.label}
            </option>
          ))}
        </select>
      </label>
      <label className="grid gap-1" htmlFor="cityId">
        {t('city')}
        <select id="cityId" className={inputClass} {...form.register('cityId')}>
          <option value="">{t('choose')}</option>
          {cities.map((item) => (
            <option key={item.id} value={item.id}>
              {item.label}
            </option>
          ))}
        </select>
      </label>
      <label className="grid gap-1" htmlFor="phone">
        {t('phone')}
        <input id="phone" type="tel" className={inputClass} aria-invalid={Boolean(errors.phone)} {...form.register('phone')} />
      </label>
      <label className="grid gap-1" htmlFor="address">
        {t('address')}
        <input id="address" className={inputClass} aria-invalid={Boolean(errors.address)} {...form.register('address')} />
      </label>
      <label className="flex items-start gap-3 text-small" htmlFor="contractAccepted">
        <input id="contractAccepted" type="checkbox" className="mt-1" {...form.register('contractAccepted')} />
        <span>{t('contract')}</span>
      </label>

      {error && (
        <p role="alert" className="flex items-start gap-2 rounded-control-ui bg-error-tint p-3 text-error">
          <span aria-hidden="true">!</span>
          <span>{error}</span>
        </p>
      )}

      <button
        type="submit"
        disabled={form.formState.isSubmitting}
        className="rounded-control-ui bg-brand px-5 py-3 font-semibold text-surface disabled:opacity-60"
      >
        {t('submitForReview')}
      </button>
    </form>
  );
}
