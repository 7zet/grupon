import { NextResponse } from 'next/server';

/**
 * Same-origin proxy for search suggestions. The browser cannot call the Go API directly (it sends no CORS
 * headers), and suggestions are public, so this route forwards the query anonymously and never caches it.
 */
function apiBaseUrl(): string {
  return process.env.NEXT_PUBLIC_API_BASE_URL ?? 'http://localhost:8080/api/v1';
}

export async function GET(request: Request) {
  const q = new URL(request.url).searchParams.get('q') ?? '';
  if (q.trim().length < 2 || q.length > 100) {
    return NextResponse.json({ items: [] });
  }
  const upstream = await fetch(`${apiBaseUrl()}/search/suggestions?${new URLSearchParams({ q })}`, {
    headers: { Accept: 'application/json' },
    cache: 'no-store',
  });
  if (!upstream.ok) return NextResponse.json({ items: [] }, { status: 200 });
  return NextResponse.json(await upstream.json());
}
