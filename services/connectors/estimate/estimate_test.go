package estimate

import (
	"context"
	"testing"

	conn "github.com/KhanbalaRashidov/ai-travel-platform/services/connectors"
)

func testGeo() conn.Geocoder {
	return conn.NewStaticGeocoder(map[string]conn.LatLng{
		"Baku":     {Lat: 40.4093, Lng: 49.8671},
		"Tbilisi":  {Lat: 41.7151, Lng: 44.8271},
		"Istanbul": {Lat: 41.0082, Lng: 28.9784},
	})
}

func TestTransport_EstimatedAndLabeled(t *testing.T) {
	e := New(testGeo(), 0)
	q, err := e.Transport(context.Background(), "Baku", "Tbilisi", conn.ModeFlight)
	if err != nil {
		t.Fatalf("transport: %v", err)
	}
	if !q.Estimated {
		t.Fatal("estimate must be labeled Estimated=true")
	}
	if q.Source != sourceName {
		t.Fatalf("source = %q", q.Source)
	}
	if q.Price.Amount <= 0 || q.Price.Currency != "EUR" {
		t.Fatalf("bad price %+v", q.Price)
	}
	if q.Duration <= 0 {
		t.Fatalf("duration should be positive, got %v", q.Duration)
	}
	if q.ExpiresAt.Before(q.FetchedAt) {
		t.Fatal("ExpiresAt must be after FetchedAt")
	}
}

func TestTransport_UnknownCity(t *testing.T) {
	e := New(testGeo(), 0)
	if _, err := e.Transport(context.Background(), "Baku", "Atlantis", conn.ModeFlight); err != conn.ErrUnavailable {
		t.Fatalf("expected ErrUnavailable, got %v", err)
	}
}

func TestEstimateLeg_ModeViability(t *testing.T) {
	// Short hop: all modes viable.
	if _, _, ok := estimateLeg(300, conn.ModeBus); !ok {
		t.Fatal("bus should be viable at 300 km")
	}
	// Long hop: bus and train drop out, flight remains.
	if _, _, ok := estimateLeg(2000, conn.ModeBus); ok {
		t.Fatal("bus should not be viable at 2000 km")
	}
	if _, _, ok := estimateLeg(2000, conn.ModeTrain); ok {
		t.Fatal("train should not be viable at 2000 km")
	}
	if _, _, ok := estimateLeg(2000, conn.ModeFlight); !ok {
		t.Fatal("flight should be viable at 2000 km")
	}
}

func TestHotel_TierClampAndLabel(t *testing.T) {
	e := New(testGeo(), 0)
	q, err := e.Hotel(context.Background(), "Baku", 9) // clamped to 5
	if err != nil {
		t.Fatalf("hotel: %v", err)
	}
	if q.Stars != 5 {
		t.Fatalf("stars should clamp to 5, got %d", q.Stars)
	}
	if !q.Estimated {
		t.Fatal("hotel estimate must be labeled Estimated=true")
	}
	if q.Nightly.Amount <= 0 {
		t.Fatalf("nightly should be positive, got %v", q.Nightly.Amount)
	}
}
