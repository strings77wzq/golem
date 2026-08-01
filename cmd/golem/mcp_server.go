package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/spf13/cobra"

	dbcore "github.com/strings77wzq/golem/core/database"
	"github.com/strings77wzq/golem/core/security"
	"github.com/strings77wzq/golem/core/tools"
	dbtools "github.com/strings77wzq/golem/core/tools/database"
	toolexec "github.com/strings77wzq/golem/core/tools/exec"
	"github.com/strings77wzq/golem/core/tools/fileops"
	"github.com/strings77wzq/golem/core/tools/websearch"
	"github.com/strings77wzq/golem/feature/mcp"
	"github.com/strings77wzq/golem/foundation/logger"
)

func newMCPServerCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "mcp-server",
		Short: "Start MCP server (stdio mode) to expose tools to other agents",
		Long:  "Start an MCP server that exposes Golem's database and tool capabilities via stdio JSON-RPC. Other agents (Claude Code, Cursor, etc.) can connect to this server.",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runMCPServer(cmd)
		},
	}

	cmd.Flags().String("tools", "", "Comma-separated list of tools to expose (default: all)")
	cmd.Flags().String("db", "", "Database path (SQLite) or connection string")
	cmd.Flags().Bool("read-only", false, "Only expose read operations (no INSERT/UPDATE/DELETE)")

	return cmd
}

func runMCPServer(cmd *cobra.Command) error {
	toolsFlag, _ := cmd.Flags().GetString("tools")
	dbFlag, _ := cmd.Flags().GetString("db")
	readOnly, _ := cmd.Flags().GetBool("read-only")

	log := logger.New(logger.DefaultOptions())
	auditFn := newAuditFn(log, logger.ComponentMCP)

	// Build tool registry
	registry := buildMCPTools(dbFlag, readOnly, toolsFlag, auditFn)

	// Create MCP server with stdio transport (official go-sdk)
	server := mcp.NewServer(registry)

	// Handle shutdown
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	fmt.Fprintln(os.Stderr, "Golem MCP Server started (stdio mode)")
	fmt.Fprintln(os.Stderr, "Tools:", registry.Count(), "registered")
	fmt.Fprintln(os.Stderr, "Read-only:", readOnly)

	return server.Run(ctx)
}

func buildMCPTools(dbPath string, readOnly bool, toolsFilter string, auditFn func(security.AuditEntry)) *tools.Registry {
	registry := tools.NewRegistry()

	// Parse tool filter
	allowedTools := make(map[string]bool)
	if toolsFilter != "" {
		for _, name := range strings.Split(toolsFilter, ",") {
			allowedTools[strings.TrimSpace(name)] = true
		}
	}
	acceptAll := len(allowedTools) == 0

	// Database tools
	if dbPath != "" {
		dbRegistry := dbcore.NewRegistry()

		driverName := "default"
		driver := dbcore.NewSQLiteDriver(driverName, dbPath)
		if err := driver.Connect(context.Background()); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to connect to database %s: %v\n", dbPath, err)
		} else {
			dbRegistry.RegisterSQL(driverName, driver)
			dbRegistry.SetDefault(driverName)

			if acceptAll || allowedTools["sql_query"] {
				sqlTool := dbtools.NewSQLQueryTool(dbRegistry)
				if auditFn != nil {
					sqlTool.SetAuditFunc(auditFn)
				}
				if err := registry.Register(sqlTool); err != nil {
					fmt.Fprintf(os.Stderr, "Warning: failed to register sql_query: %v\n", err)
				}
			}
			if acceptAll || allowedTools["sql_schema"] {
				if err := registry.Register(dbtools.NewSQLSchemaTool(dbRegistry)); err != nil {
					fmt.Fprintf(os.Stderr, "Warning: failed to register sql_schema: %v\n", err)
				}
			}
			if acceptAll || allowedTools["sql_analyze"] {
				if err := registry.Register(dbtools.NewSQLAnalyzeTool(dbRegistry)); err != nil {
					fmt.Fprintf(os.Stderr, "Warning: failed to register sql_analyze: %v\n", err)
				}
			}
		}
	}

	// Non-database tools
	workspace, err := os.Getwd()
	if err != nil {
		workspace = "."
	}
	if acceptAll || allowedTools["exec"] {
		if !readOnly {
			if err := registry.Register(toolexec.New(workspace)); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: failed to register exec: %v\n", err)
			}
		}
	}
	if acceptAll || allowedTools["file_read"] {
		if err := registry.Register(fileops.NewFileReadTool(workspace)); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to register file_read: %v\n", err)
		}
	}
	if acceptAll || allowedTools["file_write"] {
		if !readOnly {
			if err := registry.Register(fileops.NewFileWriteTool(workspace)); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: failed to register file_write: %v\n", err)
			}
		}
	}
	if acceptAll || allowedTools["web_search"] {
		if err := registry.Register(websearch.New()); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to register web_search: %v\n", err)
		}
	}

	return registry
}
