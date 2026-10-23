package planner

import "testing"

func sum(xs []int) int {
	t := 0
	for _, x := range xs {
		t += x
	}
	return t
}

func TestAllocateNights_Even(t *testing.T) {
	got := AllocateNights(6, 3, 1, nil)
	if sum(got) != 6 {
		t.Fatalf("sum=%d want 6 (%v)", sum(got), got)
	}
	for _, v := range got {
		if v != 2 {
			t.Fatalf("expected even [2 2 2], got %v", got)
		}
	}
}

func TestAllocateNights_RemainderDeterministic(t *testing.T) {
	got := AllocateNights(7, 3, 1, nil)
	if sum(got) != 7 {
		t.Fatalf("sum=%d want 7 (%v)", sum(got), got)
	}
	// largest-remainder ties break to the front: [3 2 2]
	if got[0] != 3 || got[1] != 2 || got[2] != 2 {
		t.Fatalf("expected [3 2 2], got %v", got)
	}
}

func TestAllocateNights_Weighted(t *testing.T) {
	got := AllocateNights(10, 2, 1, []float64{3, 1})
	if sum(got) != 10 || got[0] != 7 || got[1] != 3 {
		t.Fatalf("expected [7 3], got %v", got)
	}
}

func TestAllocateNights_Insufficient(t *testing.T) {
	got := AllocateNights(1, 3, 1, nil)
	if sum(got) != 1 || got[0] != 1 || got[1] != 0 || got[2] != 0 {
		t.Fatalf("expected [1 0 0], got %v", got)
	}
}
