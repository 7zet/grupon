'use client';

import type { components } from '@dealo/api-client';
import QRCode from 'qrcode';
import { useEffect, useState } from 'react';

type Voucher = components['schemas']['Voucher'];

/**
 * Returns a bounded delay until the server says the current signed QR expires.
 * The short buffer means a scanned code never races the rotation boundary.
 */
export function qrRefreshDelay(expiresAt: string, now = Date.now()): number {
  const expiry = new Date(expiresAt).getTime();
  return Math.max(0, expiry - now + 250);
}

export function VoucherQRCode({ voucher, alt }: { voucher: Voucher; alt: string }) {
  const [current, setCurrent] = useState(voucher);
  const [image, setImage] = useState('');

  useEffect(() => {
    let alive = true;
    void QRCode.toDataURL(current.qr_token, { margin: 1, width: 320 }).then((dataUrl) => {
      if (alive) setImage(dataUrl);
    });
    return () => {
      alive = false;
    };
  }, [current.qr_token]);

  useEffect(() => {
    if (!current.qr_token_expires_at) return;
    const refresh = async () => {
      const response = await fetch(`/api/bff/me/vouchers/${encodeURIComponent(current.id)}`, {
        cache: 'no-store',
      });
      if (response.ok) setCurrent((await response.json()) as Voucher);
    };
    const timer = window.setTimeout(() => void refresh(), qrRefreshDelay(current.qr_token_expires_at));
    return () => window.clearTimeout(timer);
  }, [current.id, current.qr_token_expires_at]);

  if (!image) return <div className="size-80 max-w-full rounded-card bg-surface p-3" aria-busy="true" />;
  // Generated data URL; Next image optimization does not apply.
  // eslint-disable-next-line @next/next/no-img-element
  return <img src={image} alt={alt} width={320} height={320} className="h-auto max-w-full rounded-card bg-surface p-3" />;
}
