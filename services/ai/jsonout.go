package ai

import (
	"context"
	"strings"
)

// JSONCompleter is an optional capability: an LLM that can be asked for a raw
// JSON response (no prose/fences). Gemini implements it via responseMimeType.
// Claude/Mock do not, so callers fall back to a normal completion + fence strip.
type JSONCompleter interface {
	CompleteJSON(ctx context.Context, msgs []Message) (Completion, error)
}

// completeStructured requests structured output, preferring the provider's JSON
// mode when available.
func completeStructured(ctx context.Context, llm LLM, msgs []Message) (Completion, error) {
	if jc, ok := llm.(JSONCompleter); ok {
		return jc.CompleteJSON(ctx, msgs)
	}
	return llm.Complete(ctx, msgs, nil)
}

// CompleteStructured is the exported form of completeStructured, for other
// packages (e.g. price extraction) that need JSON-mode output.
func CompleteStructured(ctx context.Context, llm LLM, msgs []Message) (Completion, error) {
	return completeStructured(ctx, llm, msgs)
}

// ExtractJSONObject is the exported, robust JSON-object extractor.
func ExtractJSONObject(s string) string { return extractJSONObject(s) }

// extractJSONObject returns the first balanced-ish JSON object in s: it strips
// code fences, then takes the span from the first "{" to the last "}". This
// tolerates a model that wraps JSON in prose despite instructions.
func extractJSONObject(s string) string {
	s = stripFences(s)
	i := strings.IndexByte(s, '{')
	j := strings.LastIndexByte(s, '}')
	if i >= 0 && j > i {
		return s[i : j+1]
	}
	return s
}

// snippet returns a short, single-line preview of raw model output for logs.
func snippet(s string, n int) string {
	s = strings.TrimSpace(strings.ReplaceAll(s, "\n", " "))
	if len(s) > n {
		return s[:n] + "…"
	}
	return s
}
