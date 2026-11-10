import type { SVGProps } from 'react';

/** Ticket/tag mark used for the favicon, PWA icons and the wordmark. */
export function TicketMark(props: SVGProps<SVGSVGElement>) {
  return (
    <svg viewBox="0 0 32 32" aria-hidden="true" focusable="false" {...props}>
      <path
        d="M4 9.5A2.5 2.5 0 0 1 6.5 7h19A2.5 2.5 0 0 1 28 9.5V13a3 3 0 0 0 0 6v3.5a2.5 2.5 0 0 1-2.5 2.5h-19A2.5 2.5 0 0 1 4 22.5V19a3 3 0 0 0 0-6z"
        fill="currentColor"
      />
      <circle cx="21" cy="16" r="2.25" fill="var(--surface, #fff)" />
      <path d="M11 12.5h6M11 16h4M11 19.5h6" stroke="var(--surface, #fff)" strokeWidth="1.6" strokeLinecap="round" opacity="0.7" />
    </svg>
  );
}

/** Wordmark: the product name from brand config, Manrope 800, with the ticket mark in brand green. */
export function Logo({ name, className = '' }: { name: string; className?: string }) {
  return (
    <span className={`inline-flex items-center gap-2 font-display font-extrabold tracking-tight text-brand-dark ${className}`}>
      <TicketMark className="size-6 text-brand" />
      <span>{name}</span>
    </span>
  );
}
