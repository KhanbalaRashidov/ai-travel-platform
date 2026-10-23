package ai

import "testing"

func TestBuildRequest_HoistsSystemAndMergesRoles(t *testing.T) {
	msgs := []Message{
		{Role: RoleSystem, Content: "sys A"},
		{Role: RoleUser, Content: "hello"},
		{Role: RoleAssistant, ToolCalls: []ToolCall{{ID: "t1", Name: "plan", Args: map[string]any{"x": 1}}}},
		{Role: RoleTool, ToolCallID: "t1", Content: "result: 120 EUR"},
		{Role: RoleSystem, Content: "sys B"},
	}
	req := buildRequest("claude-x", 512, msgs, []ToolSpec{{Name: "plan", Description: "d"}})

	if req.System != "sys A\n\nsys B" {
		t.Fatalf("system not hoisted/joined: %q", req.System)
	}
	if req.Model != "claude-x" || req.MaxTokens != 512 {
		t.Fatalf("model/maxTokens wrong: %+v", req)
	}
	// user "hello" (assistant tool_use) (user tool_result) -> 3 messages.
	if len(req.Messages) != 3 {
		t.Fatalf("expected 3 messages, got %d: %+v", len(req.Messages), req.Messages)
	}
	if req.Messages[0].Role != "user" || req.Messages[0].Content[0].Type != "text" {
		t.Fatalf("msg0 wrong: %+v", req.Messages[0])
	}
	if req.Messages[1].Role != "assistant" || req.Messages[1].Content[0].Type != "tool_use" {
		t.Fatalf("msg1 should be tool_use: %+v", req.Messages[1])
	}
	if req.Messages[2].Role != "user" || req.Messages[2].Content[0].Type != "tool_result" {
		t.Fatalf("msg2 should be tool_result: %+v", req.Messages[2])
	}
	if req.Messages[2].Content[0].ToolUseID != "t1" {
		t.Fatalf("tool_result must reference tool id: %+v", req.Messages[2].Content[0])
	}
	if len(req.Tools) != 1 || req.Tools[0].Name != "plan" {
		t.Fatalf("tool not mapped: %+v", req.Tools)
	}
	if req.Tools[0].InputSchema["type"] != "object" {
		t.Fatalf("tool input_schema must be an object: %+v", req.Tools[0].InputSchema)
	}
}

func TestBuildRequest_MergesConsecutiveToolResults(t *testing.T) {
	msgs := []Message{
		{Role: RoleTool, ToolCallID: "a", Content: "r1"},
		{Role: RoleTool, ToolCallID: "b", Content: "r2"},
	}
	req := buildRequest("m", 10, msgs, nil)
	if len(req.Messages) != 1 {
		t.Fatalf("consecutive user turns should merge into 1, got %d", len(req.Messages))
	}
	if len(req.Messages[0].Content) != 2 {
		t.Fatalf("merged message should hold 2 blocks, got %d", len(req.Messages[0].Content))
	}
}

func TestParseResponse_TextAndToolUse(t *testing.T) {
	body := `{
      "content": [
        {"type":"text","text":"Here is the plan. "},
        {"type":"tool_use","id":"tu1","name":"price","input":{"city":"Baku"}},
        {"type":"text","text":"Done."}
      ],
      "usage": {"input_tokens": 1200, "output_tokens": 300}
    }`
	comp, r, err := parseResponse([]byte(body))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if comp.Content != "Here is the plan. Done." {
		t.Fatalf("text blocks not concatenated: %q", comp.Content)
	}
	if len(comp.ToolCalls) != 1 || comp.ToolCalls[0].Name != "price" {
		t.Fatalf("tool_use not parsed: %+v", comp.ToolCalls)
	}
	if comp.ToolCalls[0].Args["city"] != "Baku" {
		t.Fatalf("tool input not parsed: %+v", comp.ToolCalls[0].Args)
	}
	if r.Usage.InputTokens != 1200 || r.Usage.OutputTokens != 300 {
		t.Fatalf("usage not parsed: %+v", r.Usage)
	}
}

func TestParseResponse_APIError(t *testing.T) {
	body := `{"error":{"type":"invalid_request_error","message":"bad key"}}`
	if _, _, err := parseResponse([]byte(body)); err == nil {
		t.Fatal("expected an error for an API error payload")
	}
}
