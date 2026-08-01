// Package mcp implements MCP server/client integration using the official
// modelcontextprotocol/go-sdk (v1.7.0). The golem tool registry is exposed
// over stdio (mcp-server subcommand) and external MCP servers are proxied
// via Manager with mcp_ prefixed tools.
package mcp

import (
	"context"
	"encoding/json"
	"fmt"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/strings77wzq/golem/core/tools"
)

// Server-side input/output limits: a remote MCP client must not be able to
// OOM this process with oversized arguments, nor drain unbounded tokens via
// oversized tool output.
const (
	maxCallArgumentsBytes = 1 << 20  // 1 MiB
	maxServerOutputBytes  = 10 << 10 // 10 KiB, mirrors core/tools/exec
)

// Server exposes golem registry tools over MCP using the official SDK.
type Server struct {
	sdk      *sdk.Server
	registry *tools.Registry
}

// NewServer creates an MCP server exposing all registry tools. Tool
// parameters are converted to a JSON Schema object (draft 2020-12) so that
// MCP clients can validate arguments before calling.
func NewServer(registry *tools.Registry) *Server {
	s := &Server{registry: registry}

	sdkServer := sdk.NewServer(&sdk.Implementation{
		Name:    "golem-mcp",
		Version: "0.7.0",
	}, nil)

	for _, tool := range registry.ListTools() {
		t := tool
		sdkServer.AddTool(&sdk.Tool{
			Name:        t.Name(),
			Description: t.Description(),
			InputSchema: inputSchemaFromParameters(t.Parameters()),
		}, func(ctx context.Context, req *sdk.CallToolRequest) (*sdk.CallToolResult, error) {
			return handleToolCall(ctx, t, req)
		})
	}

	s.sdk = sdkServer
	return s
}

// Run serves on stdio (newline-delimited JSON) until ctx is cancelled or
// stdin closes.
func (s *Server) Run(ctx context.Context) error {
	return s.sdk.Run(ctx, &sdk.StdioTransport{})
}

// Connect binds the server to an explicit transport. Used by tests with
// sdk.NewInMemoryTransports; production uses Run.
func (s *Server) Connect(ctx context.Context, transport sdk.Transport) (*sdk.ServerSession, error) {
	return s.sdk.Connect(ctx, transport, nil)
}

// inputSchemaFromParameters converts golem ToolParameters into a JSON Schema
// object (type: object + properties + required).
func inputSchemaFromParameters(params []tools.ToolParameter) map[string]any {
	props := make(map[string]any, len(params))
	required := make([]string, 0, len(params))
	for _, p := range params {
		props[p.Name] = map[string]any{
			"type":        p.Type,
			"description": p.Description,
		}
		if p.Required {
			required = append(required, p.Name)
		}
	}
	return map[string]any{
		"type":       "object",
		"properties": props,
		"required":   required,
	}
}

// handleToolCall routes a tools/call request into the golem tool and maps the
// ToolResult back to an MCP CallToolResult. Errors from Execute are mapped to
// IsError=true so the MCP connection stays usable.
func handleToolCall(ctx context.Context, tool tools.Tool, req *sdk.CallToolRequest) (*sdk.CallToolResult, error) {
	args := map[string]any{}
	if req.Params != nil && len(req.Params.Arguments) > maxCallArgumentsBytes {
		return &sdk.CallToolResult{
			Content: []sdk.Content{&sdk.TextContent{Text: "arguments too large"}},
			IsError: true,
		}, nil
	}
	if req.Params != nil && len(req.Params.Arguments) > 0 {
		if err := json.Unmarshal(req.Params.Arguments, &args); err != nil {
			return &sdk.CallToolResult{
				Content: []sdk.Content{&sdk.TextContent{Text: fmt.Sprintf("Invalid arguments: %v", err)}},
				IsError: true,
			}, nil
		}
	}

	result, err := tool.Execute(ctx, args)
	if err != nil {
		return &sdk.CallToolResult{
			Content: []sdk.Content{&sdk.TextContent{Text: fmt.Sprintf("Tool execution failed: %v", err)}},
			IsError: true,
		}, nil
	}
	if result == nil {
		return &sdk.CallToolResult{
			Content: []sdk.Content{&sdk.TextContent{Text: "tool returned no result"}},
			IsError: true,
		}, nil
	}

	// MCP consumers are LLMs, so return the full LLM-facing result.
	// ForUser is a human summary and used only as a fallback.
	text := result.ForLLM
	if text == "" {
		text = result.ForUser
	}
	if len(text) > maxServerOutputBytes {
		text = text[:maxServerOutputBytes] + "\n...[truncated]"
	}
	return &sdk.CallToolResult{
		Content: []sdk.Content{&sdk.TextContent{Text: text}},
		IsError: result.IsError,
	}, nil
}
