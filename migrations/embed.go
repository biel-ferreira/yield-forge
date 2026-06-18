// Package migrations holds the database schema migrations as embedded SQL files.
//
// Files are named NNNN_short_name.up.sql / NNNN_short_name.down.sql (zero-padded,
// monotonically increasing, one concern per migration). They are embedded into the
// binary so migrations ship with the app and can be applied without the source tree
// present (SPEC-002 FR-203).
package migrations

import "embed"

// FS is the embedded set of *.sql migration files, consumed by the migrate runner
// via golang-migrate's iofs source.
//
//go:embed *.sql
var FS embed.FS
