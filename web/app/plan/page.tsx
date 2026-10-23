"use client";

import { useEffect, useState } from "react";
import { CITIES, CURRENCIES, City, ModePref, Tier } from "../../lib/sample";
import {
  destinations,
  EditState,
  ItineraryBrief,
  planQuote,
  PlanQuote,
} from "../../lib/api";
import { LOCALE_CURRENCY, useI18n } from "../../lib/i18n";
import { AppShell } from "../../components/AppShell";
import { Badge, Button, Card, Chip, Field, NumberInput, Select } from "../../components/ui";
import { RouteTimeline, TimelineLeg } from "../../components/RouteTimeline";
import { CostDonut, CostSegment } from "../../components/CostDonut";
import { RouteMap, MapPoint } from "../../components/RouteMap";
import { AIPanel } from "../../components/AIPanel";
import { PinIcon, RouteIcon, WalletIcon } from "../../components/icons";
import Link from "next/link";

// Pending describes the next priced-plan fetch. order set => re-price that
// order without re-sequencing; order undefined => let the engine sequence.
interface Pending {
  sel: string[];
  start: string;
  mode: ModePref;
  tier: Tier;
  nights: number;
  currency: string;
  order?: string[];
}

function sameSet(a: string[], b: string[]): boolean {
  if (a.length !== b.length) return false;
  const s = new Set(a.map((x) => x.toLowerCase()));
  return b.every((x) => s.has(x.toLowerCase()));
}

