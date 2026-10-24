package pricefetch

import (
	"net/url"
	"strconv"
	"strings"
)

// DeepLink returns a REAL, MODE-SPECIFIC, route-prefilled URL on the matching
// travel site (never a generic/empty search): the route (from→to), the date(s)
// and the passenger count are placed into the URL so the site opens with the
// search already filled in (D14). All templates are name-based (work with city
// names, no IATA needed) so they resolve reliably. When the mode is unknown /
// "cheapest" it falls back to Rome2Rio (multimodal, name-based, shows prices).
func DeepLink(from, to, mode, date, returnDate string, adults, children, infants int) string {
	seg := func(s string) string { return url.PathEscape(strings.TrimSpace(s)) }
	a := adults
	if a < 1 {
		a = 1
	}
	ch := children
	if ch < 0 {
		ch = 0
	}
	inf := infants
	if inf < 0 {
		inf = 0
	}
	switch mode {
	case "train":
		// Trainline results, route + date(s) + passengers prefilled.
		q := url.Values{}
		q.Set("origin", strings.TrimSpace(from))
		q.Set("destination", strings.TrimSpace(to))
		if date != "" {
			q.Set("outboundDate", date)
		}
		if returnDate != "" {
			q.Set("inboundDate", returnDate)
		}
		q.Set("adults", strconv.Itoa(a))
		if ch > 0 {
			q.Set("children", strconv.Itoa(ch))
		}
		return "https://www.thetrainline.com/book/results?" + q.Encode()
	case "bus":
		// FlixBus search, route + date + passengers prefilled.
		q := url.Values{}
		q.Set("departureCity", strings.TrimSpace(from))
		q.Set("arrivalCity", strings.TrimSpace(to))
		if date != "" {
			q.Set("rideDate", date)
		}
		q.Set("adult", strconv.Itoa(a))
		if ch > 0 {
			q.Set("children", strconv.Itoa(ch))
		}
		return "https://global.flixbus.com/search?" + q.Encode()
	case "ferry":
		// Direct Ferries, name-based route page.
		return "https://www.directferries.com/ferry-" + seg(from) + "-" + seg(to) + ".htm"
	case "car":
		// Google Maps driving directions, route prefilled.
		q := url.Values{}
		q.Set("api", "1")
		q.Set("origin", strings.TrimSpace(from))
		q.Set("destination", strings.TrimSpace(to))
		q.Set("travelmode", "driving")
		return "https://www.google.com/maps/dir/?" + q.Encode()
	case "flight":
		// Google Flights name-based query (resolves city names without IATA; the
		// frontend upgrades this to Skyscanner with IATA codes for airport stops).
		q := "flights from " + strings.TrimSpace(from) + " to " + strings.TrimSpace(to)
		if date != "" {
			q += " on " + date
		}
		if returnDate != "" {
			q += " returning " + returnDate
		}
		return "https://www.google.com/travel/flights?q=" + url.QueryEscape(q)
	default:
		// Multimodal "cheapest": Rome2Rio, route + date + pax prefilled.
		u := "https://www.rome2rio.com/map/" + seg(from) + "/" + seg(to)
		q := url.Values{}
		if date != "" {
			q.Set("outDate", date)
		}
		if returnDate != "" {
			q.Set("returnDate", returnDate)
		}
		q.Set("pax", strconv.Itoa(a+ch+inf))
		u += "?" + q.Encode()
		return u
	}
}

// HotelLink returns a real, prefilled Booking.com search for a city + dates.
func HotelLink(city, checkin, checkout string, adults int) string {
	q := url.Values{}
	q.Set("ss", strings.TrimSpace(city))
	if checkin != "" {
		q.Set("checkin", checkin)
	}
	if checkout != "" {
		q.Set("checkout", checkout)
	}
	if adults > 0 {
		q.Set("group_adults", strconv.Itoa(adults))
	}
	return "https://www.booking.com/searchresults.html?" + q.Encode()
}
