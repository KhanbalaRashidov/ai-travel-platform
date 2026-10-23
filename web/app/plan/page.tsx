"use client";

import { useEffect, useState } from "react";
import { CURRENCIES, ModePref, Tier } from "../../lib/sample";
import {
  EditState,
  ItineraryBrief,
  Place,
  planQuote,
  PlanQuote,
  track,
} from "../../lib/api";
import { LOCALE_CURRENCY, useI18n } from "../../lib/i18n";
import { AppShell } from "../../components/AppShell";
import { Badge, Button, Card, Field, NumberInput, Select } from "../../components/ui";
import { CityAutocomplete } from "../../components/CityAutocomplete";
import { RouteTimeline, TimelineLeg } from "../../components/RouteTimeline";
import { CostDonut, CostSegment } from "../../components/CostDonut";
import { RouteMap, MapPoint } from "../../components/RouteMap";
import { AIPanel } from "../../components/AIPanel";
import { useToast } from "../../components/toast";
import { PinIcon, RouteIcon, WalletIcon } from "../../components/icons";
import Link from "next/link";

interface Pending {
  places: Place[];
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
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [pending, setPending] = useState<Pending | null>(null);
  const [copied, setCopied] = useState(false);
  const [legModes, setLegModes] = useState<Record<string, string>>({});

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
          places: pl, start: st, mode: d.m ?? "cheapest", tier: d.t ?? "standard",
          nights: 2, currency: d.cu ?? "EUR", order: undefined, legModes: {},
          startDate: d.d ?? "", returnDate: d.r ?? "",
          adults: d.ad ?? 1, children: d.ch ?? 0, infants: d.inf ?? 0,
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

  async function fetchPlan(p: Pending) {
    if (p.places.length < 2) {
      setError(t("err.minCities"));
      return;
    }
    const names = p.places.map((x) => x.name);
    const startName = names.includes(p.start) ? p.start : names[0];
    setLoading(true);
    setError(null);
    try {
      const q = await planQuote({
        cities: names,
        places: p.places.map((x) => ({ name: x.name, lat: x.lat, lng: x.lng })),
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
      });
      setPlan(q);
      track("plan_computed");
      toast(p.order ? t("toast.updated") : t("toast.ready"), "success");
    } catch {
      setError(t("err.sequence"));
      toast(t("err.sequence"), "error");
    } finally {
      setLoading(false);
    }
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
  }
  function removePlace(name: string) {
    const next = places.filter((x) => x.name !== name);
    setPlaces(next);
    if (start === name) setStart(next[0]?.name ?? "");
    setLegModes({});
    setPlan(null);
  }

  function base(order?: string[]): Pending {
    return {
      places,
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
    };
  }

  // Per-leg alternatives selector: override one leg's mode and re-price the same
  // order (no re-sequence).
  function changeLegMode(from: string, to: string, mode: string) {
    const next = { ...legModes, [`${from}|${to}`]: mode };
    setLegModes(next);
    if (plan) setPending({ ...base(plan.order), legModes: next });
  }

