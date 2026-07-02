import type { SVGProps } from "react";

// Small inline icon set (no runtime icon dependency). stroke=currentColor so they inherit
// text color. (SPEC-200 FR-2006)
function Base(props: SVGProps<SVGSVGElement>) {
  return (
    <svg
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      strokeWidth={1.7}
      strokeLinecap="round"
      strokeLinejoin="round"
      width={18}
      height={18}
      aria-hidden
      {...props}
    />
  );
}

export function IconPainel(p: SVGProps<SVGSVGElement>) {
  return (
    <Base {...p}>
      <rect x="3" y="3" width="7" height="9" rx="1.5" />
      <rect x="14" y="3" width="7" height="5" rx="1.5" />
      <rect x="14" y="12" width="7" height="9" rx="1.5" />
      <rect x="3" y="16" width="7" height="5" rx="1.5" />
    </Base>
  );
}

export function IconCarteira(p: SVGProps<SVGSVGElement>) {
  return (
    <Base {...p}>
      <rect x="3" y="6" width="18" height="13" rx="2" />
      <path d="M3 10h18M16 14h2" />
    </Base>
  );
}

export function IconInsights(p: SVGProps<SVGSVGElement>) {
  return (
    <Base {...p}>
      <circle cx="12" cy="12" r="3.5" />
      <path d="M12 3v3M12 18v3M5 12H2M22 12h-3M6 6l1.5 1.5M16.5 16.5 18 18M6 18l1.5-1.5M16.5 7.5 18 6" />
    </Base>
  );
}

export function IconSaude(p: SVGProps<SVGSVGElement>) {
  return (
    <Base {...p}>
      <path d="M20.8 5.6a5 5 0 0 0-8.8-1.2A5 5 0 0 0 3.2 5.6C1.5 8 3 12 12 20c9-8 10.5-12 8.8-14.4Z" />
    </Base>
  );
}

export function IconProjecoes(p: SVGProps<SVGSVGElement>) {
  return (
    <Base {...p}>
      <path d="M4 19V5M4 19h16M8 15l4-5 3 3 5-7" />
    </Base>
  );
}

export function IconPerfil(p: SVGProps<SVGSVGElement>) {
  return (
    <Base {...p}>
      <circle cx="12" cy="8" r="4" />
      <path d="M4 21a8 8 0 0 1 16 0" />
    </Base>
  );
}

export function IconCopilot(p: SVGProps<SVGSVGElement>) {
  return (
    <Base {...p}>
      <path d="M21 12a8 8 0 0 1-11.3 7.3L4 21l1.7-5.7A8 8 0 1 1 21 12Z" />
    </Base>
  );
}

export function IconClose(p: SVGProps<SVGSVGElement>) {
  return (
    <Base {...p}>
      <path d="M6 6l12 12M18 6 6 18" />
    </Base>
  );
}
