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
