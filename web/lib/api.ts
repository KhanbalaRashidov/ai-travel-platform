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

async function postJSON<T>(path: string, body: unknown): Promise<T> {
  const res = await fetch(`${BASE}${path}`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(body),
  });
  const json = (await res.json()) as Envelope<T>;
  if (!res.ok || json.success === false) {
    throw new Error(json.error ?? `request failed (${res.status})`);
  }
  return json.data as T;
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
