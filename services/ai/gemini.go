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

// geminiBaseURL is the Google Generative Language API root.
const geminiBaseURL = "https://generativelanguage.googleapis.com/v1beta/models"

// DefaultGeminiModel is the free, fast default.
const DefaultGeminiModel = "gemini-2.0-flash"

// GeminiLLM is a free LLM backed by Google's Generative Language API. It
// satisfies the same provider-agnostic LLM interface as ClaudeLLM. The API key
// is supplied at construction (read from the environment) and never logged.
// Gemini's free tier means per-call cost is reported as ~0.
type GeminiLLM struct {
	apiKey    string
	model     string
	fallback  string // optional second model tried when the primary stays overloaded
	maxTokens int
	baseURL   string
	http      HTTPDoer
	budget    *Budget
	meter     func(model string, u Usage)
	// delays are the backoff waits before each retry (len+1 = max attempts).
	delays []time.Duration
	sleep  func(d time.Duration) // injectable for tests
}

// NewGemini builds a Gemini adapter. Pass nil doer for a default HTTP client;
// empty model uses DefaultGeminiModel.
func NewGemini(apiKey, model string, doer HTTPDoer) *GeminiLLM {
	if doer == nil {
		doer = &http.Client{Timeout: 45 * time.Second}
	}
	if model == "" {
		model = DefaultGeminiModel
	}
	return &GeminiLLM{
		apiKey:    apiKey,
		model:     model,
		maxTokens: defaultMaxTokens,
		baseURL:   geminiBaseURL,
		http:      doer,
		// Up to 3 attempts; only transient (429/503) errors trigger a retry.
		delays: []time.Duration{500 * time.Millisecond, 1500 * time.Millisecond},
		sleep:  time.Sleep,
	}
}

// WithBudget attaches a shared budget (Gemini cost is ~0, so this rarely trips).
func (g *GeminiLLM) WithBudget(b *Budget) *GeminiLLM { g.budget = b; return g }

// WithMeter attaches a usage callback for token logging.
func (g *GeminiLLM) WithMeter(f func(model string, u Usage)) *GeminiLLM { g.meter = f; return g }

// WithFallback sets a second model to try when the primary stays overloaded.
func (g *GeminiLLM) WithFallback(model string) *GeminiLLM { g.fallback = model; return g }

// Model returns the configured model name.
func (g *GeminiLLM) Model() string { return g.model }

// --- request/response shapes ------------------------------------------------

type geminiPart struct {
	Text         string              `json:"text,omitempty"`
	FunctionCall *geminiFunctionCall `json:"functionCall,omitempty"`
	FunctionResp *geminiFunctionResp `json:"functionResponse,omitempty"`
}

type geminiFunctionCall struct {
	Name string         `json:"name"`
	Args map[string]any `json:"args,omitempty"`
}

type geminiFunctionResp struct {
	Name     string         `json:"name"`
	Response map[string]any `json:"response"`
}

type geminiContent struct {
	Role  string       `json:"role,omitempty"`
	Parts []geminiPart `json:"parts"`
}

type geminiSystem struct {
	Parts []geminiPart `json:"parts"`
}

type geminiFuncDecl struct {
	Name        string         `json:"name"`
	Description string         `json:"description,omitempty"`
	Parameters  map[string]any `json:"parameters"`
}

type geminiTool struct {
	FunctionDeclarations []geminiFuncDecl `json:"functionDeclarations"`
}

type geminiGenConfig struct {
	MaxOutputTokens  int    `json:"maxOutputTokens,omitempty"`
	ResponseMIMEType string `json:"responseMimeType,omitempty"`
}

type geminiReq struct {
	SystemInstruction *geminiSystem   `json:"systemInstruction,omitempty"`
	Contents          []geminiContent `json:"contents"`
	Tools             []geminiTool    `json:"tools,omitempty"`
	GenerationConfig  geminiGenConfig `json:"generationConfig"`
}

type geminiResp struct {
	Candidates []struct {
		Content geminiContent `json:"content"`
	} `json:"candidates"`
	UsageMetadata struct {
		PromptTokenCount     int `json:"promptTokenCount"`
		CandidatesTokenCount int `json:"candidatesTokenCount"`
	} `json:"usageMetadata"`
	Error *struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Status  string `json:"status"`
	} `json:"error"`
}

// buildGeminiRequest maps provider-agnostic messages/tools into a Gemini
// request: system messages go to systemInstruction, roles become user/model,
// and tool_use/tool_result map to functionCall/functionResponse parts.
// Consecutive same-role turns are merged. Pure — unit-tested without the network.
func buildGeminiRequest(maxTokens int, msgs []Message, tools []ToolSpec, jsonMode bool) geminiReq {
	req := geminiReq{GenerationConfig: geminiGenConfig{MaxOutputTokens: maxTokens}}
	if jsonMode {
		// Ask Gemini for raw JSON (no prose/fences) so structured extraction
		// parses reliably.
		req.GenerationConfig.ResponseMIMEType = "application/json"
	}

	var systemParts []string
	for _, m := range msgs {
		if m.Role == RoleSystem {
			if m.Content != "" {
				systemParts = append(systemParts, m.Content)
			}
			continue
		}
		role, part := mapGeminiTurn(m)
		if n := len(req.Contents); n > 0 && req.Contents[n-1].Role == role {
			req.Contents[n-1].Parts = append(req.Contents[n-1].Parts, part)
		} else {
			req.Contents = append(req.Contents, geminiContent{Role: role, Parts: []geminiPart{part}})
		}
	}
	if len(systemParts) > 0 {
		req.SystemInstruction = &geminiSystem{Parts: []geminiPart{{Text: strings.Join(systemParts, "\n\n")}}}
	}

	if len(tools) > 0 {
		decls := make([]geminiFuncDecl, 0, len(tools))
		for _, t := range tools {
			decls = append(decls, geminiFuncDecl{
				Name:        t.Name,
				Description: t.Description,
				Parameters:  map[string]any{"type": "object", "properties": map[string]any{}},
			})
		}
		req.Tools = []geminiTool{{FunctionDeclarations: decls}}
	}
	return req
}

