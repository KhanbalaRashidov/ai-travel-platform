"use client";

import { useState } from "react";
import { ApiError, fetchPOIs, POIItem } from "../lib/api";
import { useI18n } from "../lib/i18n";

function mapsLink(name: string, city: string): string {
  return `https://www.google.com/maps/search/?api=1&query=${encodeURIComponent(name + " " + city)}`;
}

// CityPOIs lazily loads REAL, open-data points of interest for a city from the
// OSM Overpass connector (provenance-tagged, not estimated). It shows a free
// image when OSM has one, the category, a Google Maps link (ratings/reviews live
// there) and the official site / Wikipedia when tagged. Opens on demand so it
// never slows the roadmap.
export function CityPOIs({ city }: { city: string }) {
  const { t } = useI18n();
  const [open, setOpen] = useState(false);
  const [items, setItems] = useState<POIItem[] | null>(null);
  const [loading, setLoading] = useState(false);
  const [err, setErr] = useState<string | null>(null);

  async function toggle() {
    const next = !open;
    setOpen(next);
    if (next && items === null && !loading) {
      setLoading(true);
      setErr(null);
      try {
        setItems(await fetchPOIs(city, 12));
      } catch (e) {
        setErr(e instanceof ApiError && e.status === 502 ? t("poi.none") : t("poi.error"));
      } finally {
        setLoading(false);
      }
    }
  }

  return (
    <div className="mt-2">
      <button
        type="button"
        onClick={toggle}
        className="text-xs font-medium text-primary hover:underline"
      >
        {open ? "▾" : "▸"} {t("poi.title")}
      </button>

      {open && (
        <div className="mt-2">
          {loading && <div className="h-20 w-full animate-pulse rounded-xl bg-surface-2" />}
          {err && <p className="text-xs text-muted">{err}</p>}
          {items && items.length === 0 && !err && <p className="text-xs text-muted">{t("poi.none")}</p>}
          {items && items.length > 0 && (
            <>
              <div className="grid grid-cols-2 gap-2 sm:grid-cols-3">
                {items.map((p, i) => (
                  <div key={i} className="overflow-hidden rounded-xl border border-border bg-surface">
                    {p.image ? (
                      // eslint-disable-next-line @next/next/no-img-element
                      <img
                        src={p.image}
                        alt={p.name}
                        loading="lazy"
                        className="h-20 w-full object-cover"
                        onError={(e) => {
                          (e.currentTarget as HTMLImageElement).style.display = "none";
                        }}
                      />
                    ) : (
                      <div className="flex h-20 w-full items-center justify-center bg-surface-2 text-xl text-muted">
                        📍
                      </div>
                    )}
                    <div className="p-2">
                      <div className="truncate text-xs font-semibold" title={p.name}>
                        {p.name}
                      </div>
                      <div className="flex items-center justify-between gap-1">
                        <span className="truncate text-[11px] text-muted">{p.category}</span>
                        {/* Rating only when a real source provided it — no fake stars. */}
                        {typeof p.rating === "number" && p.rating > 0 && (
                          <span className="shrink-0 text-[11px] text-accent">
                            ★ {p.rating.toFixed(1)}
                            {p.review_count ? ` (${p.review_count})` : ""}
                          </span>
                        )}
                      </div>
                      {p.desc && <p className="mt-0.5 line-clamp-2 text-[11px] leading-snug text-muted">{p.desc}</p>}
                      {(p.free || p.price) && (
                        <div className="mt-0.5 text-[11px]">
                          {p.free ? (
                            <span className="font-medium text-success">{t("res.free")}</span>
                          ) : (
                            <span className="text-accent">{p.price || t("res.approx")}</span>
                          )}
                        </div>
                      )}
                      <div className="mt-1 flex flex-wrap gap-x-2 text-[11px]">
                        <a
                          href={mapsLink(p.name, p.city)}
                          target="_blank"
                          rel="noopener noreferrer"
                          className="text-primary hover:underline"
                        >
                          {t("res.maps")} ↗
                        </a>
                        {p.wiki && (
                          <a href={p.wiki} target="_blank" rel="noopener noreferrer" className="text-muted hover:text-primary">
                            wiki
                          </a>
                        )}
                        {p.website && (
                          <a href={p.website} target="_blank" rel="noopener noreferrer" className="text-muted hover:text-primary">
                            {t("poi.site")}
                          </a>
                        )}
                      </div>
                    </div>
                  </div>
                ))}
              </div>
              <p className="mt-1.5 text-[11px] text-muted">
                {t("poi.source")}
                {items.some((i) => i.provenance?.source === "ai-knowledge") ? ` · ${t("poi.sourceAI")}` : ""}
              </p>
            </>
          )}
        </div>
      )}
    </div>
  );
}
