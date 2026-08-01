package main

import (
	"github.com/strings77wzq/golem/core/security"
	"github.com/strings77wzq/golem/foundation/logger"
)

// newAuditFn builds the structured-log audit callback shared by the CLI
// agent and mcp-server entry points.
func newAuditFn(log logger.Logger, component string) func(entry security.AuditEntry) {
	return func(entry security.AuditEntry) {
		log.WithComponent(component).Info("db audit",
			"operation", entry.Operation,
			"database", entry.Database,
			"table", entry.Table,
			"sql", entry.SQL,
			"status", entry.Status,
			"affected_rows", entry.AffectedRows,
			"rollback_sql", entry.RollbackSQL,
			"trace_id", entry.TraceID,
		)
	}
}
