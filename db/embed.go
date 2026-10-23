// Package migrations embeds the SQL up-migrations so the gateway can apply them
// automatically on boot (zero-touch deploy — no make/psql step).
package migrations

import "embed"

//go:embed migrations/*.up.sql
var FS embed.FS
