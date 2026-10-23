package ai

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

const okBody = `{"candidates":[{"content":{"parts":[{"text":"ok"}]}}],"usageMetadata":{"promptTokenCount":1,"candidatesTokenCount":1}}`

// scriptDoer returns a scripted (status, body) per call index + URL.
type scriptDoer struct {
	calls int
	fn    func(i int, url string) (int, string)
}

func (d *scriptDoer) Do(req *http.Request) (*http.Response, error) {
	code, body := d.fn(d.calls, req.URL.String())
	d.calls++
	return &http.Response{
		StatusCode: code,
		Body:       io.NopCloser(strings.NewReader(body)),
		Header:     make(http.Header),
	}, nil
}

func fastGemini(doer HTTPDoer) *GeminiLLM {
	g := NewGemini("k", "gemini-2.0-flash", doer)
	g.delays = []time.Duration{0, 0} // 3 attempts, no real waiting
	g.sleep = func(time.Duration) {}
	return g
}

func TestGemini_RetriesTransientThenSucceeds(t *testing.T) {
	doer := &scriptDoer{fn: func(i int, _ string) (int, string) {
		if i == 0 {
			return 503, `{"error":{"code":503,"status":"UNAVAILABLE","message":"high demand"}}`
		}
		return 200, okBody
	}}
	g := fastGemini(doer)
	comp, err := g.Complete(context.Background(), []Message{{Role: RoleUser, Content: "hi"}}, nil)
	if err != nil {
		t.Fatalf("should recover after a transient 503: %v", err)
	}
	if comp.Content != "ok" {
		t.Fatalf("unexpected content %q", comp.Content)
	}
	if doer.calls != 2 {
		t.Fatalf("expected 2 calls (1 retry), got %d", doer.calls)
	}
}

func TestGemini_NoRetryOn4xx(t *testing.T) {
	doer := &scriptDoer{fn: func(_ int, _ string) (int, string) {
		return 400, `{"error":{"code":400,"status":"INVALID_ARGUMENT","message":"bad"}}`
	}}
	g := fastGemini(doer)
	_, err := g.Complete(context.Background(), []Message{{Role: RoleUser, Content: "hi"}}, nil)
	if err == nil {
		t.Fatal("expected an error")
	}
	if IsTransient(err) {
		t.Fatal("400 must not be transient")
	}
	if doer.calls != 1 {
		t.Fatalf("4xx must not be retried, got %d calls", doer.calls)
	}
}

func TestGemini_AllTransientReturnsTransient(t *testing.T) {
	doer := &scriptDoer{fn: func(_ int, _ string) (int, string) {
		return 503, `{"error":{"code":503,"status":"UNAVAILABLE","message":"high demand"}}`
	}}
	g := fastGemini(doer)
	_, err := g.Complete(context.Background(), []Message{{Role: RoleUser, Content: "hi"}}, nil)
	if !IsTransient(err) {
		t.Fatalf("exhausted retries should surface a transient error, got %v", err)
	}
	if doer.calls != 3 {
		t.Fatalf("expected 3 attempts, got %d", doer.calls)
	}
}

func TestGemini_FallbackModelOnOverload(t *testing.T) {
	doer := &scriptDoer{fn: func(_ int, url string) (int, string) {
		if strings.Contains(url, "gemini-fallback") {
			return 200, okBody
		}
		return 503, `{"error":{"code":503,"status":"UNAVAILABLE","message":"high demand"}}`
	}}
	g := fastGemini(doer).WithFallback("gemini-fallback")
	comp, err := g.Complete(context.Background(), []Message{{Role: RoleUser, Content: "hi"}}, nil)
	if err != nil {
		t.Fatalf("fallback should succeed: %v", err)
	}
	if comp.Content != "ok" {
		t.Fatalf("unexpected content %q", comp.Content)
	}
	// 3 primary attempts (all 503) + 1 successful fallback call.
	if doer.calls != 4 {
		t.Fatalf("expected 4 calls, got %d", doer.calls)
	}
}
