import { NextResponse } from 'next/server';

/**
 * Same-origin proxy for sign-in. The Go API sets httpOnly session cookies on its own origin; this
 * route forwards the request and re-issues those cookies on the web origin so the browser keeps them
 * for every page on this site. Only the phone sign-in steps are reachable.
 */
const API_PATHS: Record<string, string> = {
  'phone-code': '/auth/phone/code',
  'phone-verify': '/auth/phone/verify',
};

function apiBaseUrl(): string {
  return process.env.NEXT_PUBLIC_API_BASE_URL ?? 'http://localhost:8080/api/v1';
}

export async function POST(request: Request, context: { params: Promise<{ action: string }> }) {
  const { action } = await context.params;
  const path = API_PATHS[action];
  if (!path) {
    return NextResponse.json({ code: 'not_found' }, { status: 404 });
  }

  const body = await request.text();
  const upstream = await fetch(`${apiBaseUrl()}${path}`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json', Accept: 'application/json' },
    body,
    cache: 'no-store',
  });

  const response = new NextResponse(await upstream.text(), {
    status: upstream.status,
    headers: { 'Content-Type': upstream.headers.get('content-type') ?? 'application/json' },
  });

  // Cookies issued for the API path are re-scoped to the whole web site.
  for (const cookie of upstream.headers.getSetCookie()) {
    response.headers.append('Set-Cookie', cookie.replace(/;\s*Path=[^;]*/i, '; Path=/'));
  }
  return response;
}
