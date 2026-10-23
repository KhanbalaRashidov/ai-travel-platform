package planner

import "testing"

func testConverter() StaticConverter {
	// values in a common numeraire: 1 EUR = 1, 1 USD = 0.9 EUR, 1 AZN = 0.53 EUR
	return StaticConverter{Rates: map[string]float64{"EUR": 1, "USD": 0.9, "AZN": 0.53}}
}

func TestPrice_MultiCurrency(t *testing.T) {
	conv := testConverter()
	in := PlanInput{
		Order: []int{0, 1},
		Legs: []PricedLeg{
			{From: 0, To: 1, Mode: ModeFlight, Price: Money{Amount: 100, Currency: "EUR"}},
			{From: 1, To: 2, Mode: ModeTrain, Price: Money{Amount: 50, Currency: "USD"}}, // 45 EUR
		},
		Hotels:          []PricedHotel{{City: 1, Stars: 3, Nights: 3, Nightly: Money{Amount: 40, Currency: "EUR"}}},
		ActivitiesTotal: Money{Amount: 30, Currency: "EUR"},
		DisplayCurrency: "EUR",
	}
	b, err := Price(in, conv)
	if err != nil {
		t.Fatal(err)
	}
	if b.Transport != 145 || b.Hotels != 120 || b.Activities != 30 || b.Total != 295 {
		t.Fatalf("breakdown = %+v, want transport145 hotels120 act30 total295", b)
	}
}

// Changing the comfort tier swaps only the hotel price; re-pricing must change
// hotels+total while transport stays identical (no re-sequencing).
func TestPrice_TierTogglePricesHotelsOnly(t *testing.T) {
	conv := testConverter()
	base := PlanInput{
		Order:           []int{0},
		Legs:            []PricedLeg{{Price: Money{Amount: 100, Currency: "EUR"}}},
		Hotels:          []PricedHotel{{Stars: 2, Nights: 3, Nightly: Money{Amount: 40, Currency: "EUR"}}},
		ActivitiesTotal: Money{Amount: 0, Currency: "EUR"},
		DisplayCurrency: "EUR",
	}
	budget, err := Price(base, conv)
	if err != nil {
		t.Fatal(err)
	}

	vip := base
	vip.Hotels = []PricedHotel{{Stars: 5, Nights: 3, Nightly: Money{Amount: 90, Currency: "EUR"}}}
	vipB, err := Price(vip, conv)
	if err != nil {
		t.Fatal(err)
	}

	if vipB.Transport != budget.Transport {
		t.Fatalf("transport changed on tier toggle: %v -> %v", budget.Transport, vipB.Transport)
	}
	if vipB.Hotels <= budget.Hotels || vipB.Total <= budget.Total {
		t.Fatalf("tier upgrade must raise hotels+total: budget=%+v vip=%+v", budget, vipB)
	}
}
