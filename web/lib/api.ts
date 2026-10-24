// Typed client for the plan endpoints. The API base is resolved at RUNTIME so
// the same build works locally and over a LAN (D11): an explicit
// NEXT_PUBLIC_API_BASE_URL wins; otherwise, in the browser, it is built from the
// page's own host (so a phone opening http://<lan-ip>:3000 calls
// http://<lan-ip>:8080) — never a hardcoded "localhost". SSR/build falls back to
// localhost, but all real requests run client-side.
function apiBase(): string {
  const env = process.env.NEXT_PUBLIC_API_BASE_URL;
  if (env && env.trim() !== "") return env;
  if (typeof window !== "undefined" && window.location?.hostname) {
    const port = process.env.NEXT_PUBLIC_API_PORT || "8080";
    return `${window.location.protocol}//${window.location.hostname}:${port}/api/v1`;
  }
  return "http://localhost:8080/api/v1";
}

export interface Money {
  amount: number;
  currency: string;
}

export interface RouteDTO {
  variant: string;
  order: number[];
  cost: number;
}

export interface PricedLeg {
  from: number;
  to: number;
  mode: string;
  price: Money;
}

export interface PricedHotel {
  city: number;
  stars: number;
  nights: number;
  nightly: Money;
}

export interface PriceInput {
  order: number[];
  legs: PricedLeg[];
  hotels: PricedHotel[];
  activities_total: Money;
  display_currency: string;
  rates: Record<string, number>;
}

export interface Breakdown {
  transport: number;
  hotels: number;
  activities: number;
  total: number;
  currency: string;
}

interface Envelope<T> {
  success: boolean;
  data?: T;
  error?: string;
}

// ApiError carries the HTTP status so callers can distinguish cases such as 503
// (e.g. AI not configured) from a generic failure.
export class ApiError extends Error {
  status: number;
  constructor(status: number, message: string) {
    super(message);
    this.name = "ApiError";
    this.status = status;
  }
}

// Default request timeout. AI calls (Gemini) can take tens of seconds, so this
// is generous; it exists to fail cleanly instead of hanging forever.
const REQUEST_TIMEOUT_MS = 60_000;

async function parseEnvelope<T>(res: Response): Promise<T> {
  let json: Envelope<T> | null = null;
  try {
    json = (await res.json()) as Envelope<T>;
  } catch {
    /* non-JSON body */
  }
  if (!res.ok || !json || json.success === false) {
    throw new ApiError(res.status, json?.error ?? `request failed (${res.status})`);
  }
  return json.data as T;
}

// withTimeout runs a fetch under an AbortController; a timeout throws an
// ApiError(408) so callers show a "try again" instead of hanging.
async function withTimeout(url: string, init: RequestInit): Promise<Response> {
  const ctrl = new AbortController();
  const timer = setTimeout(() => ctrl.abort(), REQUEST_TIMEOUT_MS);
  try {
    return await fetch(url, { ...init, signal: ctrl.signal });
  } catch (e) {
    if (e instanceof DOMException && e.name === "AbortError") {
      throw new ApiError(408, "request timed out");
    }
    throw e;
  } finally {
    clearTimeout(timer);
  }
}

async function postJSON<T>(path: string, body: unknown): Promise<T> {
  const res = await withTimeout(`${apiBase()}${path}`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(body),
  });
  return parseEnvelope<T>(res);
}

async function getJSON<T>(path: string): Promise<T> {
  const res = await withTimeout(`${apiBase()}${path}`, {});
  return parseEnvelope<T>(res);
}

// --- client-side response cache (§6: aggressive cache, no re-fetch) ----------
// A small signature-keyed cache that also de-duplicates in-flight requests, so
// toggling back to an already-computed plan / roadmap / POI set is instant and
// never re-hits the network. TTL-bounded and size-capped; results stay fresh
// enough (server fares are cached ~6h) while a session's toggles feel instant.

interface CacheEntry<T> {
  at: number;
  value?: T;
  promise?: Promise<T>;
}

const CACHE_TTL_MS = 5 * 60_000;
const CACHE_MAX = 60;
const responseCache = new Map<string, CacheEntry<unknown>>();

