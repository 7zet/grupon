import { cookies } from 'next/headers';
import { NextResponse } from 'next/server';

/**
 * Backend-for-frontend proxy for signed-in browser code. The browser sends its session cookie to this
 * same-origin route; the route turns it into a bearer token for the Go API. Only the operations listed
 * here are reachable, so the proxy is not an open relay.
 */
const UUID = '[0-9a-f-]{36}';

const ALLOWED: { method: string; pattern: RegExp }[] = [
  { method: 'POST', pattern: /^orders$/ },
  { method: 'POST', pattern: /^merchants$/ },
  { method: 'PATCH', pattern: /^merchant\/business$/ },
  { method: 'POST', pattern: /^merchant\/locations$/ },
  { method: 'PATCH', pattern: new RegExp(`^merchant\\/locations\\/${UUID}$`) },
  { method: 'DELETE', pattern: new RegExp(`^merchant\\/locations\\/${UUID}$`) },
  { method: 'POST', pattern: new RegExp(`^admin\\/deals\\/${UUID}\\/(approve|reject)$`) },
  { method: 'POST', pattern: /^merchant\/vouchers\/lookup$/ },
  { method: 'POST', pattern: new RegExp(`^merchant\\/vouchers\\/${UUID}\\/redeem$`) },
  { method: 'POST', pattern: new RegExp(`^admin\/review-reports\/${UUID}\/dismiss$`) },
  { method: 'POST', pattern: new RegExp(`^admin\/reviews\/${UUID}\/hide$`) },
  { method: 'POST', pattern: /^reviews$/ },
  { method: 'PATCH', pattern: new RegExp(`^reviews\\/${UUID}$`) },
  { method: 'DELETE', pattern: new RegExp(`^reviews\\/${UUID}$`) },
  { method: 'POST', pattern: new RegExp(`^reviews\\/${UUID}\\/report$`) },
  { method: 'POST', pattern: /^merchant\/deals$/ },
  { method: 'PUT', pattern: new RegExp(`^merchant\\/deals\\/${UUID}$`) },
  { method: 'DELETE', pattern: new RegExp(`^merchant\\/deals\\/${UUID}$`) },
  { method: 'POST', pattern: new RegExp(`^merchant\\/deals\\/${UUID}\\/submit$`) },
  { method: 'POST', pattern: /^merchant\/media$/ },
  { method: 'PUT', pattern: new RegExp(`^merchant\\/media\\/${UUID}\\/content$`) },
  { method: 'DELETE', pattern: new RegExp(`^merchant\\/media\\/${UUID}$`) },
  { method: 'POST', pattern: new RegExp(`^orders\\/${UUID}\\/pay$`) },
  { method: 'POST', pattern: new RegExp(`^payments\\/mock\\/${UUID}\\/complete$`) },
];

/** Media bytes are binary: they must be forwarded as-is, never decoded as text. */
const BINARY_UPLOAD = new RegExp(`^merchant\\/media\\/${UUID}\\/content$`);

function apiBaseUrl(): string {
  return process.env.NEXT_PUBLIC_API_BASE_URL ?? 'http://localhost:8080/api/v1';
}

async function forward(request: Request, context: { params: Promise<{ path: string[] }> }) {
  const { path } = await context.params;
  const route = path.join('/');
  const allowed = ALLOWED.some(
    (rule) => rule.method === request.method && rule.pattern.test(route),
  );
  if (!allowed) {
    return NextResponse.json({ code: 'not_found' }, { status: 404 });
  }

  const token = (await cookies()).get('access_token')?.value;
  if (!token) {
    return NextResponse.json({ code: 'unauthenticated' }, { status: 401 });
  }

  const headers: Record<string, string> = {
    Authorization: `Bearer ${token}`,
    Accept: 'application/json',
  };
  const binary = BINARY_UPLOAD.test(route);
  headers['Content-Type'] = binary
    ? (request.headers.get('content-type') ?? 'application/octet-stream')
    : 'application/json';
  const idempotencyKey = request.headers.get('idempotency-key');
  if (idempotencyKey) headers['Idempotency-Key'] = idempotencyKey;

  const upstream = await fetch(`${apiBaseUrl()}/${route}`, {
    method: request.method,
    headers,
    body: binary ? await request.arrayBuffer() : await request.text(),
    cache: 'no-store',
  });
  // A 204 response must not carry a body; the Response constructor rejects one.
  if (upstream.status === 204) return new NextResponse(null, { status: 204 });
  return new NextResponse(await upstream.text(), {
    status: upstream.status,
    headers: { 'Content-Type': upstream.headers.get('content-type') ?? 'application/json' },
  });
}

export const POST = forward;
export const PUT = forward;
export const PATCH = forward;
export const DELETE = forward;
