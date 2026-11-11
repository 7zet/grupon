import 'server-only';

import type { components } from '@dealo/api-client';
import { cache } from 'react';
import { cookies } from 'next/headers';
import { redirect } from 'next/navigation';

/** Thrown when the session is missing or rejected, so the caller can send the visitor to sign in. */
export class SessionRequiredError extends Error {}

function apiBaseUrl(): string {
  return process.env.NEXT_PUBLIC_API_BASE_URL ?? 'http://localhost:8080/api/v1';
}

/**
 * Calls an authenticated API operation on behalf of the signed-in visitor. Account data is personal,
 * so responses are never cached.
 */
export async function accountGet<T>(path: string): Promise<T> {
  const token = (await cookies()).get('access_token')?.value;
  if (!token) throw new SessionRequiredError();

  const response = await fetch(`${apiBaseUrl()}${path}`, {
    headers: { Authorization: `Bearer ${token}`, Accept: 'application/json' },
    cache: 'no-store',
  });
  if (response.status === 401) throw new SessionRequiredError();
  if (!response.ok) throw new Error(`account request ${path} failed: ${response.status}`);
  return (await response.json()) as T;
}

/** Sends the visitor to sign in and returns to the page they asked for. */
export function requireSession(locale: string, returnPath: string): never {
  redirect(`/${locale}/login?next=${encodeURIComponent(returnPath)}`);
}

/** Runs a visitor-scoped load and converts a missing or rejected session into a sign-in redirect. */
export async function loadForVisitor<T>(
  locale: string,
  returnPath: string,
  load: () => Promise<T>,
): Promise<T> {
  try {
    return await load();
  } catch (error) {
    if (error instanceof SessionRequiredError) requireSession(locale, returnPath);
    throw error;
  }
}

/** Like accountGet, but a missing resource resolves to null so the caller can show a not-found page. */
export async function accountGetOrNull<T>(path: string): Promise<T | null> {
  const token = (await cookies()).get('access_token')?.value;
  if (!token) throw new SessionRequiredError();

  const response = await fetch(`${apiBaseUrl()}${path}`, {
    headers: { Authorization: `Bearer ${token}`, Accept: 'application/json' },
    cache: 'no-store',
  });
  if (response.status === 401) throw new SessionRequiredError();
  if (response.status === 404) return null;
  if (!response.ok) throw new Error(`account request ${path} failed: ${response.status}`);
  return (await response.json()) as T;
}

/** The signed-in user, requested once per page render (layouts and pages share the result). */
export const currentUser = cache(() => accountGetOrNull<components['schemas']['User']>('/me'));

/** The signed-in user, or null for an anonymous or expired session. Never redirects. */
export async function optionalUser() {
  try {
    return await currentUser();
  } catch (error) {
    if (error instanceof SessionRequiredError) return null;
    return null;
  }
}