function cached<T>(key: string, fn: () => Promise<T>): Promise<T> {
  const now = Date.now();
  const hit = responseCache.get(key) as CacheEntry<T> | undefined;
  if (hit) {
    if (hit.value !== undefined && now - hit.at < CACHE_TTL_MS) return Promise.resolve(hit.value);
    if (hit.promise) return hit.promise; // in-flight de-dupe
  }
  const promise = fn()
    .then((v) => {
      responseCache.set(key, { at: Date.now(), value: v });
      if (responseCache.size > CACHE_MAX) {
        const oldest = responseCache.keys().next().value;
        if (oldest !== undefined) responseCache.delete(oldest);
      }
      return v;
    })
    .catch((e) => {
      responseCache.delete(key); // never cache failures
      throw e;
    });
  responseCache.set(key, { at: now, promise });
  return promise;
}

// Destination from the dynamic, knowledge-backed city list (GET /data/destinations).
export interface DestinationDTO {
  country: string;
  city: string;
  lat: number;
  lng: number;
  summary: string;
}

export function destinations(): Promise<DestinationDTO[]> {
  return getJSON("/data/destinations");
}

// Place autocomplete (GET /data/places) — real city/airport search with coords.
export interface Place {
  code: string;
  name: string;
  country: string;
  lat: number;
  lng: number;
  kind: string; // "city" | "airport"
}

export function searchPlaces(q: string, locale = "en"): Promise<Place[]> {
  return getJSON(`/data/places?q=${encodeURIComponent(q)}&locale=${encodeURIComponent(locale)}`);
}

// Real, open-data points of interest for a city (GET /data/poi) — OSM Overpass,
// provenance-tagged (not estimated). Image/website/wiki present when OSM has them.
export interface POIItem {
  city: string;
  name: string;
  category: string;
  lat: number;
  lng: number;
  image?: string;
  website?: string;
  wiki?: string;
  desc?: string;
  free?: boolean;
  price?: string;
  rating?: number; // only when a real source provided it (no fake stars)
  review_count?: number;
  review_summary?: string;
  provenance?: { source: string; fetched_at?: string; estimated?: boolean };
}

export function fetchPOIs(city: string, limit = 8): Promise<POIItem[]> {
  return cached(`poi:${city.toLowerCase()}:${limit}`, () =>
    getJSON(`/data/poi?city=${encodeURIComponent(city)}&limit=${limit}`)
  );
}

// track records a privacy-safe aggregate event (fire-and-forget, never throws).
export function track(name: string): void {
  try {
    void fetch(`${apiBase()}/analytics/event`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ name }),
      keepalive: true,
    }).catch(() => {});
  } catch {
    /* ignore */
  }
}

// --- priced plan (single source of truth: flywheel + estimator + engine) ----

export interface LegAlt {
  source: string;
  price: number;
  url: string;
  chosen: boolean;
  // Optional fare details, shown only when the source stated them.
  operator?: string;
  depart?: string;
  duration?: string;
  stops?: string;
  baggage?: string;
}

export interface PlanLeg {
  from: string;
  to: string;
  mode: string;
  price: number;
  estimated: boolean;
  source: string;
  source_tier: string;
  url?: string;
  alternatives?: LegAlt[];
  fetched_at?: string;
  requested_mode?: string; // set when the requested mode was unavailable
  mode_unavailable?: boolean;
  // Chosen-fare detail shown inline on the route (D3); only when the source
  // stated it (else "~estimate").
  operator?: string;
  depart?: string;
  duration?: string;
  stops?: string;
  baggage?: string;
}

export interface HotelAlt {
  stars: number;
  nightly: number;
  total: number;
  chosen: boolean;
}

export interface HotelLine {
  city: string;
  stars: number;
  name?: string; // real hotel name when a source has one (else star descriptor)
  nights: number;
  nightly: number;
  total: number;
  estimated: boolean;
  source: string;
  url?: string;
  fetched_at?: string;
  alts?: HotelAlt[]; // other star tiers for this city (D6), chosen marked
}

export interface PlanQuote {
  order: string[];
  variant: string;
  legs: PlanLeg[];
  hotel_lines?: HotelLine[];
  transport: number;
  hotels: number;
  activities: number;
  total: number;
  currency: string;
  hotel_stars: number;
  nights: number;
  nights_by_city?: number[];
  travelers?: number;
  per_person?: number;
  activities_per_night: number;
  estimated: boolean;
  alt_variant?: string;
  alt_cost?: number;
  alt_order?: string[];
  alt_nights_by_city?: number[];
}

