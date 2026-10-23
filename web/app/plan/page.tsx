"use client";

import { useCallback, useEffect, useState } from "react";
import {
  ACTIVITY_PER_NIGHT_EUR,
  buildMatrix,
  CITIES,
  CURRENCIES,
  City,
  INF,
  legFor,
  ModePref,
  NIGHTLY_BY_STAR,
  RATES,
  Tier,
  TIER_STARS,
} from "../../lib/sample";
import {
  Breakdown,
  destinations,
  PricedHotel,
  PricedLeg,
  price,
  RouteDTO,
  sequence,
} from "../../lib/api";
import { LOCALE_CURRENCY, useI18n } from "../../lib/i18n";
import { AppShell } from "../../components/AppShell";
import { Badge, Button, Card, Chip, Field, NumberInput, Select } from "../../components/ui";
import { RouteTimeline, TimelineLeg } from "../../components/RouteTimeline";
import { CostDonut, CostSegment } from "../../components/CostDonut";
import { RouteMap, MapPoint } from "../../components/RouteMap";
import { PinIcon, RouteIcon, WalletIcon } from "../../components/icons";
import Link from "next/link";

interface Planned {
  cities: City[];
  route: RouteDTO;
  alt: RouteDTO;
}

interface UILeg {
  mode: string; // raw mode key
  label: string; // localized
  price: number;
}

