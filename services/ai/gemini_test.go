package ai

import "testing"

func TestBuildGeminiRequest_SystemRolesAndTools(t *testing.T) {
	msgs := []Message{
		{Role: RoleSystem, Content: "sys A"},
		{Role: RoleUser, Content: "hello"},
		{Role: RoleAssistant, ToolCalls: []ToolCall{{ID: "t1", Name: "quote", Args: map[string]any{"from": "Baku"}}}},
		{Role: RoleTool, ToolCallID: "t1", ToolName: "quote", Content: "120 EUR"},
	}
	req := buildGeminiRequest(256, msgs, []ToolSpec{{Name: "quote", Description: "d"}}, false)

	if req.SystemInstruction == nil || req.SystemInstruction.Parts[0].Text != "sys A" {
		t.Fatalf("system not mapped to systemInstruction: %+v", req.SystemInstruction)
	}
	if req.GenerationConfig.MaxOutputTokens != 256 {
		t.Fatalf("maxOutputTokens wrong: %d", req.GenerationConfig.MaxOutputTokens)
	}
	// user "hello" / model functionCall / user functionResponse -> 3 contents.
	if len(req.Contents) != 3 {
		t.Fatalf("expected 3 contents, got %d: %+v", len(req.Contents), req.Contents)
	}
	if req.Contents[0].Role != "user" || req.Contents[0].Parts[0].Text != "hello" {
		t.Fatalf("content0 wrong: %+v", req.Contents[0])
	}
	if req.Contents[1].Role != "model" || req.Contents[1].Parts[0].FunctionCall == nil {
		t.Fatalf("content1 should be a functionCall (model): %+v", req.Contents[1])
	}
	if req.Contents[1].Parts[0].FunctionCall.Name != "quote" {
		t.Fatalf("functionCall name: %+v", req.Contents[1].Parts[0].FunctionCall)
	}
	fr := req.Contents[2].Parts[0].FunctionResp
	if req.Contents[2].Role != "user" || fr == nil || fr.Name != "quote" {
		t.Fatalf("content2 should be a functionResponse named quote: %+v", req.Contents[2])
	}
	if fr.Response["result"] != "120 EUR" {
		t.Fatalf("functionResponse payload: %+v", fr.Response)
	}
	if len(req.Tools) != 1 || len(req.Tools[0].FunctionDeclarations) != 1 || req.Tools[0].FunctionDeclarations[0].Name != "quote" {
		t.Fatalf("tool declaration not mapped: %+v", req.Tools)
	}
	if req.Tools[0].FunctionDeclarations[0].Parameters["type"] != "object" {
		t.Fatalf("function parameters must be an object schema: %+v", req.Tools[0].FunctionDeclarations[0].Parameters)
	}
}

func TestParseGeminiResponse_TextAndFunctionCall(t *testing.T) {
	body := `{
      "candidates": [
        {"content": {"role":"model","parts": [
          {"text": "Here. "},
          {"functionCall": {"name": "sequence", "args": {"cities": ["Baku","Tbilisi"]}}},
          {"text": "Done."}
        ]}}
      ],
      "usageMetadata": {"promptTokenCount": 800, "candidatesTokenCount": 200}
    }`
	comp, r, err := parseGeminiResponse([]byte(body))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if comp.Content != "Here. Done." {
		t.Fatalf("text parts not concatenated: %q", comp.Content)
	}
	if len(comp.ToolCalls) != 1 || comp.ToolCalls[0].Name != "sequence" {
		t.Fatalf("functionCall not parsed: %+v", comp.ToolCalls)
	}
	if r.UsageMetadata.PromptTokenCount != 800 || r.UsageMetadata.CandidatesTokenCount != 200 {
		t.Fatalf("usage not parsed: %+v", r.UsageMetadata)
	}
}

func TestParseGeminiResponse_APIError(t *testing.T) {
	body := `{"error":{"code":400,"message":"bad key","status":"INVALID_ARGUMENT"}}`
	if _, _, err := parseGeminiResponse([]byte(body)); err == nil {
		t.Fatal("expected an error for an API error payload")
	}
}

func TestBuildGeminiRequest_JSONMode(t *testing.T) {
	plain := buildGeminiRequest(100, []Message{{Role: RoleUser, Content: "hi"}}, nil, false)
	if plain.GenerationConfig.ResponseMIMEType != "" {
		t.Fatalf("plain mode should not set responseMimeType, got %q", plain.GenerationConfig.ResponseMIMEType)
	}
	j := buildGeminiRequest(100, []Message{{Role: RoleUser, Content: "hi"}}, nil, true)
	if j.GenerationConfig.ResponseMIMEType != "application/json" {
		t.Fatalf("json mode must set responseMimeType, got %q", j.GenerationConfig.ResponseMIMEType)
	}
}

func TestExtractJSONObject(t *testing.T) {
	cases := map[string]string{
		"```json\n{\"a\":1}\n```":    `{"a":1}`,
		"here you go: {\"a\":1} ok":  `{"a":1}`,
		"{\"a\":1}":                  `{"a":1}`,
		"prefix {\"a\":{\"b\":2}} s": `{"a":{"b":2}}`,
	}
	for in, want := range cases {
		if got := extractJSONObject(in); got != want {
			t.Fatalf("extractJSONObject(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestGeminiMask(t *testing.T) {
	g := NewGemini("secret-key-123", "", nil)
	masked := g.mask("oops secret-key-123 leaked")
	if masked != "oops *** leaked" {
		t.Fatalf("key not masked: %q", masked)
	}
}
