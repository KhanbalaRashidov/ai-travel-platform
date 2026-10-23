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

async function postJSON<T>(path: string, body: unknown): Promise<T> {
  const res = await fetch(`${BASE}${path}`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(body),
  });
  return parseEnvelope<T>(res);
}

async function getJSON<T>(path: string): Promise<T> {
  const res = await fetch(`${BASE}${path}`);
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
  currency: string;
  transport_total: number;
  hotels_total: number;
  activities_total: number;
  total: number;
}

export function itinerary(brief: ItineraryBrief): Promise<AIResult> {
  return postJSON("/ai/itinerary", brief);
}

export function chat(message: string): Promise<AIResult> {
  return postJSON("/ai/chat", { message });
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
