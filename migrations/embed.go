// Package migrations embeds the SQL migration files into the binary so the app
// can run migrations without the golang-migrate CLI being installed on the host.
package migrations

import "embed"

// FS holds every .sql migration, embedded at build time.
//
//go:embed *.sql
var FS embed.FS
