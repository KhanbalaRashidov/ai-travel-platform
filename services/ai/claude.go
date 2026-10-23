package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// claudeBaseURL is the Anthropic Messages API endpoint.
const claudeBaseURL = "https://api.anthropic.com/v1/messages"

// anthropicVersion is the required API version header.
const anthropicVersion = "2023-06-01"

// defaultMaxTokens caps a single completion's output.
const defaultMaxTokens = 1024

// HTTPDoer is the subset of *http.Client used here, so tests inject a fake.
type HTTPDoer interface {
	Do(req *http.Request) (*http.Response, error)
}

// Usage is the token accounting returned by a completion.
type Usage struct {
	InputTokens  int
	OutputTokens int
	CostUSD      float64
}

// ClaudeLLM is a real LLM backed by the Anthropic Messages API. The API key is
// supplied at construction (read from the environment by the caller) and never
// logged. Cost-aware: each call is priced from its token usage, optionally
// metered and charged against a fail-closed Budget.
type ClaudeLLM struct {
	apiKey    string
	route     ModelRoute
	maxTokens int
	baseURL   string
	http      HTTPDoer
	budget    *Budget
	meter     func(model string, u Usage)
}

// NewClaude builds a Claude adapter for one model route. Pass nil doer for a
// default HTTP client.
func NewClaude(apiKey string, route ModelRoute, doer HTTPDoer) *ClaudeLLM {
	if doer == nil {
		doer = &http.Client{Timeout: 60 * time.Second}
	}
	return &ClaudeLLM{
		apiKey:    apiKey,
		route:     route,
		maxTokens: defaultMaxTokens,
		baseURL:   claudeBaseURL,
		http:      doer,
	}
}

// WithBudget attaches a shared budget; calls that would exceed it fail closed.
func (c *ClaudeLLM) WithBudget(b *Budget) *ClaudeLLM { c.budget = b; return c }

// WithMeter attaches a usage callback for token/cost logging.
func (c *ClaudeLLM) WithMeter(f func(model string, u Usage)) *ClaudeLLM { c.meter = f; return c }

// --- request/response shapes ------------------------------------------------

type claudeBlock struct {
	Type string `json:"type"`
	// text
	Text string `json:"text,omitempty"`
	// tool_use
	ID    string         `json:"id,omitempty"`
	Name  string         `json:"name,omitempty"`
	Input map[string]any `json:"input,omitempty"`
	// tool_result
	ToolUseID string `json:"tool_use_id,omitempty"`
	Content   string `json:"content,omitempty"`
}

type claudeMsg struct {
	Role    string        `json:"role"`
	Content []claudeBlock `json:"content"`
}

type claudeTool struct {
	Name        string         `json:"name"`
	Description string         `json:"description,omitempty"`
	InputSchema map[string]any `json:"input_schema"`
}

type claudeReq struct {
	Model     string       `json:"model"`
	MaxTokens int          `json:"max_tokens"`
	System    string       `json:"system,omitempty"`
	Messages  []claudeMsg  `json:"messages"`
	Tools     []claudeTool `json:"tools,omitempty"`
}

type claudeResp struct {
	Content []claudeBlock `json:"content"`
	Usage   struct {
		InputTokens  int `json:"input_tokens"`
		OutputTokens int `json:"output_tokens"`
	} `json:"usage"`
	Error *struct {
		Type    string `json:"type"`
		Message string `json:"message"`
	} `json:"error"`
}

// buildRequest translates the provider-agnostic messages/tools into an Anthropic
// request. System messages are hoisted to the top-level `system` field, and
// consecutive same-role turns are merged into one message with multiple content
// blocks (the API expects alternating user/assistant turns). Pure, so it is
// unit-tested without the network.
func buildRequest(model string, maxTokens int, msgs []Message, tools []ToolSpec) claudeReq {
	req := claudeReq{Model: model, MaxTokens: maxTokens}

	var systemParts []string
	for _, m := range msgs {
		if m.Role == RoleSystem {
			if m.Content != "" {
				systemParts = append(systemParts, m.Content)
			}
			continue
		}
		role, block := mapTurn(m)
		// Merge into the previous message when the mapped role matches.
		if n := len(req.Messages); n > 0 && req.Messages[n-1].Role == role {
			req.Messages[n-1].Content = append(req.Messages[n-1].Content, block)
		} else {
			req.Messages = append(req.Messages, claudeMsg{Role: role, Content: []claudeBlock{block}})
		}
	}
	req.System = strings.Join(systemParts, "\n\n")

	for _, t := range tools {
		req.Tools = append(req.Tools, claudeTool{
			Name:        t.Name,
			Description: t.Description,
			InputSchema: map[string]any{"type": "object", "properties": map[string]any{}},
		})
	}
	return req
}

