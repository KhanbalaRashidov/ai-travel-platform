package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/KhanbalaRashidov/ai-travel-platform/services/ai"
	conn "github.com/KhanbalaRashidov/ai-travel-platform/services/connectors"
	"github.com/KhanbalaRashidov/ai-travel-platform/services/dataflow"
	"github.com/KhanbalaRashidov/ai-travel-platform/services/planner"
)

// This file exposes the planner and the data flywheel as LLM tools, so the chat
// orchestrator can ground its answers in real engine output (routes) and real,
// cached/estimated prices (quotes) instead of inventing them.

// QuoteTool answers "what does leg X->Y cost by MODE?" via the flywheel
// (hot -> warm -> cold), so any price it returns is grounded + provenance-tagged.
type QuoteTool struct {
	fw       *dataflow.RouteFlywheel
	tenantID string
}

// NewQuoteTool builds the quote tool.
func NewQuoteTool(fw *dataflow.RouteFlywheel, tenantID string) *QuoteTool {
	return &QuoteTool{fw: fw, tenantID: tenantID}
}

// Name identifies the tool to the model.
func (t *QuoteTool) Name() string { return "quote" }

// Call expects args {from, to, mode}. It returns a one-line grounded quote.
func (t *QuoteTool) Call(ctx context.Context, args map[string]any) (string, error) {
	from, _ := args["from"].(string)
	to, _ := args["to"].(string)
	mode, _ := args["mode"].(string)
	if from == "" || to == "" {
		return "", fmt.Errorf("quote: from and to are required")
	}
	if mode == "" {
		mode = "flight"
	}
	q, tier, err := t.fw.Route(ctx, t.tenantID, from, to, conn.TransportMode(mode))
	if err != nil {
		return fmt.Sprintf("No quote available for %s to %s by %s.", from, to, mode), nil
	}
	est := ""
	if q.Estimated {
		est = " (estimated)"
	}
	return fmt.Sprintf("%s to %s by %s: %.2f %s [source=%s]%s",
		from, to, q.Mode, q.Price.Amount, q.Price.Currency, tier, est), nil
}

// SequenceTool answers "what is the cheapest order to visit these cities?" by
// building a cost matrix from the estimator and running the TSP engine.
type SequenceTool struct {
	est estimateSource // injected transport estimator
}

// estimateSource is the minimal transport-estimate interface SequenceTool needs
// (satisfied by connectors/estimate.Estimator).
type estimateSource interface {
	Transport(ctx context.Context, from, to string, mode conn.TransportMode) (conn.TransportQuote, error)
}

// NewSequenceTool builds the sequence tool.
func NewSequenceTool(est estimateSource) *SequenceTool {
	return &SequenceTool{est: est}
}

// Name identifies the tool to the model.
func (t *SequenceTool) Name() string { return "sequence" }

// Call expects args {cities: [names], start: name}. It returns the cheapest
// visiting order and its total cost (grounded).
func (t *SequenceTool) Call(ctx context.Context, args map[string]any) (string, error) {
	names := toStringSlice(args["cities"])
	if len(names) < 2 {
		return "", fmt.Errorf("sequence: at least 2 cities are required")
	}
	start, _ := args["start"].(string)
	startIdx := 0
	for i, n := range names {
		if strings.EqualFold(n, start) {
			startIdx = i
			break
		}
	}

	// Build an asymmetric cheapest-mode cost matrix via the estimator.
	m := make(planner.Matrix, len(names))
	modes := []conn.TransportMode{conn.ModeFlight, conn.ModeTrain, conn.ModeBus}
	for i := range names {
		m[i] = make([]float64, len(names))
		for j := range names {
			if i == j {
				continue
			}
			best := -1.0
			for _, mode := range modes {
				q, err := t.est.Transport(ctx, names[i], names[j], mode)
				if err != nil {
					continue
				}
				if best < 0 || q.Price.Amount < best {
					best = q.Price.Amount
				}
			}
			if best < 0 {
				return fmt.Sprintf("Cannot route between %s and %s.", names[i], names[j]), nil
			}
			m[i][j] = best
		}
	}

	best, _, err := planner.Optimize(m, startIdx, -1)
	if err != nil {
		return "", fmt.Errorf("sequence: %w", err)
	}
	ordered := make([]string, len(best.Order))
	for i, idx := range best.Order {
		ordered[i] = names[idx]
	}
	return fmt.Sprintf("Cheapest order (%s): %s. Transport total: %.2f EUR.",
		best.Variant, strings.Join(ordered, " -> "), best.Cost), nil
}

// toStringSlice coerces an any (from JSON) into a []string.
func toStringSlice(v any) []string {
	switch xs := v.(type) {
	case []string:
		return xs
	case []any:
		out := make([]string, 0, len(xs))
		for _, x := range xs {
			if s, ok := x.(string); ok {
				out = append(out, s)
			}
		}
		return out
	}
	return nil
}

var (
	_ ai.Tool = (*QuoteTool)(nil)
	_ ai.Tool = (*SequenceTool)(nil)
)
