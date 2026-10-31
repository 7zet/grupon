'use client';

import { zodResolver } from '@hookform/resolvers/zod';
import { useLocale, useTranslations } from 'next-intl';
import { useRouter } from '@/i18n/navigation';
import { useState } from 'react';
import { useForm } from 'react-hook-form';
import { z } from 'zod';

const PASSWORD_MIN_LENGTH = 10;

function loginSchema() {
  return z.object({
    email: z.email({ message: 'errorEmailInvalid' }),
    password: z.string().min(1, { message: 'errorPasswordShort' }),
  });
}

function registerSchema() {
  return z.object({
    name: z.string().trim().min(1, { message: 'errorNameRequired' }),
    email: z.email({ message: 'errorEmailInvalid' }),
    password: z.string().min(PASSWORD_MIN_LENGTH, { message: 'errorPasswordShort' }),
  });
}

type Values = { name?: string; email: string; password: string };

/** Maps a stable API problem code to the translation key shown to the visitor. */
function problemKey(code: string | undefined): 'errorInvalidCredentials' | 'errorEmailTaken' | 'errorRateLimited' | 'errorGeneric' {
  switch (code) {
    case 'invalid_credentials':
      return 'errorInvalidCredentials';
    case 'email_already_registered':
      return 'errorEmailTaken';
    case 'rate_limited':
      return 'errorRateLimited';
    default:
      return 'errorGeneric';
  }
}

export function AuthForm({ mode, next }: { mode: 'login' | 'register'; next: string }) {
  const t = useTranslations('Auth');
  const locale = useLocale();
  const router = useRouter();
  const [serverError, setServerError] = useState<string | null>(null);
  const isRegister = mode === 'register';

  const form = useForm<Values>({
    resolver: zodResolver(isRegister ? registerSchema() : loginSchema()),
    defaultValues: { name: '', email: '', password: '' },
  });

  async function onSubmit(values: Values) {
    setServerError(null);
    const body = isRegister
      ? { name: values.name, email: values.email, password: values.password }
      : { email: values.email, password: values.password };
    const response = await fetch(`/api/auth/${mode}`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(
        isRegister ? { ...body, locale, currency: 'UZS' } : body,
      ),
    });
    if (!response.ok) {
      const problem = (await response.json().catch(() => ({}))) as { code?: string };
      setServerError(t(problemKey(problem.code)));
      return;
    }
    router.replace(next);
    router.refresh();
  }

  const errors = form.formState.errors;
  const fieldError = (key: keyof Values) => {
    const error = errors[key];
    return error?.message ? t(error.message as 'errorEmailInvalid') : null;
  };

  return (
    <form
      className="mx-auto mt-10 grid max-w-md gap-5"
      noValidate
      onSubmit={form.handleSubmit(onSubmit)}
    >
      {isRegister && (
        <Field id="name" label={t('name')} error={fieldError('name')}>
          <input
            id="name"
            autoComplete="name"
            className="rounded-control-ui border border-border bg-surface px-4 py-3 text-ink"
            aria-invalid={Boolean(errors.name)}
            {...form.register('name')}
          />
        </Field>
      )}
      <Field id="email" label={t('email')} error={fieldError('email')}>
        <input
          id="email"
          type="email"
          autoComplete="email"
          className="rounded-control-ui border border-border bg-surface px-4 py-3 text-ink"
          aria-invalid={Boolean(errors.email)}
          {...form.register('email')}
        />
      </Field>
      <Field
        id="password"
        label={t('password')}
        hint={isRegister ? t('passwordHint') : undefined}
        error={fieldError('password')}
      >
        <input
          id="password"
          type="password"
          autoComplete={isRegister ? 'new-password' : 'current-password'}
          className="rounded-control-ui border border-border bg-surface px-4 py-3 text-ink"
          aria-invalid={Boolean(errors.password)}
          {...form.register('password')}
        />
      </Field>

      {serverError && (
        <p role="alert" className="flex items-start gap-2 rounded-control-ui bg-error-tint p-3 text-error">
          <span aria-hidden="true">!</span>
          <span>{serverError}</span>
        </p>
      )}

      <button
        type="submit"
        disabled={form.formState.isSubmitting}
        className="rounded-control-ui bg-brand px-5 py-3 font-semibold text-surface disabled:opacity-60"
      >
        {isRegister ? t('registerSubmit') : t('loginSubmit')}
      </button>
    </form>
  );
}

function Field({
  id,
  label,
  hint,
  error,
  children,
}: {
  id: string;
  label: string;
  hint?: string | undefined;
  error: string | null;
  children: React.ReactNode;
}) {
  return (
    <div className="grid gap-2">
      <label htmlFor={id} className="font-medium text-ink">
        {label}
      </label>
      {children}
      {hint && !error && <p className="text-sm text-ink-muted">{hint}</p>}
      {error && (
        <p className="flex items-start gap-2 text-sm text-error">
          <span aria-hidden="true">!</span>
          <span>{error}</span>
        </p>
      )}
    </div>
  );
}
