import type { PropsWithChildren } from 'react';

export function BaseLayout({ children }: PropsWithChildren) {
  return <div className="min-h-dvh bg-page font-sans text-ink antialiased">{children}</div>;
}
