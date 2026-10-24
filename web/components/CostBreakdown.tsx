"use client";

import { FoodLine, HotelLine, PlanLeg } from "../lib/api";

// A source name mapped to a human label (real scrapers vs the labeled estimate).
function sourceLabel(src: string): string {
  if (!src || src === "distance-estimate" || src === "seed") return "";
  return src;
}

// mapsLink opens the actual place on Google Maps (never an empty search).
function mapsLink(name: string, city: string): string {
  return `https://www.google.com/maps/search/?api=1&query=${encodeURIComponent(name + " " + city)}`;
}

// hotelMapsLink opens a Maps search for the named hotel (or hotels in the city).
function hotelMapsLink(name: string, city: string): string {
  const q = name ? `${name} ${city}` : `hotels in ${city}`;
  return `https://www.google.com/maps/search/?api=1&query=${encodeURIComponent(q)}`;
}

function asOf(iso?: string): string {
  if (!iso) return "";
  const d = new Date(iso);
  return isNaN(d.getTime()) ? "" : d.toISOString().slice(0, 10);
}

function Row({
  left,
  value,
  currency,
  source,
  fetchedAt,
  estimated,
  url,
  labels,
  fmt,
  children,
}: {
  left: React.ReactNode;
  value: number;
  currency: string;
  source: string;
  fetchedAt?: string;
  estimated: boolean;
  url?: string;
  labels: { source: string; asOf: string; est: string };
  fmt: (n: number) => string;
  children?: React.ReactNode;
}) {
  const src = sourceLabel(source);
  const date = asOf(fetchedAt);
  return (
    <li className="flex flex-col gap-0.5 border-t border-border/60 py-1.5 text-sm first:border-t-0">
      <div className="flex items-center justify-between gap-2">
        <span className="min-w-0 truncate">{left}</span>
        <span className="shrink-0 font-semibold tabular-nums">
          {fmt(value)} {currency}
        </span>
      </div>
      <div className="flex flex-wrap items-center gap-x-2 text-xs text-muted">
        {estimated || !src ? (
          <span className="text-accent">{labels.est}</span>
        ) : (
          <span>
            {labels.source}: {src}
          </span>
        )}
        {date && <span>· {labels.asOf} {date}</span>}
        {url && (
          <a href={url} target="_blank" rel="noopener noreferrer" className="text-primary hover:underline">
            ↗
          </a>
        )}
      </div>
      {children}
    </li>
  );
}

