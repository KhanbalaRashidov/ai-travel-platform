package handler

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v2"

	"github.com/KhanbalaRashidov/ai-travel-platform/pkg/cache"
	"github.com/KhanbalaRashidov/ai-travel-platform/pkg/tenant"
	conn "github.com/KhanbalaRashidov/ai-travel-platform/services/connectors"
	"github.com/KhanbalaRashidov/ai-travel-platform/services/connectors/estimate"
	"github.com/KhanbalaRashidov/ai-travel-platform/services/connectors/places"
	"github.com/KhanbalaRashidov/ai-travel-platform/services/dataflow"
	"github.com/KhanbalaRashidov/ai-travel-platform/services/knowledge"
	"github.com/KhanbalaRashidov/ai-travel-platform/services/planner"
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
	est := estimate.New(geo, 0)
	kv := cache.NewMemKV()
	fw := dataflow.NewRouteFlywheel(kv, store, est, 0)
	poi := fakePOI{pois: []conn.POI{{City: "Baku", Name: "Old City", Category: "attraction"}}}
	conv := planner.StaticConverter{Rates: map[string]float64{"EUR": 1, "USD": 0.92, "AZN": 0.53}}
	placesDoer := placesDoerFunc(func(*http.Request) (*http.Response, error) {
		body := `[{"code":"PAR","name":"Paris","country_name":"France","type":"city","coordinates":{"lat":48.85,"lon":2.35}}]`
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(body)), Header: make(http.Header)}, nil
	})
	h := NewDataHandler(fw, poi, store, kv, conv, geo, places.New(placesDoer))

	app := fiber.New()
	app.Use(tenant.Middleware("t1"))
	app.Get("/data/transport", h.Transport)
	app.Get("/data/destinations", h.Destinations)
	app.Get("/data/poi", h.POIs)
	app.Get("/data/places", h.Places)
	app.Post("/data/plan", h.Plan)
	return app
}

type placesDoerFunc func(*http.Request) (*http.Response, error)

func (f placesDoerFunc) Do(req *http.Request) (*http.Response, error) { return f(req) }

func TestPlacesEndpoint_Autocomplete(t *testing.T) {
	app := dataApp(t)
	out := getJSON(t, app, "/data/places?q=par")
	list, ok := out["data"].([]any)
	if !ok || len(list) != 1 {
		t.Fatalf("expected 1 place, got %v", out["data"])
	}
	first, _ := list[0].(map[string]any)
	if first["name"] != "Paris" || first["code"] != "PAR" {
		t.Fatalf("unexpected place: %v", first)
	}
}

func TestPlanEndpoint_PricesFromFlywheel(t *testing.T) {
	app := dataApp(t)
	body := `{"cities":["Baku","Tbilisi","Istanbul"],"start":"Baku","mode":"cheapest","tier":"standard","nights":2,"currency":"EUR"}`
	req := httptest.NewRequest(http.MethodPost, "/data/plan", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("status %d: %s", resp.StatusCode, string(b))
	}
	var out struct {
		Data struct {
			Order     []string `json:"order"`
			Legs      []map[string]any
			Total     float64 `json:"total"`
			Estimated bool    `json:"estimated"`
			Currency  string  `json:"currency"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatal(err)
	}
	if len(out.Data.Order) != 3 || len(out.Data.Legs) != 2 {
		t.Fatalf("expected 3 stops / 2 legs, got order=%v legs=%d", out.Data.Order, len(out.Data.Legs))
	}
	if out.Data.Total <= 0 || out.Data.Currency != "EUR" {
		t.Fatalf("bad totals: total=%v cur=%s", out.Data.Total, out.Data.Currency)
	}
	if !out.Data.Estimated {
		t.Fatal("estimator-backed plan must be flagged estimated")
	}
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
