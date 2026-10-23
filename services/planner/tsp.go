package planner

import (
	"errors"
	"math"
)

// Matrix is an asymmetric transport-cost matrix: Matrix[i][j] is the cheapest
// cost to go from city i to city j. Use math.Inf(1) for "no edge".
type Matrix [][]float64

// ErrNoRoute means no feasible route exists for the given matrix/constraints.
var ErrNoRoute = errors.New("no feasible route")

// Variant is the trip shape.
type Variant string

const (
	RoundTrip Variant = "round_trip" // start == end (returns home)
	OpenJaw   Variant = "open_jaw"   // start != end
)

// exactThreshold is the largest node count solved exactly (Held-Karp). Beyond
// it, the heuristic (nearest-neighbor + 2-opt) is used to cap the blow-up.
const exactThreshold = 12

const inf = math.MaxFloat64

// Route is a solved visiting order with its cost. For RoundTrip the implicit
// return hop to the start is included in Cost but not repeated in Order.
type Route struct {
	Variant Variant `json:"variant"`
	Order   []int   `json:"order"`
	Cost    float64 `json:"cost"`
}

// Solve finds the best route for one variant with a fixed start and an optional
// fixed end (end = -1 means free end; end is ignored for RoundTrip).
func Solve(m Matrix, start, end int, variant Variant) (Route, error) {
	n := len(m)
	if n == 0 {
		return Route{}, ErrNoRoute
	}
	if start < 0 || start >= n {
		return Route{}, errors.New("start out of range")
	}
	returnToStart := variant == RoundTrip
	e := end
	if returnToStart {
		e = -1
	}
	if e >= n {
		return Route{}, errors.New("end out of range")
	}

	var (
		order []int
		cost  float64
		err   error
	)
	if n <= exactThreshold {
		order, cost, err = heldKarp(m, start, e, returnToStart)
	} else {
		order, cost, err = heuristic(m, start, e, returnToStart)
	}
	if err != nil {
		return Route{}, err
	}
	return Route{Variant: variant, Order: order, Cost: cost}, nil
}

// Optimize evaluates both RoundTrip and OpenJaw and returns the cheaper as best
// and the other as alt. end applies to the open-jaw variant (-1 = free end).
func Optimize(m Matrix, start, end int) (best, alt Route, err error) {
	rt, e1 := Solve(m, start, -1, RoundTrip)
	oj, e2 := Solve(m, start, end, OpenJaw)
	switch {
	case e1 == nil && e2 == nil:
		if rt.Cost <= oj.Cost {
			return rt, oj, nil
		}
		return oj, rt, nil
	case e1 == nil:
		return rt, Route{}, nil
	case e2 == nil:
		return oj, Route{}, nil
	default:
		return Route{}, Route{}, ErrNoRoute
	}
}

