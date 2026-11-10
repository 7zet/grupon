import '@testing-library/jest-dom/vitest';
import React from 'react';
import { vi } from 'vitest';
vi.mock('next/image', () => ({
  default: (props: Record<string, unknown>) => {
    const imageProps = { ...props };
    delete imageProps.fill;
    delete imageProps.priority;
    return React.createElement('img', imageProps);
  },
}));
vi.mock('next/font/google', () => ({
  Inter: () => ({ className: '', variable: '--font-app' }),
  Manrope: () => ({ className: '', variable: '--font-manrope' }),
}));
vi.mock('@/i18n/navigation', async () => {
  const React = await import('react');
  return {
    Link: ({ href, children, ...props }: { href: string; children: React.ReactNode }) =>
      React.createElement('a', { href, ...props }, children),
    useRouter: () => ({ push: vi.fn(), replace: vi.fn(), refresh: vi.fn() }),
    usePathname: () => '/',
  };
});