// CostBreakdown renders expandable Transport / Hotels / Activities sections with
// each line's amount, REAL source deep-link + "as of", and — for transport — the
// full provenance chain (every source checked, chosen one marked).
export function CostBreakdown({
  legs,
  hotels,
  activities,
  activityItems,
  food,
  foodTotal = 0,
  currency,
  labels,
  fmt,
  catLabels,
}: {
  legs: PlanLeg[];
  hotels: HotelLine[];
  activities: number;
  // D1: when present, the Activities section expands into one row per planned
  // place (city · name · free/~price · Maps), and its sum is the figure shown.
  activityItems?: { city: string; name: string; free: boolean; price?: string }[];
  // D4: approximate per-city restaurant spend ("~" labeled), its own category.
  food?: FoodLine[];
  foodTotal?: number;
  city?: (name: string) => string;
  currency: string;
  labels: {
    source: string;
    asOf: string;
    est: string;
    chosen: string;
    baggage: string;
    free: string;
    approx: string;
    maps: string;
    hotel: string;
  };
  catLabels: { transport: string; hotels: string; activities: string; food: string };
  fmt: (n: number) => string;
}) {
  const sum = (arr: number[]) => arr.reduce((a, b) => a + b, 0);
  const section =
    "group rounded-xl border border-border bg-surface-2/40 px-3 py-2 [&_summary]:cursor-pointer";

  return (
    <div className="mt-4 space-y-2">
      <details className={section} open>
        <summary className="flex items-center justify-between text-sm font-semibold">
          <span>{catLabels.transport}</span>
          <span className="tabular-nums">{fmt(sum(legs.map((l) => l.price)))} {currency}</span>
        </summary>
        <ul className="mt-1">
          {legs.map((l, i) => (
            <Row
              key={i}
              left={`${l.from} → ${l.to} · ${l.mode}`}
              value={l.price}
              currency={currency}
              source={l.source}
              fetchedAt={l.fetched_at}
              estimated={l.estimated}
              url={l.url}
              labels={labels}
              fmt={fmt}
            >
              {l.alternatives && l.alternatives.length > 0 && (
                <ul className="mt-1 space-y-1 pl-3 text-xs text-muted">
                  {l.alternatives.map((a, j) => (
                    <li key={j} className="flex flex-col gap-0.5">
                      <div className="flex flex-wrap items-center gap-x-2 gap-y-0.5">
                        <a href={a.url} target="_blank" rel="noopener noreferrer" className="text-primary hover:underline">
                          {a.source} ↗
                        </a>
                        <span className="tabular-nums font-medium text-text">{fmt(a.price)} {currency}</span>
                        {a.operator && <span>· {a.operator}</span>}
                        {a.depart && <span>· {a.depart}</span>}
                        {a.duration && <span>· {a.duration}</span>}
                        {a.stops && <span>· {a.stops}</span>}
                        {a.chosen && <span className="text-success">✓ {labels.chosen}</span>}
                      </div>
                      {a.baggage && (
                        <div className="text-[11px] text-muted/80">🧳 {labels.baggage}: {a.baggage}</div>
                      )}
                    </li>
                  ))}
                </ul>
              )}
            </Row>
          ))}
        </ul>
      </details>

      <details className={section}>
        <summary className="flex items-center justify-between text-sm font-semibold">
          <span>{catLabels.hotels}</span>
          <span className="tabular-nums">{fmt(sum(hotels.map((h) => h.total)))} {currency}</span>
        </summary>
        <ul className="mt-1">
          {hotels.map((h, i) => (
            <Row
              key={i}
              left={
                <span className="min-w-0">
                  <span className="font-medium">{h.name || `${h.stars}★ ${labels.hotel}`}</span>
                  <span className="block text-xs text-muted">
                    {h.city} · {h.nights} × {fmt(h.nightly)} {currency}
                  </span>
                </span>
              }
              value={h.total}
              currency={currency}
              source={h.source}
              fetchedAt={h.fetched_at}
              estimated={h.estimated}
              labels={labels}
              fmt={fmt}
            >
              <div className="mt-0.5 flex flex-wrap gap-x-3 text-xs">
                {h.url && (
                  <a href={h.url} target="_blank" rel="noopener noreferrer" className="text-primary hover:underline">
                    Booking ↗
                  </a>
                )}
                <a
                  href={hotelMapsLink(h.name ?? "", h.city)}
                  target="_blank"
                  rel="noopener noreferrer"
                  className="text-primary hover:underline"
                >
                  📍 {labels.maps}
                </a>
              </div>
            </Row>
          ))}
        </ul>
      </details>

      <details className={section}>
        <summary className="flex items-center justify-between text-sm font-semibold">
          <span>{catLabels.activities}</span>
          <span className="tabular-nums">{fmt(activities)} {currency}</span>
        </summary>
        {activityItems && activityItems.length > 0 ? (
          <>
            <ul className="mt-1">
              {activityItems.map((a, i) => (
                <li
                  key={i}
                  className="flex items-center justify-between gap-2 border-t border-border/60 py-1.5 text-sm first:border-t-0"
                >
                  <span className="min-w-0 truncate">
                    <span className="text-muted">{a.city} · </span>
                    {a.name}
                  </span>
                  <span className="flex shrink-0 items-center gap-2 text-xs">
                    {a.free ? (
                      <span className="font-medium text-success">{labels.free}</span>
                    ) : (
                      <span className="text-accent">{a.price || labels.approx}</span>
                    )}
                    <a
                      href={mapsLink(a.name, a.city)}
                      target="_blank"
                      rel="noopener noreferrer"
                      title={labels.maps}
                      className="text-primary hover:underline"
                    >
                      ↗
                    </a>
                  </span>
                </li>
              ))}
            </ul>
            <p className="mt-1 text-[11px] text-accent">{labels.est}</p>
          </>
        ) : (
          <ul className="mt-1">
            <Row
              left={catLabels.activities}
              value={activities}
              currency={currency}
              source=""
              estimated
              labels={labels}
              fmt={fmt}
            />
          </ul>
        )}
      </details>

      {food && food.length > 0 && (
        <details className={section}>
          <summary className="flex items-center justify-between text-sm font-semibold">
            <span>{catLabels.food}</span>
            <span className="tabular-nums">~{fmt(foodTotal)} {currency}</span>
          </summary>
          <ul className="mt-1">
            {food.map((f, i) => (
              <li
                key={i}
                className="flex items-center justify-between gap-2 border-t border-border/60 py-1.5 text-sm first:border-t-0"
              >
                <span className="min-w-0 truncate">
                  <span>{f.city}</span>
                  <span className="ml-1 text-xs text-muted">
                    · {f.nights} × ~{fmt(f.per_day)} {currency}
                  </span>
                </span>
                <span className="shrink-0 tabular-nums text-sm">
                  ~{fmt((f.total ?? f.per_day * f.nights))} {currency}
                </span>
              </li>
            ))}
          </ul>
          <p className="mt-1 text-[11px] text-accent">{labels.est}</p>
        </details>
      )}
    </div>
  );
}
