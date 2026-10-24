"use client";

import Link from "next/link";
import type { ReactNode } from "react";
import { LanguageSwitcher } from "../lib/i18n";
import { ThemeToggle } from "./theme";

// Brand mark: gradient rounded square + a navigation arrow (matches the promo /
// og image and favicon). Kept as inline SVG so it reads crisply light & dark.
function Brand() {
  return (
    <Link href="/" className="flex min-w-0 items-center gap-2.5">
      <svg width="36" height="36" viewBox="0 0 54 54" className="shadow-soft shrink-0 rounded-[10px]" aria-hidden>
        <defs>
          <linearGradient id="brandMark" x1="0" y1="0" x2="1" y2="1">
            <stop offset="0" stopColor="#8B7CF6" />
            <stop offset="1" stopColor="#2DD4BF" />
          </linearGradient>
        </defs>
        <rect width="54" height="54" rx="15" fill="url(#brandMark)" />
        <path d="M27 12 L38 39 L27 32 L16 39 Z" fill="#0A0E24" />
      </svg>
      {/* Hide the wordmark on very narrow phones so the logo + language switcher
          + theme toggle never overflow the header (gutter preserved). */}
      <span className="hidden truncate font-display text-base font-bold tracking-tight min-[360px]:inline">
        AI Travel
      </span>
    </Link>
  );
}

export function AppShell({ children }: { children: ReactNode }) {
  return (
    <div className="relative min-h-screen">
      <div className="aurora pointer-events-none absolute inset-x-0 top-0 h-96" />
      <header className="sticky top-0 z-30 border-b border-border/70 bg-bg/80 backdrop-blur-md print:hidden">
        <div className="mx-auto flex max-w-5xl items-center justify-between gap-2 px-4 py-3">
          <Brand />
          <div className="flex shrink-0 items-center gap-2">
            <LanguageSwitcher />
            <ThemeToggle />
          </div>
        </div>
      </header>
      {/* isolate: trap descendant stacking (e.g. Leaflet's z-index-heavy map
          controls) inside the content so nothing paints over the sticky header. */}
      <div className="relative isolate">{children}</div>
      <footer className="mx-auto max-w-5xl px-4 py-10 text-center text-xs text-muted">
        AI Travel Platform · advisory only — no booking, no payment · prices are
        indicative, confirm on the provider&apos;s site · map data © OpenStreetMap.
      </footer>
    </div>
  );
}
