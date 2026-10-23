// Typed client for the plan endpoints. The API base comes from the environment
// so the same build works against local dev and deployed backends.

const BASE =
  process.env.NEXT_PUBLIC_API_BASE_URL ?? "http://localhost:8080/api/v1";

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
  const res = await withTimeout(`${BASE}${path}`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(body),
  });
  return parseEnvelope<T>(res);
}

async function getJSON<T>(path: string): Promise<T> {
  const res = await withTimeout(`${BASE}${path}`, {});
  return parseEnvelope<T>(res);
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

// track records a privacy-safe aggregate event (fire-and-forget, never throws).
export function track(name: string): void {
  try {
    void fetch(`${BASE}/analytics/event`, {
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

export interface PlanLeg {
  from: string;
  to: string;
  mode: string;
  price: number;
  estimated: boolean;
  source: string;
  source_tier: string;
}

export interface PlanQuote {
  order: string[];
  variant: string;
  legs: PlanLeg[];
  transport: number;
  hotels: number;
  activities: number;
  total: number;
  currency: string;
  hotel_stars: number;
  nights: number;
  nights_by_city?: number[];
  activities_per_night: number;
  estimated: boolean;
  alt_variant?: string;
  alt_cost?: number;
}

export interface PlanQuoteReq {
  cities: string[];
  places?: { name: string; lat: number; lng: number }[];
  start: string;
  mode: string;
  tier: string;
  nights: number;
  currency: string;
  order?: string[]; // supplied => re-price that order without re-sequencing
  variant?: string;
  start_date?: string;
  return_date?: string;
  adults?: number;
  children?: number;
  infants?: number;
}

export function planQuote(req: PlanQuoteReq): Promise<PlanQuote> {
  return postJSON("/data/plan", req);
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
  legs: { from: string; to: string; mode: string; price: number }[];
  nights: number;
  nights_by_city?: number[];
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
