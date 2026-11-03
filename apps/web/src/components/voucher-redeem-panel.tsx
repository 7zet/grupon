'use client';

import type { components } from '@grupon/api-client';
import { BrowserQRCodeReader, type IScannerControls } from '@zxing/browser';
import { useTranslations } from 'next-intl';
import { useRouter } from '@/i18n/navigation';
import { useEffect, useRef, useState } from 'react';

import { pickText } from '@/lib/localized';

type Voucher = components['schemas']['Voucher'];
type Location = { id: string; label: string };

const inputClass = 'rounded-control-ui border border-border bg-surface px-4 py-3 text-ink text-lg';

/** Problem codes the counter needs to explain. Anything else shows the generic message. */
const PROBLEM_KEYS = [
  'already_redeemed',
  'voucher_expired',
  'voucher_refunded',
  'voucher_voided',
  'voucher_on_hold',
  'wrong_merchant',
  'voucher_not_found',
  'invalid_qr',
  'location_not_found',
] as const;

type ProblemKey = (typeof PROBLEM_KEYS)[number] | 'errorGeneric';

function problemKey(code: string | undefined): ProblemKey {
  return (PROBLEM_KEYS as readonly string[]).includes(code ?? '')
    ? (code as ProblemKey)
    : 'errorGeneric';
}

async function call(
  path: string,
  body: unknown,
): Promise<{ ok: true; voucher: Voucher } | { ok: false; code?: string }> {
  const response = await fetch(`/api/bff/${path}`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(body),
  });
  const data = (await response.json().catch(() => ({}))) as Voucher & { code?: string };
  return response.ok ? { ok: true, voucher: data } : { ok: false, code: data.code };
}

export function VoucherRedeemPanel({
  locations,
  locale,
}: {
  locations: Location[];
  locale: string;
}) {
  const t = useTranslations('Redeem');
  const router = useRouter();
  const videoRef = useRef<HTMLVideoElement>(null);
  const [code, setCode] = useState('');
  const [locationId, setLocationId] = useState(locations[0]?.id ?? '');
  const [voucher, setVoucher] = useState<Voucher | null>(null);
  const [scanning, setScanning] = useState(false);
  const [message, setMessage] = useState<ProblemKey | null>(null);
  const [done, setDone] = useState(false);

  async function lookup(body: { code: string } | { qr_token: string }) {
    setMessage(null);
    setDone(false);
    const result = await call('merchant/vouchers/lookup', body);
    if (!result.ok) {
      setVoucher(null);
      setMessage(problemKey(result.code));
      return;
    }
    setVoucher(result.voucher);
  }

  // The camera runs only while the scan mode is on, and stops as soon as a code is read.
  useEffect(() => {
    if (!scanning || !videoRef.current) return;
    let controls: IScannerControls | undefined;
    let stopped = false;
    new BrowserQRCodeReader()
      .decodeFromVideoDevice(undefined, videoRef.current, (result) => {
        if (!result || stopped) return;
        stopped = true;
        controls?.stop();
        setScanning(false);
        void lookup({ qr_token: result.getText() });
      })
      .then((value) => {
        controls = value;
        if (stopped) value.stop();
      })
      .catch(() => {
        setScanning(false);
        setMessage('errorGeneric');
      });
    return () => {
      stopped = true;
      controls?.stop();
    };
    // lookup is recreated each render but only reads state setters, so the scanner must not restart on it.
  }, [scanning]);

  async function confirm() {
    if (!voucher || !locationId) return;
    setMessage(null);
    const result = await call(`merchant/vouchers/${voucher.id}/redeem`, {
      location_id: locationId,
    });
    if (!result.ok) {
      setMessage(problemKey(result.code));
      return;
    }
    setVoucher(result.voucher);
    setDone(true);
    router.refresh();
  }

  const status = voucher?.status;
  return (
    <div className="grid gap-6">
      <section className="grid max-w-xl gap-3">
        <label className="grid gap-1 text-sm">
          {t('locationLabel')}
          <select
            className={inputClass}
            value={locationId}
            onChange={(event) => setLocationId(event.target.value)}
          >
            {locations.map((location) => (
              <option key={location.id} value={location.id}>
                {location.label}
              </option>
            ))}
          </select>
        </label>
        <form
          className="grid gap-3"
          onSubmit={(event) => {
            event.preventDefault();
            void lookup({ code: code.trim().toUpperCase() });
          }}
        >
          <label className="grid gap-1 text-sm">
            {t('codeLabel')}
            <input
              className={inputClass}
              placeholder="XXXX-XXXX"
              autoCapitalize="characters"
              value={code}
              onChange={(event) => setCode(event.target.value)}
            />
          </label>
          <div className="flex flex-wrap gap-3">
            <button
              type="submit"
              className="rounded-control-ui bg-brand px-5 py-3 font-semibold text-surface"
            >
              {t('check')}
            </button>
            <button
              type="button"
              onClick={() => setScanning((on) => !on)}
              className="rounded-control-ui border border-border px-5 py-3 font-semibold"
            >
              {scanning ? t('stopScan') : t('scan')}
            </button>
          </div>
        </form>
        {scanning && (
          <video
            ref={videoRef}
            className="aspect-square w-full rounded-card bg-ink object-cover"
            muted
            playsInline
          />
        )}
      </section>

      {message && (
        <p
          role="alert"
          className="flex max-w-xl items-start gap-2 rounded-control-ui bg-error-tint p-3 text-error"
        >
          <span aria-hidden="true">!</span>
          <span>{t(message)}</span>
        </p>
      )}

      {voucher && (
        <section
          aria-labelledby="voucher-result"
          className="grid max-w-xl gap-3 rounded-card border border-border bg-surface p-5"
        >
          <h2 id="voucher-result" className="font-semibold">
            {pickText(voucher.deal.title, locale)}
          </h2>
          <p className="text-sm text-ink-muted">{pickText(voucher.option.title, locale)}</p>
          <p className="text-sm">
            {t('code')}: <span className="font-semibold">{voucher.code}</span>
          </p>
          <p role="status" className="font-semibold">
            {done ? t('redeemedNow') : t(`status_${status}` as 'status_active')}
          </p>
          {status === 'active' && !done && (
            <button
              type="button"
              onClick={confirm}
              disabled={!locationId}
              className="rounded-control-ui bg-brand px-5 py-4 text-lg font-semibold text-surface"
            >
              {t('confirm')}
            </button>
          )}
        </section>
      )}
    </div>
  );
}
