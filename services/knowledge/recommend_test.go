package knowledge

import (
	"context"
	"testing"
)

func TestMemStore_PopularityRanking(t *testing.T) {
	ctx := context.Background()
	s := NewMemStore()
	for i := 0; i < 3; i++ {
		_ = s.RecordVisit(ctx, "t1", "Baku", "city")
	}
	_ = s.RecordVisit(ctx, "t1", "Istanbul", "city")

	top, _ := s.TopDestinations(ctx, "t1", 10)
	if len(top) != 2 || top[0].Destination != "Baku" || top[0].Hits != 3 {
		t.Fatalf("expected Baku(3) first, got %+v", top)
	}
	if top[1].Destination != "Istanbul" || top[1].Hits != 1 {
		t.Fatalf("expected Istanbul(1) second, got %+v", top)
	}
	if limited, _ := s.TopDestinations(ctx, "t1", 1); len(limited) != 1 {
		t.Fatalf("limit not applied: %+v", limited)
	}
}

func TestService_LearnAndSuggest(t *testing.T) {
	ctx := context.Background()
	svc := NewService(NewMemStore())

	_ = svc.Learn(ctx, "t1", []Destination{{Country: "AZ", City: "Baku"}, {Country: "TR", City: "Istanbul"}})
	_ = svc.Learn(ctx, "t1", []Destination{{Country: "AZ", City: "Baku"}, {Country: "GE", City: "Tbilisi"}})

	recs, err := svc.Suggest(ctx, "t1", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(recs) != 3 || recs[0].Destination != "Baku" || recs[0].Hits != 2 {
		t.Fatalf("expected Baku(2) top of 3 recs, got %+v", recs)
	}
	if recs[0].Reason == "" {
		t.Fatal("recommendation should carry a reason")
	}
	if limited, _ := svc.Suggest(ctx, "t1", 2); len(limited) != 2 {
		t.Fatalf("limit not applied: %+v", limited)
	}
}