// mapTurn maps one provider-agnostic message to an Anthropic role + content block.
func mapTurn(m Message) (string, claudeBlock) {
	switch {
	case m.Role == RoleTool:
		return "user", claudeBlock{Type: "tool_result", ToolUseID: m.ToolCallID, Content: m.Content}
	case m.Role == RoleAssistant && len(m.ToolCalls) > 0:
		// Note: only the first tool call is represented per block; the orchestrator
		// emits one assistant message per tool-call batch, which is the common case.
		tc := m.ToolCalls[0]
		return "assistant", claudeBlock{Type: "tool_use", ID: tc.ID, Name: tc.Name, Input: tc.Args}
	case m.Role == RoleAssistant:
		return "assistant", claudeBlock{Type: "text", Text: m.Content}
	default:
		return "user", claudeBlock{Type: "text", Text: m.Content}
	}
}

// parseResponse turns an Anthropic response body into a Completion + Usage. Pure.
func parseResponse(body []byte) (Completion, claudeResp, error) {
	var r claudeResp
	if err := json.Unmarshal(body, &r); err != nil {
		return Completion{}, r, fmt.Errorf("decode claude response: %w", err)
	}
	if r.Error != nil {
		return Completion{}, r, fmt.Errorf("claude error: %s: %s", r.Error.Type, r.Error.Message)
	}
	var comp Completion
	var text strings.Builder
	for _, b := range r.Content {
		switch b.Type {
		case "text":
			text.WriteString(b.Text)
		case "tool_use":
			comp.ToolCalls = append(comp.ToolCalls, ToolCall{ID: b.ID, Name: b.Name, Args: b.Input})
		}
	}
	comp.Content = text.String()
	return comp, r, nil
}

// Complete calls the Anthropic Messages API.
func (c *ClaudeLLM) Complete(ctx context.Context, msgs []Message, tools []ToolSpec) (Completion, error) {
	if c.budget != nil && c.budget.Remaining() <= 0 {
		return Completion{}, fmt.Errorf("ai budget exhausted (spent $%.4f of $%.2f)", c.budget.SpentUSD, c.budget.LimitUSD)
	}

	payload, err := json.Marshal(buildRequest(c.route.Name, MaxTokensFrom(ctx, c.maxTokens), msgs, tools))
	if err != nil {
		return Completion{}, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL, bytes.NewReader(payload))
	if err != nil {
		return Completion{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", c.apiKey)
	req.Header.Set("anthropic-version", anthropicVersion)

	resp, err := c.http.Do(req)
	if err != nil {
		return Completion{}, err
	}
	defer func() { _ = resp.Body.Close() }()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return Completion{}, err
	}
	if resp.StatusCode != http.StatusOK {
		// Surface the API's error message (body already read).
		comp, _, perr := parseResponse(raw)
		if perr != nil {
			return Completion{}, fmt.Errorf("claude http %d: %s", resp.StatusCode, truncate(string(raw), 300))
		}
		return comp, fmt.Errorf("claude http %d", resp.StatusCode)
	}

	comp, r, err := parseResponse(raw)
	if err != nil {
		return Completion{}, err
	}

	u := Usage{
		InputTokens:  r.Usage.InputTokens,
		OutputTokens: r.Usage.OutputTokens,
		CostUSD:      EstimateCostUSD(c.route, r.Usage.InputTokens, r.Usage.OutputTokens),
	}
	if c.budget != nil {
		c.budget.Add(u.CostUSD)
	}
	if c.meter != nil {
		c.meter(c.route.Name, u)
	}
	return comp, nil
}

func truncate(s string, n int) string {
	if len(s) > n {
		return s[:n]
	}
	return s
}

var _ LLM = (*ClaudeLLM)(nil)
