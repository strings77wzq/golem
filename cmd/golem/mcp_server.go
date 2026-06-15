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
	"github.com/strings77wzq/golem/core/tools"
	dbtools "github.com/strings77wzq/golem/core/tools/database"
	toolexec "github.com/strings77wzq/golem/core/tools/exec"
	"github.com/strings77wzq/golem/core/tools/fileops"
	"github.com/strings77wzq/golem/core/tools/websearch"
	"github.com/strings77wzq/golem/feature/mcp"
)

func newMCPServerCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "mcp-server",
		Short: "Start MCP server (stdio mode) to expose tools to other agents",
		Long:  "Start an MCP server that exposes Golem's database and tool capabilities via stdio JSON-RPC. Other agents (Claude Code, Cursor, etc.) can connect to this server.",
		RunE: func(cmd *cobra.Command, args []string) error {
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

	// Build tool registry
	registry := buildMCPTools(dbFlag, readOnly, toolsFlag)

	// Create MCP server with stdio transport
	server := mcp.NewServer(os.Stdin, os.Stdout, registry)

	// Handle shutdown
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	fmt.Fprintln(os.Stderr, "Golem MCP Server started (stdio mode)")
	fmt.Fprintln(os.Stderr, "Tools:", registry.Count(), "registered")
	fmt.Fprintln(os.Stderr, "Read-only:", readOnly)

	return server.Start(ctx)
}

func buildMCPTools(dbPath string, readOnly bool, toolsFilter string) *tools.Registry {
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
				registry.Register(dbtools.NewSQLQueryTool(dbRegistry))
			}
			if acceptAll || allowedTools["sql_schema"] {
				registry.Register(dbtools.NewSQLSchemaTool(dbRegistry))
			}
			if acceptAll || allowedTools["sql_analyze"] {
				registry.Register(dbtools.NewSQLAnalyzeTool(dbRegistry))
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
			registry.Register(toolexec.New(workspace))
		}
	}
	if acceptAll || allowedTools["file_read"] {
		registry.Register(fileops.NewFileReadTool(workspace))
	}
	if acceptAll || allowedTools["file_write"] {
		if !readOnly {
			registry.Register(fileops.NewFileWriteTool(workspace))
		}
	}
	if acceptAll || allowedTools["web_search"] {
		registry.Register(websearch.New())
	}

	return registry
}
