package planner

import "testing"

func eur(a float64) Money { return Money{Amount: a, Currency: "EUR"} }

func TestSelectHotel_InBandCheapest(t *testing.T) {
	cands := []HotelCandidate{
		{Name: "b2", Stars: 2, Nightly: eur(40)},
		{Name: "b3", Stars: 3, Nightly: eur(60)},
		{Name: "b4", Stars: 4, Nightly: eur(90)},
		{Name: "b5", Stars: 5, Nightly: eur(150)},
	}
	h, ok := SelectHotel(cands, TierStandard, false) // 3-4 stars
	if !ok || h.Name != "b3" || h.Stars != 3 {
		t.Fatalf("expected in-band cheapest b3, got %+v", h)
	}
}

func TestSelectHotel_TieBreaksCentral(t *testing.T) {
	cands := []HotelCandidate{
		{Name: "far", Stars: 3, Nightly: eur(60), CentralityKm: 5},
		{Name: "near", Stars: 3, Nightly: eur(60), CentralityKm: 1},
	}
	h, _ := SelectHotel(cands, TierStandard, true)
	if h.Name != "near" {
		t.Fatalf("expected central tie-break 'near', got %+v", h)
	}
}

func TestSelectHotel_FallbackWhenNoneInBand(t *testing.T) {
	cands := []HotelCandidate{
		{Name: "b2", Stars: 2, Nightly: eur(40)},
		{Name: "b3", Stars: 3, Nightly: eur(60)},
	}
	// VIP wants 4-5 stars; none in band, none >= 4 -> cheapest overall.
	h, ok := SelectHotel(cands, TierVIP, false)
	if !ok || h.Name != "b2" {
		t.Fatalf("expected fallback cheapest b2, got %+v (ok=%v)", h, ok)
	}
}

func TestSelectHotel_Empty(t *testing.T) {
	if _, ok := SelectHotel(nil, TierBudget, false); ok {
		t.Fatal("expected ok=false for no candidates")
	}
}
