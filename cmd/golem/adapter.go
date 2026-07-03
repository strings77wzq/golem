// Package main — feature-specific wiring functions.
// These are kept in cmd/ because they depend on feature package internals.
package main

import (
	"context"
	"fmt"

	"github.com/strings77wzq/golem/core/config"
	"github.com/strings77wzq/golem/core/tools"
	"github.com/strings77wzq/golem/feature/mcp"
	"github.com/strings77wzq/golem/foundation/logger"
)

// loadMCPTools loads MCP tools into the registry from a flag value.
func loadMCPTools(ctx context.Context, flagValue string, registry *tools.Registry, log logger.Logger) error {
	if flagValue == "" {
		return nil
	}
	mcpCfg, err := ParseMCPConfig(flagValue)
	if err != nil {
		return fmt.Errorf("parsing MCP config: %w", err)
	}
	mcpManager, err := LoadMCPTools(ctx, mcpCfg)
	if err != nil {
		return fmt.Errorf("loading MCP tools: %w", err)
	}
	mcpProxies, err := MCPToolsToRegistry(mcpManager)
	if err != nil {
		return fmt.Errorf("converting MCP tools: %w", err)
	}
	for _, proxy := range mcpProxies {
		if err := registry.Register(proxy); err != nil {
			log.Warn("failed to register MCP tool", "tool", proxy.Name(), "error", err)
		}
	}
	log.Info("loaded MCP tools", "count", len(mcpProxies))
	return nil
}

// loadRAGTools loads RAG tools into the registry from a flag value.
func loadRAGTools(ctx context.Context, cfg *config.Config, flagValue string, registry *tools.Registry) error {
	if flagValue == "" {
		return nil
	}
	ragCfg, err := ParseRagConfig(flagValue)
	if err != nil {
		return fmt.Errorf("parsing RAG config: %w", err)
	}
	if ragCfg.APIKey == "" {
		if defaultModel, _ := cfg.FindModel(cfg.Agents.Defaults.ModelName); defaultModel != nil {
			ragCfg.APIKey = defaultModel.APIKey
		}
	}
	ragRegistry, err := LoadRAGTools(ctx, ragCfg)
	if err != nil {
		return fmt.Errorf("loading RAG tools: %w", err)
	}
	for _, t := range ragRegistry.ListTools() {
		registry.Register(t) //nolint:errcheck
	}
	return nil
}

// loadMemoryTools loads memory tools into the registry from a flag value.
func loadMemoryTools(ctx context.Context, flagValue string, registry *tools.Registry, log logger.Logger) error {
	if flagValue == "" {
		return nil
	}
	memCfg, err := ParseMemoryConfig(flagValue)
	if err != nil {
		return fmt.Errorf("parsing memory config: %w", err)
	}
	memRegistry, _, err := LoadMemoryTools(ctx, memCfg)
	if err != nil {
		return fmt.Errorf("loading memory tools: %w", err)
	}
	for _, t := range memRegistry.ListTools() {
		registry.Register(t) //nolint:errcheck
	}
	log.Info("loaded memory tools")
	return nil
}

// MCPManager is an alias for convenience.
type MCPManager = mcp.Manager
