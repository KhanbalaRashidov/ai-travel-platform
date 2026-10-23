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
  PricedHotel,
  PricedLeg,
  price,
  RouteDTO,
  sequence,
} from "../../lib/api";

interface Planned {
  cities: City[];
  route: RouteDTO;
  alt: RouteDTO;
}

export default function PlanPage() {
  const [selected, setSelected] = useState<string[]>(["BAK", "TBS", "IST"]);
  const [start, setStart] = useState("BAK");
  const [tier, setTier] = useState<Tier>("standard");
  const [pref, setPref] = useState<ModePref>("cheapest");
  const [nights, setNights] = useState(2);
  const [currency, setCurrency] = useState("EUR");

  const [planned, setPlanned] = useState<Planned | null>(null);
  const [legs, setLegs] = useState<Array<{ from: string; to: string; mode: string; price: number }>>([]);
  const [breakdown, setBreakdown] = useState<Breakdown | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  function toggleCity(code: string) {
    setPlanned(null); // city-set change invalidates the sequence
    setBreakdown(null);
    setSelected((cur) =>
      cur.includes(code) ? cur.filter((c) => c !== code) : [...cur, code]
    );
  }

  // Re-price a fixed order. Does NOT re-sequence (refinement #2): called on
  // tier/mode/nights/currency changes.
  const reprice = useCallback(
    async (p: Planned) => {
      const order = p.route.order;
      const stars = TIER_STARS[tier];
      const nightly = NIGHTLY_BY_STAR[stars];

      const uiLegs: Array<{ from: string; to: string; mode: string; price: number }> = [];
      const apiLegs: PricedLeg[] = [];
      for (let i = 0; i + 1 < order.length; i++) {
        const a = p.cities[order[i]];
        const b = p.cities[order[i + 1]];
        const leg = legFor(a, b, pref);
        if (leg.price >= INF) {
          setError(`"${pref}" bu marşrut üçün mümkün deyil (${a.name} → ${b.name}). Başqa nəqliyyat seçin.`);
          return;
        }
        uiLegs.push({ from: a.name, to: b.name, mode: leg.mode, price: leg.price });
        apiLegs.push({ from: order[i], to: order[i + 1], mode: leg.mode, price: { amount: leg.price, currency: "EUR" } });
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
      } catch (e) {
        setError(e instanceof Error ? e.message : "qiymətləndirmə alınmadı");
      }
    },
    [tier, pref, nights, currency]
  );

  async function onPlan() {
    if (selected.length < 2) {
      setError("Ən azı 2 şəhər seçin.");
      return;
    }
    if (!selected.includes(start)) {
      setError("Başlanğıc şəhər seçilmiş şəhərlər arasında olmalıdır.");
      return;
    }
    const cities = CITIES.filter((c) => selected.includes(c.code));
    const startIdx = cities.findIndex((c) => c.code === start);
    const matrix = buildMatrix(cities, pref);
    setLoading(true);
    setError(null);
    try {
      const res = await sequence(matrix, startIdx, -1);
      const p: Planned = { cities, route: res.best, alt: res.alt };
      setPlanned(p);
      await reprice(p);
    } catch (e) {
      setError(e instanceof Error ? e.message : "marşrut hesablanmadı");
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

  const fmt = (n: number) => n.toLocaleString(undefined, { maximumFractionDigits: 2 });

  return (
    <main className="mx-auto max-w-3xl px-4 py-10">
      <h1 className="text-2xl font-bold">Ən sərfəli marşrut</h1>
      <p className="mt-1 text-sm text-gray-500">
        Şəhərləri seç, sistem ən ucuz sıranı çıxarır; rahatlıq/nəqliyyat dəyişdikcə total anında yenilənir.
      </p>

      {/* Wizard */}
      <section className="mt-6 space-y-4 rounded-lg border border-gray-200 p-4">
        <div>
          <div className="text-sm font-medium">Şəhərlər</div>
          <div className="mt-2 flex flex-wrap gap-2">
            {CITIES.map((c) => (
              <button
                key={c.code}
                onClick={() => toggleCity(c.code)}
                className={`rounded-full border px-3 py-1 text-sm ${
                  selected.includes(c.code)
                    ? "border-blue-600 bg-blue-50 text-blue-700"
                    : "border-gray-300 text-gray-600"
                }`}
              >
                {c.name}
              </button>
            ))}
          </div>
        </div>

        <div className="grid grid-cols-2 gap-4 sm:grid-cols-4">
          <label className="text-sm">
            Başlanğıc
            <select
              value={start}
              onChange={(e) => setStart(e.target.value)}
              className="mt-1 w-full rounded border border-gray-300 p-1.5"
            >
              {CITIES.filter((c) => selected.includes(c.code)).map((c) => (
                <option key={c.code} value={c.code}>
                  {c.name}
                </option>
              ))}
            </select>
          </label>

          <label className="text-sm">
            Rahatlıq
            <select
              value={tier}
              onChange={(e) => setTier(e.target.value as Tier)}
              className="mt-1 w-full rounded border border-gray-300 p-1.5"
            >
              <option value="budget">Budget (3★)</option>
              <option value="standard">Standart (4★)</option>
              <option value="vip">VIP (5★)</option>
            </select>
          </label>

          <label className="text-sm">
            Nəqliyyat
            <select
              value={pref}
              onChange={(e) => setPref(e.target.value as ModePref)}
              className="mt-1 w-full rounded border border-gray-300 p-1.5"
            >
              <option value="cheapest">Ən ucuz</option>
              <option value="flight">Təyyarə</option>
              <option value="train">Qatar</option>
              <option value="bus">Avtobus</option>
            </select>
          </label>

          <label className="text-sm">
            Gecə / şəhər
            <input
              type="number"
              min={1}
              max={10}
              value={nights}
              onChange={(e) => setNights(Math.max(1, Number(e.target.value)))}
              className="mt-1 w-full rounded border border-gray-300 p-1.5"
            />
          </label>
        </div>

        <div className="flex items-center gap-3">
          <button
            onClick={onPlan}
            disabled={loading}
            className="rounded bg-blue-600 px-4 py-2 text-sm font-medium text-white disabled:opacity-50"
          >
            {loading ? "Hesablanır..." : "Marşrutu hesabla"}
          </button>
          <label className="text-sm text-gray-600">
            Valyuta{" "}
            <select
              value={currency}
              onChange={(e) => setCurrency(e.target.value)}
              className="rounded border border-gray-300 p-1"
            >
              {CURRENCIES.map((c) => (
                <option key={c} value={c}>
                  {c}
                </option>
              ))}
            </select>
          </label>
        </div>
      </section>

      {error && (
        <p className="mt-4 rounded bg-red-50 px-3 py-2 text-sm text-red-700">{error}</p>
      )}

      {/* Results */}
      {planned && breakdown && (
        <section className="mt-6 grid gap-4 md:grid-cols-2">
          <div className="rounded-lg border border-gray-200 p-4">
            <div className="flex items-center justify-between">
              <h2 className="font-semibold">Marşrut</h2>
              <span className="rounded bg-gray-100 px-2 py-0.5 text-xs text-gray-600">
                {planned.route.variant === "open_jaw" ? "açıq-çənə" : "gediş-dönüş"}
              </span>
            </div>
            <ol className="mt-3 space-y-2 text-sm">
              {planned.route.order.map((idx, i) => (
                <li key={i}>
                  <span className="font-medium">{planned.cities[idx].name}</span>
                  {i < legs.length && (
                    <span className="text-gray-500">
                      {" "}→ {legs[i].mode} ({fmt(legs[i].price)} EUR)
                    </span>
                  )}
                </li>
              ))}
            </ol>
            {planned.alt && planned.alt.order && planned.alt.order.length > 0 && (
              <p className="mt-3 text-xs text-gray-400">
                Alternativ ({planned.alt.variant === "open_jaw" ? "açıq-çənə" : "gediş-dönüş"}):
                nəqliyyat ~{fmt(planned.alt.cost)} EUR
              </p>
            )}
          </div>

          <div className="rounded-lg border border-gray-200 p-4">
            <h2 className="font-semibold">Canlı xərc ({breakdown.currency})</h2>
            <dl className="mt-3 space-y-1 text-sm">
              <Row label="Nəqliyyat" value={fmt(breakdown.transport)} />
              <Row label="Otellər" value={fmt(breakdown.hotels)} />
              <Row label="Aktivliklər" value={fmt(breakdown.activities)} />
              <div className="mt-2 flex justify-between border-t pt-2 font-semibold">
                <dt>Cəmi</dt>
                <dd>
                  {fmt(breakdown.total)} {breakdown.currency}
                </dd>
              </div>
            </dl>
            <p className="mt-3 text-xs text-gray-400">
              Qiymətlər indikativdir (demo data). Rahatlıq/nəqliyyat/valyuta dəyişdikcə yalnız yenidən
              qiymətləndirilir — marşrut sırası dəyişmir.
            </p>
          </div>
        </section>
      )}
    </main>
  );
}

function Row({ label, value }: { label: string; value: string }) {
  return (
    <div className="flex justify-between">
      <dt className="text-gray-600">{label}</dt>
      <dd>{value}</dd>
    </div>
  );
}