export default function PlanPage() {
  const { t, locale } = useI18n();

  // City list is dynamic: fetched from the knowledge base (GET /data/destinations)
  // with the demo dataset as an offline fallback — no hardcoded chip list.
  const [cityList, setCityList] = useState<City[]>(CITIES);
  const [selected, setSelected] = useState<string[]>(["BAK", "TBS", "IST"]);
  const [start, setStart] = useState("BAK");
  const [tier, setTier] = useState<Tier>("standard");
  const [pref, setPref] = useState<ModePref>("cheapest");
  const [nights, setNights] = useState(2);
  const [currency, setCurrency] = useState<string>(LOCALE_CURRENCY[locale]);

  const [planned, setPlanned] = useState<Planned | null>(null);
  const [legs, setLegs] = useState<UILeg[]>([]);
  const [breakdown, setBreakdown] = useState<Breakdown | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  // Load the dynamic city list once. If the API is unreachable we keep the demo
  // list, so the page stays fully functional offline.
  useEffect(() => {
    let cancelled = false;
    destinations()
      .then((ds) => {
        if (cancelled || !ds || ds.length === 0) return;
        const mapped: City[] = ds.map((d) => ({
          code: d.city,
          name: d.city,
          lat: d.lat,
          lng: d.lng,
        }));
        setCityList(mapped);
        const codes = mapped.map((c) => c.code);
        setSelected(codes.slice(0, Math.min(3, codes.length)));
        setStart(codes[0]);
        setPlanned(null);
        setBreakdown(null);
      })
      .catch(() => {
        /* API offline — keep the demo list */
      });
    return () => {
      cancelled = true;
    };
  }, []);

  function toggleCity(code: string) {
    setPlanned(null); // city-set change invalidates the sequence
    setBreakdown(null);
    setSelected((cur) =>
      cur.includes(code) ? cur.filter((c) => c !== code) : [...cur, code]
    );
  }

  const reprice = useCallback(
    async (p: Planned) => {
      const order = p.route.order;
      const stars = TIER_STARS[tier];
      const nightly = NIGHTLY_BY_STAR[stars];

      const uiLegs: UILeg[] = [];
      const apiLegs: PricedLeg[] = [];
      for (let i = 0; i + 1 < order.length; i++) {
        const a = p.cities[order[i]];
        const b = p.cities[order[i + 1]];
        const leg = legFor(a, b, pref);
        if (leg.price >= INF) {
          setError(
            t("err.modeImpossible", { mode: t(`mode.${pref}`), from: a.name, to: b.name })
          );
          return;
        }
        uiLegs.push({ mode: leg.mode, label: t(`mode.${leg.mode}`), price: leg.price });
        apiLegs.push({
          from: order[i],
          to: order[i + 1],
          mode: leg.mode,
          price: { amount: leg.price, currency: "EUR" },
        });
      }

      const hotels: PricedHotel[] = order.map((idx) => ({
        city: idx,
        stars,
        nights,
        nightly: { amount: nightly, currency: "EUR" },
      }));
      const activities = ACTIVITY_PER_NIGHT_EUR * nights * order.length;

      try {
        setError(null);
        const b = await price({
          order,
          legs: apiLegs,
          hotels,
          activities_total: { amount: activities, currency: "EUR" },
          display_currency: currency,
          rates: RATES,
        });
        setLegs(uiLegs);
        setBreakdown(b);
      } catch {
        setError(t("err.price"));
      }
    },
    [tier, pref, nights, currency, t]
  );

  async function onPlan() {
    if (selected.length < 2) {
      setError(t("err.minCities"));
      return;
    }
    if (!selected.includes(start)) {
      setError(t("err.startInvalid"));
      return;
    }
    const cities = cityList.filter((c) => selected.includes(c.code));
    const startIdx = cities.findIndex((c) => c.code === start);
    const matrix = buildMatrix(cities, pref);
    setLoading(true);
    setError(null);
    try {
      const res = await sequence(matrix, startIdx, -1);
      const p: Planned = { cities, route: res.best, alt: res.alt };
      setPlanned(p);
      await reprice(p);
    } catch {
      setError(t("err.sequence"));
      setPlanned(null);
    } finally {
      setLoading(false);
    }
  }

  // Live re-price when tier/mode/nights/currency change (order untouched).
  useEffect(() => {
    if (planned) void reprice(planned);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [tier, pref, nights, currency]);

  const fmt = (n: number) =>
    n.toLocaleString(undefined, { maximumFractionDigits: 2 });
  const variantTone = (v: string) => (v === "open_jaw" ? "accent" : "primary");
  const variantLabel = (v: string) => (v === "open_jaw" ? t("res.openjaw") : t("res.roundtrip"));

  const timelineLegs: TimelineLeg[] = legs.map((l) => ({
    mode: l.mode,
    label: l.label,
    price: `${fmt(l.price)} EUR`,
  }));
  const stops = planned ? planned.route.order.map((idx) => planned.cities[idx].name) : [];
  const mapPoints: MapPoint[] = planned
    ? planned.route.order.map((idx) => {
        const c = planned.cities[idx];
        return { name: c.name, lat: c.lat, lng: c.lng };
      })
    : [];

  const segments: CostSegment[] = breakdown
    ? [
        { key: "transport", label: t("res.transport"), value: breakdown.transport, colorVar: "--transport" },
        { key: "hotels", label: t("res.hotels"), value: breakdown.hotels, colorVar: "--hotels" },
        { key: "activities", label: t("res.activities"), value: breakdown.activities, colorVar: "--activities" },
      ]
    : [];

  return (
    <AppShell>
      <main className="mx-auto max-w-5xl px-4 pb-16">
        <section className="pt-10">
          <Link href="/" className="text-sm font-medium text-muted transition-colors hover:text-primary">
            {t("plan.back")}
          </Link>
          <h1 className="mt-3 font-display text-3xl font-extrabold tracking-tight">
            {t("plan.title")}
          </h1>
          <p className="mt-1.5 max-w-2xl text-sm text-muted">{t("plan.subtitle")}</p>
        </section>

        {/* Wizard */}
        <Card className="mt-6 animate-fade-up">
          <div className="flex items-center gap-2 text-sm font-semibold">
            <PinIcon width={16} height={16} className="text-primary" />
            {t("plan.cities")}
          </div>
          <div className="mt-3 flex flex-wrap gap-2">
            {cityList.map((c) => (
              <Chip key={c.code} active={selected.includes(c.code)} onClick={() => toggleCity(c.code)}>
                {c.name}
              </Chip>
            ))}
          </div>

          <div className="mt-6 grid grid-cols-2 gap-4 sm:grid-cols-4">
            <Field label={t("plan.start")}>
              <Select value={start} onChange={(e) => setStart(e.target.value)}>
                {cityList.filter((c) => selected.includes(c.code)).map((c) => (
                  <option key={c.code} value={c.code}>{c.name}</option>
                ))}
              </Select>
            </Field>
            <Field label={t("plan.comfort")}>
              <Select value={tier} onChange={(e) => setTier(e.target.value as Tier)}>
                <option value="budget">{t("tier.budget")}</option>
                <option value="standard">{t("tier.standard")}</option>
                <option value="vip">{t("tier.vip")}</option>
              </Select>
            </Field>
            <Field label={t("plan.transport")}>
              <Select value={pref} onChange={(e) => setPref(e.target.value as ModePref)}>
                <option value="cheapest">{t("mode.cheapest")}</option>
                <option value="flight">{t("mode.flight")}</option>
                <option value="train">{t("mode.train")}</option>
                <option value="bus">{t("mode.bus")}</option>
              </Select>
            </Field>
            <Field label={t("plan.nights")}>
              <NumberInput
                type="number"
                min={1}
                max={10}
                value={nights}
                onChange={(e) => setNights(Math.max(1, Number(e.target.value)))}
              />
            </Field>
          </div>

          <div className="mt-6 flex flex-wrap items-center gap-4">
            <Button onClick={onPlan} disabled={loading}>
              {loading ? (
                <>
                  <span className="h-4 w-4 animate-spin rounded-full border-2 border-primary-fg/40 border-t-primary-fg" />
                  {t("plan.calculating")}
                </>
              ) : (
                <>
                  <RouteIcon width={16} height={16} />
                  {t("plan.calc")}
                </>
              )}
            </Button>
            <label className="flex items-center gap-2 text-sm text-muted">
              {t("plan.currency")}
              <Select
                value={currency}
                onChange={(e) => setCurrency(e.target.value)}
                className="w-auto py-1.5"
              >
                {CURRENCIES.map((c) => (
                  <option key={c} value={c}>{c}</option>
                ))}
              </Select>
            </label>
          </div>
        </Card>

        {error && (
          <p className="mt-4 animate-fade-in rounded-xl border border-danger/30 bg-danger/10 px-4 py-3 text-sm font-medium text-danger">
            {error}
          </p>
        )}

        {/* Empty state */}
        {!planned && !error && (
          <Card className="mt-6 flex flex-col items-center py-14 text-center">
            <span className="grid h-14 w-14 place-items-center rounded-2xl bg-primary/10 text-primary">
              <RouteIcon width={26} height={26} />
            </span>
            <h2 className="mt-4 font-display text-lg font-bold">{t("res.emptyTitle")}</h2>
            <p className="mt-1.5 max-w-sm text-sm text-muted">{t("res.emptyDesc")}</p>
          </Card>
        )}

        {/* Results */}
        {planned && breakdown && (
          <section className="mt-6 space-y-4">
            <div className="grid gap-4 lg:grid-cols-5">
              {/* Roadmap + timeline */}
              <Card className="animate-fade-up lg:col-span-3">
                <div className="flex items-center justify-between">
                  <h2 className="font-display text-lg font-bold">{t("res.planTitle")}</h2>
                  <Badge tone={variantTone(planned.route.variant)}>
                    {variantLabel(planned.route.variant)}
                  </Badge>
                </div>
                <div className="mt-4">
                  <RouteTimeline stops={stops} legs={timelineLegs} />
                </div>
                {planned.alt && planned.alt.order && planned.alt.order.length > 0 && (
                  <p className="mt-4 border-t border-border pt-3 text-xs text-muted">
                    {t("res.alt")} ({variantLabel(planned.alt.variant)}): ~{fmt(planned.alt.cost)} EUR
                  </p>
                )}
              </Card>

              {/* Live cost */}
              <Card className="animate-fade-up lg:col-span-2" >
                <div className="flex items-center gap-2">
                  <WalletIcon width={18} height={18} className="text-activities" />
                  <h2 className="font-display text-lg font-bold">{t("res.liveCost")}</h2>
                </div>
                <div className="mt-5">
                  <CostDonut
                    segments={segments}
                    total={breakdown.total}
                    currency={breakdown.currency}
                    fmt={fmt}
                  />
                </div>
                <p className="mt-4 text-xs leading-relaxed text-muted">{t("res.disclaimer")}</p>
              </Card>
            </div>

            {/* Map */}
            <Card className="animate-fade-up">
              <div className="mb-3 flex items-center gap-2">
                <PinIcon width={18} height={18} className="text-primary" />
                <h2 className="font-display text-lg font-bold">{t("res.map")}</h2>
              </div>
              <RouteMap points={mapPoints} closed={planned.route.variant === "round_trip"} />
            </Card>
          </section>
        )}
      </main>
    </AppShell>
  );
}
