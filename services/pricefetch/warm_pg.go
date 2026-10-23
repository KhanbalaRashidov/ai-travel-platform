package pricefetch

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	conn "github.com/KhanbalaRashidov/ai-travel-platform/services/connectors"
)

// PgWarm is a Postgres-backed Warm tier (durable, date-keyed fare cache). Table:
// db/migrations/0003_fare_cache.
type PgWarm struct {
	pool *pgxpool.Pool
	now  func() time.Time
}

// NewPgWarm wraps a pool as a warm fare cache.
func NewPgWarm(pool *pgxpool.Pool) *PgWarm { return &PgWarm{pool: pool, now: time.Now} }

// GetFare returns a non-expired cached fare for (tenant, key).
func (w *PgWarm) GetFare(ctx context.Context, tenantID, key string) (Candidate, bool) {
	var (
		source, url, currency string
		amount                float64
		fetchedAt             time.Time
	)
	err := w.pool.QueryRow(ctx,
		`SELECT source, url, amount, currency, fetched_at
		   FROM fare_cache
		  WHERE tenant_id = $1 AND fare_key = $2 AND expires_at > now()`,
		tenantID, key).Scan(&source, &url, &amount, &currency, &fetchedAt)
	if err != nil || amount <= 0 {
		return Candidate{}, false
	}
	return Candidate{
		Source: source, URL: url,
		Price:     conn.Money{Amount: amount, Currency: currency},
		FetchedAt: fetchedAt,
	}, true
}

// PutFare upserts a fare with a TTL.
func (w *PgWarm) PutFare(ctx context.Context, tenantID, key string, c Candidate, ttl time.Duration) {
	exp := w.now().UTC().Add(ttl)
	_, _ = w.pool.Exec(ctx,
		`INSERT INTO fare_cache (tenant_id, fare_key, source, url, amount, currency, fetched_at, expires_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
		 ON CONFLICT (tenant_id, fare_key) DO UPDATE
		   SET source=$3, url=$4, amount=$5, currency=$6, fetched_at=$7, expires_at=$8`,
		tenantID, key, c.Source, c.URL, c.Price.Amount, c.Price.Currency, c.FetchedAt, exp)
}

var _ Warm = (*PgWarm)(nil)
