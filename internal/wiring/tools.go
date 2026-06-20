// Package wiring provides dependency creation functions for the CLI.
// This centralizes all wiring logic so cmd/ only needs to call these functions.
package wiring

import (
	"context"
	"fmt"
	"os"

	"github.com/strings77wzq/golem/core/database"
	"github.com/strings77wzq/golem/core/security"
	"github.com/strings77wzq/golem/core/tools"
	dbtools "github.com/strings77wzq/golem/core/tools/database"
	toolexec "github.com/strings77wzq/golem/core/tools/exec"
	"github.com/strings77wzq/golem/core/tools/fileops"
	"github.com/strings77wzq/golem/core/tools/infra"
	"github.com/strings77wzq/golem/core/tools/think"
	"github.com/strings77wzq/golem/core/tools/websearch"
)

// BuildToolRegistry creates the default tool registry with built-in tools.
func BuildToolRegistry(workspace string, execOpts ...toolexec.Option) *tools.Registry {
	registry := tools.NewRegistry()
	registry.Register(think.New())
	registry.Register(toolexec.New(workspace, execOpts...))
	registry.Register(fileops.NewFileReadTool(workspace))
	registry.Register(fileops.NewFileWriteTool(workspace))
	registry.Register(fileops.NewFileListTool(workspace))
	registry.Register(websearch.New())
	return registry
}

// BuildDBTools creates database tools from a database path.
// Returns nil registries if dbPath is empty or connection fails.
func BuildDBTools(dbPath string, auditFn func(entry security.AuditEntry), secHandler security.SecurityEventHandler) (*database.Registry, *tools.Registry) {
	if dbPath == "" {
		return nil, nil
	}

	dbRegistry := database.NewRegistry()
	driverName := "default"
	driver := database.NewSQLiteDriver(driverName, dbPath)
	if err := driver.Connect(context.Background()); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to connect to database %s: %v\n", dbPath, err)
		return nil, nil
	}
	dbRegistry.RegisterSQL(driverName, driver)
	dbRegistry.SetDefault(driverName)

	toolRegistry := tools.NewRegistry()
	sqlTool := dbtools.NewSQLQueryTool(dbRegistry)
	if auditFn != nil {
		sqlTool.SetAuditFunc(auditFn)
	}
	if secHandler != nil {
		sqlTool.SetSecurityEventHandler(secHandler)
	}
	toolRegistry.Register(sqlTool)
	toolRegistry.Register(dbtools.NewSQLSchemaTool(dbRegistry))
	toolRegistry.Register(dbtools.NewSQLAnalyzeTool(dbRegistry))

	return dbRegistry, toolRegistry
}

// RegisterInfraTools adds infrastructure tools (kubectl, docker, helm) to a registry.
func RegisterInfraTools(registry *tools.Registry) {
	registry.Register(infra.NewKubectlTool())
	registry.Register(infra.NewDockerTool())
	registry.Register(infra.NewHelmTool())
}
