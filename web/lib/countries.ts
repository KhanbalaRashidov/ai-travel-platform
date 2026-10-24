// Localized country autocomplete data (D19). No external keyed API: country
// names are resolved from ISO 3166-1 alpha-2 codes via Intl.DisplayNames in the
// UI language (AZ/EN/RU/TR), with an English fallback. Search is diacritic- and
// case-insensitive and matches the local name, the English name or the code, so
// a user can type "almaniy" (AZ) and find Almaniya. The selected country keeps
// its code (canonical) for the AI, and its local name for display.

// A broad, tourism-oriented set of ISO alpha-2 codes.
export const COUNTRY_CODES = [
  "AL", "AD", "AM", "AR", "AU", "AT", "AZ", "BH", "BD", "BY", "BE", "BA", "BR", "BG", "KH",
  "CA", "CL", "CN", "CO", "CR", "HR", "CU", "CY", "CZ", "DK", "DO", "EC", "EG", "EE", "FI",
  "FR", "GE", "DE", "GR", "GT", "HK", "HU", "IS", "IN", "ID", "IR", "IQ", "IE", "IL", "IT",
  "JP", "JO", "KZ", "KE", "KW", "KG", "LA", "LV", "LB", "LT", "LU", "MO", "MY", "MV", "MT",
  "MX", "MD", "MC", "ME", "MA", "NP", "NL", "NZ", "MK", "NO", "OM", "PK", "PA", "PE", "PH",
  "PL", "PT", "QA", "RO", "RU", "SA", "RS", "SG", "SK", "SI", "ZA", "KR", "ES", "LK", "SE",
  "CH", "TW", "TJ", "TH", "TN", "TR", "TM", "UA", "AE", "GB", "US", "UY", "UZ", "VN",
];

// Map a UI locale to the BCP-47 tag Intl.DisplayNames uses.
function intlLocale(locale: string): string {
  switch (locale) {
    case "az":
      return "az";
    case "ru":
      return "ru";
    case "tr":
      return "tr";
    default:
      return "en";
  }
}

const cache = new Map<string, Intl.DisplayNames>();
function displayNames(locale: string): Intl.DisplayNames | null {
  const key = intlLocale(locale);
  if (cache.has(key)) return cache.get(key)!;
  try {
    const dn = new Intl.DisplayNames([key], { type: "region" });
    cache.set(key, dn);
    return dn;
  } catch {
    return null;
  }
}

// countryName resolves a code to its name in the given locale (English fallback).
export function countryName(code: string, locale: string): string {
  const dn = displayNames(locale);
  try {
    const n = dn?.of(code);
    if (n && n !== code) return n;
  } catch {
    /* ignore */
  }
  try {
    const en = displayNames("en")?.of(code);
    if (en && en !== code) return en;
  } catch {
    /* ignore */
  }
  return code;
}

// norm lowercases and strips diacritics for accent-insensitive matching.
function norm(s: string): string {
  return s
    .normalize("NFD")
    .replace(/\p{Diacritic}/gu, "")
    .toLowerCase()
    .trim();
}

export interface CountryOption {
  code: string;
  name: string; // localized
}

// searchCountries returns localized matches for a query (local name, English
// name, or ISO code), prefix matches first. Empty for a blank query.
export function searchCountries(query: string, locale: string, limit = 10): CountryOption[] {
  const nq = norm(query);
  if (nq.length < 1) return [];
  const scored: { opt: CountryOption; score: number }[] = [];
  for (const code of COUNTRY_CODES) {
    const local = countryName(code, locale);
    const en = countryName(code, "en");
    const nl = norm(local);
    const ne = norm(en);
    let score = -1;
    if (nl.startsWith(nq) || ne.startsWith(nq) || code.toLowerCase() === nq) score = 2;
    else if (nl.includes(nq) || ne.includes(nq)) score = 1;
    if (score >= 0) scored.push({ opt: { code, name: local }, score });
  }
  scored.sort((a, b) => b.score - a.score || a.opt.name.localeCompare(b.opt.name));
  return scored.slice(0, limit).map((s) => s.opt);
}

// canonicalName returns the English name for a code — the stable form sent to the
// AI so suggestions resolve correctly regardless of UI language.
export function canonicalName(code: string): string {
  return countryName(code, "en");
}

const ALL_LOCALES = ["az", "en", "ru", "tr"];

// codeByName reverse-maps a country name (in any of the 4 UI languages, English,
// or an ISO code) back to its code — so an AI panel heading can be re-localized
// to the current UI language (D20). Returns "" when unknown.
export function codeByName(name: string): string {
  const n = norm(name);
  if (!n) return "";
  for (const code of COUNTRY_CODES) {
    if (code.toLowerCase() === n) return code;
    for (const loc of ALL_LOCALES) {
      if (norm(countryName(code, loc)) === n) return code;
    }
  }
  return "";
}

// localizeCountry returns `name` rendered in `locale` when it maps to a known
// country, otherwise the original string (so unknown AI labels still show).
export function localizeCountry(name: string, locale: string): string {
  const code = codeByName(name);
  return code ? countryName(code, locale) : name;
}
