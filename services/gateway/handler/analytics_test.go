package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v2"
)

func analyticsApp() *fiber.App {
	h := NewAnalyticsHandler()
	app := fiber.New()
	app.Post("/analytics/event", h.Event)
	app.Get("/analytics/stats", h.Stats)
	return app
}

func postEvent(app *fiber.App, body string) int {
	req := httptest.NewRequest(http.MethodPost, "/analytics/event", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := app.Test(req, -1)
	return resp.StatusCode
}

func TestAnalytics_AllowlistAndCount(t *testing.T) {
	app := analyticsApp()
	if postEvent(app, `{"name":"plan_computed"}`) != http.StatusOK {
		t.Fatal("allowlisted event should be accepted")
	}
	postEvent(app, `{"name":"plan_computed"}`)
	if postEvent(app, `{"name":"evil; DROP TABLE"}`) != http.StatusBadRequest {
		t.Fatal("non-allowlisted event must be rejected (no PII/free-form)")
	}

	resp, _ := app.Test(httptest.NewRequest(http.MethodGet, "/analytics/stats", nil), -1)
	var out struct {
		Data map[string]int64 `json:"data"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&out)
	if out.Data["plan_computed"] != 2 {
		t.Fatalf("expected plan_computed=2, got %d", out.Data["plan_computed"])
	}
	if _, ok := out.Data["evil; DROP TABLE"]; ok {
		t.Fatal("rejected event must not be recorded")
	}
}
