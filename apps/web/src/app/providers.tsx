'use client';

import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { NuqsAdapter } from 'nuqs/adapters/next/app';
import dynamic from 'next/dynamic';
import { useState, type PropsWithChildren } from 'react';

const BrowserMockProvider = dynamic(() => import('@/mocks/browser-provider'), { ssr: false });

export function Providers({ children }: PropsWithChildren) {
  const [queryClient] = useState(() => new QueryClient());
  return (
    <QueryClientProvider client={queryClient}>
      <NuqsAdapter>
        {process.env.NEXT_PUBLIC_USE_MSW === 'true' && <BrowserMockProvider />}
        {children}
      </NuqsAdapter>
    </QueryClientProvider>
  );
}
