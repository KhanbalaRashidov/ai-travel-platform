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

// groqBaseURL is Groq's OpenAI-compatible chat-completions endpoint.
const groqBaseURL = "https://api.groq.com/openai/v1/chat/completions"

// DefaultGroqModel is a fast, capable free default.
const DefaultGroqModel = "llama-3.3-70b-versatile"

// GroqLLM is a free LLM backed by Groq's OpenAI-compatible API. It satisfies the
// same provider-agnostic LLM interface; the key is supplied at construction and
// never logged. Groq's free tier -> per-call cost ~0.
type GroqLLM struct {
	apiKey    string
	model     string
	maxTokens int
	baseURL   string
	http      HTTPDoer
	budget    *Budget
	meter     func(model string, u Usage)
	delays    []time.Duration
	sleep     func(d time.Duration)
}

// NewGroq builds a Groq adapter. Pass nil doer for a default HTTP client; empty
// model uses DefaultGroqModel.
func NewGroq(apiKey, model string, doer HTTPDoer) *GroqLLM {
	if doer == nil {
		doer = &http.Client{Timeout: 45 * time.Second}
	}
	if model == "" {
		model = DefaultGroqModel
	}
	return &GroqLLM{
		apiKey:    apiKey,
		model:     model,
		maxTokens: defaultMaxTokens,
		baseURL:   groqBaseURL,
		http:      doer,
		delays:    []time.Duration{500 * time.Millisecond, 1500 * time.Millisecond},
		sleep:     time.Sleep,
	}
}

// WithBudget attaches a shared budget.
func (g *GroqLLM) WithBudget(b *Budget) *GroqLLM { g.budget = b; return g }

// WithMeter attaches a usage callback for token logging.
func (g *GroqLLM) WithMeter(f func(model string, u Usage)) *GroqLLM { g.meter = f; return g }

// Model returns the configured model name.
func (g *GroqLLM) Model() string { return g.model }

// --- OpenAI chat shapes -----------------------------------------------------

type oaiToolCall struct {
	ID       string `json:"id,omitempty"`
	Type     string `json:"type,omitempty"`
	Function struct {
		Name      string `json:"name"`
		Arguments string `json:"arguments"` // JSON-encoded args
	} `json:"function"`
}

type oaiMessage struct {
	Role       string        `json:"role"`
	Content    string        `json:"content"`
	ToolCalls  []oaiToolCall `json:"tool_calls,omitempty"`
	ToolCallID string        `json:"tool_call_id,omitempty"`
}

type oaiTool struct {
	Type     string `json:"type"`
	Function struct {
		Name        string         `json:"name"`
		Description string         `json:"description,omitempty"`
		Parameters  map[string]any `json:"parameters"`
	} `json:"function"`
}

type oaiRespFormat struct {
	Type string `json:"type"`
}

type groqReq struct {
	Model          string         `json:"model"`
	Messages       []oaiMessage   `json:"messages"`
	Tools          []oaiTool      `json:"tools,omitempty"`
	MaxTokens      int            `json:"max_tokens,omitempty"`
	ResponseFormat *oaiRespFormat `json:"response_format,omitempty"`
}

