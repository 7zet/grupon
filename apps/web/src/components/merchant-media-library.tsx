'use client';

import { useTranslations } from 'next-intl';
import { useEffect, useRef, useState } from 'react';

import { useRouter } from '@/i18n/navigation';

export type MediaAsset = {
  id: string;
  url: string;
  content_type: string;
  size: number;
  width?: number;
  height?: number;
};

const MAX_BYTES = 10 * 1024 * 1024;
const ALLOWED_TYPES = ['image/jpeg', 'image/png', 'image/webp'];
const RATIO = 4 / 3;
const MAX_SIDE = 1600;
const inputClass = 'w-full';

/** Upload URLs from the API start with /api/v1; the browser reaches them through the same-origin BFF. */
function bffPath(uploadUrl: string) {
  return uploadUrl.replace(/^.*\/api\/v1\//, '/api/bff/');
}

type Crop = { zoom: number; x: number; y: number };

/** The source rectangle picked by the owner: a 4:3 window, zoomed in and moved inside the photo. */
function cropRect(width: number, height: number, crop: Crop) {
  const baseW = Math.min(width, height * RATIO);
  const cropW = baseW / crop.zoom;
  const cropH = cropW / RATIO;
  return { sx: (width - cropW) * crop.x, sy: (height - cropH) * crop.y, sw: cropW, sh: cropH };
}

/** Lets the merchant choose the frame themselves, so the server never has to guess a bad crop. */
function ImageCropper({
  file,
  onCancel,
  onConfirm,
}: {
  file: File;
  onCancel: () => void;
  onConfirm: (blob: Blob) => void;
}) {
  const t = useTranslations('Deals');
  const canvasRef = useRef<HTMLCanvasElement>(null);
  const [image, setImage] = useState<HTMLImageElement | null>(null);
  const [crop, setCrop] = useState<Crop>({ zoom: 1, x: 0.5, y: 0.5 });

  useEffect(() => {
    const url = URL.createObjectURL(file);
    const img = new Image();
    img.onload = () => setImage(img);
    img.src = url;
    return () => URL.revokeObjectURL(url);
  }, [file]);

  useEffect(() => {
    const canvas = canvasRef.current;
    if (!image || !canvas) return;
    const rect = cropRect(image.naturalWidth, image.naturalHeight, crop);
    canvas.width = 800;
    canvas.height = 600;
    canvas
      .getContext('2d')
      ?.drawImage(image, rect.sx, rect.sy, rect.sw, rect.sh, 0, 0, canvas.width, canvas.height);
  }, [image, crop]);

  function confirm() {
    if (!image) return;
    const rect = cropRect(image.naturalWidth, image.naturalHeight, crop);
    const scale = Math.min(1, MAX_SIDE / rect.sw);
    const output = document.createElement('canvas');
    output.width = Math.round(rect.sw * scale);
    output.height = Math.round(rect.sh * scale);
    output
      .getContext('2d')
      ?.drawImage(image, rect.sx, rect.sy, rect.sw, rect.sh, 0, 0, output.width, output.height);
    output.toBlob((blob) => blob && onConfirm(blob), 'image/jpeg', 0.9);
  }

  return (
    <div className="grid max-w-md gap-4 rounded-card border border-border bg-surface p-4">
      <p className="text-sm text-ink-muted">{t('cropHint')}</p>
      <canvas
        ref={canvasRef}
        className="aspect-[4/3] w-full rounded-control-ui bg-ink"
        aria-label={t('cropTitle')}
      />
      <label className="grid gap-1 text-sm">
        {t('cropZoom')}
        <input
          type="range"
          min={1}
          max={3}
          step={0.05}
          value={crop.zoom}
          className={inputClass}
          onChange={(e) => setCrop({ ...crop, zoom: Number(e.target.value) })}
        />
      </label>
      <label className="grid gap-1 text-sm">
        {t('cropX')}
        <input
          type="range"
          min={0}
          max={1}
          step={0.01}
          value={crop.x}
          className={inputClass}
          onChange={(e) => setCrop({ ...crop, x: Number(e.target.value) })}
        />
      </label>
      <label className="grid gap-1 text-sm">
        {t('cropY')}
        <input
          type="range"
          min={0}
          max={1}
          step={0.01}
          value={crop.y}
          className={inputClass}
          onChange={(e) => setCrop({ ...crop, y: Number(e.target.value) })}
        />
      </label>
      <div className="flex gap-3">
        <button
          type="button"
          disabled={!image}
          onClick={confirm}
          className="rounded-control-ui bg-brand px-5 py-3 font-semibold text-surface disabled:opacity-60"
        >
          {t('cropConfirm')}
        </button>
        <button
          type="button"
          onClick={onCancel}
          className="rounded-control-ui border border-border px-5 py-3"
        >
          {t('cropCancel')}
        </button>
      </div>
    </div>
  );
}

export function MerchantMediaLibrary({ assets }: { assets: MediaAsset[] }) {
  const t = useTranslations('Deals');
  const router = useRouter();
  const [message, setMessage] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);
  const [pending, setPending] = useState<File | null>(null);

  function pick(file: File) {
    setMessage(null);
    if (!ALLOWED_TYPES.includes(file.type)) return setMessage(t('mediaType'));
    if (file.size < 1 || file.size > MAX_BYTES) return setMessage(t('mediaSize'));
    setPending(file);
  }

  async function upload(blob: Blob, filename: string) {
    setPending(null);
    setBusy(true);
    const created = await fetch('/api/bff/merchant/media', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        filename: filename.slice(0, 255),
        content_type: blob.type,
        size: blob.size,
      }),
    });
    if (!created.ok) {
      setBusy(false);
      return setMessage(t('errorGeneric'));
    }
    const upload = (await created.json()) as { asset: MediaAsset; upload_url: string };
    const stored = await fetch(bffPath(upload.upload_url), {
      method: 'PUT',
      headers: { 'Content-Type': blob.type },
      body: blob,
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
      {pending ? (
        <ImageCropper
          file={pending}
          onCancel={() => setPending(null)}
          onConfirm={(blob) => void upload(blob, pending.name)}
        />
      ) : (
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
              if (file) pick(file);
              event.target.value = '';
            }}
          />
        </label>
      )}
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
                className="aspect-[4/3] w-full rounded-control-ui object-cover"
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