// heldKarp solves exactly via bitmask DP. With returnToStart it finds the min
// Hamiltonian cycle from start; otherwise the min Hamiltonian path from start
// to the fixed end (or to the cheapest end when end < 0).
func heldKarp(m Matrix, start, end int, returnToStart bool) ([]int, float64, error) {
	n := len(m)
	full := (1 << n) - 1

	dp := make([][]float64, 1<<n)
	par := make([][]int, 1<<n)
	for i := range dp {
		dp[i] = make([]float64, n)
		par[i] = make([]int, n)
		for j := range dp[i] {
			dp[i][j] = inf
			par[i][j] = -1
		}
	}
	dp[1<<start][start] = 0

	for mask := 0; mask <= full; mask++ {
		if mask&(1<<start) == 0 {
			continue
		}
		for j := 0; j < n; j++ {
			if mask&(1<<j) == 0 || dp[mask][j] == inf {
				continue
			}
			for k := 0; k < n; k++ {
				if mask&(1<<k) != 0 {
					continue
				}
				c := m[j][k]
				if math.IsInf(c, 1) || c >= inf {
					continue
				}
				nm := mask | (1 << k)
				if nd := dp[mask][j] + c; nd < dp[nm][k] {
					dp[nm][k] = nd
					par[nm][k] = j
				}
			}
		}
	}

	bestCost := inf
	bestEnd := -1
	switch {
	case returnToStart:
		for j := 0; j < n; j++ {
			if dp[full][j] == inf {
				continue
			}
			c := m[j][start]
			if math.IsInf(c, 1) {
				continue
			}
			if dp[full][j]+c < bestCost {
				bestCost = dp[full][j] + c
				bestEnd = j
			}
		}
	case end >= 0:
		bestCost = dp[full][end]
		bestEnd = end
	default:
		for j := 0; j < n; j++ {
			if j == start && n > 1 {
				continue
			}
			if dp[full][j] < bestCost {
				bestCost = dp[full][j]
				bestEnd = j
			}
		}
	}
	if bestEnd < 0 || bestCost >= inf {
		return nil, 0, ErrNoRoute
	}

	order := make([]int, 0, n)
	for mask, j := full, bestEnd; j != -1; {
		order = append(order, j)
		pj := par[mask][j]
		mask &^= 1 << j
		j = pj
	}
	for i, k := 0, len(order)-1; i < k; i, k = i+1, k-1 {
		order[i], order[k] = order[k], order[i]
	}
	return order, bestCost, nil
}

// heuristic builds a nearest-neighbor path then improves it with 2-opt, keeping
// the start fixed (and the end fixed for an open path when end >= 0).
func heuristic(m Matrix, start, end int, returnToStart bool) ([]int, float64, error) {
	order, ok := nearestNeighbor(m, start, end, returnToStart)
	if !ok {
		return nil, 0, ErrNoRoute
	}
	order = twoOpt(order, m, returnToStart, !returnToStart && end >= 0)
	c := pathCost(order, m, returnToStart)
	if math.IsInf(c, 1) || c >= inf {
		return nil, 0, ErrNoRoute
	}
	return order, c, nil
}

func nearestNeighbor(m Matrix, start, end int, returnToStart bool) ([]int, bool) {
	n := len(m)
	visited := make([]bool, n)
	order := []int{start}
	visited[start] = true
	cur := start
	deferEnd := !returnToStart && end >= 0

	for len(order) < n {
		next := -1
		best := math.Inf(1)
		for k := 0; k < n; k++ {
			if visited[k] {
				continue
			}
			if deferEnd && k == end && len(order) < n-1 {
				continue // keep the fixed end for last
			}
			if m[cur][k] < best {
				best = m[cur][k]
				next = k
			}
		}
		if next == -1 {
			if deferEnd && !visited[end] {
				next = end
			} else {
				return nil, false
			}
		}
		order = append(order, next)
		visited[next] = true
		cur = next
	}
	return order, true
}

// twoOpt reverses segments to reduce cost. Index 0 (start) is never moved; when
// fixedEnd is true the last index is never moved either.
func twoOpt(order []int, m Matrix, returnToStart, fixedEnd bool) []int {
	n := len(order)
	if n < 4 {
		return order
	}
	best := pathCost(order, m, returnToStart)
	for improved := true; improved; {
		improved = false
		hi := n - 1
		if fixedEnd {
			hi = n - 2
		}
		for i := 1; i <= hi-1; i++ {
			for j := i + 1; j <= hi; j++ {
				cand := make([]int, n)
				copy(cand, order)
				for a, b := i, j; a < b; a, b = a+1, b-1 {
					cand[a], cand[b] = cand[b], cand[a]
				}
				if c := pathCost(cand, m, returnToStart); c+1e-9 < best {
					best = c
					order = cand
					improved = true
				}
			}
		}
	}
	return order
}

// pathCost sums leg costs along order, adding the return hop for a round trip.
func pathCost(order []int, m Matrix, returnToStart bool) float64 {
	c := 0.0
	for i := 0; i+1 < len(order); i++ {
		c += m[order[i]][order[i+1]]
	}
	if returnToStart && len(order) > 1 {
		c += m[order[len(order)-1]][order[0]]
	}
	return c
}
