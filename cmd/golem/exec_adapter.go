package main

import (
	"encoding/json"
	"fmt"

	"github.com/strings77wzq/golem/internal/security"
)

// SandboxValidator adapts internal/security.Sandbox to core/tools/exec.CommandValidator.
type SandboxValidator struct {
	sandbox *security.Sandbox
}

// NewSandboxValidator creates a validator wrapping the security sandbox.
func NewSandboxValidator(cfg security.SandboxConfig) *SandboxValidator {
	return &SandboxValidator{sandbox: security.NewSandbox(cfg)}
}

// ValidateCommand checks if the command is allowed by sandbox rules.
func (v *SandboxValidator) ValidateCommand(cmd string) error {
	return v.sandbox.ValidateCommand(cmd)
}

// ValidatePath checks if a file path is allowed by sandbox rules.
func (v *SandboxValidator) ValidatePath(path string) error {
	return v.sandbox.ValidatePath(path)
}

// ParseSandboxConfig parses a sandbox config from JSON flag value.
func ParseSandboxConfig(flagValue string) (*security.SandboxConfig, error) {
	if flagValue == "" {
		return nil, fmt.Errorf("empty sandbox config")
	}
	var cfg struct {
		AllowedPaths   []string `json:"allowed_paths"`
		DeniedPaths    []string `json:"denied_paths"`
		DeniedCommands []string `json:"denied_commands"`
	}
	if err := json.Unmarshal([]byte(flagValue), &cfg); err != nil {
		return nil, fmt.Errorf("parsing sandbox config: %w", err)
	}
	return &security.SandboxConfig{
		AllowedPaths:   cfg.AllowedPaths,
		DeniedPaths:    cfg.DeniedPaths,
		DeniedCommands: cfg.DeniedCommands,
	}, nil
}

// DefaultSandboxConfig returns a safe default sandbox configuration.
func DefaultSandboxConfig() security.SandboxConfig {
	return security.SandboxConfig{
		DeniedCommands: []string{"rm", "shutdown", "reboot", "halt", "init", "systemctl"},
	}
}
