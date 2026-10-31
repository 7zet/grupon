'use client';
import { useEffect } from 'react';

let workerStart: Promise<unknown> | undefined;

export default function BrowserMockProvider() {
  useEffect(() => {
    workerStart ??= import('./browser').then(({ mockWorker }) =>
      mockWorker.start({
        serviceWorker: { url: '/mockServiceWorker.js' },
        onUnhandledFrame: 'bypass',
        quiet: true,
      }),
    );
  }, []);
  return null;
}
