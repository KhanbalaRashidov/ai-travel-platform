// Pre-filled, real deep-links for hotels (D5/D14). Each opens the site with the
// search ALREADY filled in (city + dates + guests) so the user never re-types.
// We only ever provide links here — never a fabricated per-site price (guardrail:
// a price is shown only when a source actually returned it).

export interface SiteLink {
  name: string;
  url: string;
}

function enc(s: string): string {
  return encodeURIComponent(s.trim());
}

// yymmdd formats YYYY-MM-DD as the yyMMdd Skyscanner expects ("" if unparseable).
function yymmdd(date?: string): string {
  if (!date) return "";
  const m = date.match(/^(\d{4})-(\d{2})-(\d{2})$/);
  return m ? m[1].slice(2) + m[2] + m[3] : "";
}

// skyscannerFlightLink builds a prefilled Skyscanner search from IATA/airport
// codes (D14): route + date(s) + passengers. Returns "" when codes are missing
// so the caller can keep the name-based fallback (Google Flights).
export function skyscannerFlightLink(
  fromCode?: string,
  toCode?: string,
  startDate?: string,
  returnDate?: string,
  adults = 1,
  children = 0,
  infants = 0
): string {
  const f = (fromCode ?? "").trim().toLowerCase();
  const t = (toCode ?? "").trim().toLowerCase();
  if (f.length < 3 || t.length < 3) return "";
  const out = yymmdd(startDate);
  const ret = yymmdd(returnDate);
  const path = ["https://www.skyscanner.net/transport/flights", f, t, out, ret]
    .filter((s, i) => i < 3 || s) // always keep host/from/to; dates only if set
    .join("/");
  const q = new URLSearchParams({
    adults: String(Math.max(1, adults)),
    children: String(Math.max(0, children)),
    infants: String(Math.max(0, infants)),
  });
  return `${path}/?${q.toString()}`;
}

// hotelSites returns prefilled searches on the major hotel sites for a city and
// stay. checkin/checkout are YYYY-MM-DD (optional); adults >= 1.
export function hotelSites(
  city: string,
  checkin?: string,
  checkout?: string,
  adults = 1
): SiteLink[] {
  const a = Math.max(1, adults);
  const ci = checkin ?? "";
  const co = checkout ?? "";
  const links: SiteLink[] = [];

  // Booking.com
  {
    const q = new URLSearchParams({ ss: city, group_adults: String(a) });
    if (ci) q.set("checkin", ci);
    if (co) q.set("checkout", co);
    links.push({ name: "Booking", url: `https://www.booking.com/searchresults.html?${q.toString()}` });
  }
  // Hotels.com
  {
    const q = new URLSearchParams({ destination: city, adults: String(a) });
    if (ci) q.set("startDate", ci);
    if (co) q.set("endDate", co);
    links.push({ name: "Hotels.com", url: `https://www.hotels.com/Hotel-Search?${q.toString()}` });
  }
  // Agoda
  {
    const q = new URLSearchParams({ textToSearch: city, adults: String(a) });
    if (ci) q.set("checkIn", ci);
    if (co) q.set("checkOut", co);
    links.push({ name: "Agoda", url: `https://www.agoda.com/search?${q.toString()}` });
  }
  // Trivago
  links.push({ name: "Trivago", url: `https://www.trivago.com/en-US/srl?query=${enc(city)}` });
  // Hotellook
  {
    const q = new URLSearchParams({ destination: city, adults: String(a) });
    if (ci) q.set("checkIn", ci);
    if (co) q.set("checkOut", co);
    links.push({ name: "Hotellook", url: `https://search.hotellook.com/?${q.toString()}` });
  }
  return links;
}
