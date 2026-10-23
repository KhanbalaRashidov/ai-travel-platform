// Small inline icon set (no icon dependency). Each takes standard SVG props and
// inherits color via currentColor.
import type { SVGProps } from "react";

type P = SVGProps<SVGSVGElement>;

const base = {
  viewBox: "0 0 24 24",
  fill: "none",
  stroke: "currentColor",
  strokeWidth: 2,
  strokeLinecap: "round" as const,
  strokeLinejoin: "round" as const,
};

export function PlaneIcon(p: P) {
  return (
    <svg {...base} {...p}>
      <path d="M17.8 19.2 16 11l3.5-3.5a2.1 2.1 0 0 0-3-3L13 8 4.8 6.2a.6.6 0 0 0-.6 1l5 3.4-2.9 2.9-2.1-.4a.6.6 0 0 0-.5 1l1.9 1.9 1.9 1.9a.6.6 0 0 0 1-.5l-.4-2.1 2.9-2.9 3.4 5a.6.6 0 0 0 1-.6z" />
    </svg>
  );
}

export function TrainIcon(p: P) {
  return (
    <svg {...base} {...p}>
      <rect x="4" y="3" width="16" height="15" rx="2" />
      <path d="M4 11h16M12 3v8M8 19l-2 2M16 19l2 2" />
      <circle cx="8.5" cy="14.5" r="0.6" fill="currentColor" />
      <circle cx="15.5" cy="14.5" r="0.6" fill="currentColor" />
    </svg>
  );
}

export function BusIcon(p: P) {
  return (
    <svg {...base} {...p}>
      <rect x="4" y="4" width="16" height="13" rx="2" />
      <path d="M4 11h16M8 17v2M16 17v2" />
      <circle cx="8" cy="14" r="0.6" fill="currentColor" />
      <circle cx="16" cy="14" r="0.6" fill="currentColor" />
    </svg>
  );
}

export function FerryIcon(p: P) {
  return (
    <svg {...base} {...p}>
      <path d="M3 18c1.5 1 3 1 4.5 0s3-1 4.5 0 3 1 4.5 0 3-1 4.5 0" />
      <path d="M5 14l2-5h10l2 5M12 4v5M4 14h16l-2 4H6z" />
    </svg>
  );
}

export function PinIcon(p: P) {
  return (
    <svg {...base} {...p}>
      <path d="M12 21s-7-5.5-7-11a7 7 0 1 1 14 0c0 5.5-7 11-7 11z" />
      <circle cx="12" cy="10" r="2.5" />
    </svg>
  );
}

export function RouteIcon(p: P) {
  return (
    <svg {...base} {...p}>
      <circle cx="6" cy="19" r="2.5" />
      <circle cx="18" cy="5" r="2.5" />
      <path d="M8.5 19H15a3.5 3.5 0 0 0 0-7H9a3.5 3.5 0 0 1 0-7h6.5" />
    </svg>
  );
}

export function WalletIcon(p: P) {
  return (
    <svg {...base} {...p}>
      <rect x="3" y="6" width="18" height="13" rx="2" />
      <path d="M3 10h18M16 14h2" />
    </svg>
  );
}

export function SparkIcon(p: P) {
  return (
    <svg {...base} {...p}>
      <path d="M12 3v4M12 17v4M3 12h4M17 12h4M6 6l2.5 2.5M15.5 15.5 18 18M18 6l-2.5 2.5M8.5 15.5 6 18" />
    </svg>
  );
}

export function GlobeIcon(p: P) {
  return (
    <svg {...base} {...p}>
      <circle cx="12" cy="12" r="9" />
      <path d="M3 12h18M12 3c3 3.5 3 14.5 0 18M12 3c-3 3.5-3 14.5 0 18" />
    </svg>
  );
}

export function ArrowRightIcon(p: P) {
  return (
    <svg {...base} {...p}>
      <path d="M5 12h14M13 6l6 6-6 6" />
    </svg>
  );
}

export function ModeIcon({ mode, ...p }: P & { mode: string }) {
  if (mode === "train") return <TrainIcon {...p} />;
  if (mode === "bus") return <BusIcon {...p} />;
  if (mode === "ferry") return <FerryIcon {...p} />;
  return <PlaneIcon {...p} />;
}
