'use client';

import { zodResolver } from '@hookform/resolvers/zod';
import { useTranslations } from 'next-intl';
import { useRouter } from '@/i18n/navigation';
import { useState } from 'react';
import { useForm } from 'react-hook-form';
import { z } from 'zod';
import QRCode from 'qrcode';
import Image from 'next/image';

/** Uzbekistan mobile numbers: +998 followed by nine digits. The prefix is fixed in the UI. */
const phoneSchema = z.object({
  digits: z.string().regex(/^[0-9]{9}$/, { message: 'errorPhoneInvalid' }),
});
const codeSchema = z.object({
  code: z.string().regex(/^[0-9]{6}$/, { message: 'errorCodeInvalid' }),
});
type PhoneValues = z.infer<typeof phoneSchema>;
type CodeValues = z.infer<typeof codeSchema>;
const twoFactorSchema = z.object({ code: z.string().min(6).max(32) });
type TwoFactorValues = z.infer<typeof twoFactorSchema>;

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
  const [challenge, setChallenge] = useState<string | null>(null);
  const [enrolling, setEnrolling] = useState(false);
  const [qr, setQr] = useState<string | null>(null);
  const [secret, setSecret] = useState<string | null>(null);
  const [recoveryCodes, setRecoveryCodes] = useState<string[] | null>(null);

  const phoneForm = useForm<PhoneValues>({
    resolver: zodResolver(phoneSchema),
    defaultValues: { digits: '' },
  });
  const codeForm = useForm<CodeValues>({
    resolver: zodResolver(codeSchema),
    defaultValues: { code: '' },
  });
  const twoFactorForm = useForm<TwoFactorValues>({resolver: zodResolver(twoFactorSchema), defaultValues: {code: ''}});

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
    if (response.status === 202) {
      const pending = (await response.json()) as {challenge_token: string; enrollment_required: boolean};
      setChallenge(pending.challenge_token);
      setEnrolling(pending.enrollment_required);
      if (pending.enrollment_required) {
        const setupResponse = await fetch('/api/auth/2fa-enroll', {method: 'POST', headers: {'Content-Type': 'application/json'}, body: JSON.stringify({challenge_token: pending.challenge_token})});
        if (!setupResponse.ok) { setServerError(t('errorGeneric')); return; }
        const setup = (await setupResponse.json()) as {otpauth_uri: string; secret: string};
        setSecret(setup.secret);
        setQr(await QRCode.toDataURL(setup.otpauth_uri, {margin: 1, width: 240}));
      }
      return;
    }
    router.replace(next);
    router.refresh();
  }

  async function verifyTwoFactor(values: TwoFactorValues) {
    setServerError(null);
    const action = enrolling ? '2fa-enable' : '2fa-verify';
    const response = await fetch(`/api/auth/${action}`, {method: 'POST', headers: {'Content-Type': 'application/json'}, body: JSON.stringify({challenge_token: challenge, code: values.code})});
    if (!response.ok) { setServerError(t('errorTwoFactorInvalid')); return; }
    if (enrolling) {
      const enabled = (await response.json()) as {recovery_codes: string[]};
      setRecoveryCodes(enabled.recovery_codes);
      return;
    }
    router.replace(next); router.refresh();
  }

  if (recoveryCodes) return <div className="mx-auto mt-10 grid max-w-md gap-4"><h2 className="text-title font-bold">{t('recoveryTitle')}</h2><p className="text-ink-muted">{t('recoveryHint')}</p><ul className="grid grid-cols-2 gap-2 rounded-control-ui bg-surface p-4 font-mono">{recoveryCodes.map((item) => <li key={item}>{item}</li>)}</ul><button className="rounded-control-ui bg-brand px-5 py-3 font-semibold text-surface" onClick={() => {router.replace(next); router.refresh();}}>{t('continue')}</button></div>;

  if (challenge) return <form className="mx-auto mt-10 grid max-w-md gap-5" onSubmit={twoFactorForm.handleSubmit(verifyTwoFactor)}><h2 className="text-title font-bold">{enrolling ? t('setupTwoFactor') : t('twoFactorTitle')}</h2>{qr && <Image src={qr} alt={t('qrAlt')} width={240} height={240} unoptimized className="mx-auto" />}{secret && <code className="break-all rounded-control-ui bg-surface p-3 text-center">{secret}</code>}<label htmlFor="two-factor-code" className="font-medium text-ink">{t('twoFactorCode')}</label><input id="two-factor-code" autoComplete="one-time-code" className="rounded-control-ui border border-border bg-surface px-4 py-3 text-center tracking-widest" {...twoFactorForm.register('code')} />{serverError && <ErrorBox message={serverError} />}<button type="submit" className="rounded-control-ui bg-brand px-5 py-3 font-semibold text-surface">{t('verifyCode')}</button></form>;

  if (phone === null) {
    const errors = phoneForm.formState.errors;
    return (
      <>
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
      </>
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
