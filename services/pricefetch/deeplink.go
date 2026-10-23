package pricefetch

import (
	"net/url"
	"strconv"
	"strings"
)

// DeepLink returns a REAL, route-prefilled search URL on a travel site (never a
// generic search engine). Rome2Rio is used because it is name-based (works with
// city names, no IATA needed), multimodal, and shows prices — so clicking opens
// the actual route with results the user can verify.
func DeepLink(from, to, mode, date string, adults int) string {
	seg := func(s string) string { return url.PathEscape(strings.TrimSpace(s)) }
	u := "https://www.rome2rio.com/map/" + seg(from) + "/" + seg(to)
	q := url.Values{}
	if date != "" {
		q.Set("outDate", date)
	}
	if adults > 0 {
		q.Set("pax", strconv.Itoa(adults))
	}
	if len(q) > 0 {
		u += "?" + q.Encode()
	}
	return u
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