type groqResp struct {
	Choices []struct {
		Message struct {
			Content   string        `json:"content"`
			ToolCalls []oaiToolCall `json:"tool_calls"`
		} `json:"message"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
	} `json:"usage"`
	Error *struct {
		Message string `json:"message"`
		Type    string `json:"type"`
	} `json:"error"`
}

// buildGroqRequest maps provider-agnostic messages/tools into an OpenAI chat
// request. Pure, so it is unit-tested without the network.
func buildGroqRequest(model string, maxTokens int, msgs []Message, tools []ToolSpec, jsonMode bool) groqReq {
	req := groqReq{Model: model, MaxTokens: maxTokens}
	if jsonMode {
		req.ResponseFormat = &oaiRespFormat{Type: "json_object"}
	}
	for _, m := range msgs {
		req.Messages = append(req.Messages, mapGroqMessage(m))
	}
	for _, t := range tools {
		var ot oaiTool
		ot.Type = "function"
		ot.Function.Name = t.Name
		ot.Function.Description = t.Description
		ot.Function.Parameters = map[string]any{"type": "object", "properties": map[string]any{}}
		req.Tools = append(req.Tools, ot)
	}
	return req
}

func mapGroqMessage(m Message) oaiMessage {
	switch {
	case m.Role == RoleTool:
		return oaiMessage{Role: "tool", ToolCallID: m.ToolCallID, Content: m.Content}
	case m.Role == RoleAssistant && len(m.ToolCalls) > 0:
		out := oaiMessage{Role: "assistant"}
		for _, tc := range m.ToolCalls {
			var oc oaiToolCall
			oc.ID = tc.ID
			oc.Type = "function"
			oc.Function.Name = tc.Name
			if b, err := json.Marshal(tc.Args); err == nil {
				oc.Function.Arguments = string(b)
			}
			out.ToolCalls = append(out.ToolCalls, oc)
		}
		return out
	case m.Role == RoleAssistant:
		return oaiMessage{Role: "assistant", Content: m.Content}
	case m.Role == RoleSystem:
		return oaiMessage{Role: "system", Content: m.Content}
	default:
		return oaiMessage{Role: "user", Content: m.Content}
	}
}

// parseGroqResponse turns an OpenAI chat response into a Completion + usage. Pure.
func parseGroqResponse(body []byte) (Completion, groqResp, error) {
	var r groqResp
	if err := json.Unmarshal(body, &r); err != nil {
		return Completion{}, r, fmt.Errorf("decode groq response: %w", err)
	}
	if r.Error != nil {
		return Completion{}, r, fmt.Errorf("groq error: %s: %s", r.Error.Type, r.Error.Message)
	}
	var comp Completion
	if len(r.Choices) > 0 {
		msg := r.Choices[0].Message
		comp.Content = msg.Content
		for _, tc := range msg.ToolCalls {
			args := map[string]any{}
			if tc.Function.Arguments != "" {
				_ = json.Unmarshal([]byte(tc.Function.Arguments), &args)
			}
			comp.ToolCalls = append(comp.ToolCalls, ToolCall{ID: tc.ID, Name: tc.Function.Name, Args: args})
		}
	}
	return comp, r, nil
}

// Complete calls Groq's chat-completions endpoint.
func (g *GroqLLM) Complete(ctx context.Context, msgs []Message, tools []ToolSpec) (Completion, error) {
	return g.complete(ctx, msgs, tools, false)
}

// CompleteJSON forces OpenAI JSON mode (response_format=json_object), no tools.
func (g *GroqLLM) CompleteJSON(ctx context.Context, msgs []Message) (Completion, error) {
	return g.complete(ctx, msgs, nil, true)
}

func (g *GroqLLM) complete(ctx context.Context, msgs []Message, tools []ToolSpec, jsonMode bool) (Completion, error) {
	if g.budget != nil && g.budget.Remaining() <= 0 {
		return Completion{}, fmt.Errorf("ai budget exhausted (spent $%.4f of $%.2f)", g.budget.SpentUSD, g.budget.LimitUSD)
	}
	payload, err := json.Marshal(buildGroqRequest(g.model, g.maxTokens, msgs, tools, jsonMode))
	if err != nil {
		return Completion{}, err
	}
	return retryTransient(ctx, g.delays, g.sleep, func() (Completion, error) {
		return g.doOnce(ctx, payload)
	})
}

func (g *GroqLLM) doOnce(ctx context.Context, payload []byte) (Completion, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, g.baseURL, bytes.NewReader(payload))
	if err != nil {
		return Completion{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+g.apiKey)

	resp, err := g.http.Do(req)
	if err != nil {
		return Completion{}, err
	}
	defer func() { _ = resp.Body.Close() }()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return Completion{}, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return Completion{}, &APIError{Status: resp.StatusCode, Message: g.mask(truncate(string(raw), 300))}
	}

	comp, r, err := parseGroqResponse(raw)
	if err != nil {
		return Completion{}, err
	}
	route := ModelRoute{Name: g.model} // free -> cost ~0
	usage := Usage{
		InputTokens:  r.Usage.PromptTokens,
		OutputTokens: r.Usage.CompletionTokens,
		CostUSD:      EstimateCostUSD(route, r.Usage.PromptTokens, r.Usage.CompletionTokens),
	}
	if g.budget != nil {
		g.budget.Add(usage.CostUSD)
	}
	if g.meter != nil {
		g.meter(g.model, usage)
	}
	return comp, nil
}

// mask redacts the API key if it ever appears in an error body.
func (g *GroqLLM) mask(s string) string {
	if g.apiKey == "" {
		return s
	}
	return strings.ReplaceAll(s, g.apiKey, "***")
}

var (
	_ LLM           = (*GroqLLM)(nil)
	_ JSONCompleter = (*GroqLLM)(nil)
)
