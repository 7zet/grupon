'use client';

import { useEffect } from 'react';

import { useRouter } from '@/i18n/navigation';

/** Re-renders the server page while the webhook has not yet marked the order paid. */
export function PaymentPoller({ intervalMs = 2000, maxTries = 15 }: { intervalMs?: number; maxTries?: number }) {
  const router = useRouter();
  useEffect(() => {
    let tries = 0;
    const timer = window.setInterval(() => {
      tries += 1;
      router.refresh();
      if (tries >= maxTries) window.clearInterval(timer);
    }, intervalMs);
    return () => window.clearInterval(timer);
  }, [router, intervalMs, maxTries]);
  return null;
}
