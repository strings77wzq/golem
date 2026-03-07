package main

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/strings77wzq/golem/feature/mcp"
)

type MCPConfig struct {
	Servers []mcp.ServerConfig `json:"servers"`
}

// LoadMCPTools loads tools from MCP servers
func LoadMCPTools(ctx context.Context, cfg MCPConfig) (*mcp.Manager, error) {
	if len(cfg.Servers) == 0 {
		return nil, nil
	}

	manager := mcp.NewManager()
	for _, server := range cfg.Servers {
		if err := manager.AddServer(server); err != nil {
			return nil, fmt.Errorf("adding MCP server %s: %w", server.Name, err)
		}
	}

	if err := manager.Start(ctx); err != nil {
		return nil, fmt.Errorf("starting MCP servers: %w", err)
	}

	return manager, nil
}

// MCPToolsToRegistry converts MCP manager's tools to a tools.Registry
func MCPToolsToRegistry(manager *mcp.Manager) (map[string]mcp.MCPToolProxy, error) {
	if manager == nil {
		return nil, nil
	}

	proxies, err := manager.DiscoverTools(context.Background())
	if err != nil {
		return nil, fmt.Errorf("discovering MCP tools: %w", err)
	}

	result := make(map[string]mcp.MCPToolProxy)
	for _, proxy := range proxies {
		result[proxy.Name()] = proxy
	}

	return result, nil
}

// ParseMCPConfig parses MCP configuration from JSON string
func ParseMCPConfig(jsonStr string) (MCPConfig, error) {
	cfg := MCPConfig{}

	if jsonStr == "" {
		return cfg, nil
	}

	if err := json.Unmarshal([]byte(jsonStr), &cfg); err != nil {
		return cfg, fmt.Errorf("parsing MCP config JSON: %w", err)
	}

	return cfg, nil
}
