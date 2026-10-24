"use client";

export interface CostSegment {
  key: string;
  label: string;
  value: number;
  colorVar: string; // CSS var name, e.g. "--transport"
}

// Donut chart drawn with stacked SVG circle strokes (one arc per segment) plus a
// legend. The total sits in the hole. Pure SVG — no chart dependency.
export function CostDonut({
  segments = [],
  total,
  currency,
  fmt,
}: {
  segments?: CostSegment[];
  total: number;
  currency: string;
  fmt: (n: number) => string;
}) {
  const r = 52;
  const c = 2 * Math.PI * r;
  const sum = segments.reduce((a, s) => a + s.value, 0) || 1;
  let offset = 0;

  return (
    <div className="flex flex-col items-center gap-5 sm:flex-row sm:items-center">
      <svg
        viewBox="0 0 140 140"
        className="h-36 w-36 shrink-0 -rotate-90"
        role="img"
        aria-label="Cost breakdown"
      >
        <circle
          cx="70"
          cy="70"
          r={r}
          fill="none"
          stroke="rgb(var(--border))"
          strokeWidth="14"
        />
        {segments.map((s) => {
          const frac = s.value / sum;
          const dash = frac * c;
          const el = (
            <circle
              key={s.key}
              cx="70"
              cy="70"
              r={r}
              fill="none"
              stroke={`rgb(var(${s.colorVar}))`}
              strokeWidth="14"
              strokeLinecap="round"
              strokeDasharray={`${dash} ${c - dash}`}
              strokeDashoffset={-offset}
              className="transition-[stroke-dasharray] duration-700 ease-out"
            />
          );
          offset += dash;
          return el;
        })}
      </svg>

      <div className="w-full">
        <div className="mb-3 rotate-0">
          <div className="text-xs font-semibold uppercase tracking-wide text-muted">
            {currency}
          </div>
          <div className="font-display text-2xl font-bold tabular-nums">
            {fmt(total)}
          </div>
        </div>
        <ul className="space-y-2">
          {segments.map((s) => (
            <li key={s.key} className="flex items-center justify-between gap-2 text-sm">
              <span className="flex min-w-0 items-center gap-2 text-muted">
                <span
                  className="h-2.5 w-2.5 shrink-0 rounded-full"
                  style={{ background: `rgb(var(${s.colorVar}))` }}
                />
                <span className="truncate">{s.label}</span>
              </span>
              <span className="shrink-0 font-semibold tabular-nums">{fmt(s.value)}</span>
            </li>
          ))}
        </ul>
      </div>
    </div>
  );
}
