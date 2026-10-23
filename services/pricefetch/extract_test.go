package pricefetch

import "testing"

func TestRegexExtract_CurrencyMarkedPrices(t *testing.T) {
	raw := "Flights from €1,234.50 round trip. Also 99 USD one-way. Bus tickets AZN 25. " +
		"Call 1234567 for info. Hotel from £80 per night."
	got := RegexExtract(raw)

	want := map[string]float64{"EUR": 1234.50, "USD": 99, "AZN": 25, "GBP": 80}
	found := map[string]float64{}
	for _, e := range got {
		found[e.Currency] = e.Price
	}
	for cur, amt := range want {
		if found[cur] != amt {
			t.Fatalf("expected %s=%.2f, got %v (all: %+v)", cur, amt, found[cur], got)
		}
	}
	// A bare number with no currency marker (the phone number) must NOT appear.
	for _, e := range got {
		if e.Price == 1234567 {
			t.Fatalf("picked up a non-currency number: %+v", got)
		}
	}
}

func TestRegexExtract_Empty(t *testing.T) {
	if got := RegexExtract("no prices here, just words"); len(got) != 0 {
		t.Fatalf("expected no matches, got %+v", got)
	}
}

func TestParseAmount_Separators(t *testing.T) {
	cases := map[string]float64{
		"1,234.50": 1234.50,
		"1.234,50": 1234.50,
		"1 234":    1234,
		"99":       99,
		"80.00":    80,
	}
	for in, want := range cases {
		if got := parseAmount(in); got != want {
			t.Fatalf("parseAmount(%q) = %v, want %v", in, got, want)
		}
	}
}