// mapGeminiTurn maps one message to a Gemini role ("user"/"model") + part.
func mapGeminiTurn(m Message) (string, geminiPart) {
	switch {
	case m.Role == RoleTool:
		return "user", geminiPart{FunctionResp: &geminiFunctionResp{
			Name:     m.ToolName,
			Response: map[string]any{"result": m.Content},
		}}
	case m.Role == RoleAssistant && len(m.ToolCalls) > 0:
		tc := m.ToolCalls[0]
		return "model", geminiPart{FunctionCall: &geminiFunctionCall{Name: tc.Name, Args: tc.Args}}
	case m.Role == RoleAssistant:
		return "model", geminiPart{Text: m.Content}
	default:
		return "user", geminiPart{Text: m.Content}
	}
}

// parseGeminiResponse turns a Gemini response body into a Completion + token
// usage. Pure.
func parseGeminiResponse(body []byte) (Completion, geminiResp, error) {
	var r geminiResp
	if err := json.Unmarshal(body, &r); err != nil {
		return Completion{}, r, fmt.Errorf("decode gemini response: %w", err)
	}
	if r.Error != nil {
		return Completion{}, r, fmt.Errorf("gemini error: %s: %s", r.Error.Status, r.Error.Message)
	}
	var comp Completion
	var text strings.Builder
	if len(r.Candidates) > 0 {
		for _, p := range r.Candidates[0].Content.Parts {
			if p.Text != "" {
				text.WriteString(p.Text)
			}
			if p.FunctionCall != nil {
				comp.ToolCalls = append(comp.ToolCalls, ToolCall{Name: p.FunctionCall.Name, Args: p.FunctionCall.Args})
			}
		}
	}
	comp.Content = text.String()
	return comp, r, nil
}

// Complete calls the Gemini generateContent endpoint.
func (g *GeminiLLM) Complete(ctx context.Context, msgs []Message, tools []ToolSpec) (Completion, error) {
	return g.complete(ctx, msgs, tools, false)
}

// CompleteJSON forces Gemini's JSON response mode (no tools), so callers that
// need structured output get clean JSON without prose or code fences. Satisfies
// the optional JSONCompleter interface.
func (g *GeminiLLM) CompleteJSON(ctx context.Context, msgs []Message) (Completion, error) {
	return g.complete(ctx, msgs, nil, true)
}

func (g *GeminiLLM) complete(ctx context.Context, msgs []Message, tools []ToolSpec, jsonMode bool) (Completion, error) {
	if g.budget != nil && g.budget.Remaining() <= 0 {
		return Completion{}, fmt.Errorf("ai budget exhausted (spent $%.4f of $%.2f)", g.budget.SpentUSD, g.budget.LimitUSD)
	}
	payload, err := json.Marshal(buildGeminiRequest(g.maxTokens, msgs, tools, jsonMode))
	if err != nil {
		return Completion{}, err
	}

	comp, err := g.attempt(ctx, g.model, payload)
	// If the primary model stays overloaded, try the fallback once (with its own
	// retry budget) before giving up.
	if err != nil && IsTransient(err) && g.fallback != "" && g.fallback != g.model {
		if c2, e2 := g.attempt(ctx, g.fallback, payload); e2 == nil {
			return c2, nil
		}
	}
	return comp, err
}

// attempt performs one logical call to a model, retrying only on transient
// (429/503) errors with exponential backoff + jitter.
func (g *GeminiLLM) attempt(ctx context.Context, model string, payload []byte) (Completion, error) {
	return retryTransient(ctx, g.delays, g.sleep, func() (Completion, error) {
		return g.doOnce(ctx, model, payload)
	})
}

// doOnce performs a single HTTP call + parse, metering on success.
func (g *GeminiLLM) doOnce(ctx context.Context, model string, payload []byte) (Completion, error) {
	// The key is a query param; keep it out of the body/logs.
	u := fmt.Sprintf("%s/%s:generateContent?key=%s", g.baseURL, model, g.apiKey)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u, bytes.NewReader(payload))
	if err != nil {
		return Completion{}, err
	}
	req.Header.Set("Content-Type", "application/json")

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

	comp, r, err := parseGeminiResponse(raw)
	if err != nil {
		return Completion{}, err
	}

	// Gemini's free tier -> treat cost as ~0 (route priced at 0).
	route := ModelRoute{Name: model}
	usage := Usage{
		InputTokens:  r.UsageMetadata.PromptTokenCount,
		OutputTokens: r.UsageMetadata.CandidatesTokenCount,
		CostUSD:      EstimateCostUSD(route, r.UsageMetadata.PromptTokenCount, r.UsageMetadata.CandidatesTokenCount),
	}
	if g.budget != nil {
		g.budget.Add(usage.CostUSD)
	}
	if g.meter != nil {
		g.meter(model, usage)
	}
	return comp, nil
}

// mask redacts the API key if it ever appears in an error body.
func (g *GeminiLLM) mask(s string) string {
	if g.apiKey == "" {
		return s
	}
	return strings.ReplaceAll(s, g.apiKey, "***")
}

var _ LLM = (*GeminiLLM)(nil)
