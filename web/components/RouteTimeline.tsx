"use client";

import { ModeIcon } from "./icons";

export interface TimelineLeg {
  mode: string; // raw mode key: flight | train | bus
  label: string; // localized mode name
  price: string; // formatted price + currency
  href?: string; // optional info/search deep-link
  hrefLabel?: string; // localized label for the link
  modeValue?: string; // current override selection (for the per-leg selector)
  onModeChange?: (mode: string) => void;
}

export interface ModeOption {
  value: string;
  label: string;
}

// Vertical, visual route: numbered stops connected by transport legs that carry
// a mode icon and the leg price. Each row fades up in sequence.
export function RouteTimeline({
  stops = [],
  legs = [],
  nights = [],
  airports = [],
  airportLabel,
  nightLabel,
  modeOptions,
}: {
  stops?: string[];
  legs?: TimelineLeg[];
  nights?: number[];
  airports?: boolean[]; // per-stop: true when the selected place is an airport
  airportLabel?: string;
  nightLabel?: (n: number) => string;
  modeOptions?: ModeOption[];
}) {
  return (
    <ol className="relative">
      {stops.map((stop, i) => (
        <li
          key={i}
          className="animate-fade-up"
          style={{ animationDelay: `${i * 70}ms` }}
        >
          <div className="flex items-center gap-3">
            <span className="z-10 grid h-8 w-8 shrink-0 place-items-center rounded-full bg-primary text-xs font-bold text-primary-fg shadow-soft">
              {i + 1}
            </span>
            <span className="font-display text-base font-semibold">{stop}</span>
            {airports[i] && (
              <span className="inline-flex items-center gap-1 rounded-full bg-accent/10 px-2 py-0.5 text-xs font-medium text-accent">
                ✈ {airportLabel ?? "airport"}
              </span>
            )}
            {typeof nights[i] === "number" && nights[i] > 0 && (
              <span className="rounded-full bg-surface-2 px-2 py-0.5 text-xs font-medium text-muted">
                {nightLabel ? nightLabel(nights[i]) : `${nights[i]}n`}
              </span>
            )}
          </div>

          {i < legs.length && (
            <div className="ml-4 flex items-stretch gap-3 border-l-2 border-dashed border-border py-2 pl-6">
              <div className="flex items-center gap-2 text-sm text-muted">
                <span className="grid h-7 w-7 place-items-center rounded-lg bg-transport/10 text-transport">
                  <ModeIcon mode={legs[i].mode} width={15} height={15} />
                </span>
                <span>{legs[i].label}</span>
                <span className="text-border">·</span>
                <span className="font-semibold text-text">{legs[i].price}</span>
                {legs[i].href && (
                  <>
                    <span className="text-border">·</span>
                    <a
                      href={legs[i].href}
                      target="_blank"
                      rel="noopener noreferrer"
                      className="text-primary underline-offset-2 hover:underline"
                    >
                      {legs[i].hrefLabel ?? "search"} ↗
                    </a>
                  </>
                )}
                {legs[i].onModeChange && modeOptions && (
                  <>
                    <span className="text-border">·</span>
                    <select
                      value={legs[i].modeValue ?? legs[i].mode}
                      onChange={(e) => legs[i].onModeChange?.(e.target.value)}
                      aria-label="change transport mode"
                      className="rounded-md border border-border bg-surface px-1.5 py-0.5 text-xs text-muted outline-none hover:border-primary/40 focus:border-primary print:hidden"
                    >
                      {modeOptions.map((o) => (
                        <option key={o.value} value={o.value}>{o.label}</option>
                      ))}
                    </select>
                  </>
                )}
              </div>
            </div>
          )}
        </li>
      ))}
    </ol>
  );
}
