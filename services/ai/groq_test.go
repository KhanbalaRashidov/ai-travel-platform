package ai

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestBuildGroqRequest_RolesToolsAndJSONMode(t *testing.T) {
	msgs := []Message{
		{Role: RoleSystem, Content: "sys"},
		{Role: RoleUser, Content: "hello"},
		{Role: RoleAssistant, ToolCalls: []ToolCall{{ID: "c1", Name: "quote", Args: map[string]any{"from": "Baku"}}}},
		{Role: RoleTool, ToolCallID: "c1", ToolName: "quote", Content: "120 EUR"},
	}
	req := buildGroqRequest("llama", 256, msgs, []ToolSpec{{Name: "quote", Description: "d"}}, true)

	if req.Model != "llama" || req.MaxTokens != 256 {
		t.Fatalf("model/max wrong: %+v", req)
	}
	if req.ResponseFormat == nil || req.ResponseFormat.Type != "json_object" {
		t.Fatalf("json mode must set response_format: %+v", req.ResponseFormat)
	}
	if len(req.Messages) != 4 {
		t.Fatalf("expected 4 messages, got %d", len(req.Messages))
	}
	if req.Messages[0].Role != "system" || req.Messages[1].Role != "user" {
		t.Fatalf("role mapping wrong: %+v", req.Messages[:2])
	}
	a := req.Messages[2]
	if a.Role != "assistant" || len(a.ToolCalls) != 1 || a.ToolCalls[0].Function.Name != "quote" {
		t.Fatalf("assistant tool_calls wrong: %+v", a)
	}
	if !strings.Contains(a.ToolCalls[0].Function.Arguments, "Baku") {
		t.Fatalf("tool args should be JSON-encoded: %q", a.ToolCalls[0].Function.Arguments)
	}
	tool := req.Messages[3]
	if tool.Role != "tool" || tool.ToolCallID != "c1" || tool.Content != "120 EUR" {
		t.Fatalf("tool message wrong: %+v", tool)
	}
	if len(req.Tools) != 1 || req.Tools[0].Type != "function" || req.Tools[0].Function.Name != "quote" {
		t.Fatalf("tool decl wrong: %+v", req.Tools)
	}
}

func TestParseGroqResponse_TextAndToolCalls(t *testing.T) {
	body := `{
      "choices":[{"message":{"role":"assistant","content":"Hello.",
        "tool_calls":[{"id":"t1","type":"function","function":{"name":"sequence","arguments":"{\"cities\":[\"Baku\"]}"}}]}}],
      "usage":{"prompt_tokens":500,"completion_tokens":120}
    }`
	comp, r, err := parseGroqResponse([]byte(body))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if comp.Content != "Hello." {
		t.Fatalf("content: %q", comp.Content)
	}
	if len(comp.ToolCalls) != 1 || comp.ToolCalls[0].Name != "sequence" {
		t.Fatalf("tool_calls: %+v", comp.ToolCalls)
	}
	if comp.ToolCalls[0].Args["cities"] == nil {
		t.Fatalf("arguments JSON not parsed: %+v", comp.ToolCalls[0].Args)
	}
	if r.Usage.PromptTokens != 500 || r.Usage.CompletionTokens != 120 {
		t.Fatalf("usage: %+v", r.Usage)
	}
}

func TestParseGroqResponse_APIError(t *testing.T) {
	if _, _, err := parseGroqResponse([]byte(`{"error":{"message":"bad key","type":"invalid_request_error"}}`)); err == nil {
		t.Fatal("expected error for error payload")
	}
}

func fastGroq(doer HTTPDoer) *GroqLLM {
	g := NewGroq("k", "llama", doer)
	g.delays = []time.Duration{0, 0}
	g.sleep = func(time.Duration) {}
	return g
}

const groqOK = `{"choices":[{"message":{"role":"assistant","content":"ok"}}],"usage":{"prompt_tokens":1,"completion_tokens":1}}`

func TestGroq_RetriesTransientThenSucceeds(t *testing.T) {
	doer := &scriptDoer{fn: func(i int, _ string) (int, string) {
		if i == 0 {
			return 503, `{"error":{"message":"over capacity","type":"service_unavailable"}}`
		}
		return 200, groqOK
	}}
	comp, err := fastGroq(doer).Complete(context.Background(), []Message{{Role: RoleUser, Content: "hi"}}, nil)
	if err != nil {
		t.Fatalf("should recover: %v", err)
	}
	if comp.Content != "ok" || doer.calls != 2 {
		t.Fatalf("content=%q calls=%d", comp.Content, doer.calls)
	}
}

func TestGroq_NoRetryOn4xx(t *testing.T) {
	doer := &scriptDoer{fn: func(_ int, _ string) (int, string) {
		return 401, `{"error":{"message":"invalid api key","type":"invalid_request_error"}}`
	}}
	_, err := fastGroq(doer).Complete(context.Background(), []Message{{Role: RoleUser, Content: "hi"}}, nil)
	if err == nil || IsTransient(err) {
		t.Fatalf("401 must be a non-transient error, got %v", err)
	}
	if doer.calls != 1 {
		t.Fatalf("4xx must not retry, got %d calls", doer.calls)
	}
}

func TestGroq_Authorization(t *testing.T) {
	var gotAuth string
	doer := doerFunc(func(req *http.Request) (*http.Response, error) {
		gotAuth = req.Header.Get("Authorization")
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(groqOK)), Header: make(http.Header)}, nil
	})
	g := NewGroq("secret-123", "llama", doer)
	g.sleep = func(time.Duration) {}
	if _, err := g.Complete(context.Background(), []Message{{Role: RoleUser, Content: "hi"}}, nil); err != nil {
		t.Fatal(err)
	}
	if gotAuth != "Bearer secret-123" {
		t.Fatalf("Authorization header wrong: %q", gotAuth)
	}
}

// doerFunc adapts a func to HTTPDoer.
type doerFunc func(*http.Request) (*http.Response, error)

func (f doerFunc) Do(req *http.Request) (*http.Response, error) { return f(req) }
