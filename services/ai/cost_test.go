package ai

import "testing"

func TestRouteModel(t *testing.T) {
	if RouteModel(TaskBulk).Name != ModelBulk.Name {
		t.Fatal("bulk should route to the cheap model")
	}
	if RouteModel(TaskReasoning).Name != ModelPrimary.Name {
		t.Fatal("reasoning should route to the primary model")
	}
}

func TestEstimateCostUSD(t *testing.T) {
	// 6000 in + 1500 out on primary (2/10 per 1M) = 0.012 + 0.015 = 0.027
	got := EstimateCostUSD(ModelPrimary, 6000, 1500)
	if got != 0.027 {
		t.Fatalf("expected 0.027, got %v", got)
	}
}

func TestBudget_EnforcesCap(t *testing.T) {
	b := NewBudget(0.05)
	if !b.Add(0.027) {
		t.Fatal("first charge should fit")
	}
	if b.Add(0.03) { // 0.057 > 0.05 -> rejected
		t.Fatal("over-cap charge must be rejected")
	}
	if b.SpentUSD != 0.027 {
		t.Fatalf("spent should stay 0.027 after rejection, got %v", b.SpentUSD)
	}
	if r := b.Remaining(); r < 0.022-1e-9 || r > 0.023+1e-9 {
		t.Fatalf("remaining ~0.023, got %v", r)
	}
}

func TestBudget_Unlimited(t *testing.T) {
	b := NewBudget(0)
	if !b.Add(1000) || b.Remaining() == 0 {
		t.Fatal("unlimited budget should accept any charge")
	}
}
