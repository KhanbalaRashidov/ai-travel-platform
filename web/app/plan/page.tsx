"use client";

import { useEffect, useRef, useState } from "react";
import { CURRENCIES, ModePref, Tier } from "../../lib/sample";
import {
  ApiError,
  EditState,
  ItineraryBrief,
  Place,
  planQuote,
  PlanQuote,
  Roadmap,
  track,
} from "../../lib/api";
import { LOCALE_CURRENCY, useI18n } from "../../lib/i18n";
import { hotelSites, skyscannerFlightLink } from "../../lib/links";
import { AppShell } from "../../components/AppShell";
import { Badge, Button, Card, Field, NumberInput, Select } from "../../components/ui";
import { CityAutocomplete } from "../../components/CityAutocomplete";
import { CountryMode } from "../../components/CountryMode";
import { RouteTimeline, TimelineLeg } from "../../components/RouteTimeline";
import { CostDonut, CostSegment } from "../../components/CostDonut";
import { CostBreakdown } from "../../components/CostBreakdown";
import { RoadmapPrint } from "../../components/RoadmapPrint";
import { RouteMap, MapPoint } from "../../components/RouteMap";
import { AIPanel } from "../../components/AIPanel";
import { useToast } from "../../components/toast";
import { PinIcon, RouteIcon, WalletIcon } from "../../components/icons";
import Link from "next/link";

interface Pending {
  places: Place[];
  origin: Place | null;
  budget: number;
  start: string;
  mode: ModePref;
  tier: Tier;
  nights: number;
  currency: string;
  order?: string[];
  legModes: Record<string, string>;
  startDate: string;
  returnDate: string;
  adults: number;
  children: number;
  infants: number;
  cityNights: Record<string, number>;
  cityStars: Record<string, number>;
}

function sameNames(a: Place[], b: Place[]): boolean {
  if (a.length !== b.length) return false;
  const s = new Set(a.map((x) => x.name.toLowerCase()));
  return b.every((x) => s.has(x.name.toLowerCase()));
}

// Small +/- stepper for passenger counts.
function Stepper({
  label,
  value,
  min,
  onChange,
}: {
  label: string;
  value: number;
  min: number;
  onChange: (v: number) => void;
}) {
  const btn =
    "grid h-7 w-7 place-items-center rounded-lg border border-border text-muted transition-colors hover:border-primary/40 hover:text-text disabled:opacity-40";
  return (
    <div className="flex items-center justify-between gap-2">
      <span className="text-sm">{label}</span>
      <div className="flex items-center gap-2">
        <button type="button" className={btn} disabled={value <= min} onClick={() => onChange(value - 1)}>
          −
        </button>
        <span className="w-5 text-center text-sm font-semibold tabular-nums">{value}</span>
        <button type="button" className={btn} onClick={() => onChange(value + 1)}>
          +
        </button>
      </div>
    </div>
  );
}

