// Package migrate applies embedded SQL up-migrations idempotently at startup.
// Each migration runs once (tracked in schema_migrations), in a transaction,
// fail-fast — so `docker compose up -d` needs no separate migrate step.
package migrate

import (
	"context"
	"fmt"
	"io/fs"
	"sort"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

type entry struct {
	version string // e.g. "0001_init"
	path    string // e.g. "migrations/0001_init.up.sql"
}

// list returns the up-migrations sorted by filename (version order).
func list(fsys fs.FS) ([]entry, error) {
	files, err := fs.ReadDir(fsys, "migrations")
	if err != nil {
		return nil, err
	}
	var out []entry
	for _, f := range files {
		name := f.Name()
		if !strings.HasSuffix(name, ".up.sql") {
			continue
		}
		out = append(out, entry{
			version: strings.TrimSuffix(name, ".up.sql"),
			path:    "migrations/" + name,
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].version < out[j].version })
	return out, nil
}

// Apply runs every not-yet-applied up-migration in order. Idempotent (safe to
// run on every boot) and fail-fast (returns on the first error).
func Apply(ctx context.Context, pool *pgxpool.Pool, fsys fs.FS) error {
	if _, err := pool.Exec(ctx,
		`CREATE TABLE IF NOT EXISTS schema_migrations (
			version    TEXT PRIMARY KEY,
			applied_at TIMESTAMPTZ NOT NULL DEFAULT now()
		)`); err != nil {
		return fmt.Errorf("migrate: tracking table: %w", err)
	}

	entries, err := list(fsys)
	if err != nil {
		return fmt.Errorf("migrate: list: %w", err)
	}

	for _, e := range entries {
		var exists bool
		if err := pool.QueryRow(ctx,
			`SELECT EXISTS (SELECT 1 FROM schema_migrations WHERE version = $1)`, e.version,
		).Scan(&exists); err != nil {
			return fmt.Errorf("migrate: check %s: %w", e.version, err)
		}
		if exists {
			continue
		}

		sqlBytes, err := fs.ReadFile(fsys, e.path)
		if err != nil {
			return fmt.Errorf("migrate: read %s: %w", e.version, err)
		}

		tx, err := pool.Begin(ctx)
		if err != nil {
			return fmt.Errorf("migrate: begin %s: %w", e.version, err)
		}
		// No args => pgx uses the simple protocol, which allows the multiple
		// statements a migration file contains.
		if _, err := tx.Exec(ctx, string(sqlBytes)); err != nil {
			_ = tx.Rollback(ctx)
			return fmt.Errorf("migrate: apply %s: %w", e.version, err)
		}
		if _, err := tx.Exec(ctx, `INSERT INTO schema_migrations (version) VALUES ($1)`, e.version); err != nil {
			_ = tx.Rollback(ctx)
			return fmt.Errorf("migrate: record %s: %w", e.version, err)
		}
		if err := tx.Commit(ctx); err != nil {
			return fmt.Errorf("migrate: commit %s: %w", e.version, err)
		}
	}
	return nil
}
