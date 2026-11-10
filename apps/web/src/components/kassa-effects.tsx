'use client';

import { ScanLine, XCircle } from 'lucide-react';
import { motion, useReducedMotion } from 'motion/react';
import { useEffect, type ReactNode, type RefObject } from 'react';

const EASE = [0.22, 1, 0.36, 1] as const;

/** The cashier's main action: a large round button with a slow pulsing ring. */
export function ScanButton({ label, onClick }: { label: string; onClick: () => void }) {
  return (
    <div className="flex justify-center py-4">
      <button
        type="button"
        onClick={onClick}
        className="relative flex size-44 flex-col items-center justify-center gap-2 rounded-round-ui bg-brand font-display text-lead font-extrabold text-surface shadow-lift transition-transform active:scale-95 motion-reduce:transition-none"
      >
        <span aria-hidden="true" className="kassa-pulse absolute inset-0 rounded-round-ui border-4 border-brand" />
        <ScanLine className="size-12" aria-hidden="true" />
        <span className="px-4 text-center leading-tight">{label}</span>
      </button>
    </div>
  );
}

/** Camera view with an animated scan line. */
export function ScanFrame({ videoRef }: { videoRef: RefObject<HTMLVideoElement | null> }) {
  return (
    <div className="relative overflow-hidden rounded-card-ui bg-ink">
      <video ref={videoRef} className="aspect-square w-full object-cover" muted playsInline />
      <span aria-hidden="true" className="pointer-events-none absolute inset-6 rounded-card-ui border-2 border-surface/70" />
      <span aria-hidden="true" className="kassa-scanline pointer-events-none absolute inset-x-6 top-6 h-0.5 bg-brand-100 shadow-[0_0_12px_2px_var(--brand-100)]" />
    </div>
  );
}

/**
 * Full-screen confirmation after a redemption: a green circle with a check that draws itself, and a short
 * vibration on phones that support it. It closes by itself; the text status stays in the result card.
 */
export function SuccessOverlay({ label, onClose }: { label: string; onClose: () => void }) {
  const reduced = useReducedMotion();
  // onClose must be stable (useCallback in the caller), so the page refresh does not restart the timer.
  useEffect(() => {
    if (typeof navigator !== 'undefined' && 'vibrate' in navigator) navigator.vibrate(80);
    const timer = window.setTimeout(onClose, 1600);
    return () => window.clearTimeout(timer);
  }, [onClose]);
  return (
    <motion.div
      aria-hidden="true"
      onClick={onClose}
      initial={reduced ? false : { opacity: 0 }}
      animate={{ opacity: 1 }}
      transition={{ duration: 0.2 }}
      className="fixed inset-0 z-[60] flex flex-col items-center justify-center gap-6 bg-brand-dark text-surface"
    >
      <motion.svg
        viewBox="0 0 120 120"
        className="size-40"
        initial={reduced ? false : { scale: 0.7 }}
        animate={{ scale: 1 }}
        transition={{ duration: 0.4, ease: EASE }}
      >
        <circle cx="60" cy="60" r="56" fill="var(--brand-700)" />
        <motion.path
          d="M36 62 L53 78 L86 44"
          fill="none"
          stroke="white"
          strokeWidth="10"
          strokeLinecap="round"
          strokeLinejoin="round"
          initial={reduced ? false : { pathLength: 0 }}
          animate={{ pathLength: 1 }}
          transition={{ duration: 0.45, delay: 0.15, ease: EASE }}
        />
      </motion.svg>
      <p className="font-display text-title font-extrabold">{label}</p>
    </motion.div>
  );
}

/** Error message with a red cross that shakes once, so the cashier notices it. */
export function ErrorShake({ children, shakeKey }: { children: ReactNode; shakeKey: string }) {
  const reduced = useReducedMotion();
  return (
    <motion.p
      key={shakeKey}
      role="alert"
      {...(reduced ? {} : { initial: { x: 0 }, animate: { x: [0, -10, 10, -6, 6, 0] } })}
      transition={{ duration: 0.4 }}
      className="flex max-w-xl items-start gap-3 rounded-control-ui border border-error/30 bg-error-tint p-4 text-error"
    >
      <XCircle className="mt-0.5 size-6 shrink-0" aria-hidden="true" />
      <span className="font-semibold">{children}</span>
    </motion.p>
  );
}
