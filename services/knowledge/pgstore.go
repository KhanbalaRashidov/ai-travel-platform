package knowledge

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	conn "github.com/KhanbalaRashidov/ai-travel-platform/services/connectors"
)

// PgStore is the Postgres-backed Store (production), using the 0002_knowledge
// tables. It is written against the schema but exercised only once the dev
// Docker Postgres has the migrations applied.
type PgStore struct {
	pool *pgxpool.Pool
}

// NewPgStore wraps a pgx pool as a Store.
func NewPgStore(pool *pgxpool.Pool) *PgStore { return &PgStore{pool: pool} }

func nullTime(t time.Time) *time.Time {
	if t.IsZero() {
		return nil
	}
	return &t
}

func (s *PgStore) PutRoute(ctx context.Context, tenantID string, q conn.TransportQuote) error {
	const sql = `
		INSERT INTO route_cache
			(tenant_id, from_city, to_city, mode, price_amount, price_currency,
			 duration_seconds, deep_link, estimated, source, fetched_at, expires_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)
		ON CONFLICT (tenant_id, from_city, to_city, mode) DO UPDATE SET
			price_amount=EXCLUDED.price_amount, price_currency=EXCLUDED.price_currency,
			duration_seconds=EXCLUDED.duration_seconds, deep_link=EXCLUDED.deep_link,
			estimated=EXCLUDED.estimated, source=EXCLUDED.source,
			fetched_at=EXCLUDED.fetched_at, expires_at=EXCLUDED.expires_at`
	_, err := s.pool.Exec(ctx, sql, tenantID, q.From, q.To, string(q.Mode),
		q.Price.Amount, q.Price.Currency, int(q.Duration.Seconds()), q.DeepLink,
		q.Estimated, q.Source, q.FetchedAt, nullTime(q.ExpiresAt))
	return err
}

func (s *PgStore) GetRoute(ctx context.Context, tenantID, from, to string, mode conn.TransportMode) (conn.TransportQuote, bool, error) {
	const sql = `
		SELECT price_amount, price_currency, duration_seconds, deep_link,
		       estimated, source, fetched_at, expires_at
		FROM route_cache
		WHERE tenant_id=$1 AND from_city=$2 AND to_city=$3 AND mode=$4
		  AND (expires_at IS NULL OR expires_at > now())`
	var (
		amount   float64
		currency string
		durSec   int
		deepLink string
		est      bool
		source   string
		fetched  time.Time
		expires  *time.Time
	)
	err := s.pool.QueryRow(ctx, sql, tenantID, from, to, string(mode)).
		Scan(&amount, &currency, &durSec, &deepLink, &est, &source, &fetched, &expires)
	if errors.Is(err, pgx.ErrNoRows) {
		return conn.TransportQuote{}, false, nil
	}
	if err != nil {
		return conn.TransportQuote{}, false, err
	}
	q := conn.TransportQuote{
		From: from, To: to, Mode: mode,
		Price:    conn.Money{Amount: amount, Currency: currency},
		Duration: time.Duration(durSec) * time.Second,
		DeepLink: deepLink,
		Provenance: conn.Provenance{
			Source: source, FetchedAt: fetched, Estimated: est,
		},
	}
	if expires != nil {
		q.ExpiresAt = *expires
	}
	return q, true, nil
}

func (s *PgStore) PutFx(ctx context.Context, tenantID string, r conn.FxRate) error {
	const sql = `
		INSERT INTO fx_rates (tenant_id, base, quote, rate, source, fetched_at, expires_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7)
		ON CONFLICT (tenant_id, base, quote) DO UPDATE SET
			rate=EXCLUDED.rate, source=EXCLUDED.source,
			fetched_at=EXCLUDED.fetched_at, expires_at=EXCLUDED.expires_at`
	_, err := s.pool.Exec(ctx, sql, tenantID, r.Base, r.Quote, r.Rate, r.Source, r.FetchedAt, nullTime(r.ExpiresAt))
	return err
}

func (s *PgStore) GetFx(ctx context.Context, tenantID, base, quote string) (conn.FxRate, bool, error) {
	const sql = `
		SELECT rate, source, fetched_at, expires_at
		FROM fx_rates
		WHERE tenant_id=$1 AND base=$2 AND quote=$3
		  AND (expires_at IS NULL OR expires_at > now())`
	var (
		rate    float64
		source  string
		fetched time.Time
		expires *time.Time
	)
	err := s.pool.QueryRow(ctx, sql, tenantID, base, quote).Scan(&rate, &source, &fetched, &expires)
	if errors.Is(err, pgx.ErrNoRows) {
		return conn.FxRate{}, false, nil
	}
	if err != nil {
		return conn.FxRate{}, false, err
	}
	r := conn.FxRate{Base: base, Quote: quote, Rate: rate,
		Provenance: conn.Provenance{Source: source, FetchedAt: fetched}}
	if expires != nil {
		r.ExpiresAt = *expires
	}
	return r, true, nil
}

func (s *PgStore) UpsertDestination(ctx context.Context, tenantID string, d Destination) error {
	const sql = `
		INSERT INTO destinations (tenant_id, country, city, lat, lng, summary, source, fetched_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
		ON CONFLICT (tenant_id, country, city) DO UPDATE SET
			lat=EXCLUDED.lat, lng=EXCLUDED.lng, summary=EXCLUDED.summary,
			source=EXCLUDED.source, fetched_at=EXCLUDED.fetched_at`
	fetched := d.FetchedAt
	if fetched.IsZero() {
		fetched = time.Now()
	}
	_, err := s.pool.Exec(ctx, sql, tenantID, d.Country, d.City, d.Lat, d.Lng, d.Summary, d.Source, fetched)
	return err
}

func (s *PgStore) ListDestinations(ctx context.Context, tenantID string) ([]Destination, error) {
	const sql = `
		SELECT country, city, lat, lng, summary, source, fetched_at
		FROM destinations WHERE tenant_id=$1 ORDER BY country, city`
	rows, err := s.pool.Query(ctx, sql, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Destination
	for rows.Next() {
		var (
			d        Destination
			lat, lng *float64
		)
		if err := rows.Scan(&d.Country, &d.City, &lat, &lng, &d.Summary, &d.Source, &d.FetchedAt); err != nil {
			return nil, err
		}
		if lat != nil {
			d.Lat = *lat
		}
		if lng != nil {
			d.Lng = *lng
		}
		out = append(out, d)
	}
	return out, rows.Err()
}

func (s *PgStore) RecordVisit(ctx context.Context, tenantID, destination, kind string) error {
	const sql = `
		INSERT INTO popularity_rank (tenant_id, destination, kind, hits, updated_at)
		VALUES ($1,$2,$3,1,now())
		ON CONFLICT (tenant_id, destination, kind) DO UPDATE SET
			hits = popularity_rank.hits + 1, updated_at = now()`
	_, err := s.pool.Exec(ctx, sql, tenantID, destination, kind)
	return err
}

func (s *PgStore) TopDestinations(ctx context.Context, tenantID string, limit int) ([]Popularity, error) {
	if limit <= 0 {
		limit = 50
	}
	const sql = `
		SELECT destination, kind, hits, updated_at
		FROM popularity_rank WHERE tenant_id=$1
		ORDER BY hits DESC, destination ASC
		LIMIT $2`
	rows, err := s.pool.Query(ctx, sql, tenantID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Popularity
	for rows.Next() {
		var p Popularity
		if err := rows.Scan(&p.Destination, &p.Kind, &p.Hits, &p.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

var _ Store = (*PgStore)(nil)
