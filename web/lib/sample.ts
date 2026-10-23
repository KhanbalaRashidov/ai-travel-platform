// Demo dataset for the E5 wizard. Until the real connectors/scraping are wired,
// intercity costs are derived from a simple distance+mode model and hotel
// nightly rates from a star table. This keeps the UI fully functional end-to-end
// against the real (pure) planner engine.

export type Mode = "flight" | "train" | "bus";

export interface City {
  code: string;
  name: string;
  lat: number;
  lng: number;
}

export const CITIES: City[] = [
  { code: "BAK", name: "Baku", lat: 40.4093, lng: 49.8671 },
  { code: "TBS", name: "Tbilisi", lat: 41.7151, lng: 44.8271 },
  { code: "BUS", name: "Batumi", lat: 41.6168, lng: 41.6367 },
  { code: "EVN", name: "Yerevan", lat: 40.1792, lng: 44.4991 },
  { code: "IST", name: "Istanbul", lat: 41.0082, lng: 28.9784 },
];

// Must match the planner's "no edge" sentinel (math.MaxFloat64).
export const INF = Number.MAX_VALUE;

function haversineKm(a: City, b: City): number {
  const R = 6371;
  const dLat = ((b.lat - a.lat) * Math.PI) / 180;
  const dLng = ((b.lng - a.lng) * Math.PI) / 180;
  const la1 = (a.lat * Math.PI) / 180;
  const la2 = (b.lat * Math.PI) / 180;
  const h =
    Math.sin(dLat / 2) ** 2 +
    Math.cos(la1) * Math.cos(la2) * Math.sin(dLng / 2) ** 2;
  return 2 * R * Math.asin(Math.sqrt(h));
}

// legPrice returns an indicative EUR price for a leg by mode, or INF if that
// mode is not viable for the distance.
export function legPrice(a: City, b: City, mode: Mode): number {
  if (a.code === b.code) return 0;
  const km = haversineKm(a, b);
  switch (mode) {
    case "flight":
      return Math.round(30 + 0.08 * km);
    case "train":
      return km <= 1500 ? Math.round(0.05 * km) : INF;
    case "bus":
      return km <= 1200 ? Math.round(0.03 * km) : INF;
  }
}

export function cheapestMode(a: City, b: City): { mode: Mode; price: number } {
  const modes: Mode[] = ["flight", "train", "bus"];
  let best: { mode: Mode; price: number } = { mode: "flight", price: INF };
  for (const m of modes) {
    const p = legPrice(a, b, m);
    if (p < best.price) best = { mode: m, price: p };
  }
  return best;
}

export type ModePref = "cheapest" | Mode;

export function buildMatrix(cities: City[], pref: ModePref): number[][] {
  return cities.map((a) =>
    cities.map((b) => {
      if (a.code === b.code) return 0;
      return pref === "cheapest" ? cheapestMode(a, b).price : legPrice(a, b, pref);
    })
  );
}

// Pick the mode actually used for a leg given the preference.
export function legFor(a: City, b: City, pref: ModePref): { mode: Mode; price: number } {
  if (pref === "cheapest") return cheapestMode(a, b);
  return { mode: pref, price: legPrice(a, b, pref) };
}

export type Tier = "budget" | "standard" | "vip";
export const TIER_STARS: Record<Tier, number> = { budget: 3, standard: 4, vip: 5 };
export const NIGHTLY_BY_STAR: Record<number, number> = { 2: 35, 3: 55, 4: 90, 5: 150 };

// StaticConverter rates: value of 1 unit in a common numeraire (EUR).
export const RATES: Record<string, number> = { EUR: 1, USD: 0.92, AZN: 0.53 };
export const CURRENCIES = ["EUR", "USD", "AZN"];
export const ACTIVITY_PER_NIGHT_EUR = 20;
