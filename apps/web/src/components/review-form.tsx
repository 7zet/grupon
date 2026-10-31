'use client';

import { zodResolver } from '@hookform/resolvers/zod';
import { useTranslations } from 'next-intl';
import { useRouter } from '@/i18n/navigation';
import { useState } from 'react';
import { useForm } from 'react-hook-form';
import { z } from 'zod';

const schema = z.object({
  rating: z.number().int().min(1).max(5),
  text: z.string().trim().min(20).max(2000),
});
type Values = z.infer<typeof schema>;

const inputClass = 'rounded-control-ui border border-border bg-surface px-4 py-3 text-ink';

/** Written once after a redeemed voucher. The API enforces one review per voucher and the redeemed state. */
export function ReviewForm({ voucherId, locale }: { voucherId: string; locale: string }) {
  const t = useTranslations('Reviews');
  const router = useRouter();
  const [message, setMessage] = useState<string | null>(null);
  const form = useForm<Values>({
    resolver: zodResolver(schema),
    defaultValues: { rating: 5, text: '' },
  });

  async function submit(values: Values) {
    setMessage(null);
    const response = await fetch('/api/bff/reviews', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        voucher_id: voucherId,
        rating: values.rating,
        text: values.text,
        locale,
      }),
    });
    if (!response.ok) {
      const problem = (await response.json().catch(() => ({}))) as { code?: string };
      setMessage(problem.code === 'review_exists' ? t('alreadyReviewed') : t('errorGeneric'));
      return;
    }
    router.refresh();
  }

  const errors = form.formState.errors;
  return (
    <form
      className="grid w-full gap-4 rounded-card border border-border bg-surface p-5 text-left"
      noValidate
      onSubmit={form.handleSubmit(submit)}
    >
      <h3 className="font-semibold">{t('formTitle')}</h3>
      <label className="grid gap-1 text-sm">
        {t('rating')}
        <select className={inputClass} {...form.register('rating', { valueAsNumber: true })}>
          {[5, 4, 3, 2, 1].map((value) => (
            <option key={value} value={value}>
              {'★'.repeat(value)}
            </option>
          ))}
        </select>
      </label>
      <label className="grid gap-1 text-sm">
        {t('text')}
        <textarea
          className={inputClass}
          aria-invalid={Boolean(errors.text)}
          {...form.register('text')}
        />
      </label>
      {errors.text && <p className="text-sm text-error">{t('textLength')}</p>}
      {message && (
        <p
          role="alert"
          className="flex items-start gap-2 rounded-control-ui bg-error-tint p-3 text-error"
        >
          <span aria-hidden="true">!</span>
          <span>{message}</span>
        </p>
      )}
      <button
        type="submit"
        disabled={form.formState.isSubmitting}
        className="w-fit rounded-control-ui bg-brand px-5 py-3 font-semibold text-surface disabled:opacity-60"
      >
        {t('submit')}
      </button>
    </form>
  );
}
