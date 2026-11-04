'use client';

import { zodResolver } from '@hookform/resolvers/zod';
import { useTranslations } from 'next-intl';
import { useRouter } from '@/i18n/navigation';
import { useState } from 'react';
import { useForm } from 'react-hook-form';
import { z } from 'zod';

/** Uzbekistan mobile numbers: +998 followed by nine digits. The prefix is fixed in the UI. */
const phoneSchema = z.object({
  digits: z.string().regex(/^[0-9]{9}$/, { message: 'errorPhoneInvalid' }),
});
const codeSchema = z.object({
  code: z.string().regex(/^[0-9]{6}$/, { message: 'errorCodeInvalid' }),
});
type PhoneValues = z.infer<typeof phoneSchema>;
type CodeValues = z.infer<typeof codeSchema>;

const PHONE_PREFIX = '+998';

/** Maps a stable API problem code to the translation key shown to the visitor. */
function problemKey(code: string | undefined) {
  switch (code) {
    case 'invalid_phone':
      return 'errorPhoneInvalid' as const;
    case 'otp_cooldown':
      return 'errorOtpCooldown' as const;
    case 'otp_invalid':
      return 'errorOtpInvalid' as const;
    case 'otp_attempts_exceeded':
      return 'errorOtpAttempts' as const;
    case 'otp_unavailable':
      return 'errorOtpUnavailable' as const;
    case 'rate_limited':
      return 'errorRateLimited' as const;
    default:
      return 'errorGeneric' as const;
  }
}

/** Phone sign-in: send a code, then enter it. The account is created on first use. */
export function AuthForm({ next }: { mode?: 'login' | 'register'; next: string }) {
  const t = useTranslations('Auth');
  const router = useRouter();
  const [phone, setPhone] = useState<string | null>(null);
  const [devCode, setDevCode] = useState<string | null>(null);
  const [serverError, setServerError] = useState<string | null>(null);

  const phoneForm = useForm<PhoneValues>({
    resolver: zodResolver(phoneSchema),
    defaultValues: { digits: '' },
  });
  const codeForm = useForm<CodeValues>({
    resolver: zodResolver(codeSchema),
    defaultValues: { code: '' },
  });

  async function sendCode(values: PhoneValues) {
    setServerError(null);
    const full = `${PHONE_PREFIX}${values.digits}`;
    const response = await fetch('/api/auth/phone-code', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ phone: full }),
    });
    if (!response.ok) {
      const problem = (await response.json().catch(() => ({}))) as { code?: string };
      setServerError(t(problemKey(problem.code)));
      return;
    }
    const sent = (await response.json().catch(() => ({}))) as { dev_code?: string };
    setPhone(full);
    setDevCode(sent.dev_code ?? null);
  }

  async function verifyCode(values: CodeValues) {
    setServerError(null);
    const response = await fetch('/api/auth/phone-verify', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ phone, code: values.code }),
    });
    if (!response.ok) {
      const problem = (await response.json().catch(() => ({}))) as { code?: string };
      setServerError(t(problemKey(problem.code)));
      return;
    }
    router.replace(next);
    router.refresh();
  }

  if (phone === null) {
    const errors = phoneForm.formState.errors;
    return (
      <form
        className="mx-auto mt-10 grid max-w-md gap-5"
        noValidate
        onSubmit={phoneForm.handleSubmit(sendCode)}
      >
        <div className="grid gap-2">
          <label htmlFor="phone" className="font-medium text-ink">
            {t('phoneLabel')}
          </label>
          <div className="flex items-center gap-2">
            <span className="rounded-control-ui border border-border bg-page px-4 py-3 text-ink-muted">
              {PHONE_PREFIX}
            </span>
            <input
              id="phone"
              type="tel"
              inputMode="numeric"
              autoComplete="tel-national"
              maxLength={9}
              placeholder="901234567"
              className="min-w-0 flex-1 rounded-control-ui border border-border bg-surface px-4 py-3 text-ink"
              aria-invalid={Boolean(errors.digits)}
              {...phoneForm.register('digits')}
            />
          </div>
          {errors.digits && (
            <p className="text-sm text-error">{t(errors.digits.message as 'errorPhoneInvalid')}</p>
          )}
        </div>
        {serverError && <ErrorBox message={serverError} />}
        <button
          type="submit"
          disabled={phoneForm.formState.isSubmitting}
          className="rounded-control-ui bg-brand px-5 py-3 font-semibold text-surface disabled:opacity-60"
        >
          {t('sendCode')}
        </button>
      </form>
    );
  }

  const errors = codeForm.formState.errors;
  return (
    <form
      className="mx-auto mt-10 grid max-w-md gap-5"
      noValidate
      onSubmit={codeForm.handleSubmit(verifyCode)}
    >
      <p className="text-ink-muted">{t('codeSent', { phone })}</p>
      {devCode && <p className="text-sm text-ink-muted">{t('devCodeHint', { code: devCode })}</p>}
      <div className="grid gap-2">
        <label htmlFor="code" className="font-medium text-ink">
          {t('codeLabel')}
        </label>
        <input
          id="code"
          inputMode="numeric"
          autoComplete="one-time-code"
          maxLength={6}
          className="rounded-control-ui border border-border bg-surface px-4 py-3 text-center text-lead tracking-widest text-ink"
          aria-invalid={Boolean(errors.code)}
          {...codeForm.register('code')}
        />
        {errors.code && (
          <p className="text-sm text-error">{t(errors.code.message as 'errorCodeInvalid')}</p>
        )}
      </div>
      {serverError && <ErrorBox message={serverError} />}
      <button
        type="submit"
        disabled={codeForm.formState.isSubmitting}
        className="rounded-control-ui bg-brand px-5 py-3 font-semibold text-surface disabled:opacity-60"
      >
        {t('verifyCode')}
      </button>
      <button type="button" className="text-sm text-brand" onClick={() => setPhone(null)}>
        {t('changePhone')}
      </button>
    </form>
  );
}

function ErrorBox({ message }: { message: string }) {
  return (
    <p
      role="alert"
      className="flex items-start gap-2 rounded-control-ui bg-error-tint p-3 text-error"
    >
      <span aria-hidden="true">!</span>
      <span>{message}</span>
    </p>
  );
}
