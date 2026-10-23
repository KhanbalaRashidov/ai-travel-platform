"use client";

import { Roadmap } from "../lib/api";
import { PinIcon } from "./icons";
import { POIMap, POIPoint } from "./POIMap";
import { CityPOIs } from "./CityPOIs";

// mapsLink builds a real Google Maps place search for a POI (opens the actual
// place, never an empty search).
function mapsLink(name: string, city: string): string {
  return `https://www.google.com/maps/search/?api=1&query=${encodeURIComponent(name + " " + city)}`;
}

// RoadmapCards renders the structured, day-by-day, location-based itinerary:
// a POI map on top, then one card per day grouped under its city (with the
// city's summary, best season and nights). No raw markdown — every POI is a row
// with a real Google Maps link.
export function RoadmapCards({
  roadmap,
  labels,
}: {
  roadmap: Roadmap;
  labels: {
    day: (n: number) => string;
    nights: (n: number) => string;
    season: string;
    free: string;
    approx: string;
    maps: string;
  };
}) {
  const cityInfo = new Map(roadmap.cities.map((c) => [c.city.toLowerCase(), c]));

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

      <div className="space-y-3">
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
                    {info.nights > 0 && (
                      <span className="rounded-full bg-primary/10 px-2.5 py-1 text-xs font-medium text-primary">
                        {labels.nights(info.nights)}
                      </span>
                    )}
                    {info.best_season && (
                      <span className="rounded-full bg-accent/10 px-2.5 py-1 text-xs font-medium text-accent">
                        {labels.season}: {info.best_season}
                      </span>
                    )}
                  </div>
                  {info.summary && <p className="mt-1 text-sm leading-relaxed text-muted">{info.summary}</p>}
                  <CityPOIs city={info.city} />
                </div>
              )}

              <div
                className="animate-fade-up rounded-2xl border border-border bg-surface-2/40 p-4"
                style={{ animationDelay: `${Math.min(i, 8) * 50}ms` }}
              >
                <div className="flex items-center gap-2">
                  <span className="grid h-7 w-7 shrink-0 place-items-center rounded-full bg-primary text-xs font-bold text-primary-fg tabular-nums">
                    {d.day}
                  </span>
                  <span className="text-sm font-semibold">
                    {labels.day(d.day)} · {d.city}
                    {d.title ? ` — ${d.title}` : ""}
                  </span>
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
    </div>
  );
}
