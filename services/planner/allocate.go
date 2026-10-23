package planner

import (
	"math"
	"sort"
)

// AllocateNights distributes total nights across n cities, giving each at least
// minPer, proportional to weights (equal weighting when weights is nil), using
// the largest-remainder method so the result sums to total exactly and is
// deterministic. If total < n*minPer it fills from the front as far as it can.
func AllocateNights(total, n, minPer int, weights []float64) []int {
	if n <= 0 {
		return nil
	}
	res := make([]int, n)

	if total < n*minPer {
		left := total
		for i := 0; i < n; i++ {
			g := minPer
			if g > left {
				g = left
			}
			if g < 0 {
				g = 0
			}
			res[i] = g
			left -= g
		}
		return res
	}

	for i := range res {
		res[i] = minPer
	}
	remaining := total - n*minPer
	if remaining == 0 {
		return res
	}

	w := make([]float64, n)
	sum := 0.0
	for i := 0; i < n; i++ {
		if weights != nil && i < len(weights) && weights[i] > 0 {
			w[i] = weights[i]
		} else {
			w[i] = 1
		}
		sum += w[i]
	}

	type frac struct {
		idx int
		rem float64
	}
	fracs := make([]frac, n)
	assigned := 0
	for i := 0; i < n; i++ {
		ideal := float64(remaining) * w[i] / sum
		b := int(math.Floor(ideal))
		res[i] += b
		assigned += b
		fracs[i] = frac{idx: i, rem: ideal - float64(b)}
	}

	leftover := remaining - assigned
	sort.Slice(fracs, func(a, b int) bool {
		if fracs[a].rem != fracs[b].rem {
			return fracs[a].rem > fracs[b].rem
		}
		return fracs[a].idx < fracs[b].idx
	})
	for k := 0; k < leftover && k < n; k++ {
		res[fracs[k].idx]++
	}
	return res
}