export default function PlanPage() {
  const { t, locale } = useI18n();

  // City list is dynamic (GET /data/destinations) with the demo set as fallback.
  const [cityList, setCityList] = useState<City[]>(CITIES);
  const [selected, setSelected] = useState<string[]>(["BAK", "TBS", "IST"]);
  const [start, setStart] = useState("BAK");
  const [tier, setTier] = useState<Tier>("standard");
  const [pref, setPref] = useState<ModePref>("cheapest");
  const [nights, setNights] = useState(2);
  const [currency, setCurrency] = useState<string>(LOCALE_CURRENCY[locale]);

  // Everything priced server-side (flywheel + estimator + engine): one source.
  const [plan, setPlan] = useState<PlanQuote | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [pending, setPending] = useState<Pending | null>(null);

  // Load the dynamic city list once (offline fallback keeps the page working).
  useEffect(() => {
    let cancelled = false;
    destinations()
      .then((ds) => {
        if (cancelled || !ds || ds.length === 0) return;
        const mapped: City[] = ds.map((d) => ({ code: d.city, name: d.city, lat: d.lat, lng: d.lng }));
        setCityList(mapped);
        const codes = mapped.map((c) => c.code);
        setSelected(codes.slice(0, Math.min(3, codes.length)));
        setStart(codes[0]);
        setPlan(null);
      })
      .catch(() => {
        /* API offline — keep the demo list */
      });
    return () => {
      cancelled = true;
    };
  }, []);

  // Single place that talks to the backend pricer. Params are explicit so it is
  // never stale against React state.
  async function fetchPlan(p: Pending) {
    const names = cityList.filter((c) => p.sel.includes(c.code)).map((c) => c.name);
    if (names.length < 2) {
      setError(t("err.minCities"));
      return;
    }
    const startName = cityList.find((c) => c.code === p.start)?.name ?? names[0];
    setLoading(true);
    setError(null);
    try {
      const q = await planQuote({
        cities: names,
        start: startName,
        mode: p.mode,
        tier: p.tier,
        nights: p.nights,
        currency: p.currency,
        order: p.order,
        variant: p.order ? plan?.variant : undefined,
      });
      setPlan(q);
    } catch {
      setError(t("err.sequence"));
    } finally {
      setLoading(false);
    }
  }

  // Run queued fetches one at a time.
  useEffect(() => {
    if (!pending) return;
    const p = pending;
    setPending(null);
    void fetchPlan(p);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [pending]);

  function toggleCity(code: string) {
    setPlan(null); // city-set change invalidates the sequence; recompute manually
    setSelected((cur) => (cur.includes(code) ? cur.filter((c) => c !== code) : [...cur, code]));
  }

  function onPlan() {
    if (selected.length < 2) {
      setError(t("err.minCities"));
      return;
    }
    if (!selected.includes(start)) {
      setError(t("err.startInvalid"));
      return;
    }
    setPending({ sel: selected, start, mode: pref, tier, nights, currency });
  }

  // Toggles re-price the SAME order (no re-sequence) when a plan exists.
  const reprice = (patch: Partial<Pending>) => {
    if (!plan) return;
    setPending({ sel: selected, start, mode: pref, tier, nights, currency, order: plan.order, ...patch });
  };
  function changeTier(v: Tier) {
    setTier(v);
    reprice({ tier: v });
  }
  function changeMode(v: ModePref) {
    setPref(v);
    reprice({ mode: v });
  }
  function changeNights(v: number) {
    const n = Math.max(1, v);
    setNights(n);
    reprice({ nights: n });
  }
  function changeCurrency(v: string) {
    setCurrency(v);
    reprice({ currency: v });
  }

  // Apply an AI-parsed edit: map names->codes, update state, and queue a fetch
  // (re-sequence if the city set changed, else re-price the current order).
  function applyRefine(s: EditState) {
    const byName = (n: string) => cityList.find((c) => c.name.toLowerCase() === n.toLowerCase());
    const newSelected = (s.cities ?? []).map(byName).filter((c): c is City => !!c).map((c) => c.code);
    const newTier = (["budget", "standard", "vip"].includes(s.tier) ? s.tier : tier) as Tier;
    const newPref = (["cheapest", "flight", "train", "bus"].includes(s.mode) ? s.mode : pref) as ModePref;
    const newNights = s.nights > 0 ? s.nights : nights;
    const startCity = byName(s.start);
    const newStart = startCity ? startCity.code : newSelected[0] ?? start;

    setSelected(newSelected);
    setStart(newStart);
    setTier(newTier);
    setPref(newPref);
    setNights(newNights);
    if (newSelected.length >= 2) {
      const resequence = !sameSet(newSelected, selected);
      setPending({
        sel: newSelected,
        start: newStart,
        mode: newPref,
        tier: newTier,
        nights: newNights,
        currency,
        order: resequence ? undefined : plan?.order,
      });
    }
  }

  const fmt = (n: number) => n.toLocaleString(undefined, { maximumFractionDigits: 2 });
  const variantTone = (v: string) => (v === "open_jaw" ? "accent" : "primary");
  const variantLabel = (v: string) => (v === "open_jaw" ? t("res.openjaw") : t("res.roundtrip"));

  const coordsByName = (name: string) => cityList.find((c) => c.name.toLowerCase() === name.toLowerCase());

  const timelineLegs: TimelineLeg[] = plan
    ? (plan.legs ?? []).map((l) => ({
        mode: l.mode,
        label: t(`mode.${l.mode}`),
        price: `${l.estimated ? "~" : ""}${fmt(l.price)} ${plan.currency}`,
      }))
    : [];

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
        currency: plan.currency,
        transport_total: plan.transport,
        hotels_total: plan.hotels,
        activities_total: plan.activities,
        total: plan.total,
      }
    : null;

  const refineState: EditState = {
    cities: cityList.filter((c) => selected.includes(c.code)).map((c) => c.name),
    start: cityList.find((c) => c.code === start)?.name ?? "",
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
            <Field label={t("plan.nights")}>
              <NumberInput
                type="number"
                min={1}
                max={10}
                value={nights}
                onChange={(e) => changeNights(Number(e.target.value))}
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
              <Select value={currency} onChange={(e) => changeCurrency(e.target.value)} className="w-auto py-1.5">
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
            <div className="grid gap-4 lg:grid-cols-5">
              {/* Roadmap + timeline */}
              <Card className="animate-fade-up lg:col-span-3">
                <div className="flex items-center justify-between">
                  <h2 className="font-display text-lg font-bold">{t("res.planTitle")}</h2>
                  <Badge tone={variantTone(plan.variant)}>{variantLabel(plan.variant)}</Badge>
                </div>
                <div className="mt-4">
                  <RouteTimeline stops={plan.order ?? []} legs={timelineLegs} />
                </div>
                {plan.alt_variant && (
                  <p className="mt-4 border-t border-border pt-3 text-xs text-muted">
                    {t("res.alt")} ({variantLabel(plan.alt_variant)}): ~{fmt(plan.alt_cost ?? 0)} {plan.currency}
                  </p>
                )}
              </Card>

              {/* Live cost */}
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

            {/* Map */}
            <Card className="animate-fade-up">
              <div className="mb-3 flex items-center gap-2">
                <PinIcon width={18} height={18} className="text-primary" />
                <h2 className="font-display text-lg font-bold">{t("res.map")}</h2>
              </div>
              <RouteMap points={mapPoints} closed={plan.variant === "round_trip"} />
            </Card>

            {/* AI: itinerary writer + conversational refine */}
            <AIPanel brief={aiBrief} refineState={refineState} onRefine={(s) => applyRefine(s)} />
          </section>
        )}
      </main>
    </AppShell>
  );
}
