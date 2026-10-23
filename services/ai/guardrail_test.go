package ai

import "testing"

func TestValidate_Grounded(t *testing.T) {
	v := Validate("Flights €120 and hotel 40 EUR.", []float64{120, 40})
	if len(v) != 0 {
		t.Fatalf("expected no violations, got %+v", v)
	}
}

func TestValidate_FabricatedPrice(t *testing.T) {
	v := Validate("Your total is 550 EUR.", []float64{120, 40})
	if len(v) != 1 || v[0].Value != 550 {
		t.Fatalf("expected one violation of 550, got %+v", v)
	}
}

func TestValidate_WithinTolerance(t *testing.T) {
	if v := Validate("Total €120.00", []float64{120}); len(v) != 0 {
		t.Fatalf("120.00 should match grounded 120, got %+v", v)
	}
}
