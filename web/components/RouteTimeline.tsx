"use client";

import { ModeIcon } from "./icons";

export interface TimelineLeg {
  mode: string; // raw mode key: flight | train | bus
  label: string; // localized mode name
  price: string; // formatted price + currency
}

// Vertical, visual route: numbered stops connected by transport legs that carry
// a mode icon and the leg price. Each row fades up in sequence.
export function RouteTimeline({
  stops = [],
  legs = [],
  nights = [],
  nightLabel,
}: {
  stops?: string[];
  legs?: TimelineLeg[];
  nights?: number[];
  nightLabel?: (n: number) => string;
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
            {typeof nights[i] === "number" && (
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
              </div>
            </div>
          )}
        </li>
      ))}
    </ol>
  );
}
