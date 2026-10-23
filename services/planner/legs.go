package planner

import (
	"errors"
	"sort"
)

// LegOption is one transport choice for a city->city hop.
type LegOption struct {
	Mode            TransportMode `json:"mode"`
	Price           Money         `json:"price"`
	DurationMinutes int           `json:"duration_minutes"`
}

// LegInput is a leg with the options to compare.
type LegInput struct {
	From    string      `json:"from"`
	To      string      `json:"to"`
	Options []LegOption `json:"options"`
}

// LegComparison is the result for one leg: the cheapest (default) and fastest
// choices plus all options sorted cheapest-first.
type LegComparison struct {
	From     string      `json:"from"`
	To       string      `json:"to"`
	Cheapest LegOption   `json:"cheapest"`
	Fastest  LegOption   `json:"fastest"`
	Options  []LegOption `json:"options"`
}

// ErrNoOptions means a leg had no transport options to compare.
var ErrNoOptions = errors.New("leg has no options")

// CompareLeg picks the cheapest (ties -> shorter) and fastest (ties -> cheaper)
// options and returns all options sorted cheapest-first. Pure.
func CompareLeg(from, to string, opts []LegOption) (LegComparison, error) {
	if len(opts) == 0 {
		return LegComparison{}, ErrNoOptions
	}
	sorted := append([]LegOption(nil), opts...)
	sort.SliceStable(sorted, func(i, j int) bool {
		if sorted[i].Price.Amount != sorted[j].Price.Amount {
			return sorted[i].Price.Amount < sorted[j].Price.Amount
		}
		return sorted[i].DurationMinutes < sorted[j].DurationMinutes
	})

	fastest := opts[0]
	for _, o := range opts[1:] {
		if o.DurationMinutes < fastest.DurationMinutes ||
			(o.DurationMinutes == fastest.DurationMinutes && o.Price.Amount < fastest.Price.Amount) {
			fastest = o
		}
	}

	return LegComparison{
		From:     from,
		To:       to,
		Cheapest: sorted[0],
		Fastest:  fastest,
		Options:  sorted,
	}, nil
}

// CompareLegs compares every leg; it fails if any leg has no options.
func CompareLegs(legs []LegInput) ([]LegComparison, error) {
	out := make([]LegComparison, 0, len(legs))
	for _, l := range legs {
		cmp, err := CompareLeg(l.From, l.To, l.Options)
		if err != nil {
			return nil, err
		}
		out = append(out, cmp)
	}
	return out, nil
}
