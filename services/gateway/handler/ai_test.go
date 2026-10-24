package handler

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v2"

	"github.com/KhanbalaRashidov/ai-travel-platform/services/ai"
)

func aiApp(primary ai.LLM) *fiber.App {
	app := fiber.New()
	h := NewAIHandler(primary, primary, nil, nil)
	app.Post("/ai/refine", h.Refine)
	app.Post("/ai/chat", h.Chat)
	return app
}

func TestRefine_AppliesEdit(t *testing.T) {
	// LLM parses the message into a structured edit.
	llm := &ai.MockLLM{Script: []ai.Completion{
		{Content: `{"cheaper":true,"remove_cities":["Istanbul"]}`},
	}}
	app := aiApp(llm)

	body := `{"message":"make it cheaper and drop Istanbul","state":{"cities":["Baku","Tbilisi","Istanbul"],"start":"Baku","tier":"vip","mode":"flight","nights":2}}`
	req := httptest.NewRequest(http.MethodPost, "/ai/refine", strings.NewReader(body))
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
			State   ai.EditState `json:"state"`
			Changed []string     `json:"changed"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatal(err)
	}
	st := out.Data.State
	if st.Mode != "cheapest" || st.Tier != "standard" {
		t.Fatalf("cheaper not applied: mode=%s tier=%s", st.Mode, st.Tier)
	}
	for _, c := range st.Cities {
		if strings.EqualFold(c, "Istanbul") {
			t.Fatal("Istanbul should have been removed")
		}
	}
	if len(out.Data.Changed) == 0 {
		t.Fatal("expected change codes")
	}
}

func TestRefine_503WhenNoLLM(t *testing.T) {
	app := aiApp(nil)
	resp, _ := app.Test(httptest.NewRequest(http.MethodPost, "/ai/refine",
		strings.NewReader(`{"message":"cheaper","state":{}}`)), -1)
	if resp.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("expected 503 when AI is not configured, got %d", resp.StatusCode)
	}
}