  function onPlan() {
    if (places.length < 2) {
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
      });
    }
  }

  const fmt = (n: number) => n.toLocaleString(undefined, { maximumFractionDigits: 2 });
  const variantTone = (v: string) => (v === "open_jaw" ? "accent" : "primary");
  const variantLabel = (v: string) => (v === "open_jaw" ? t("res.openjaw") : t("res.roundtrip"));

  const coordsByName = (name: string) => places.find((p) => p.name.toLowerCase() === name.toLowerCase());

  const timelineLegs: TimelineLeg[] = plan
    ? (plan.legs ?? []).map((l) => ({
        mode: l.mode,
        label: t(`mode.${l.mode}`),
        price:
          l.price > 0
            ? `${l.estimated ? "~" : ""}${fmt(l.price)} ${plan.currency}`
            : t("res.transfer"),
        // Plain informational search link (no booking, no affiliate).
        href: `https://www.google.com/search?q=${encodeURIComponent(`${l.from} to ${l.to} ${l.mode}`)}`,
        hrefLabel: t("res.searchInfo"),
        modeValue: legModes[`${l.from}|${l.to}`] ?? l.mode,
        onModeChange: (m: string) => changeLegMode(l.from, l.to, m),
      }))
    : [];

  const modeOptions = [
    { value: "cheapest", label: t("mode.cheapest") },
    { value: "flight", label: t("mode.flight") },
    { value: "train", label: t("mode.train") },
    { value: "bus", label: t("mode.bus") },
  ];

  const mapPoints: MapPoint[] = plan
    ? (plan.order ?? [])
        .map((name) => {
          const c = coordsByName(name);
          return c ? { name, lat: c.lat, lng: c.lng } : null;
        })
        .filter((p): p is MapPoint => !!p)
    : [];

  const aiBrief: ItineraryBrief | null = plan
    ? {
        cities: plan.order ?? [],
        legs: (plan.legs ?? []).map((l) => ({ from: l.from, to: l.to, mode: l.mode, price: l.price })),
        nights: plan.nights,
        nights_by_city: plan.nights_by_city,
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
        { key: "activities", label: t("res.activities"), value: plan.activities, colorVar: "--activities" },
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
              {places.map((p) => (
                <span
                  key={p.name}
                  className="inline-flex items-center gap-1.5 rounded-full border border-primary/30 bg-primary/10 px-3 py-1.5 text-sm font-medium text-primary"
                >
                  <PinIcon width={13} height={13} />
                  {p.name}
                  <button
                    type="button"
                    onClick={() => removePlace(p.name)}
                    aria-label={`remove ${p.name}`}
                    className="ml-0.5 text-primary/70 transition-colors hover:text-primary"
                  >
                    ×
                  </button>
                </span>
              ))}
            </div>
          )}

          <div className="mt-6 grid grid-cols-2 gap-4 sm:grid-cols-3">
            <Field label={t("plan.depart")}>
              <NumberInput type="date" value={startDate} onChange={(e) => setStartDate(e.target.value)} />
            </Field>
            <Field label={t("plan.return")}>
              <NumberInput type="date" value={returnDate} onChange={(e) => setReturnDate(e.target.value)} />
            </Field>
            <Field label={t("plan.start")}>
              <Select value={start} onChange={(e) => setStart(e.target.value)}>
                {places.map((p) => (
                  <option key={p.name} value={p.name}>{p.name}</option>
                ))}
              </Select>
            </Field>
          </div>

          <div className="mt-4 grid grid-cols-2 gap-4 sm:grid-cols-4">
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
              <Stepper label={t("plan.adults")} value={adults} min={1} onChange={setAdults} />
              <Stepper label={t("plan.children")} value={children} min={0} onChange={setChildren} />
              <Stepper label={t("plan.infants")} value={infants} min={0} onChange={setInfants} />
            </div>
          </div>

          <div className="mt-6">
            <Button onClick={onPlan} disabled={loading || places.length < 2}>
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
          <p className="mt-4 animate-fade-in rounded-xl border border-danger/30 bg-danger/10 px-4 py-3 text-sm font-medium text-danger">
            {error}
          </p>
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
            <div className="flex flex-wrap justify-end gap-2 print:hidden">
              <Button variant="ghost" onClick={share}>
                {copied ? t("res.copied") : t("res.share")}
              </Button>
              <Button variant="ghost" onClick={() => window.print()}>
                {t("res.print")}
              </Button>
            </div>
            <div className="grid gap-4 lg:grid-cols-5">
              <Card className="animate-fade-up lg:col-span-3">
                <div className="flex items-center justify-between">
                  <h2 className="font-display text-lg font-bold">{t("res.planTitle")}</h2>
                  <Badge tone={variantTone(plan.variant)}>{variantLabel(plan.variant)}</Badge>
                </div>
                <div className="mt-4">
                  <RouteTimeline
                    stops={plan.order ?? []}
                    legs={timelineLegs}
                    nights={plan.nights_by_city ?? []}
                    nightLabel={(n) => t("res.nights", { n })}
                    modeOptions={modeOptions}
                  />
                </div>
                {plan.alt_variant && (
                  <p className="mt-4 border-t border-border pt-3 text-xs text-muted">
                    {t("res.alt")} ({variantLabel(plan.alt_variant)}): ~{fmt(plan.alt_cost ?? 0)} {plan.currency}
                  </p>
                )}
              </Card>

              <Card className="animate-fade-up lg:col-span-2">
                <div className="flex items-center justify-between">
                  <div className="flex items-center gap-2">
                    <WalletIcon width={18} height={18} className="text-activities" />
                    <h2 className="font-display text-lg font-bold">{t("res.liveCost")}</h2>
                  </div>
                  {plan.estimated && <Badge tone="accent">{t("res.estimated")}</Badge>}
                </div>
                <div className="mt-5">
                  <CostDonut segments={segments} total={plan.total} currency={plan.currency} fmt={fmt} />
                </div>
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
              <AIPanel brief={aiBrief} refineState={refineState} onRefine={(s) => applyRefine(s)} />
            </div>
          </section>
        )}
      </main>
    </AppShell>
  );
}
