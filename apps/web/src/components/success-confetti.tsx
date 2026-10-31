'use client';

import { useEffect } from 'react';

/**
 * One burst of confetti after a successful purchase. The library is loaded only here, so it never
 * blocks the page, and nothing plays when the visitor prefers reduced motion.
 */
export function SuccessConfetti() {
  useEffect(() => {
    if (window.matchMedia('(prefers-reduced-motion: reduce)').matches) return;
    let cancelled = false;
    void import('canvas-confetti').then(({ default: confetti }) => {
      if (!cancelled) confetti({ particleCount: 120, spread: 70, origin: { y: 0.6 } });
    });
    return () => {
      cancelled = true;
    };
  }, []);
  return null;
}
