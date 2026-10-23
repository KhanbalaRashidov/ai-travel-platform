"use client";

import Link from "next/link";
import type { ReactNode } from "react";
import { LanguageSwitcher } from "../lib/i18n";
import { ThemeToggle } from "./theme";

// Brand mark: a small compass/route glyph in a gradient tile.
function Brand() {
  return (
    <Link href="/" className="flex items-center gap-2.5">
      <span className="grid h-9 w-9 place-items-center rounded-xl bg-gradient-to-br from-primary to-accent text-primary-fg shadow-soft">
        <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
          <circle cx="12" cy="12" r="9" />
          <path d="M14.5 9.5 11 11l-1.5 3.5L13 13z" fill="currentColor" />
        </svg>
      </span>
      <span className="font-display text-base font-bold tracking-tight">
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
        <div className="mx-auto flex max-w-5xl items-center justify-between px-4 py-3">
          <Brand />
          <div className="flex items-center gap-2">
            <LanguageSwitcher />
            <ThemeToggle />
          </div>
        </div>
      </header>
      <div className="relative">{children}</div>
      <footer className="mx-auto max-w-5xl px-4 py-10 text-center text-xs text-muted">
        AI Travel Platform · advisory only — no booking, no payment · prices are
        indicative, confirm on the provider&apos;s site · map data © OpenStreetMap.
      </footer>
    </div>
  );
}
