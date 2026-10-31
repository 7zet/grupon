'use client';

import { useEffect, useRef } from 'react';

export function HeroMotion({ children }: { children: React.ReactNode }) {
  const ref = useRef<HTMLDivElement>(null);
  useEffect(() => {
    if (matchMedia('(prefers-reduced-motion: reduce)').matches) return;
    void import('gsap').then(({ gsap }) => {
      gsap.fromTo(
        ref.current?.querySelectorAll('[data-hero]') ?? [],
        { opacity: 0, y: 16 },
        { opacity: 1, y: 0, stagger: 0.08, duration: 0.5, ease: 'power2.out' },
      );
    });
  }, []);
  return <div ref={ref}>{children}</div>;
}
