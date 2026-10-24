"use client";

import { HotelLine, Roadmap } from "../lib/api";
import { PinIcon } from "./icons";
import { POIMap, POIPoint } from "./POIMap";
import { CityPOIs } from "./CityPOIs";

// mapsLink builds a real Google Maps place search for a POI (opens the actual
// place, never an empty search).
function mapsLink(name: string, city: string): string {
  return `https://www.google.com/maps/search/?api=1&query=${encodeURIComponent(name + " " + city)}`;
}

function hotelMapsLink(name: string, city: string): string {
  const q = name ? `${name} ${city}` : `hotels in ${city}`;
  return `https://www.google.com/maps/search/?api=1&query=${encodeURIComponent(q)}`;
}

// priceNum pulls the first number out of an approximate price string like
// "~€15", "€12-18" or "15 EUR" (0 when none / free).
function priceNum(price?: string): number {
  if (!price) return 0;
  const m = price.replace(",", ".").match(/\d+(\.\d+)?/);
  return m ? parseFloat(m[0]) : 0;
}

function round2(n: number): number {
  return Math.round(n * 100) / 100;
}

// RoadmapCards renders the structured, day-by-day, location-based itinerary:
// a POI map on top, then one card per day grouped under its city (with the
// city's summary, best season and nights). No raw markdown — every POI is a row
// with a real Google Maps link.
export function RoadmapCards({
  roadmap,
  travelers = 1,
  hotels,
  onNights,
  nightsFor,
  labels,
}: {
  roadmap: Roadmap;
  travelers?: number;
  hotels?: HotelLine[];
  // D9: editable per-city nights, synced with the D7 list. Editing re-plans; the
  // "Regenerate" button then rebuilds the roadmap with the new nights.
  onNights?: (city: string, n: number) => void;
  nightsFor?: (city: string) => number;
  labels: {
    day: (n: number) => string;
    nights: (n: number) => string;
    season: string;
    free: string;
    approx: string;
    maps: string;
    activitiesEst: string;
    perPerson: string;
    hotel: string;
    perNight: string;
    regenHint: string;
  };
}) {
  const cityInfo = new Map(roadmap.cities.map((c) => [c.city.toLowerCase(), c]));
  const hotelByCity = new Map((hotels ?? []).map((h) => [h.city.toLowerCase(), h]));

  // Per-person activities estimate = sum of listed POI prices (approximate).
  const perPersonActivities = round2(
    roadmap.days.reduce((s, d) => s + (d.pois ?? []).reduce((a, p) => a + priceNum(p.price), 0), 0)
  );
  const groupActivities = round2(perPersonActivities * Math.max(1, travelers));
  const dayCost = (d: Roadmap["days"][number]) =>
    round2((d.pois ?? []).reduce((a, p) => a + priceNum(p.price), 0));

  // All POIs with coordinates -> map pins (with a real Maps link).
  const points: POIPoint[] = [];
  for (const d of roadmap.days) {
    for (const p of d.pois ?? []) {
      if ((p.lat ?? 0) !== 0 || (p.lng ?? 0) !== 0) {
        points.push({ name: p.name, lat: p.lat ?? 0, lng: p.lng ?? 0, day: d.day, mapsUrl: mapsLink(p.name, d.city) });
      }
    }
  }

  let lastCity = "";

  return (
    <div className="space-y-4">
      {points.length > 0 && <POIMap points={points} />}

      <div className="relative space-y-3 pl-8">
        {/* Vertical rail connecting the whole roadmap (days read as one flow). */}
        {roadmap.days.length > 0 && (
          <span
            aria-hidden
            className="pointer-events-none absolute bottom-3 left-[13px] top-3 w-0.5 bg-gradient-to-b from-primary/50 via-primary/30 to-accent/40"
          />
        )}
        {roadmap.days.map((d, i) => {
          const isNewCity = d.city.toLowerCase() !== lastCity;
          lastCity = d.city.toLowerCase();
          const info = cityInfo.get(d.city.toLowerCase());

          return (
            <div key={i}>
              {isNewCity && info && (
                <div className="mb-2 mt-1">
                  <div className="flex flex-wrap items-center gap-2">
                    <h3 className="font-display text-base font-bold">{info.city}</h3>
                    {onNights ? (
                      // Editable nights (D9), bound to the live value so it stays
                      // in sync with the D7 list; editing re-plans.
                      <span className="inline-flex items-center gap-1 rounded-full bg-primary/10 px-1.5 py-0.5 text-xs font-medium text-primary">
                        <button
                          type="button"
                          onClick={() => onNights(info.city, (nightsFor ? nightsFor(info.city) : info.nights) - 1)}
                          aria-label={`fewer nights in ${info.city}`}
                          className="grid h-4 w-4 place-items-center rounded-full hover:bg-primary/20"
                        >
                          −
                        </button>
                        <span className="tabular-nums">{labels.nights(nightsFor ? nightsFor(info.city) : info.nights)}</span>
                        <button
                          type="button"
                          onClick={() => onNights(info.city, (nightsFor ? nightsFor(info.city) : info.nights) + 1)}
                          aria-label={`more nights in ${info.city}`}
                          className="grid h-4 w-4 place-items-center rounded-full hover:bg-primary/20"
                        >
                          +
                        </button>
                      </span>
                    ) : (
                      info.nights > 0 && (
                        <span className="rounded-full bg-primary/10 px-2.5 py-1 text-xs font-medium text-primary">
                          {labels.nights(info.nights)}
                        </span>
                      )
                    )}
                    {/* Hint when the live nights no longer match this roadmap. */}
                    {onNights && nightsFor && nightsFor(info.city) !== info.nights && (
                      <span className="rounded-full bg-accent/10 px-2 py-0.5 text-[11px] font-medium text-accent">
                        {labels.regenHint}
                      </span>
                    )}
                    {info.best_season && (
                      <span className="rounded-full bg-accent/10 px-2.5 py-1 text-xs font-medium text-accent">
                        {labels.season}: {info.best_season}
                      </span>
                    )}
                  </div>
                  {info.summary && <p className="mt-1 text-sm leading-relaxed text-muted">{info.summary}</p>}
                  {(() => {
                    const h = hotelByCity.get(d.city.toLowerCase());
                    if (!h) return null;
                    return (
                      <div className="mt-1.5 flex flex-wrap items-center gap-x-3 gap-y-1 rounded-xl border border-border bg-surface-2/50 px-3 py-1.5 text-xs">
                        <span className="font-medium text-text">🏨 {h.name || `${h.stars}★ ${labels.hotel}`}</span>
                        <span className="text-muted">
                          ~{h.nightly} / {labels.perNight}
                        </span>
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
                    );
                  })()}
                  <CityPOIs city={info.city} />
                </div>
              )}

              <div
                className="animate-fade-up relative rounded-2xl border border-border bg-surface-2/40 p-4"
                style={{ animationDelay: `${Math.min(i, 8) * 50}ms` }}
              >
                {/* Day node sitting on the rail. */}
                <span className="absolute -left-[26px] top-4 z-10 grid h-6 w-6 place-items-center rounded-full bg-primary text-xs font-bold text-primary-fg shadow-soft ring-2 ring-surface tabular-nums">
                  {d.day}
                </span>
                <div className="flex items-center justify-between gap-2">
                  <div className="flex min-w-0 items-center gap-2">
                    <span className="truncate text-sm font-semibold">
                      {labels.day(d.day)} · {d.city}
                      {d.title ? ` — ${d.title}` : ""}
                    </span>
                  </div>
                  {dayCost(d) > 0 && (
                    <span className="shrink-0 rounded-full bg-activities/10 px-2 py-0.5 text-xs font-medium text-activities tabular-nums">
                      ~€{dayCost(d)}
                    </span>
                  )}
                </div>

                {d.pois && d.pois.length > 0 && (
                  <ol className="mt-3 space-y-1.5">
                    {d.pois.map((p, j) => (
                      <li
                        key={j}
                        className="flex items-start justify-between gap-2 rounded-xl bg-surface px-3 py-2 text-sm"
                      >
                        <span className="flex min-w-0 items-start gap-2">
                          <a
                            href={mapsLink(p.name, d.city)}
                            target="_blank"
                            rel="noopener noreferrer"
                            title={labels.maps}
                            className="mt-0.5 shrink-0 text-muted hover:text-primary"
                          >
                            <PinIcon width={14} height={14} />
                          </a>
                          <span className="min-w-0">
                            <span className="font-medium">{p.name}</span>
                            {p.type && <span className="ml-1.5 text-xs text-muted">· {p.type}</span>}
                            {p.desc && <span className="block text-xs leading-snug text-muted">{p.desc}</span>}
                          </span>
                        </span>
                        <span className="shrink-0 text-xs">
                          {p.free ? (
                            <span className="rounded-full bg-success/10 px-2 py-0.5 font-medium text-success">
                              {labels.free}
                            </span>
                          ) : (
                            <span className="text-accent">{p.price || labels.approx}</span>
                          )}
                        </span>
                      </li>
                    ))}
                  </ol>
                )}
              </div>
            </div>
          );
        })}
      </div>

      {perPersonActivities > 0 && (
        <div className="flex flex-wrap items-center justify-between gap-2 rounded-2xl border border-activities/30 bg-activities/5 px-4 py-3 text-sm">
          <span className="font-medium text-activities">{labels.activitiesEst}</span>
          <span className="tabular-nums">
            <span className="font-semibold">~€{groupActivities}</span>
            {travelers > 1 && (
              <span className="ml-2 text-muted">
                (~€{perPersonActivities} / {labels.perPerson})
              </span>
            )}
          </span>
        </div>
      )}
    </div>
  );
}
