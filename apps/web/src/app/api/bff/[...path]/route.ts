import { cookies } from 'next/headers';
import { NextResponse } from 'next/server';

/**
 * Backend-for-frontend proxy for signed-in browser code. The browser sends its session cookie to this
 * same-origin route; the route turns it into a bearer token for the Go API. Only the operations listed
 * here are reachable, so the proxy is not an open relay.
 */
const UUID = '[0-9a-f-]{36}';

const ALLOWED: { method: string; pattern: RegExp }[] = [
  { method: 'GET', pattern: /^me$/ },
  { method: 'POST', pattern: /^orders$/ },
  { method: 'POST', pattern: /^merchants$/ },
  { method: 'POST', pattern: /^merchant\/submit$/ },
  { method: 'GET', pattern: /^me\/notifications$/ },
  { method: 'POST', pattern: new RegExp(`^me\\/vouchers\\/${UUID}\\/dispute$`) },
  { method: 'GET', pattern: /^merchant\/business$/ },
  { method: 'GET', pattern: /^merchant\/documents$/ },
  { method: 'PUT', pattern: /^merchant\/documents\/(certificate|bank_letter|price_proof)$/ },
  { method: 'GET', pattern: /^admin\/merchants$/ },
  { method: 'GET', pattern: /^admin\/finance\/queue$/ },
  { method: 'POST', pattern: new RegExp(`^admin\\/payouts\\/${UUID}\\/(approve|reconcile)$`) },
  { method: 'POST', pattern: new RegExp(`^admin\\/refunds\\/${UUID}\\/(approve|paid)$`) },
  { method: 'POST', pattern: new RegExp(`^admin\\/disputes\\/${UUID}\\/resolve$`) },
  { method: 'POST', pattern: new RegExp(`^admin\\/merchants\\/${UUID}\\/(start-review|approve|reject)$`) },
  { method: 'GET', pattern: new RegExp(`^admin\\/merchants\\/${UUID}\\/documents\\/(certificate|bank_letter|price_proof)$`) },
  { method: 'PATCH', pattern: /^merchant\/business$/ },
  { method: 'GET', pattern: /^merchant\/payouts$/ },
  { method: 'GET', pattern: /^merchant\/customers$/ },
  { method: 'GET', pattern: /^merchant\/analytics$/ },
  { method: 'GET', pattern: /^merchant\/reviews$/ },
  { method: 'PUT', pattern: new RegExp(`^merchant\\/reviews\\/${UUID}\\/reply$`) },
  { method: 'GET', pattern: /^merchant\/orders$/ },
  { method: 'GET', pattern: /^merchant\/settings$/ },
  { method: 'PATCH', pattern: /^merchant\/settings$/ },
  { method: 'GET', pattern: /^merchant\/staff$/ },
  { method: 'POST', pattern: /^merchant\/staff$/ },
  { method: 'PATCH', pattern: new RegExp(`^merchant\\/staff\\/${UUID}$`) },
  { method: 'DELETE', pattern: new RegExp(`^merchant\\/staff\\/${UUID}$`) },
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
const BINARY_UPLOAD = new RegExp(
  `^(merchant\\/media\\/${UUID}\\/content|merchant\\/documents\\/(certificate|bank_letter|price_proof))$`,
);

/** Business documents are downloaded as files, so the bytes must not be decoded as text. */
const BINARY_DOWNLOAD = new RegExp(`^admin\\/merchants\\/${UUID}\\/documents\\/(certificate|bank_letter|price_proof)$`);

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
    body:
      request.method === 'GET'
        ? null
        : binary
          ? await request.arrayBuffer()
          : await request.text(),
    cache: 'no-store',
  });
  // A 204 response must not carry a body; the Response constructor rejects one.
  if (upstream.status === 204) return new NextResponse(null, { status: 204 });
  if (BINARY_DOWNLOAD.test(route) && upstream.ok) {
    return new NextResponse(await upstream.arrayBuffer(), {
      status: upstream.status,
      headers: {
        'Content-Type': upstream.headers.get('content-type') ?? 'application/octet-stream',
        'Content-Disposition': 'attachment',
        'Cache-Control': 'private, no-store',
      },
    });
  }
  return new NextResponse(await upstream.text(), {
    status: upstream.status,
    headers: { 'Content-Type': upstream.headers.get('content-type') ?? 'application/json' },
  });
}

export const GET = forward;
export const POST = forward;
export const PUT = forward;
export const PATCH = forward;
export const DELETE = forward;
