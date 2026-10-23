package pricefetch

import conn "github.com/KhanbalaRashidov/ai-travel-platform/services/connectors"

// builtinSpec is the shipped, zero-config source list (no PRICE_SOURCES needed).
// These are well-known public metasearch/operator pages. Many apply anti-bot
// measures and will simply be skipped gracefully at runtime (we never bypass
// them); reachable ones feed the verbatim-guarded extractor. The operator
// accepts the ToS of whichever sources actually respond. URL patterns are
// best-effort and are easy to tune via PRICE_SOURCES (which is merged on top).
//
// Format per entry: {name, tos, url-template, modes}. {from}/{to} are IATA/city
// strings, {date} is YYYY-MM-DD.
type builtinEntry struct {
	name  string
	tos   string
	tmpl  string
	modes []conn.TransportMode
}

var builtinEntries = []builtinEntry{
	// Flights (metasearch) — ToS generally restricts scraping; skipped if blocked.
	{"skyscanner", "flight metasearch; ToS restricts scraping", "https://www.skyscanner.net/transport/flights/{from}/{to}/{date}/", []conn.TransportMode{conn.ModeFlight}},
	{"kayak", "flight metasearch; ToS restricts scraping", "https://www.kayak.com/flights/{from}-{to}/{date}", []conn.TransportMode{conn.ModeFlight}},
	{"googleflights", "flight metasearch; ToS restricts scraping", "https://www.google.com/travel/flights?q=flights%20from%20{from}%20to%20{to}%20on%20{date}", []conn.TransportMode{conn.ModeFlight}},
	{"momondo", "flight metasearch; ToS restricts scraping", "https://www.momondo.com/flight-search/{from}-{to}/{date}", []conn.TransportMode{conn.ModeFlight}},
	{"kiwi", "flight aggregator; ToS restricts scraping", "https://www.kiwi.com/en/search/results/{from}/{to}/{date}", []conn.TransportMode{conn.ModeFlight}},
	{"aviasales", "flight metasearch; ToS restricts scraping", "https://www.aviasales.com/search/{from}{to}{date}", []conn.TransportMode{conn.ModeFlight}},
	{"jetradar", "flight metasearch; ToS restricts scraping", "https://www.jetradar.com/flights/{from}-{to}/", []conn.TransportMode{conn.ModeFlight}},

	// Ground transport (train/bus) — mixed ToS; skipped if blocked.
	{"rome2rio", "multimodal aggregator; ToS restricts scraping", "https://www.rome2rio.com/map/{from}/{to}", []conn.TransportMode{conn.ModeFlight, conn.ModeTrain, conn.ModeBus}},
	{"omio", "train/bus aggregator; ToS restricts scraping", "https://www.omio.com/search-front/results?departureCity={from}&arrivalCity={to}&departureDate={date}", []conn.TransportMode{conn.ModeTrain, conn.ModeBus}},
	{"flixbus", "bus operator; ToS restricts scraping", "https://global.flixbus.com/search?departureCity={from}&arrivalCity={to}&rideDate={date}", []conn.TransportMode{conn.ModeBus}},
	{"twelvego", "train/bus aggregator; ToS restricts scraping", "https://12go.asia/en/travel/{from}/{to}", []conn.TransportMode{conn.ModeTrain, conn.ModeBus}},
	{"busbud", "bus aggregator; ToS restricts scraping", "https://www.busbud.com/en/bus-{from}-{to}", []conn.TransportMode{conn.ModeBus}},
}

// BuiltinSources returns the shipped source list wired to the given renderers.
func BuiltinSources(httpR, headless Renderer) []PriceSource {
	out := make([]PriceSource, 0, len(builtinEntries))
	for _, e := range builtinEntries {
		tmpl := e.tmpl
		src := NewHTTPSource(e.name, e.tos, func(q LegQuery) string { return buildURL(tmpl, q) }, httpR, headless).
			WithModes(e.modes...)
		out = append(out, src)
	}
	return out
}

// DefaultSources returns the built-in list plus any PRICE_SOURCES overrides.
func DefaultSources(envSpec string, httpR, headless Renderer) []PriceSource {
	sources := BuiltinSources(httpR, headless)
	sources = append(sources, SourcesFromSpec(envSpec, httpR, headless)...)
	return sources
}
