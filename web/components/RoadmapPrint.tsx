"use client";

import { FoodLine, HotelLine, PlanLeg, Roadmap } from "../lib/api";
import { useI18n } from "../lib/i18n";

// RoadmapPrint renders the AI day-by-day roadmap as a clean, structured,
// print-friendly document (D15): per-city info (nights + summary + best season),
// day-by-day places (name, free/paid, ~price, Google Maps URL), hotels and a
// cost summary. It is hidden on screen and shown only when printing; blocks
// avoid mid-page breaks and link URLs are printed in full. Same content as the
// on-screen roadmap — never raw markdown.
function mapsLink(name: string, city: string): string {
  return `https://www.google.com/maps/search/?api=1&query=${encodeURIComponent(name + " " + city)}`;
}

export function RoadmapPrint({
  roadmap,
  legs,
  hotels,
  activities,
  food,
  foodTotal,
  transport,
  hotelsTotal,
  total,
  perPerson,
  travelers,
  currency,
  fmt,
}: {
  roadmap: Roadmap;
  legs: PlanLeg[];
  hotels: HotelLine[];
  activities: number;
  food: FoodLine[];
  foodTotal: number;
  transport: number;
  hotelsTotal: number;
  total: number;
  perPerson: number;
  travelers: number;
  currency: string;
  fmt: (n: number) => string;
}) {
  const { t } = useI18n();
  const hotelByCity = new Map(hotels.map((h) => [h.city.toLowerCase(), h]));
  const daysByCity = new Map<string, Roadmap["days"]>();
  for (const d of roadmap.days) {
    const k = d.city.toLowerCase();
    daysByCity.set(k, [...(daysByCity.get(k) ?? []), d]);
  }

  return (
    <div className="print-doc hidden print:block">
      <h1 className="mb-1 text-xl font-bold">{t("print.title")}</h1>
      <p className="mb-4 text-sm">
        {t("res.nights", { n: roadmap.cities.reduce((a, c) => a + c.nights, 0) })} ·{" "}
        {roadmap.cities.length} {t("plan.cities")} · {fmt(total)} {currency}
        {travelers > 1 ? ` · ${fmt(perPerson)} ${currency}/${t("res.perPerson")}` : ""}
      </p>

      {roadmap.cities.map((c, i) => {
        const h = hotelByCity.get(c.city.toLowerCase());
        const days = daysByCity.get(c.city.toLowerCase()) ?? [];
        return (
          <section key={i} className="print-block mb-4 border-b border-black/20 pb-3">
            <h2 className="text-lg font-bold">
              {c.city} — {t("res.nights", { n: c.nights })}
              {c.best_season ? ` · ${t("res.season")}: ${c.best_season}` : ""}
            </h2>
            {c.summary && <p className="text-sm">{c.summary}</p>}
            {h && (
              <p className="mt-1 text-sm">
                🏨{" "}
                {h.url ? (
                  <a href={h.url}>{h.name || `${h.stars}★ ${t("res.hotelWord")}`}</a>
                ) : (
                  h.name || `${h.stars}★ ${t("res.hotelWord")}`
                )}{" "}
                · ~{fmt(h.nightly)} {currency}/{t("res.perNight")}
              </p>
            )}
            {days.map((d, k) => (
              <div key={k} className="print-block mt-2">
                <h3 className="text-sm font-semibold">
                  {t("res.day", { n: d.day })} · {d.city}
                  {d.title ? ` — ${d.title}` : ""}
                </h3>
                <ul className="ml-4 list-disc text-sm">
                  {(d.pois ?? []).map((p, j) => (
                    <li key={j}>
                      <a className="font-medium" href={mapsLink(p.name, d.city)}>
                        {p.name}
                      </a>
                      {p.type ? ` (${p.type})` : ""} · {p.free ? t("res.free") : p.price || t("res.approx")}
                    </li>
                  ))}
                </ul>
              </div>
            ))}
          </section>
        );
      })}

      {/* Transport legs */}
      <section className="print-block mb-4">
        <h2 className="text-lg font-bold">{t("res.transport")}</h2>
        <ul className="ml-4 list-disc text-sm">
          {legs.map((l, i) => (
            <li key={i}>
              {l.url ? (
                <a href={l.url}>
                  {l.from} → {l.to}
                </a>
              ) : (
                `${l.from} → ${l.to}`
              )}{" "}
              · {t(`mode.${l.mode}`)} · {l.estimated ? "~" : ""}
              {fmt(l.price)} {currency}
              {l.operator ? ` · ${l.operator}` : ""}
              {l.depart ? ` · ${l.depart}` : ""}
              {l.duration ? ` · ${l.duration}` : ""}
            </li>
          ))}
        </ul>
      </section>

      {/* Cost summary */}
      <section className="print-block">
        <h2 className="text-lg font-bold">{t("res.liveCost")}</h2>
        <table className="w-full text-sm">
          <tbody>
            <tr>
              <td>{t("res.transport")}</td>
              <td className="text-right">{fmt(transport)} {currency}</td>
            </tr>
            <tr>
              <td>{t("res.hotels")}</td>
              <td className="text-right">{fmt(hotelsTotal)} {currency}</td>
            </tr>
            <tr>
              <td>{t("res.activities")}</td>
              <td className="text-right">{fmt(activities)} {currency}</td>
            </tr>
            {foodTotal > 0 && (
              <tr>
                <td>{t("res.food")}</td>
                <td className="text-right">~{fmt(foodTotal)} {currency}</td>
              </tr>
            )}
            <tr className="font-bold">
              <td>{t("res.total")}</td>
              <td className="text-right">{fmt(total)} {currency}</td>
            </tr>
            {travelers > 1 && (
              <tr>
                <td>{t("res.perPerson")}</td>
                <td className="text-right">{fmt(perPerson)} {currency}</td>
              </tr>
            )}
          </tbody>
        </table>
      </section>
    </div>
  );
}
