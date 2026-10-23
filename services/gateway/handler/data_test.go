package handler

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"

	"github.com/KhanbalaRashidov/ai-travel-platform/pkg/cache"
	"github.com/KhanbalaRashidov/ai-travel-platform/pkg/tenant"
	conn "github.com/KhanbalaRashidov/ai-travel-platform/services/connectors"
	"github.com/KhanbalaRashidov/ai-travel-platform/services/connectors/estimate"
	"github.com/KhanbalaRashidov/ai-travel-platform/services/dataflow"
	"github.com/KhanbalaRashidov/ai-travel-platform/services/knowledge"
)

type fakePOI struct{ pois []conn.POI }

func (f fakePOI) Name() string { return "fake-poi" }
func (f fakePOI) POIs(_ context.Context, city string, limit int) ([]conn.POI, error) {
	if len(f.pois) > limit {
		return f.pois[:limit], nil
	}
	return f.pois, nil
}

func dataApp(t *testing.T) *fiber.App {
	t.Helper()
	store := knowledge.NewMemStore()
	if err := knowledge.Seed(context.Background(), store, "t1"); err != nil {
		t.Fatal(err)
	}
	geo := knowledge.SeedGeocoder()
	fw := dataflow.NewRouteFlywheel(cache.NewMemKV(), store, estimate.New(geo, 0), 0)
	poi := fakePOI{pois: []conn.POI{{City: "Baku", Name: "Old City", Category: "attraction"}}}
	h := NewDataHandler(fw, poi, store)

	app := fiber.New()
	app.Use(tenant.Middleware("t1"))
	app.Get("/data/transport", h.Transport)
	app.Get("/data/destinations", h.Destinations)
	app.Get("/data/poi", h.POIs)
	return app
}

func getJSON(t *testing.T, app *fiber.App, path string) map[string]any {
	t.Helper()
	resp, err := app.Test(httptest.NewRequest(http.MethodGet, path, nil), -1)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("GET %s -> status %d: %s", path, resp.StatusCode, string(b))
	}
	var out map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatal(err)
	}
	return out
}

func TestTransportEndpoint_ColdEstimateThenHot(t *testing.T) {
	app := dataApp(t)

	// First call: cold (estimator), labeled estimated.
	out := getJSON(t, app, "/data/transport?from=Baku&to=Tbilisi&mode=flight")
	data, _ := out["data"].(map[string]any)
	if data["source_tier"] != "cold" {
		t.Fatalf("first lookup should be cold, got %v", data["source_tier"])
	}
	q, _ := data["quote"].(map[string]any)
	prov, _ := q["provenance"].(map[string]any)
	if prov["estimated"] != true {
		t.Fatalf("estimate must be labeled estimated=true, got %v", prov["estimated"])
	}

	// Second identical call: served hot from the flywheel cache.
	out2 := getJSON(t, app, "/data/transport?from=Baku&to=Tbilisi&mode=flight")
	data2, _ := out2["data"].(map[string]any)
	if data2["source_tier"] != "hot" {
		t.Fatalf("second lookup should be hot, got %v", data2["source_tier"])
	}
}

func TestDestinationsEndpoint_SeededDynamicList(t *testing.T) {
	app := dataApp(t)
	out := getJSON(t, app, "/data/destinations")
	list, ok := out["data"].([]any)
	if !ok || len(list) == 0 {
		t.Fatalf("expected a non-empty destination list, got %v", out["data"])
	}
}

func TestPOIEndpoint(t *testing.T) {
	app := dataApp(t)
	out := getJSON(t, app, "/data/poi?city=Baku&limit=5")
	list, ok := out["data"].([]any)
	if !ok || len(list) != 1 {
		t.Fatalf("expected 1 POI, got %v", out["data"])
	}
}

func TestTransportEndpoint_BadMode(t *testing.T) {
	app := dataApp(t)
	resp, _ := app.Test(httptest.NewRequest(http.MethodGet, "/data/transport?from=Baku&to=Tbilisi&mode=boat", nil), -1)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 for bad mode, got %d", resp.StatusCode)
	}
}