export default function PlanPage() {
  const { t, locale } = useI18n();
  const { toast } = useToast();

  // No hardcoded cities: the user searches and adds places (with coordinates).
  const [places, setPlaces] = useState<Place[]>([]);
  const [origin, setOrigin] = useState<Place | null>(null); // departure (no hotel)
  const [budget, setBudget] = useState(0); // EUR; enables minimal mode (AI suggests)
  const [start, setStart] = useState("");
  const [startDate, setStartDate] = useState("");
  const [returnDate, setReturnDate] = useState("");
  const [adults, setAdults] = useState(1);
  const [children, setChildren] = useState(0);
  const [infants, setInfants] = useState(0);
  const [tier, setTier] = useState<Tier>("standard");
  const [pref, setPref] = useState<ModePref>("cheapest");
  const [nights, setNights] = useState(2);
  const [currency, setCurrency] = useState<string>(LOCALE_CURRENCY[locale]);

  const [plan, setPlan] = useState<PlanQuote | null>(null);
  const [roadmap, setRoadmap] = useState<Roadmap | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [pending, setPending] = useState<Pending | null>(null);
  const [copied, setCopied] = useState(false);
  const [legModes, setLegModes] = useState<Record<string, string>>({});
  const [cityNights, setCityNights] = useState<Record<string, number>>({});
  const [cityStars, setCityStars] = useState<Record<string, number>>({});
  const [dragIdx, setDragIdx] = useState<number | null>(null);

  // Local "today" (not UTC) for the date inputs' min — visual validation only:
  // depart ≥ today, return ≥ depart. The backend still owns real validation.
  const todayStr = (() => {
    const d = new Date();
    return new Date(d.getTime() - d.getTimezoneOffset() * 60000).toISOString().slice(0, 10);
  })();

  // Restore a shared plan from ?s= (base64 of the selection) and auto-compute.
  useEffect(() => {
    try {
      const token = new URLSearchParams(window.location.search).get("s");
      if (!token) return;
      const d = JSON.parse(decodeURIComponent(atob(token)));
      const pl: Place[] = (d.p ?? []).map((x: Record<string, unknown>) => ({
        name: String(x.n ?? ""),
        lat: Number(x.a ?? 0),
        lng: Number(x.o ?? 0),
        code: String(x.c ?? ""),
        country: String(x.co ?? ""),
        kind: String(x.k ?? "city"),
      }));
      if (pl.length === 0) return;
      const st = typeof d.s === "string" ? d.s : pl[0].name;
      setPlaces(pl);
      setStart(st);
      if (d.d) setStartDate(d.d);
      if (d.r) setReturnDate(d.r);
      if (["budget", "standard", "vip"].includes(d.t)) setTier(d.t);
      if (["cheapest", "flight", "train", "bus"].includes(d.m)) setPref(d.m);
      if (d.cu) setCurrency(d.cu);
      if (typeof d.ad === "number") setAdults(d.ad);
      if (typeof d.ch === "number") setChildren(d.ch);
      if (typeof d.inf === "number") setInfants(d.inf);
      if (pl.length >= 2) {
        setPending({
          places: pl, origin: null, budget: 0, start: st, mode: d.m ?? "cheapest", tier: d.t ?? "standard",
          nights: 2, currency: d.cu ?? "EUR", order: undefined, legModes: {},
          startDate: d.d ?? "", returnDate: d.r ?? "",
          adults: d.ad ?? 1, children: d.ch ?? 0, infants: d.inf ?? 0, cityNights: {}, cityStars: {},
        });
      }
    } catch {
      /* ignore a malformed share link */
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  function share() {
    try {
      const payload = {
        p: places.map((x) => ({ n: x.name, a: x.lat, o: x.lng, c: x.code, co: x.country, k: x.kind })),
        s: start, d: startDate, r: returnDate, t: tier, m: pref, cu: currency,
        ad: adults, ch: children, inf: infants,
      };
      const token = btoa(encodeURIComponent(JSON.stringify(payload)));
      const url = `${window.location.origin}${window.location.pathname}?s=${token}`;
      void navigator.clipboard?.writeText(url);
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    } catch {
      /* clipboard may be unavailable */
    }
  }

  // Enough to plan: 2+ destinations, or an origin with 1+ destination, or an
  // origin + budget (minimal mode — AI suggests destinations).
  const enoughToPlan = (ps: Place[], o: Place | null, b: number) =>
    ps.length >= 2 || (!!o && (ps.length >= 1 || b > 0));

  // Monotonic request id: a superseded (stale) plan response is ignored so the
  // latest toggle always wins, with no wasted re-render (§6).
  const planSeq = useRef(0);
  // Last attempted request, so the error state can offer a one-click retry.
  const lastTry = useRef<Pending | null>(null);

  async function fetchPlan(p: Pending) {
    if (!enoughToPlan(p.places, p.origin, p.budget)) {
      setError(t("err.minCities"));
      return;
    }
    lastTry.current = p;
    const seq = ++planSeq.current;
    const names = p.places.map((x) => x.name);
    const startName = names.includes(p.start) ? p.start : names[0];
    // Origin (if any) ships its coords in `places` so the backend can geolocate it.
    const coords = [
      ...(p.origin ? [{ name: p.origin.name, lat: p.origin.lat, lng: p.origin.lng }] : []),
      ...p.places.map((x) => ({ name: x.name, lat: x.lat, lng: x.lng })),
    ];
    setLoading(true);
    setError(null);
    try {
      const q = await planQuote({
        cities: names,
        places: coords,
        origin: p.origin?.name,
        budget_eur: p.budget || undefined,
        start: startName,
        mode: p.mode,
        tier: p.tier,
        nights: p.nights,
        currency: p.currency,
        order: p.order,
        variant: p.order ? plan?.variant : undefined,
        leg_modes: p.legModes,
        start_date: p.startDate || undefined,
        return_date: p.returnDate || undefined,
        adults: p.adults,
        children: p.children,
        infants: p.infants,
        city_nights: Object.keys(p.cityNights).length ? p.cityNights : undefined,
        city_stars: Object.keys(p.cityStars).length ? p.cityStars : undefined,
      });
      if (seq !== planSeq.current) return; // a newer request superseded this one
      setPlan(q);
      track("plan_computed");
      toast(p.order ? t("toast.updated") : t("toast.ready"), "success");
    } catch (e) {
      if (seq !== planSeq.current) return;
      const msg = e instanceof ApiError && e.status === 408 ? t("err.timeout") : t("err.plan");
      setError(msg);
      toast(msg, "error");
    } finally {
      if (seq === planSeq.current) setLoading(false);
    }
  }

  // Re-run the last attempted plan (error-state retry button).
  function retryPlan() {
    if (lastTry.current) setPending(lastTry.current);
  }

  // Print/PDF (D15): expand every collapsed <details> (so the cost sections'
  // content is captured) and include the AI day-by-day roadmap, then print.
  function printPlan() {
    try {
      document.querySelectorAll("details").forEach((d) => d.setAttribute("open", ""));
    } catch {
      /* ignore */
    }
    window.print();
  }

  useEffect(() => {
    if (!pending) return;
    const p = pending;
    setPending(null);
    void fetchPlan(p);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [pending]);

  function addPlace(p: Place) {
    if (places.some((x) => x.name.toLowerCase() === p.name.toLowerCase())) return;
    const next = [...places, p];
    setPlaces(next);
    if (!start) setStart(p.name);
    setLegModes({});
    setPlan(null); // city-set change -> recompute manually
    setRoadmap(null); // stale: the city set changed
  }
  function removePlace(name: string) {
    const next = places.filter((x) => x.name !== name);
    setPlaces(next);
    if (start === name) setStart(next[0]?.name ?? "");
    setLegModes({});
    setPlan(null);
    setRoadmap(null);
  }

  function base(order?: string[]): Pending {
    return {
      places,
      origin,
      budget,
      start,
      mode: pref,
      tier,
      nights,
      currency,
      order,
      legModes,
      startDate,
      returnDate,
      adults,
      children,
      infants,
      cityNights,
      cityStars,
    };
  }

  const originThenNames = (ps: Place[]) => [...(origin ? [origin.name] : []), ...ps.map((p) => p.name)];

  // Drag-drop destination reorder: price MY order (no re-sequence). Dropping
  // destination `from` onto `to` reorders and re-plans that exact order.
  function reorderDest(from: number, to: number) {
    if (from === to || from < 0 || to < 0) return;
    const next = [...places];
    const [moved] = next.splice(from, 1);
    next.splice(to, 0, moved);
    setPlaces(next);
    setLegModes({});
    if (plan) setPending({ ...base(), order: originThenNames(next) });
  }
  // "Optimal order": re-sequence via the engine (drop the manual order).
  function optimalOrder() {
    setPending({ ...base(), order: undefined });
  }

  // Per-leg alternatives selector: override one leg's mode and re-price the same
  // order (no re-sequence).
  function changeLegMode(from: string, to: string, mode: string) {
    const next = { ...legModes, [`${from}|${to}`]: mode };
    setLegModes(next);
    if (plan) setPending({ ...base(plan.order), legModes: next });
  }

  function onPlan() {
    if (!enoughToPlan(places, origin, budget)) {
      setError(t("err.minCities"));
      return;
    }
    setPending(base());
  }

  // Toggles re-price the SAME order (no re-sequence) when a plan exists.
  const reprice = (patch: Partial<Pending>) => {
    if (!plan) return;
    setPending({ ...base(plan.order), ...patch });
  };
  function changeTier(v: Tier) {
    setTier(v);
    reprice({ tier: v });
  }
  function changeMode(v: ModePref) {
    setPref(v);
    reprice({ mode: v });
  }
  function changeCurrency(v: string) {
    setCurrency(v);
    reprice({ currency: v });
  }
  // Current nights for a city: user override, else the AI/date-allocated value
  // from the last plan, else a sensible default.
  const nightsFor = (name: string): number => {
    if (cityNights[name] != null) return cityNights[name];
    if (plan) {
      const idx = (plan.order ?? []).findIndex((c) => c.toLowerCase() === name.toLowerCase());
      if (idx >= 0 && (plan.nights_by_city?.[idx] ?? 0) > 0) return plan.nights_by_city![idx];
    }
    return 2;
  };
  // Per-city nights stepper (C8): set the override, re-price, and the roadmap's
  // POIs auto-adjust (more nights -> more day POIs).
  function changeCityNights(name: string, n: number) {
    const next = { ...cityNights, [name]: Math.max(1, n) };
    setCityNights(next);
    if (plan) setPending({ ...base(plan.order), cityNights: next });
  }
  // Per-city hotel star selector (D6): override one city's hotel tier and
  // re-price the same order live (total + per-person update).
  function changeCityStars(name: string, stars: number) {
    const next = { ...cityStars, [name]: stars };
    setCityStars(next);
    if (plan) setPending({ ...base(plan.order), cityStars: next });
  }
  // Passenger changes re-price the same order live (per-person + totals update).
  function changeAdults(v: number) {
    const n = Math.max(1, v);
    setAdults(n);
    reprice({ adults: n });
  }
  function changeChildren(v: number) {
    const n = Math.max(0, v);
    setChildren(n);
    reprice({ children: n });
  }
  function changeInfants(v: number) {
    const n = Math.max(0, v);
    setInfants(n);
    reprice({ infants: n });
  }

  // Apply an AI-parsed edit: match names to the currently selected places
  // (chat can remove/keep; adding a brand-new city is done via autocomplete).
  function applyRefine(s: EditState) {
    const want = (s.cities ?? []).map((n) => n.toLowerCase());
    const newPlaces = places.filter((p) => want.includes(p.name.toLowerCase()));
    const newTier = (["budget", "standard", "vip"].includes(s.tier) ? s.tier : tier) as Tier;
    const newPref = (["cheapest", "flight", "train", "bus"].includes(s.mode) ? s.mode : pref) as ModePref;
    const newNights = s.nights > 0 ? s.nights : nights;
    const newStart = newPlaces.some((p) => p.name === s.start) ? s.start : newPlaces[0]?.name ?? "";

    setPlaces(newPlaces);
    setStart(newStart);
    setTier(newTier);
    setPref(newPref);
    setNights(newNights);
    if (newPlaces.length >= 2) {
      const resequence = !sameNames(newPlaces, places);
      if (resequence) setLegModes({});
      setPending({
        places: newPlaces,
        origin,
        budget,
        start: newStart,
        mode: newPref,
        tier: newTier,
        nights: newNights,
        currency,
        order: resequence ? undefined : plan?.order,
        legModes: resequence ? {} : legModes,
        startDate,
        returnDate,
        adults,
        children,
        infants,
        cityNights,
        cityStars,
      });
    }
  }

  const fmt = (n: number) => n.toLocaleString(undefined, { maximumFractionDigits: 2 });

  // Activities (D1): when a roadmap exists, itemize its POIs (city · place ·
  // free/~price · maps) and derive the activities figure from those lines
  // instead of the flat per-night estimate — the sum comes from the rows.
  const priceNum = (price?: string): number => {
    if (!price) return 0;
    const m = price.replace(",", ".").match(/\d+(\.\d+)?/);
    return m ? parseFloat(m[0]) : 0;
  };
  const round2 = (n: number) => Math.round(n * 100) / 100;
  const travelers = plan?.travelers ?? 1;
  const activityItems = roadmap
    ? roadmap.days.flatMap((d) =>
        (d.pois ?? []).map((p) => ({ city: d.city, name: p.name, free: !!p.free, price: p.price }))
      )
    : [];
  const perPersonActivities = round2(activityItems.reduce((a, p) => a + priceNum(p.price), 0));
  // Food estimate (D4): per-city approximate restaurant spend from the roadmap.
  const foodItems = roadmap?.food ?? [];
  const perPersonFood = round2(foodItems.reduce((a, f) => a + (f.per_day || 0) * (f.nights || 0), 0));
  const groupActivities = round2(perPersonActivities * Math.max(1, travelers));
  const groupFood = round2(perPersonFood * Math.max(1, travelers));
  // Coherent figures: when the roadmap is present the live cost and total use the
  // itemized activities (+ food) sum; otherwise the backend per-night estimate.
  const activitiesVal = roadmap ? groupActivities : plan?.activities ?? 0;
  const foodVal = roadmap ? groupFood : 0;
  const totalVal = plan ? round2(plan.transport + plan.hotels + activitiesVal + foodVal) : 0;
  const perPersonVal = plan ? round2(totalVal / Math.max(1, travelers)) : 0;

  const variantTone = (v: string) => (v === "open_jaw" ? "accent" : "primary");
  const variantLabel = (v: string) => (v === "open_jaw" ? t("res.openjaw") : t("res.roundtrip"));

  const coordsByName = (name: string) => places.find((p) => p.name.toLowerCase() === name.toLowerCase());

  const placeByName = (name: string) =>
    places.find((p) => p.name.toLowerCase() === name.toLowerCase()) ??
    (origin && origin.name.toLowerCase() === name.toLowerCase() ? origin : undefined);

  // flightHref upgrades a flight leg's link to a prefilled Skyscanner search
  // (IATA codes) when the backend only has a generic fallback; otherwise keeps
  // the backend's real/source-specific deep-link (D14).
  const flightHref = (l: PlanQuote["legs"][number]): string | undefined => {
    const base = l.url;
    const generic = !base || /google\.com\/travel|rome2rio\.com/.test(base);
    if (l.mode === "flight" && generic) {
      const f = placeByName(l.from);
      const tp = placeByName(l.to);
      const sk = skyscannerFlightLink(f?.code, tp?.code, startDate, returnDate, adults, children, infants);
      if (sk) return sk;
    }
    return base;
  };

  const timelineLegs: TimelineLeg[] = plan
    ? (plan.legs ?? []).map((l) => ({
        mode: l.mode,
        label: t(`mode.${l.mode}`),
        price:
          l.price > 0
            ? `${l.estimated ? "~" : ""}${fmt(l.price)} ${plan.currency}`
            : t("res.transfer"),
        // Real, route-prefilled source deep-link (never a generic search). For
        // flight legs between airports we upgrade a generic fallback to a
        // prefilled Skyscanner search using the IATA codes (D14), but keep a
        // real fetched source page when the backend has one.
        href: flightHref(l),
        hrefLabel: t("res.searchInfo"),
        modeValue: legModes[`${l.from}|${l.to}`] ?? l.mode,
        onModeChange: (m: string) => changeLegMode(l.from, l.to, m),
        unavailableNote: l.mode_unavailable
          ? t("res.modeUnavailable", { m: t(`mode.${l.requested_mode}`) })
          : undefined,
        operator: l.operator,
        depart: l.depart,
        duration: l.duration,
        stops: l.stops,
        baggage: l.baggage,
        estimated: l.estimated,
        estimateNote: t("res.estimatedLine"),
        baggageLabel: t("res.baggage"),
      }))
    : [];

  const modeOptions = [
    { value: "cheapest", label: t("mode.cheapest") },
    { value: "flight", label: t("mode.flight") },
    { value: "train", label: t("mode.train") },
    { value: "bus", label: t("mode.bus") },
    { value: "ferry", label: t("mode.ferry") },
    { value: "car", label: t("mode.car") },
  ];

  const mapPoints: MapPoint[] = plan
    ? (plan.order ?? [])
        .map((name) => {
          const c = coordsByName(name);
          return c ? { name, lat: c.lat, lng: c.lng } : null;
        })
        .filter((p): p is MapPoint => !!p)
    : [];

  // Per-stop airport flag (B1): mark a stop whose selected place is an airport.
  const stopAirports = (plan?.order ?? []).map((name) => placeByName(name)?.kind === "airport");

  const aiBrief: ItineraryBrief | null = plan
    ? {
        cities: plan.order ?? [],
        places: (plan.order ?? [])
          .map((name) => {
            const c = placeByName(name);
            return c ? { name, lat: c.lat, lng: c.lng } : null;
          })
          .filter((p): p is { name: string; lat: number; lng: number } => !!p),
        legs: (plan.legs ?? []).map((l) => ({ from: l.from, to: l.to, mode: l.mode, price: l.price })),
        nights: plan.nights,
        nights_by_city: plan.nights_by_city,
        travelers: plan.travelers,
        currency: plan.currency,
        transport_total: plan.transport,
        hotels_total: plan.hotels,
        activities_total: plan.activities,
        total: plan.total,
      }
    : null;

  const refineState: EditState = {
    cities: places.map((p) => p.name),
    start,
    tier,
    mode: pref,
    nights,
  };

  const segments: CostSegment[] = plan
    ? [
        { key: "transport", label: t("res.transport"), value: plan.transport, colorVar: "--transport" },
        { key: "hotels", label: t("res.hotels"), value: plan.hotels, colorVar: "--hotels" },
        { key: "activities", label: t("res.activities"), value: activitiesVal, colorVar: "--activities" },
        ...(foodVal > 0
          ? [{ key: "food", label: t("res.food"), value: foodVal, colorVar: "--accent" } as CostSegment]
          : []),
      ]
    : [];

  return (
    <AppShell>
      <main className="mx-auto max-w-5xl px-4 pb-16">
        <section className="pt-10">
          <Link href="/" className="text-sm font-medium text-muted transition-colors hover:text-primary">
            {t("plan.back")}
          </Link>
          <h1 className="mt-3 font-display text-3xl font-extrabold tracking-tight">{t("plan.title")}</h1>
          <p className="mt-1.5 max-w-2xl text-sm text-muted">{t("plan.subtitle")}</p>
        </section>

        {/* Search wizard */}
        <Card className="mt-6 animate-fade-up print:hidden">
          <Field label={t("plan.origin")}>
            {origin ? (
              <div className="flex flex-wrap items-center gap-2">
                <span className="inline-flex max-w-full items-center gap-1.5 rounded-full border border-accent/30 bg-accent/10 px-3 py-1.5 text-sm font-medium text-accent">
                  <PinIcon width={13} height={13} className="shrink-0" />
                  <span className="min-w-0 truncate">{origin.name}</span>
                  <span className="shrink-0 text-accent/70">· {t("plan.originNote")}</span>
                  <button
                    type="button"
                    onClick={() => {
                      setOrigin(null);
                      setPlan(null);
                    }}
                    aria-label="remove origin"
                    className="ml-0.5 hover:text-accent"
                  >
                    ×
                  </button>
                </span>
              </div>
            ) : (
              <CityAutocomplete
                onAdd={(p) => {
                  setOrigin(p);
                  setPlan(null);
                }}
                placeholder={t("plan.origin")}
                isSelected={() => false}
              />
            )}
          </Field>

          <div className="mt-4">
            <Field label={t("plan.budget")}>
              <NumberInput
                type="number"
                min={0}
                value={budget || ""}
                onChange={(e) => setBudget(Math.max(0, Number(e.target.value)))}
              />
            </Field>
          </div>

          {/* D12: country mode — type a country, let the AI suggest cities. */}
          <CountryMode
            budget={budget}
            currency={currency}
            isSelected={(name) => places.some((p) => p.name.toLowerCase() === name.toLowerCase())}
            onAddCity={addPlace}
          />

          <div className="mt-4" />
          <Field label={t("plan.cities")}>
            <CityAutocomplete
              onAdd={addPlace}
              placeholder={t("plan.searchCity")}
              isSelected={(name) => places.some((p) => p.name.toLowerCase() === name.toLowerCase())}
            />
          </Field>

          {places.length === 0 ? (
            <p className="mt-3 text-sm text-muted">{t("plan.pickCities")}</p>
          ) : (
            <div className="mt-3 flex flex-wrap gap-2">
              {places.map((p, i) => (
                <span
                  key={p.name}
                  draggable
                  onDragStart={() => setDragIdx(i)}
                  onDragOver={(e) => e.preventDefault()}
                  onDrop={() => {
                    if (dragIdx !== null) reorderDest(dragIdx, i);
                    setDragIdx(null);
                  }}
                  className="inline-flex max-w-full cursor-grab items-center gap-1.5 rounded-full border border-primary/30 bg-primary/10 px-3 py-1.5 text-sm font-medium text-primary active:cursor-grabbing"
                  title={t("plan.dragHint")}
                >
                  <span className="shrink-0 text-primary/50">⠿</span>
                  <PinIcon width={13} height={13} className="shrink-0" />
                  <span className="min-w-0 truncate">{p.name}</span>
                  <button
                    type="button"
                    onClick={() => removePlace(p.name)}
                    aria-label={`remove ${p.name}`}
                    className="ml-0.5 shrink-0 text-primary/70 transition-colors hover:text-primary"
                  >
                    ×
                  </button>
                </span>
              ))}
            </div>
          )}

          {/* D7: a clearly visible, editable per-city nights list — each row has
              a drag-handle to reorder, the city, and a −N+ stepper + typed
              number. Changing re-prices the plan + roadmap + POIs live. When a
              city has no explicit nights, the AI/date-allocated value is shown. */}
          {places.length > 0 && (
            <div className="mt-4 rounded-xl border border-border bg-surface-2/50 p-3">
              <div className="mb-2 flex flex-wrap items-baseline justify-between gap-x-2 gap-y-1">
                <span className="text-xs font-semibold uppercase tracking-wide text-muted">
                  {t("plan.nightsTitle")}
                </span>
                <span className="min-w-0 text-[11px] text-muted">{t("plan.nightsHint")}</span>
              </div>
              <ul className="space-y-1.5">
                {places.map((p, i) => (
                  <li
                    key={p.name}
                    draggable
                    onDragStart={() => setDragIdx(i)}
                    onDragOver={(e) => e.preventDefault()}
                    onDrop={() => {
                      if (dragIdx !== null) reorderDest(dragIdx, i);
                      setDragIdx(null);
                    }}
                    className="flex cursor-grab items-center gap-2 rounded-lg bg-surface px-2.5 py-1.5 active:cursor-grabbing"
                    title={t("plan.dragHint")}
                  >
                    <span className="text-muted/50">⠿</span>
                    <span className="min-w-0 flex-1 truncate text-sm font-medium">{p.name}</span>
                    {cityNights[p.name] == null && (
                      <span className="shrink-0 rounded-full bg-accent/10 px-1.5 py-0.5 text-[10px] font-medium text-accent">
                        {t("plan.aiNights")}
                      </span>
                    )}
                    <div className="flex shrink-0 items-center gap-1.5">
                      <button
                        type="button"
                        onClick={() => changeCityNights(p.name, nightsFor(p.name) - 1)}
                        aria-label={`fewer nights in ${p.name}`}
                        className="grid h-6 w-6 place-items-center rounded-lg border border-border text-muted transition-colors hover:border-primary/40 hover:text-text"
                      >
                        −
                      </button>
                      <input
                        type="number"
                        min={1}
                        value={nightsFor(p.name)}
                        onChange={(e) => changeCityNights(p.name, Math.max(1, Number(e.target.value) || 1))}
                        aria-label={`nights in ${p.name}`}
                        className="w-12 rounded-lg border border-border bg-surface-2 px-1 py-0.5 text-center text-sm tabular-nums outline-none focus:border-primary"
                      />
                      <button
                        type="button"
                        onClick={() => changeCityNights(p.name, nightsFor(p.name) + 1)}
                        aria-label={`more nights in ${p.name}`}
                        className="grid h-6 w-6 place-items-center rounded-lg border border-border text-muted transition-colors hover:border-primary/40 hover:text-text"
                      >
                        +
                      </button>
                      <span className="w-10 text-right text-[11px] text-muted">{t("res.nights", { n: nightsFor(p.name) })}</span>
                    </div>
                  </li>
                ))}
              </ul>
            </div>
          )}

          <div className="mt-6 grid grid-cols-1 gap-4 sm:grid-cols-3">
            <Field label={t("plan.depart")}>
              <NumberInput type="date" min={todayStr} value={startDate} onChange={(e) => setStartDate(e.target.value)} />
            </Field>
            <Field label={t("plan.return")}>
              <NumberInput type="date" min={startDate || todayStr} value={returnDate} onChange={(e) => setReturnDate(e.target.value)} />
            </Field>
            <Field label={t("plan.start")}>
              <Select value={start} onChange={(e) => setStart(e.target.value)}>
                {places.map((p) => (
                  <option key={p.name} value={p.name}>{p.name}</option>
                ))}
              </Select>
            </Field>
          </div>
          <p className="mt-1.5 text-xs text-muted">{t("plan.startHint")}</p>

          <div className="mt-4 grid grid-cols-1 gap-4 sm:grid-cols-3">
            <Field label={t("plan.comfort")}>
              <Select value={tier} onChange={(e) => changeTier(e.target.value as Tier)}>
                <option value="budget">{t("tier.budget")}</option>
                <option value="standard">{t("tier.standard")}</option>
                <option value="vip">{t("tier.vip")}</option>
              </Select>
            </Field>
            <Field label={t("plan.transport")}>
              <Select value={pref} onChange={(e) => changeMode(e.target.value as ModePref)}>
                <option value="cheapest">{t("mode.cheapest")}</option>
                <option value="flight">{t("mode.flight")}</option>
                <option value="train">{t("mode.train")}</option>
                <option value="bus">{t("mode.bus")}</option>
                <option value="ferry">{t("mode.ferry")}</option>
                <option value="car">{t("mode.car")}</option>
              </Select>
            </Field>
            <Field label={t("plan.currency")}>
              <Select value={currency} onChange={(e) => changeCurrency(e.target.value)}>
                {CURRENCIES.map((c) => (
                  <option key={c} value={c}>{c}</option>
                ))}
              </Select>
            </Field>
          </div>

          <div className="mt-4 rounded-xl border border-border bg-surface-2 p-3">
            <div className="mb-2 text-xs font-semibold uppercase tracking-wide text-muted">
              {t("plan.passengers")}
            </div>
            <div className="grid grid-cols-1 gap-2 sm:grid-cols-3">
              <Stepper label={t("plan.adults")} value={adults} min={1} onChange={changeAdults} />
              <Stepper label={t("plan.children")} value={children} min={0} onChange={changeChildren} />
              <Stepper label={t("plan.infants")} value={infants} min={0} onChange={changeInfants} />
            </div>
          </div>

          <div className="mt-6">
            <Button onClick={onPlan} disabled={loading || !enoughToPlan(places, origin, budget)}>
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
          </div>
        </Card>

        {error && (
          <div className="mt-4 flex flex-wrap items-center justify-between gap-3 rounded-xl border border-danger/30 bg-danger/10 px-4 py-3 text-sm font-medium text-danger">
            <span>{error}</span>
            <Button variant="ghost" onClick={retryPlan} disabled={loading}>
              {t("plan.retry")}
            </Button>
          </div>
        )}

        {/* Empty state */}
        {!plan && !error && (
          <Card className="mt-6 flex flex-col items-center py-14 text-center">
            <span className="grid h-14 w-14 place-items-center rounded-2xl bg-primary/10 text-primary">
              <RouteIcon width={26} height={26} />
            </span>
            <h2 className="mt-4 font-display text-lg font-bold">{t("res.emptyTitle")}</h2>
            <p className="mt-1.5 max-w-sm text-sm text-muted">{t("res.emptyDesc")}</p>
          </Card>
        )}

        {/* Results */}
        {plan && (
          <section className="mt-6 space-y-4">
            {/* Trip summary header */}
            <Card className="animate-fade-up flex flex-wrap items-center justify-between gap-3">
              <div className="flex flex-wrap items-center gap-x-4 gap-y-1 text-sm">
                <span className="font-semibold">{t("res.nights", { n: plan.nights })}</span>
                <span className="text-muted">·</span>
                <span>{(plan.hotel_lines ?? []).length} {t("plan.cities")}</span>
                <span className="text-muted">·</span>
                <span className="font-display text-lg font-bold">
                  {fmt(totalVal)} {plan.currency}
                </span>
                {(plan.travelers ?? 1) > 1 && (
                  <span className="text-muted">
                    ({fmt(perPersonVal)} {plan.currency} / {t("res.perPerson")})
                  </span>
                )}
              </div>
              <div className="flex items-center gap-2">
                {loading && (
                  <span className="inline-flex items-center gap-1.5 rounded-full bg-surface-2 px-2.5 py-1 text-xs font-medium text-muted">
                    <span className="h-3 w-3 animate-spin rounded-full border-2 border-muted/40 border-t-muted" />
                    {t("res.updating")}
                  </span>
                )}
                <Badge tone={variantTone(plan.variant)}>{variantLabel(plan.variant)}</Badge>
              </div>
            </Card>

            <div className="flex flex-wrap justify-end gap-2 print:hidden">
              <Button variant="ghost" onClick={optimalOrder}>
                {t("res.optimalOrder")}
              </Button>
              <Button variant="ghost" onClick={share}>
                {copied ? t("res.copied") : t("res.share")}
              </Button>
              <Button variant="ghost" onClick={printPlan}>
                {t("res.print")}
              </Button>
            </div>
            <div className="grid gap-4 lg:grid-cols-5">
              <Card className="animate-fade-up lg:col-span-3">
                <div className="flex flex-wrap items-center justify-between gap-2">
                  <h2 className="min-w-0 break-words font-display text-lg font-bold">{t("res.planTitle")}</h2>
                  <Badge tone={variantTone(plan.variant)}>{variantLabel(plan.variant)}</Badge>
                </div>
                <div className="mt-4">
                  <RouteTimeline
                    // Round-trip: close the loop by repeating the origin as a
                    // final stop, so the return leg (last -> origin) reads clearly (E6).
                    stops={
                      plan.variant === "round_trip" && (plan.order?.length ?? 0) > 0
                        ? [...(plan.order ?? []), plan.order![0]]
                        : plan.order ?? []
                    }
                    legs={timelineLegs}
                    nights={
                      plan.variant === "round_trip"
                        ? [...(plan.nights_by_city ?? []), 0]
                        : plan.nights_by_city ?? []
                    }
                    airports={
                      plan.variant === "round_trip"
                        ? [...stopAirports, stopAirports[0] ?? false]
                        : stopAirports
                    }
                    airportLabel={t("res.airport")}
                    nightLabel={(n) => t("res.nights", { n })}
                    modeOptions={modeOptions}
                  />
                </div>
                {plan.alt_variant && (
                  <div className="mt-4 border-t border-border pt-3">
                    <div className="flex flex-wrap items-center justify-between gap-2">
                      <span className="text-xs font-semibold text-muted">
                        {t("res.alt")} · {variantLabel(plan.alt_variant)}
                      </span>
                      <span className="text-sm font-semibold tabular-nums">
                        ~{fmt(plan.alt_cost ?? 0)} {plan.currency}
                      </span>
                    </div>
                    {plan.alt_order && plan.alt_order.length > 0 && (
                      <div className="mt-2 flex flex-wrap items-center gap-x-1 gap-y-1 text-xs text-muted">
                        {plan.alt_order.map((city, i) => (
                          <span key={i} className="inline-flex items-center gap-1">
                            <span className="rounded-full bg-surface-2 px-2 py-0.5 font-medium text-text">{city}</span>
                            {(plan.alt_nights_by_city?.[i] ?? 0) > 0 && (
                              <span className="text-[11px]">{t("res.nights", { n: plan.alt_nights_by_city![i] })}</span>
                            )}
                            {i < plan.alt_order!.length - 1 && <span className="text-border">→</span>}
                          </span>
                        ))}
                        {plan.alt_variant === "round_trip" && plan.alt_order[0] && (
                          <span className="inline-flex items-center gap-1">
                            <span className="text-border">→</span>
                            <span className="rounded-full bg-surface-2 px-2 py-0.5 font-medium text-text">{plan.alt_order[0]}</span>
                          </span>
                        )}
                      </div>
                    )}
                  </div>
                )}
              </Card>

              <Card className="animate-fade-up lg:col-span-2">
                <div className="flex items-center gap-2">
                  <WalletIcon width={18} height={18} className="text-activities" />
                  <h2 className="font-display text-lg font-bold">{t("res.liveCost")}</h2>
                </div>
                <div className="mt-5">
                  <CostDonut segments={segments} total={totalVal} currency={plan.currency} fmt={fmt} />
                </div>
                <CostBreakdown
                  legs={plan.legs ?? []}
                  hotels={plan.hotel_lines ?? []}
                  activities={activitiesVal}
                  activityItems={activityItems}
                  food={foodItems}
                  foodTotal={foodVal}
                  hotelSites={(c: string) => hotelSites(c, startDate || undefined, returnDate || undefined, adults)}
                  onHotelStars={changeCityStars}
                  city={(name: string) => name}
                  currency={plan.currency}
                  fmt={fmt}
                  labels={{ source: t("res.source"), asOf: t("res.asOf"), est: t("res.estimatedLine"), chosen: t("res.chosen"), baggage: t("res.baggage"), free: t("res.free"), approx: t("res.approx"), maps: t("res.maps"), hotel: t("res.hotelWord") }}
                  catLabels={{ transport: t("res.transport"), hotels: t("res.hotels"), activities: t("res.activities"), food: t("res.food") }}
                />
                {(plan.travelers ?? 1) > 1 && (
                  <div className="mt-3 flex items-center justify-between border-t border-border pt-3 text-sm">
                    <span className="text-muted">{t("res.perPerson")}</span>
                    <span className="font-semibold tabular-nums">
                      {fmt(perPersonVal)} {plan.currency}
                    </span>
                  </div>
                )}
                <p className="mt-4 text-xs leading-relaxed text-muted">{t("res.disclaimer")}</p>
              </Card>
            </div>

            <Card className="animate-fade-up">
              <div className="mb-3 flex items-center gap-2">
                <PinIcon width={18} height={18} className="text-primary" />
                <h2 className="font-display text-lg font-bold">{t("res.map")}</h2>
              </div>
              <RouteMap points={mapPoints} closed={plan.variant === "round_trip"} />
            </Card>

            <div className="print:hidden">
              <AIPanel
                brief={aiBrief}
                refineState={refineState}
                onRefine={(s) => applyRefine(s)}
                roadmap={roadmap}
                onRoadmap={setRoadmap}
                hotels={plan.hotel_lines ?? []}
                onCityNights={changeCityNights}
                nightsFor={nightsFor}
              />
            </div>

            {/* Print/PDF only: the AI day-by-day roadmap in a structured layout (D15). */}
            {roadmap && roadmap.days.length > 0 && (
              <RoadmapPrint
                roadmap={roadmap}
                legs={plan.legs ?? []}
                hotels={plan.hotel_lines ?? []}
                activities={activitiesVal}
                food={foodItems}
                foodTotal={foodVal}
                transport={plan.transport}
                hotelsTotal={plan.hotels}
                total={totalVal}
                perPerson={perPersonVal}
                travelers={travelers}
                currency={plan.currency}
                fmt={fmt}
              />
            )}
          </section>
        )}
      </main>
    </AppShell>
  );
}
