package ai

import (
	"context"
	"errors"
)

// Role is a chat message role.
type Role string

const (
	RoleSystem    Role = "system"
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"
	RoleTool      Role = "tool"
)

// ToolCall is the model's request to invoke a tool.
type ToolCall struct {
	ID   string         `json:"id"`
	Name string         `json:"name"`
	Args map[string]any `json:"args"`
}

// Message is one turn in the conversation.
type Message struct {
	Role       Role       `json:"role"`
	Content    string     `json:"content,omitempty"`
	ToolCalls  []ToolCall `json:"tool_calls,omitempty"`
	ToolCallID string     `json:"tool_call_id,omitempty"` // for RoleTool results (Claude)
	ToolName   string     `json:"tool_name,omitempty"`    // for RoleTool results (Gemini functionResponse)
}

// ToolSpec describes a tool to the model.
type ToolSpec struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

// Completion is the model's reply: either tool calls or final content.
type Completion struct {
	Content   string     `json:"content"`
	ToolCalls []ToolCall `json:"tool_calls,omitempty"`
}

// LLM is the provider-agnostic model interface. Real implementations wrap
// Anthropic/OpenAI/Gemini; tests use MockLLM.
type LLM interface {
	Complete(ctx context.Context, msgs []Message, tools []ToolSpec) (Completion, error)
}

// MockLLM replays a scripted sequence of completions (for tests / dev).
type MockLLM struct {
	Script []Completion
	i      int
	Calls  int
}

// Complete returns the next scripted completion.
func (m *MockLLM) Complete(_ context.Context, _ []Message, _ []ToolSpec) (Completion, error) {
	if m.i >= len(m.Script) {
		return Completion{}, errors.New("mock llm: script exhausted")
	}
	c := m.Script[m.i]
	m.i++
	m.Calls++
	return c, nil
}

var _ LLM = (*MockLLM)(nil)
