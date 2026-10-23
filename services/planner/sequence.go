package planner

// Sequencer computes the best route (via Optimize) and caches the result by an
// order-independent city-set key. This enforces refinement #2: the expensive
// TSP runs ONLY when the city set changes. Tier/star/mode toggles reuse the
// cached order and merely re-price (see Price), so the UI stays instant.
type Sequencer struct {
	cache    map[string]seqResult
	computes int // observability: how many times Optimize actually ran
}

type seqResult struct {
	best Route
	alt  Route
}

// NewSequencer builds an empty sequencer.
func NewSequencer() *Sequencer {
	return &Sequencer{cache: make(map[string]seqResult)}
}

// Sequence returns the best + alternative route for the city set identified by
// key. On a cache hit it does not re-run the solver.
func (s *Sequencer) Sequence(key string, m Matrix, start, end int) (best, alt Route, err error) {
	if r, ok := s.cache[key]; ok {
		return r.best, r.alt, nil
	}
	best, alt, err = Optimize(m, start, end)
	if err != nil {
		return Route{}, Route{}, err
	}
	s.cache[key] = seqResult{best: best, alt: alt}
	s.computes++
	return best, alt, nil
}

// Computes reports how many times the solver actually ran (for tests/metrics).
func (s *Sequencer) Computes() int { return s.computes }

// Invalidate drops a cached city set (e.g. when its transport prices refresh).
func (s *Sequencer) Invalidate(key string) { delete(s.cache, key) }
