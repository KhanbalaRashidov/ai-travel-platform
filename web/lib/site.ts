// Resolve the public site origin for metadata/OG/sitemap/robots. Guards against
// the value being an empty string (not just undefined) — e.g. a Docker build arg
// `NEXT_PUBLIC_SITE_URL=` arrives as "" and `?? fallback` would NOT replace it,
// so `new URL("")` used to throw and crash the whole build at /_not-found. Using
// a truthy guard + a try/catch makes `next build` pass with any NEXT_PUBLIC_*.
const FALLBACK = "http://localhost:3000";

export function siteUrl(): string {
  const raw = (process.env.NEXT_PUBLIC_SITE_URL || "").trim();
  if (raw === "") return FALLBACK;
  try {
    // Validate; drop any trailing slash for consistent concatenation.
    return new URL(raw).toString().replace(/\/$/, "");
  } catch {
    return FALLBACK;
  }
}

// A guaranteed-valid URL object for Next's metadataBase.
export function siteUrlObject(): URL {
  return new URL(siteUrl());
}
