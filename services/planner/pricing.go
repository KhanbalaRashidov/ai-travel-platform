package planner

import (
	"fmt"
	"math"
)

// Converter converts an amount between currencies. The real implementation
// wraps the FX service (connectors); a static one is used for tests/dev.
type Converter interface {
	Convert(amount float64, from, to string) (float64, error)
}

// StaticConverter holds each currency's value in a common numeraire, so
// Convert(a, from, to) = a * Rates[from] / Rates[to].
type StaticConverter struct {
	Rates map[string]float64
}

// Convert implements Converter.
func (c StaticConverter) Convert(amount float64, from, to string) (float64, error) {
	if from == to {
		return amount, nil
	}
	rf, ok1 := c.Rates[from]
	rt, ok2 := c.Rates[to]
	if !ok1 || !ok2 || rt == 0 {
		return 0, fmt.Errorf("missing FX rate for %s or %s", from, to)
	}
	return amount * rf / rt, nil
}

// Price computes the deterministic cost breakdown in the display currency. It
// does no sequencing: changing tiers/stars/modes only changes Legs/Hotels and
// a re-call here re-prices instantly without touching the route order.
func Price(in PlanInput, conv Converter) (Breakdown, error) {
	cur := in.DisplayCurrency
	var transport, hotels float64

	for _, l := range in.Legs {
		v, err := conv.Convert(l.Price.Amount, l.Price.Currency, cur)
		if err != nil {
			return Breakdown{}, err
		}
		transport += v
	}
	for _, h := range in.Hotels {
		v, err := conv.Convert(h.Nightly.Amount*float64(h.Nights), h.Nightly.Currency, cur)
		if err != nil {
			return Breakdown{}, err
		}
		hotels += v
	}
	act, err := conv.Convert(in.ActivitiesTotal.Amount, in.ActivitiesTotal.Currency, cur)
	if err != nil {
		return Breakdown{}, err
	}

	transport, hotels, act = round2(transport), round2(hotels), round2(act)
	return Breakdown{
		Transport:  transport,
		Hotels:     hotels,
		Activities: act,
		Total:      round2(transport + hotels + act),
		Currency:   cur,
	}, nil
}

func round2(f float64) float64 { return math.Round(f*100) / 100 }
