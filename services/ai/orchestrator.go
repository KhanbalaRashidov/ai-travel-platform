package ai

import (
	"context"
	"errors"
)

// Tool is something the model can call for real data (connectors, planner,
// knowledge base). Its string result is fed back to the model, and any
// currency amounts in it become grounding facts.
type Tool interface {
	Name() string
	Call(ctx context.Context, args map[string]any) (string, error)
}

// Result is the outcome of an orchestration run.
type Result struct {
	Text       string      `json:"text"`
	Grounding  []float64   `json:"grounding"`
	Violations []Violation `json:"violations"`
	Steps      int         `json:"steps"`
}

// Orchestrator drives the LLM<->tools loop and enforces the price guardrail.
type Orchestrator struct {
	llm      LLM
	tools    map[string]Tool
	specs    []ToolSpec
	maxSteps int
	maxRegen int
}

// NewOrchestrator wires the orchestrator. maxSteps caps the tool loop; maxRegen
// caps how many times an ungrounded final answer is rewritten.
func NewOrchestrator(llm LLM, maxSteps, maxRegen int, tools ...Tool) *Orchestrator {
	o := &Orchestrator{
		llm:      llm,
		tools:    make(map[string]Tool, len(tools)),
		maxSteps: maxSteps,
		maxRegen: maxRegen,
	}
	for _, t := range tools {
		o.tools[t.Name()] = t
		o.specs = append(o.specs, ToolSpec{Name: t.Name()})
	}
	return o
}

// Run executes the conversation: the model may call tools (whose results are
// harvested for grounding), then produces final text that must pass the price
// guardrail. Ungrounded text is rewritten up to maxRegen times; if it still
// fails, the result is returned WITH its violations rather than silently
// trusting the model.
func (o *Orchestrator) Run(ctx context.Context, systemPrompt, userPrompt string) (Result, error) {
	msgs := []Message{
		{Role: RoleSystem, Content: systemPrompt},
		{Role: RoleUser, Content: userPrompt},
	}
	var grounding []float64
	regenLeft := o.maxRegen

	for steps := 1; steps <= o.maxSteps; steps++ {
		comp, err := o.llm.Complete(ctx, msgs, o.specs)
		if err != nil {
			return Result{}, err
		}

		if len(comp.ToolCalls) > 0 {
			msgs = append(msgs, Message{Role: RoleAssistant, ToolCalls: comp.ToolCalls})
			for _, tc := range comp.ToolCalls {
				res := o.callTool(ctx, tc)
				grounding = append(grounding, ExtractMoneyValues(res)...)
				msgs = append(msgs, Message{Role: RoleTool, ToolCallID: tc.ID, ToolName: tc.Name, Content: res})
			}
			continue
		}

		// final answer
		violations := Validate(comp.Content, grounding)
		if len(violations) > 0 && regenLeft > 0 {
			regenLeft--
			msgs = append(msgs,
				Message{Role: RoleAssistant, Content: comp.Content},
				Message{Role: RoleSystem, Content: "One or more prices were not grounded in tool results. Rewrite using ONLY the prices returned by the tools; do not invent numbers."},
			)
			continue
		}
		return Result{Text: comp.Content, Grounding: grounding, Violations: violations, Steps: steps}, nil
	}
	return Result{}, errors.New("orchestrator: max steps exceeded")
}

func (o *Orchestrator) callTool(ctx context.Context, tc ToolCall) string {
	t, ok := o.tools[tc.Name]
	if !ok {
		return "error: unknown tool " + tc.Name
	}
	out, err := t.Call(ctx, tc.Args)
	if err != nil {
		return "error: " + err.Error()
	}
	return out
}
