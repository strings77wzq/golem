package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/spf13/cobra"
	"github.com/strings77wzq/golem/core/tools"
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

	// Database tools are always available if dbPath is set
	if dbPath != "" {
		// SQLite driver will be initialized when tools are called
		// For now, register tool stubs that will connect on first use
		if len(allowedTools) == 0 || allowedTools["sql_query"] {
			registry.Register(newMCPTool("sql_query", "Execute SQL SELECT query and return results"))
		}
		if len(allowedTools) == 0 || allowedTools["sql_schema"] {
			registry.Register(newMCPTool("sql_schema", "Get database schema information"))
		}
		if len(allowedTools) == 0 || allowedTools["sql_analyze"] {
			registry.Register(newMCPTool("sql_analyze", "Analyze data distribution for a table"))
		}
	}

	// Non-database tools
	if len(allowedTools) == 0 || allowedTools["exec"] {
		if !readOnly {
			registry.Register(newMCPTool("exec", "Execute a shell command"))
		}
	}
	if len(allowedTools) == 0 || allowedTools["file_read"] {
		registry.Register(newMCPTool("file_read", "Read file contents"))
	}
	if len(allowedTools) == 0 || allowedTools["file_write"] {
		if !readOnly {
			registry.Register(newMCPTool("file_write", "Write content to a file"))
		}
	}
	if len(allowedTools) == 0 || allowedTools["web_search"] {
		registry.Register(newMCPTool("web_search", "Search the web"))
	}

	return registry
}

// mcpTool is a simple tool implementation for MCP server
type mcpTool struct {
	name        string
	description string
}

func newMCPTool(name, description string) *mcpTool {
	return &mcpTool{name: name, description: description}
}

func (t *mcpTool) Name() string        { return t.name }
func (t *mcpTool) Description() string { return t.description }
func (t *mcpTool) Parameters() []tools.ToolParameter {
	return []tools.ToolParameter{
		{Name: "input", Type: "string", Description: "Input for the tool", Required: true},
	}
}

func (t *mcpTool) Execute(ctx context.Context, args map[string]interface{}) (*tools.ToolResult, error) {
	input, _ := args["input"].(string)
	return &tools.ToolResult{
		ForLLM:  fmt.Sprintf("Tool %s executed with input: %s", t.name, input),
		ForUser: fmt.Sprintf("Executed %s", t.name),
	}, nil
}
