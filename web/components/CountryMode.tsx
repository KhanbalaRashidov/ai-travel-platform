"use client";

import { useState } from "react";
import {
  ApiError,
  DestinationSuggestions,
  Place,
  searchPlaces,
  suggestDestinations,
} from "../lib/api";
import { canonicalName, countryName, CountryOption, localizeCountry, searchCountries } from "../lib/countries";
import { useI18n } from "../lib/i18n";
import { Button } from "./ui";
import { PinIcon, SparkIcon } from "./icons";

// CountryMode (D12): the user types one or more COUNTRIES (+ optional budget);
// the AI suggests the most-recommended cities per country (reason + best season,
// and ratings ONLY when a real source has them — never invented). Picking a city
// resolves its real coordinates via the places search and adds it to the trip,
// where the normal flow (nights, sequence, pricing, roadmap) takes over.
export function CountryMode({
  budget,
  currency,
  isSelected,
  onAddCity,
}: {
  budget: number;
  currency: string;
  isSelected: (name: string) => boolean;
  onAddCity: (p: Place) => void;
}) {
  const { t, locale } = useI18n();
  const [open, setOpen] = useState(false);
  const [input, setInput] = useState("");
  // Store only ISO codes; the display name is always derived from the CURRENT
  // locale (D20), so switching language re-labels chips/suggestions instantly.
  const [codes, setCodes] = useState<string[]>([]);
  const [data, setData] = useState<DestinationSuggestions | null>(null);
  const [loading, setLoading] = useState(false);
  const [err, setErr] = useState<string | null>(null);
  const [picking, setPicking] = useState<string | null>(null);

  const matches = searchCountries(input, locale, 8).filter((m) => !codes.includes(m.code));

  function addCountry(c: CountryOption) {
    if (!codes.includes(c.code)) setCodes([...codes, c.code]);
    setInput("");
  }
  function removeCountry(code: string) {
    setCodes(codes.filter((c) => c !== code));
  }

  async function suggest() {
    if (codes.length === 0 || loading) return;
    setLoading(true);
    setErr(null);
    try {
      // Send canonical English names so the AI resolves them regardless of UI language.
      setData(await suggestDestinations(codes.map((c) => canonicalName(c)), budget, currency, locale));
    } catch (e) {
      setErr(e instanceof ApiError && (e.status === 503 || e.status === 429) ? t("ai.busy") : t("ai.error"));
    } finally {
      setLoading(false);
    }
  }

  // Resolve a suggested city to a real place (coords) and add it to the trip.
  async function pick(city: string, country: string) {
    if (isSelected(city)) return;
    setPicking(city);
    try {
      const results = await searchPlaces(city, locale);
      const match =
        results.find((r) => r.kind === "city" && r.country?.toLowerCase() === country.toLowerCase()) ??
        results.find((r) => r.kind === "city") ??
        results[0];
      if (match) onAddCity(match);
      else setErr(t("country.noCoords", { city }));
    } catch {
      setErr(t("country.noCoords", { city }));
    } finally {
      setPicking(null);
    }
  }

  return (
    <div className="mt-4 rounded-xl border border-border bg-surface-2/50 p-3">
      <button
        type="button"
        onClick={() => setOpen(!open)}
        className="flex w-full items-center justify-between gap-2 text-left"
      >
        <span className="flex items-center gap-2 text-sm font-semibold">
          <SparkIcon width={16} height={16} className="text-accent" />
          {t("country.title")}
        </span>
        <span className="text-muted">{open ? "▾" : "▸"}</span>
      </button>

      {open && (
        <div className="mt-3">
          <p className="mb-2 text-xs text-muted">{t("country.hint")}</p>
          <div className="relative">
            <input
              value={input}
              onChange={(e) => setInput(e.target.value)}
              onKeyDown={(e) => {
                if (e.key === "Enter" && matches.length > 0) {
                  e.preventDefault();
                  addCountry(matches[0]);
                }
              }}
              placeholder={t("country.placeholder")}
              className="w-full rounded-xl border border-border bg-surface px-3 py-2 text-sm outline-none focus:border-primary"
            />
            {input.trim() !== "" && matches.length > 0 && (
              <ul className="absolute z-20 mt-1 max-h-56 w-full overflow-auto rounded-xl border border-border bg-surface shadow-soft">
                {matches.map((m) => (
                  <li key={m.code}>
                    <button
                      type="button"
                      // Select on pointer-down so a re-render (input change /
                      // blur) can't drop the pick between mousedown and click.
                      onMouseDown={(e) => {
                        e.preventDefault();
                        addCountry(m);
                      }}
                      className="flex w-full items-center justify-between gap-2 px-3 py-2 text-left text-sm hover:bg-surface-2"
                    >
                      <span className="min-w-0 truncate">{m.name}</span>
                      <span className="shrink-0 text-[11px] text-muted">{m.code}</span>
                    </button>
                  </li>
                ))}
              </ul>
            )}
          </div>

          {codes.length > 0 && (
            <div className="mt-2 flex flex-wrap gap-2">
              {codes.map((code) => {
                const name = countryName(code, locale);
                return (
                  <span
                    key={code}
                    className="inline-flex items-center gap-1.5 rounded-full border border-accent/30 bg-accent/10 px-3 py-1 text-sm font-medium text-accent"
                  >
                    {name}
                    <button type="button" onClick={() => removeCountry(code)} aria-label={`remove ${name}`} className="hover:text-accent">
                      ×
                    </button>
                  </span>
                );
              })}
            </div>
          )}

          <div className="mt-3">
            <Button onClick={suggest} disabled={codes.length === 0 || loading}>
              {loading ? (
                <>
                  <span className="h-4 w-4 animate-spin rounded-full border-2 border-primary-fg/40 border-t-primary-fg" />
                  {t("country.loading")}
                </>
              ) : (
                <>
                  <SparkIcon width={16} height={16} />
                  {t("country.suggest")}
                </>
              )}
            </Button>
          </div>

          {err && <p className="mt-2 text-sm font-medium text-danger">{err}</p>}

          {data && (
            <div className="mt-4 space-y-4">
              {data.budget_advice && (
                <div className="rounded-xl border border-primary/20 bg-primary/5 p-3 text-sm">
                  <div className="mb-1 text-xs font-semibold uppercase tracking-wide text-primary">
                    {t("country.budgetAdvice")}
                  </div>
                  <p className="leading-relaxed text-text">{data.budget_advice}</p>
                </div>
              )}

              {data.countries.map((cs, i) => (
                <div key={i}>
                  <h4 className="mb-2 font-display text-sm font-bold">{localizeCountry(cs.country, locale)}</h4>
                  <div className="grid gap-2 sm:grid-cols-2">
                    {cs.cities.map((c, j) => {
                      const added = isSelected(c.city);
                      return (
                        <div key={j} className="flex flex-col gap-1 rounded-xl border border-border bg-surface p-3">
                          <div className="flex items-center justify-between gap-2">
                            <span className="flex min-w-0 items-center gap-1.5 font-semibold">
                              <span className="grid h-5 w-5 shrink-0 place-items-center rounded-full bg-accent/15 text-[10px] font-bold text-accent">
                                {j + 1}
                              </span>
                              <span className="truncate">{c.city}</span>
                            </span>
                            {/* Rating only when a real source provided it — no fake stars. */}
                            {typeof c.rating === "number" && c.rating > 0 && (
                              <span className="shrink-0 text-xs text-accent">
                                ★ {c.rating.toFixed(1)}
                                {c.review_count ? ` (${c.review_count})` : ""}
                              </span>
                            )}
                          </div>
                          {c.reason && <p className="text-xs leading-snug text-muted">{c.reason}</p>}
                          {c.best_season && (
                            <span className="text-[11px] text-muted">
                              {t("res.season")}: {c.best_season}
                            </span>
                          )}
                          <div className="mt-1">
                            <Button
                              variant={added ? "ghost" : "primary"}
                              onClick={() => pick(c.city, c.country || cs.country)}
                              disabled={added || picking === c.city}
                            >
                              <PinIcon width={13} height={13} />
                              {added ? t("country.added") : picking === c.city ? t("country.adding") : t("country.add")}
                            </Button>
                          </div>
                        </div>
                      );
                    })}
                  </div>
                </div>
              ))}
            </div>
          )}
        </div>
      )}
    </div>
  );
}
