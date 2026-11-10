'use client';

import { animate, inView, motion, useReducedMotion } from 'motion/react';
import { useEffect, useRef, type ReactNode } from 'react';

const EASE = [0.22, 1, 0.36, 1] as const;

/** One hero block that rises in after the ones before it (60 ms apart). */
export function HeroItem({ children, index, className }: { children: ReactNode; index: number; className?: string }) {
  const reduced = useReducedMotion();
  return (
    <motion.div
      className={className}
      initial={reduced ? false : { opacity: 0, y: 16 }}
      animate={{ opacity: 1, y: 0 }}
      transition={{ duration: 0.5, delay: 0.25 + index * 0.06, ease: EASE }}
    >
      {children}
    </motion.div>
  );
}

/** The hero headline; each word rises in with a short stagger. */
export function HeroTitle({ text, className }: { text: string; className?: string }) {
  const reduced = useReducedMotion();
  const words = text.split(' ');
  return (
    <h1 className={className} aria-label={text}>
      {words.map((word, index) => (
        <motion.span
          key={`${word}-${index}`}
          aria-hidden="true"
          className={index < words.length - 1 ? 'mr-[0.25em] inline-block' : 'inline-block'}
          initial={reduced ? false : { opacity: 0, y: 24 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ duration: 0.55, delay: index * 0.06, ease: EASE }}
        >
          {word}
          {index < words.length - 1 ? ' ' : ''}
        </motion.span>
      ))}
    </h1>
  );
}

/**
 * A page section that fades up 16 px once, when it scrolls into view. The section is visible in the server
 * HTML (so it never stays hidden without JavaScript, for crawlers or in full-page screenshots); only sections
 * that start below the fold are hidden on the client and then revealed.
 */
export function Reveal({ children, className }: { children: ReactNode; className?: string }) {
  const ref = useRef<HTMLElement>(null);
  const reduced = useReducedMotion();
  useEffect(() => {
    const el = ref.current;
    if (!el || reduced || el.getBoundingClientRect().top < window.innerHeight) return;
    el.style.opacity = '0';
    el.style.transform = 'translateY(16px)';
    return inView(
      el,
      () => {
        void animate(el, { opacity: 1, transform: 'translateY(0px)' }, { duration: 0.5, ease: EASE });
      },
      { margin: '0px 0px -80px 0px' },
    );
  }, [reduced]);
  return (
    <section ref={ref} className={className}>
      {children}
    </section>
  );
}
