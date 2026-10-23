package connectors

import "context"

// FxChain tries its FX sources in order and returns the first successful rate.
// This is the resilience pattern: free source -> scrape source -> (future)
// labeled estimate. If every source fails it returns ErrUnavailable.
type FxChain struct {
	sources []FxSource
}

// NewFxChain builds an FX fallback chain in priority order.
func NewFxChain(sources ...FxSource) *FxChain {
	return &FxChain{sources: sources}
}

// Name identifies the chain.
func (c *FxChain) Name() string { return "fx-chain" }

// Rate returns the first source's successful rate, else ErrUnavailable.
func (c *FxChain) Rate(ctx context.Context, base, quote string) (FxRate, error) {
	for _, s := range c.sources {
		if r, err := s.Rate(ctx, base, quote); err == nil {
			return r, nil
		}
	}
	return FxRate{}, ErrUnavailable
}

var _ FxSource = (*FxChain)(nil)
