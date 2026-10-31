import { createHash, timingSafeEqual } from 'node:crypto';
import { revalidateTag } from 'next/cache';
import { NextResponse } from 'next/server';
import { z } from 'zod';

import { CATALOG_CACHE_TAG } from '@grupon/api-client';

/** Tags the Go API is allowed to revalidate. Anything else is rejected. */
const KNOWN_TAGS = [CATALOG_CACHE_TAG] as const;

const bodySchema = z.object({
  tags: z.array(z.enum(KNOWN_TAGS)).min(1).max(KNOWN_TAGS.length),
});

const SECRET_HEADER = 'x-revalidate-secret';

/** Compares digests so the check is constant-time and independent of input length. */
function secretMatches(provided: string | null): boolean {
  const expected = process.env.WEB_REVALIDATE_SECRET;
  if (!expected || !provided) return false;
  const a = createHash('sha256').update(provided).digest();
  const b = createHash('sha256').update(expected).digest();
  return timingSafeEqual(a, b);
}

export async function POST(request: Request) {
  if (!secretMatches(request.headers.get(SECRET_HEADER))) {
    return NextResponse.json({ code: 'unauthorized' }, { status: 401 });
  }

  const parsed = bodySchema.safeParse(await request.json().catch(() => null));
  if (!parsed.success) {
    return NextResponse.json({ code: 'invalid_body' }, { status: 400 });
  }

  for (const tag of parsed.data.tags) {
    revalidateTag(tag, 'max');
  }

  return NextResponse.json({ revalidated: parsed.data.tags }, { status: 202 });
}
