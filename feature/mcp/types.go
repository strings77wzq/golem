package mcp

import "encoding/json"

// MCPTool represents a tool available from an external MCP server.
// The InputSchema is preserved as raw JSON so that it can be converted
// into golem ToolParameters via MCPToolProxy.Parameters.
type MCPTool struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	InputSchema json.RawMessage `json:"inputSchema"`
}
