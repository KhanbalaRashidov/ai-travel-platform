package handler

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v2"

	"github.com/KhanbalaRashidov/ai-travel-platform/pkg/tenant"
	"github.com/KhanbalaRashidov/ai-travel-platform/services/knowledge"
)

func recApp() *fiber.App {
	app := fiber.New()
	app.Use(tenant.Middleware("t1")) // fixed tenant for the test
	h := NewRecommendHandler(knowledge.NewService(knowledge.NewMemStore()))
	app.Get("/recommendations", h.List)
	app.Post("/recommendations/visit", h.Visit)
	return app
}

func call(t *testing.T, app *fiber.App, method, path, body string) *http.Response {
	t.Helper()
	var r *http.Request
	if body == "" {
		r = httptest.NewRequest(method, path, nil)
	} else {
		r = httptest.NewRequest(method, path, strings.NewReader(body))
		r.Header.Set("Content-Type", "application/json")
	}
	resp, err := app.Test(r, -1)
	if err != nil {
		t.Fatal(err)
	}
	return resp
}

func TestRecommendations_VisitThenList(t *testing.T) {
	app := recApp()
	for i := 0; i < 3; i++ {
		if resp := call(t, app, http.MethodPost, "/recommendations/visit", `{"destination":"Baku"}`); resp.StatusCode != 200 {
			t.Fatalf("visit status %d", resp.StatusCode)
		}
	}
	_ = call(t, app, http.MethodPost, "/recommendations/visit", `{"destination":"Istanbul"}`)

	resp := call(t, app, http.MethodGet, "/recommendations?limit=5", "")
	if resp.StatusCode != 200 {
		t.Fatalf("list status %d", resp.StatusCode)
	}
	var out struct {
		Data []struct {
			Destination string `json:"destination"`
			Hits        int64  `json:"hits"`
		} `json:"data"`
	}
	b, _ := io.ReadAll(resp.Body)
	if err := json.Unmarshal(b, &out); err != nil {
		t.Fatal(err)
	}
	if len(out.Data) != 2 || out.Data[0].Destination != "Baku" || out.Data[0].Hits != 3 {
		t.Fatalf("expected Baku(3) first, got %s", string(b))
	}
}

func TestRecommendations_VisitValidation(t *testing.T) {
	app := recApp()
	if resp := call(t, app, http.MethodPost, "/recommendations/visit", `{}`); resp.StatusCode != 400 {
		t.Fatalf("expected 400 for missing destination, got %d", resp.StatusCode)
	}
}
