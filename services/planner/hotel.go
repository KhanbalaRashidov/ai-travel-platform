package planner

// HotelCandidate is one hotel option for a city.
type HotelCandidate struct {
	Name         string  `json:"name"`
	Stars        int     `json:"stars"`
	Nightly      Money   `json:"nightly"`
	CentralityKm float64 `json:"centrality_km"` // distance from the city center (optional preference)
}

// SelectHotel picks the cheapest hotel whose star rating falls within the
// tier's band; ties break toward the more central hotel when preferCentral is
// set. If no candidate is in-band it falls back to the cheapest at-or-above the
// minimum stars, else the cheapest overall. Pure.
func SelectHotel(candidates []HotelCandidate, tier ComfortTier, preferCentral bool) (HotelCandidate, bool) {
	if len(candidates) == 0 {
		return HotelCandidate{}, false
	}
	minS, maxS := TierStars(tier)

	pool := filterStars(candidates, minS, maxS)
	if len(pool) == 0 {
		pool = filterStars(candidates, minS, 5) // allow an upgrade above the band
	}
	if len(pool) == 0 {
		pool = candidates // last resort: anything
	}

	best := pool[0]
	for _, h := range pool[1:] {
		if betterHotel(h, best, preferCentral) {
			best = h
		}
	}
	return best, true
}

func filterStars(cands []HotelCandidate, minS, maxS int) []HotelCandidate {
	var out []HotelCandidate
	for _, h := range cands {
		if h.Stars >= minS && h.Stars <= maxS {
			out = append(out, h)
		}
	}
	return out
}

func betterHotel(a, b HotelCandidate, preferCentral bool) bool {
	if a.Nightly.Amount != b.Nightly.Amount {
		return a.Nightly.Amount < b.Nightly.Amount
	}
	if preferCentral && a.CentralityKm != b.CentralityKm {
		return a.CentralityKm < b.CentralityKm
	}
	return false
}
