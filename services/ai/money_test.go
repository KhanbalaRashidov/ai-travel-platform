package ai

import "testing"

func toSet(vs []float64) map[float64]bool {
	m := make(map[float64]bool, len(vs))
	for _, v := range vs {
		m[v] = true
	}
	return m
}

func TestExtractMoneyValues(t *testing.T) {
	got := toSet(ExtractMoneyValues("flight 120 EUR, hotel €40, fee 1,84 AZN, stay 3 days, taxi 15.50₼"))
	for _, want := range []float64{120, 40, 1.84, 15.5} {
		if !got[want] {
			t.Fatalf("expected %v to be extracted, got %v", want, got)
		}
	}
	if got[3] {
		t.Fatal("plain count '3 days' must NOT be treated as money")
	}
}
