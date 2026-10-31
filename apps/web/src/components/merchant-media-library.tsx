'use client';

import { useTranslations } from 'next-intl';
import { useRouter } from '@/i18n/navigation';
import { useState } from 'react';

export type MediaAsset = { id: string; url: string; content_type: string; size: number };

const MAX_BYTES = 10 * 1024 * 1024;
const ALLOWED_TYPES = ['image/jpeg', 'image/png', 'image/webp'];

/** Upload URLs from the API start with /api/v1; the browser reaches them through the same-origin BFF. */
function bffPath(uploadUrl: string) {
  return uploadUrl.replace(/^.*\/api\/v1\//, '/api/bff/');
}

export function MerchantMediaLibrary({ assets }: { assets: MediaAsset[] }) {
  const t = useTranslations('Deals');
  const router = useRouter();
  const [message, setMessage] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);

  async function upload(file: File) {
    setMessage(null);
    if (!ALLOWED_TYPES.includes(file.type)) return setMessage(t('mediaType'));
    if (file.size < 1 || file.size > MAX_BYTES) return setMessage(t('mediaSize'));
    setBusy(true);
    const created = await fetch('/api/bff/merchant/media', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        filename: file.name.slice(0, 255),
        content_type: file.type,
        size: file.size,
      }),
    });
    if (!created.ok) {
      setBusy(false);
      return setMessage(t('errorGeneric'));
    }
    const upload = (await created.json()) as { asset: MediaAsset; upload_url: string };
    const stored = await fetch(bffPath(upload.upload_url), {
      method: 'PUT',
      headers: { 'Content-Type': file.type },
      body: file,
    });
    setBusy(false);
    if (!stored.ok) return setMessage(t('errorGeneric'));
    router.refresh();
  }

  async function remove(id: string) {
    setMessage(null);
    const response = await fetch(`/api/bff/merchant/media/${id}`, { method: 'DELETE' });
    if (!response.ok) return setMessage(t('mediaInUse'));
    router.refresh();
  }

  return (
    <section className="grid gap-6">
      <label className="grid w-fit gap-2">
        <span className="rounded-control-ui bg-brand px-5 py-3 font-semibold text-surface">
          {t('mediaUpload')}
        </span>
        <input
          type="file"
          accept={ALLOWED_TYPES.join(',')}
          disabled={busy}
          className="sr-only"
          onChange={(event) => {
            const file = event.target.files?.[0];
            if (file) void upload(file);
            event.target.value = '';
          }}
        />
      </label>
      {message && (
        <p
          role="alert"
          className="flex items-start gap-2 rounded-control-ui bg-error-tint p-3 text-error"
        >
          <span aria-hidden="true">!</span>
          <span>{message}</span>
        </p>
      )}
      {assets.length === 0 ? (
        <p className="text-ink-muted">{t('mediaEmpty')}</p>
      ) : (
        <ul className="grid grid-cols-2 gap-4 md:grid-cols-4">
          {assets.map((asset) => (
            <li
              key={asset.id}
              className="grid gap-2 rounded-card border border-border bg-surface p-3"
            >
              {/* eslint-disable-next-line @next/next/no-img-element -- merchant uploads are served by the API, not optimized */}
              <img
                src={asset.url}
                alt=""
                className="aspect-square w-full rounded-control-ui object-cover"
              />
              <button
                type="button"
                onClick={() => remove(asset.id)}
                className="text-sm font-semibold text-error"
              >
                {t('remove')}
              </button>
            </li>
          ))}
        </ul>
      )}
    </section>
  );
}