export interface PlanQuoteReq {
  cities: string[];
  places?: { name: string; lat: number; lng: number }[];
  origin?: string;
  budget_eur?: number;
  start: string;
  mode: string;
  tier: string;
  nights: number;
  currency: string;
  order?: string[]; // supplied => re-price that order without re-sequencing
  variant?: string;
  leg_modes?: Record<string, string>; // per-leg mode override ("from|to" -> mode)
  start_date?: string;
  return_date?: string;
  adults?: number;
  children?: number;
  infants?: number;
  city_nights?: Record<string, number>; // per-city nights override (stepper)
  city_stars?: Record<string, number>; // per-city hotel star override (D6)
}

export function planQuote(req: PlanQuoteReq): Promise<PlanQuote> {
  return cached("plan:" + JSON.stringify(req), () => postJSON("/data/plan", req));
}

// --- AI (Claude) -----------------------------------------------------------

export interface Violation {
  value: number;
}

// AIResult mirrors the backend ai.Result envelope.
export interface AIResult {
  text: string;
  grounding?: number[];
  violations?: Violation[];
  steps?: number;
}

export interface ItineraryBrief {
  cities: string[];
  places?: { name: string; lat: number; lng: number }[]; // city coords for roadmap pins
  legs: { from: string; to: string; mode: string; price: number }[];
  nights: number;
  nights_by_city?: number[];
  travelers?: number; // party size, for the per-person activities estimate
  currency: string;
  transport_total: number;
  hotels_total: number;
  activities_total: number;
  total: number;
}

export function itinerary(brief: ItineraryBrief, locale = "en"): Promise<AIResult> {
  return postJSON("/ai/itinerary", { ...brief, locale });
}

export function chat(message: string, locale = "en"): Promise<AIResult> {
  return postJSON("/ai/chat", { message, locale });
}

// --- structured roadmap (§7): day-by-day, location-based, never raw markdown -

export interface RoadmapPOI {
  name: string;
  type?: string;
  desc?: string;
  free: boolean;
  price?: string; // approximate, "~"-marked (model knowledge), optional
  lat?: number;
  lng?: number;
}

export interface RoadmapDay {
  day: number;
  city: string;
  title?: string;
  pois: RoadmapPOI[];
}

export interface RoadmapCity {
  city: string;
  nights: number;
  summary?: string;
  best_season?: string;
  lat?: number;
  lng?: number;
}

// FoodLine is an approximate per-city restaurant spend (D4): a mid-range daily
// cost × nights, always "~"-labeled (model knowledge, not a quoted price).
export interface FoodLine {
  city: string;
  per_day: number; // per person, approximate, EUR-normalized to display currency
  nights: number;
  total?: number;
}

export interface Roadmap {
  cities: RoadmapCity[];
  days: RoadmapDay[];
  food?: FoodLine[];
}

export function roadmap(
  cities: string[],
  nights_by_city: number[],
  places: { name: string; lat: number; lng: number }[],
  locale = "en",
  currency = "EUR"
): Promise<Roadmap> {
  const body = { cities, nights_by_city, places, locale, currency };
  return cached("roadmap:" + JSON.stringify(body), () => postJSON("/ai/roadmap", body));
}

// Conversational refine: the backend LLM parses the edit, applies it
// deterministically, and returns the new wizard state + machine-readable change
// codes. The client re-runs the engine with the returned state.
export interface EditState {
  cities: string[];
  start: string;
  tier: string;
  mode: string;
  nights: number;
}

export interface RefineResult {
  state: EditState;
  changed: string[];
}

export function refine(message: string, state: EditState): Promise<RefineResult> {
  return postJSON("/ai/refine", { message, state });
}

export function sequence(
  matrix: number[][],
  start: number,
  end: number
): Promise<{ best: RouteDTO; alt: RouteDTO }> {
  return postJSON("/plan/sequence", { matrix, start, end });
}

export function price(input: PriceInput): Promise<Breakdown> {
  return postJSON("/plan/price", input);
}
