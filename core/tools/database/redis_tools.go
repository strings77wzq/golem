package database

import (
	"context"
	"fmt"
	"time"

	"github.com/strings77wzq/golem/core/database"
	"github.com/strings77wzq/golem/core/tools"
)

// RedisGetTool retrieves a value by key.
type RedisGetTool struct {
	registry *database.Registry
}

func NewRedisGetTool(registry *database.Registry) *RedisGetTool {
	return &RedisGetTool{registry: registry}
}

func (t *RedisGetTool) Name() string        { return "redis_get" }
func (t *RedisGetTool) Description() string { return "Get value by key from Redis" }
func (t *RedisGetTool) Parameters() []tools.ToolParameter {
	return []tools.ToolParameter{
		{Name: "database", Type: "string", Description: "Redis instance name", Required: false},
		{Name: "key", Type: "string", Description: "Key to get", Required: true},
	}
}

func (t *RedisGetTool) Execute(ctx context.Context, args map[string]interface{}) (*tools.ToolResult, error) {
	dbName := t.getDefaultDB(args)
	key, _ := args["key"].(string)
	if key == "" {
		return &tools.ToolResult{ForLLM: "Error: key is required", IsError: true}, nil
	}

	driver, err := t.registry.GetRedis(dbName)
	if err != nil {
		return &tools.ToolResult{ForLLM: fmt.Sprintf("Error: %v", err), IsError: true}, nil
	}

	val, err := driver.Get(ctx, key)
	if err != nil {
		return &tools.ToolResult{ForLLM: fmt.Sprintf("Error: %v", err), IsError: true}, nil
	}

	return &tools.ToolResult{ForLLM: val, ForUser: fmt.Sprintf("Key: %s = %s", key, val)}, nil
}

func (t *RedisGetTool) getDefaultDB(args map[string]interface{}) string {
	if db, ok := args["database"].(string); ok && db != "" {
		return db
	}
	return t.registry.Default()
}

// RedisSetTool sets a key-value pair.
type RedisSetTool struct {
	registry *database.Registry
}

func NewRedisSetTool(registry *database.Registry) *RedisSetTool {
	return &RedisSetTool{registry: registry}
}

func (t *RedisSetTool) Name() string        { return "redis_set" }
func (t *RedisSetTool) Description() string { return "Set key-value pair in Redis" }
func (t *RedisSetTool) Parameters() []tools.ToolParameter {
	return []tools.ToolParameter{
		{Name: "database", Type: "string", Description: "Redis instance name", Required: false},
		{Name: "key", Type: "string", Description: "Key to set", Required: true},
		{Name: "value", Type: "string", Description: "Value to set", Required: true},
		{Name: "ttl", Type: "number", Description: "TTL in seconds (0 = no expiry)", Required: false},
	}
}

func (t *RedisSetTool) Execute(ctx context.Context, args map[string]interface{}) (*tools.ToolResult, error) {
	dbName := t.getDefaultDB(args)
	key, _ := args["key"].(string)
	value, _ := args["value"].(string)
	if key == "" || value == "" {
		return &tools.ToolResult{ForLLM: "Error: key and value are required", IsError: true}, nil
	}

	driver, err := t.registry.GetRedis(dbName)
	if err != nil {
		return &tools.ToolResult{ForLLM: fmt.Sprintf("Error: %v", err), IsError: true}, nil
	}

	var ttl time.Duration
	if t, ok := args["ttl"].(float64); ok && t > 0 {
		ttl = time.Duration(t) * time.Second
	}

	err = driver.Set(ctx, key, value, ttl)
	if err != nil {
		return &tools.ToolResult{ForLLM: fmt.Sprintf("Error: %v", err), IsError: true}, nil
	}
	return &tools.ToolResult{ForLLM: "OK", ForUser: fmt.Sprintf("Set %s", key)}, nil
}

func (t *RedisSetTool) getDefaultDB(args map[string]interface{}) string {
	if db, ok := args["database"].(string); ok && db != "" {
		return db
	}
	return t.registry.Default()
}

// RedisKeysTool searches keys by pattern.
type RedisKeysTool struct {
	registry *database.Registry
}

func NewRedisKeysTool(registry *database.Registry) *RedisKeysTool {
	return &RedisKeysTool{registry: registry}
}

func (t *RedisKeysTool) Name() string        { return "redis_keys" }
func (t *RedisKeysTool) Description() string { return "Search Redis keys by pattern" }
func (t *RedisKeysTool) Parameters() []tools.ToolParameter {
	return []tools.ToolParameter{
		{Name: "database", Type: "string", Description: "Redis instance name", Required: false},
		{Name: "pattern", Type: "string", Description: "Key pattern (e.g. user:*)", Required: true},
	}
}

func (t *RedisKeysTool) Execute(ctx context.Context, args map[string]interface{}) (*tools.ToolResult, error) {
	dbName := t.getDefaultDB(args)
	pattern, _ := args["pattern"].(string)
	if pattern == "" {
		return &tools.ToolResult{ForLLM: "Error: pattern is required", IsError: true}, nil
	}

	driver, err := t.registry.GetRedis(dbName)
	if err != nil {
		return &tools.ToolResult{ForLLM: fmt.Sprintf("Error: %v", err), IsError: true}, nil
	}

	keys, err := driver.Keys(ctx, pattern)
	if err != nil {
		return &tools.ToolResult{ForLLM: fmt.Sprintf("Error: %v", err), IsError: true}, nil
	}

	result := fmt.Sprintf("Found %d keys matching %q:\n", len(keys), pattern)
	for _, key := range keys {
		result += fmt.Sprintf("  %s\n", key)
	}

	return &tools.ToolResult{ForLLM: result, ForUser: fmt.Sprintf("Found %d keys", len(keys))}, nil
}

func (t *RedisKeysTool) getDefaultDB(args map[string]interface{}) string {
	if db, ok := args["database"].(string); ok && db != "" {
		return db
	}
	return t.registry.Default()
}
