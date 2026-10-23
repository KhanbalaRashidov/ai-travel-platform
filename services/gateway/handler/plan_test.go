package handler

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v2"

	"github.com/KhanbalaRashidov/ai-travel-platform/services/gateway/service"
)

func planApp() *fiber.App {
	app := fiber.New()
	h := NewPlanHandler(service.NewPlanService())
	app.Post("/plan/sequence", h.Sequence)
	app.Post("/plan/price", h.Price)
	app.Post("/plan/legs", h.Legs)
	return app
}

func doJSON(t *testing.T, app *fiber.App, path, body string) map[string]any {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
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
	var out map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatal(err)
	}
	return out
}

func TestSequenceEndpoint_OpenJaw(t *testing.T) {
	app := planApp()
	// asym matrix: expensive return -> open-jaw (cost 2) beats round-trip.
	body := `{"matrix":[[0,1,5],[1,0,1],[100,100,0]],"start":0,"end":-1}`
	out := doJSON(t, app, "/plan/sequence", body)

	data, _ := out["data"].(map[string]any)
	best, _ := data["best"].(map[string]any)
	if best["variant"] != "open_jaw" {
		t.Fatalf("expected open_jaw best, got %v", best["variant"])
	}
	if best["cost"].(float64) != 2 {
		t.Fatalf("expected best cost 2, got %v", best["cost"])
	}
}

func TestPriceEndpoint(t *testing.T) {
	app := planApp()
	body := `{
		"order":[0,1],
		"legs":[{"from":0,"to":1,"mode":"flight","price":{"amount":100,"currency":"EUR"}}],
		"hotels":[{"city":1,"stars":3,"nights":2,"nightly":{"amount":40,"currency":"EUR"}}],
		"activities_total":{"amount":0,"currency":"EUR"},
		"display_currency":"EUR",
		"rates":{"EUR":1}
	}`
	out := doJSON(t, app, "/plan/price", body)

	data, _ := out["data"].(map[string]any)
	if data["transport"].(float64) != 100 || data["hotels"].(float64) != 80 || data["total"].(float64) != 180 {
		t.Fatalf("unexpected breakdown: %v", data)
	}
}

func TestLegsEndpoint(t *testing.T) {
	app := planApp()
	body := `{"legs":[{"from":"BAK","to":"TBS","options":[
		{"mode":"flight","price":{"amount":120,"currency":"EUR"},"duration_minutes":90},
		{"mode":"bus","price":{"amount":25,"currency":"EUR"},"duration_minutes":720}
	]}]}`
	out := doJSON(t, app, "/plan/legs", body)

	data, _ := out["data"].([]any)
	if len(data) != 1 {
		t.Fatalf("expected 1 leg, got %v", data)
	}
	leg, _ := data[0].(map[string]any)
	cheapest, _ := leg["cheapest"].(map[string]any)
	fastest, _ := leg["fastest"].(map[string]any)
	if cheapest["mode"] != "bus" {
		t.Fatalf("expected cheapest bus, got %v", cheapest["mode"])
	}
	if fastest["mode"] != "flight" {
		t.Fatalf("expected fastest flight, got %v", fastest["mode"])
	}
}
