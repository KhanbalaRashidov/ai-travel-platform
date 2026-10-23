package planner

import "testing"

func TestCitySetKey_OrderIndependent(t *testing.T) {
	if CitySetKey([]string{"A", "B", "C"}) != CitySetKey([]string{"C", "A", "B"}) {
		t.Fatal("key must be order-independent")
	}
}

// Refinement #2: the solver runs only when the city SET changes. Repeated calls
// with the same key (as happens on tier/mode toggles) must hit the cache.
func TestSequencer_RecomputesOnlyOnSetChange(t *testing.T) {
	s := NewSequencer()
	key4 := CitySetKey([]string{"A", "B", "C", "D"})

	if _, _, err := s.Sequence(key4, square, 0, -1); err != nil {
		t.Fatal(err)
	}
	// same set again (simulating tier/mode toggles) -> cache hit, no recompute
	if _, _, err := s.Sequence(key4, square, 0, -1); err != nil {
		t.Fatal(err)
	}
	if s.Computes() != 1 {
		t.Fatalf("expected 1 compute for same set, got %d", s.Computes())
	}

	// different set -> recompute
	key3 := CitySetKey([]string{"X", "Y", "Z"})
	if _, _, err := s.Sequence(key3, asym, 0, -1); err != nil {
		t.Fatal(err)
	}
	if s.Computes() != 2 {
		t.Fatalf("expected 2 computes after set change, got %d", s.Computes())
	}

	// invalidation forces recompute for the same set
	s.Invalidate(key4)
	if _, _, err := s.Sequence(key4, square, 0, -1); err != nil {
		t.Fatal(err)
	}
	if s.Computes() != 3 {
		t.Fatalf("expected 3 computes after invalidate, got %d", s.Computes())
	}
}
