package knowledge

import "context"

// Service composes the knowledge base into the two self-growing behaviors:
// Learn (store what a trip taught us) and Suggest (recommend from aggregate
// popularity). It depends only on the Store port, so it works with either the
// in-memory or the Postgres implementation.
type Service struct {
	store Store
}

// NewService wires the knowledge service.
func NewService(s Store) *Service { return &Service{store: s} }

// Learn persists the destinations of a generated trip and bumps their
// popularity, so later users benefit from what this trip discovered.
func (s *Service) Learn(ctx context.Context, tenantID string, destinations []Destination) error {
	for _, d := range destinations {
		if err := s.store.UpsertDestination(ctx, tenantID, d); err != nil {
			return err
		}
		if err := s.store.RecordVisit(ctx, tenantID, d.City, "city"); err != nil {
			return err
		}
	}
	return nil
}

// Suggest returns dynamic recommendations from aggregate popularity
// ("travelers like you chose ...").
func (s *Service) Suggest(ctx context.Context, tenantID string, limit int) ([]Recommendation, error) {
	top, err := s.store.TopDestinations(ctx, tenantID, limit)
	if err != nil {
		return nil, err
	}
	recs := make([]Recommendation, 0, len(top))
	for _, p := range top {
		recs = append(recs, Recommendation{
			Destination: p.Destination,
			Hits:        p.Hits,
			Reason:      "popular among recent travelers",
		})
	}
	return recs, nil
}
