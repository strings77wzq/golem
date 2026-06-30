// Package database provides SQL tools for database operations.
package database

import (
	"context"
	"fmt"
	"regexp"

	"github.com/strings77wzq/golem/core/database"
)

// identPattern is the strict charset a SQL identifier (table/column name) is
// allowed to match when a tool interpolates an LLM-supplied identifier into a
// query. Anything outside this set is rejected before it reaches the driver,
// so an LLM cannot smuggle SQL (e.g. "users; DROP TABLE users--") past the guard.
var identPattern = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

// validateTableIdentifier confirms an LLM-supplied table name is safe to
// interpolate into a query: it must match the strict identifier charset AND be
// present in the live driver schema (probe via GetSchemaForTable). The
// validated name is returned verbatim on success; a non-nil typed error is
// returned (and no count query should be issued) on any failure.
func validateTableIdentifier(ctx context.Context, driver database.SQLDriver, table string) (string, error) {
	if !identPattern.MatchString(table) {
		// Do NOT echo the raw, possibly-malicious payload back to the LLM —
		// a user-visible error containing "DROP TABLE" would leak the
		// injection attempt into the model context. Report only that the
		// charset check failed (length still aids debugging without trusting
		// the content).
		return "", fmt.Errorf("table identifier failed charset check (rejected %d-char identifier)", len(table))
	}
	if _, err := driver.GetSchemaForTable(ctx, table); err != nil {
		return "", fmt.Errorf("table not present in schema: %w", err)
	}
	return table, nil
}
