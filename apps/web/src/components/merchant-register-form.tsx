'use client';

import { zodResolver } from '@hookform/resolvers/zod';
import { useTranslations } from 'next-intl';
import { useState } from 'react';
import { useForm } from 'react-hook-form';
import { z } from 'zod';

import { Link } from '@/i18n/navigation';

type Option = { id: string; label: string };
type City = Option & { lat: number; lng: number };

const schema = z.object({
  legalName: z.string().trim().min(2),
  displayName: z.string().trim().min(2),
  categoryId: z.string().min(1),
  cityId: z.string().min(1),
  phone: z.string().trim().min(7),
  address: z.string().trim().min(3),
});
type Values = z.infer<typeof schema>;

/** Maps API problem codes to translation keys. */
function problemKey(code: string | undefined) {
  switch (code) {
    case 'city_not_found':
      return 'errorCity' as const;
    case 'category_not_found':
      return 'errorCategory' as const;
    case 'invalid_phone':
      return 'errorPhone' as const;
    default:
      return 'errorGeneric' as const;
  }
}

const inputClass = 'rounded-control-ui border border-border bg-surface px-4 py-3 text-ink';

export function MerchantRegisterForm({ categories, cities }: { categories: Option[]; cities: City[] }) {
  const t = useTranslations('MerchantRegister');
  const [error, setError] = useState<string | null>(null);
  const [done, setDone] = useState(false);

  const form = useForm<Values>({
    resolver: zodResolver(schema),
    defaultValues: { legalName: '', displayName: '', categoryId: '', cityId: '', phone: '', address: '' },
  });

  // Coordinates default to the city centre; the map pin on the business page refines them later.
  async function submit(values: Values) {
    setError(null);
    const city = cities.find((item) => item.id === values.cityId);
    const response = await fetch('/api/bff/merchants', {
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
      }),
    });
    if (!response.ok) {
      const problem = (await response.json().catch(() => ({}))) as { code?: string };
      setError(t(problemKey(problem.code)));
      return;
    }
    setDone(true);
  }

  if (done) {
    return (
      <div className="mt-10 grid gap-5">
        <p role="status" className="text-lead text-ink">
          {t('success')}
        </p>
        <Link href="/merchant" className="font-semibold text-brand">
          {t('goToBusiness')}
        </Link>
      </div>
    );
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
        {t('submit')}
      </button>
    </form>
  );
}
