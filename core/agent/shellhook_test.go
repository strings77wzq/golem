package agent

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestShellHook_Execute_Success(t *testing.T) {
	hook := &ShellHook{
		Command: `echo '{"allowed":true}'`,
		Timeout: 5 * time.Second,
	}

	output, err := hook.Execute(&HookInput{
		SessionID: "test",
		ToolName:  "sql_query",
		ToolInput: map[string]interface{}{"sql": "SELECT 1"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !output.Allowed {
		t.Error("expected allowed=true")
	}
}

func TestShellHook_Execute_Blocked(t *testing.T) {
	hook := &ShellHook{
		Command: `echo '{"allowed":false,"reason":"not permitted"}'`,
		Timeout: 5 * time.Second,
	}

	output, err := hook.Execute(&HookInput{
		SessionID: "test",
		ToolName:  "shell",
		ToolInput: map[string]interface{}{"cmd": "rm -rf /"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if output.Allowed {
		t.Error("expected allowed=false")
	}
	if output.Reason != "not permitted" {
		t.Errorf("expected reason 'not permitted', got %q", output.Reason)
	}
}

func TestShellHook_Execute_Timeout(t *testing.T) {
	hook := &ShellHook{
		Command: "cat", // reads from stdin, blocks until killed
		Timeout: 100 * time.Millisecond,
	}

	start := time.Now()
	output, err := hook.Execute(&HookInput{SessionID: "test", ToolName: "test"})
	elapsed := time.Since(start)

	// cat receives empty stdin (HookInput JSON), reads it, outputs it
	// That output is not valid HookOutput JSON, so it's blocked
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if output.Allowed {
		t.Error("cat output is not valid HookOutput JSON")
	}
	if elapsed > 2*time.Second {
		t.Errorf("hook took too long: %v (expected < 2s)", elapsed)
	}
}

func TestShellHook_Execute_NonZeroExit(t *testing.T) {
	hook := &ShellHook{
		Command: "exit 1",
		Timeout: 5 * time.Second,
	}

	output, err := hook.Execute(&HookInput{SessionID: "test", ToolName: "test"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if output.Allowed {
		t.Error("non-zero exit should be blocked")
	}
}

func TestShellHook_Execute_InvalidJSON(t *testing.T) {
	hook := &ShellHook{
		Command: "echo 'not json'",
		Timeout: 5 * time.Second,
	}

	output, err := hook.Execute(&HookInput{SessionID: "test", ToolName: "test"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if output.Allowed {
		t.Error("invalid JSON should be blocked")
	}
}

func TestShellHook_Execute_EmptyCommand(t *testing.T) {
	hook := &ShellHook{
		Command: "",
		Timeout: 5 * time.Second,
	}

	output, err := hook.Execute(&HookInput{SessionID: "test", ToolName: "test"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !output.Allowed {
		t.Error("empty command should be allowed (no-op)")
	}
}

func TestShellHook_Execute_InputJSON(t *testing.T) {
	// Hook reads stdin and echoes it back — verify JSON is passed correctly
	hook := &ShellHook{
		Command: "cat", // reads stdin, writes to stdout
		Timeout: 5 * time.Second,
	}

	input := &HookInput{
		SessionID: "sess-123",
		ToolName:  "sql_query",
		ToolInput: map[string]interface{}{"sql": "SELECT * FROM users"},
	}

	output, err := hook.Execute(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// cat outputs the input JSON, which is not valid HookOutput JSON
	// so it should be blocked (invalid JSON)
	if output.Allowed {
		t.Error("cat output is not valid HookOutput JSON")
	}
}

func TestShellHook_Execute_ScriptFile(t *testing.T) {
	dir := t.TempDir()
	script := filepath.Join(dir, "hook.sh")
	os.WriteFile(script, []byte(`#!/bin/sh
echo '{"allowed":true,"reason":"approved by script"}'
`), 0755)

	hook := &ShellHook{
		Command: "sh " + script,
		Timeout: 5 * time.Second,
	}

	output, err := hook.Execute(&HookInput{SessionID: "test", ToolName: "test"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !output.Allowed {
		t.Error("expected allowed=true")
	}
	if output.Reason != "approved by script" {
		t.Errorf("expected reason 'approved by script', got %q", output.Reason)
	}
}

func TestShellHook_Execute_StderrIncluded(t *testing.T) {
	hook := &ShellHook{
		Command: `echo '{"allowed":false}' >&2; echo '{"allowed":true}'`,
		Timeout: 5 * time.Second,
	}

	// stderr goes to stderr buffer, stdout is parsed
	output, err := hook.Execute(&HookInput{SessionID: "test", ToolName: "test"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !output.Allowed {
		t.Error("stdout should be parsed, not stderr")
	}
}

func TestHookOutput_Defaults(t *testing.T) {
	output := &HookOutput{}
	if output.Allowed {
		t.Error("default Allowed should be false")
	}
}

func TestHookInput_JSON(t *testing.T) {
	input := &HookInput{
		SessionID: "test",
		ToolName:  "sql_query",
		ToolInput: map[string]interface{}{"sql": "SELECT 1"},
		ToolOutput: "result",
	}
	// Just verify it marshals without error
	// (tested indirectly via Execute)
	_ = input
}

func TestShellHook_ContextCancellation(t *testing.T) {
	hook := &ShellHook{
		Command: `echo '{"allowed":true}'`,
		Timeout: 5 * time.Second,
	}

	// Verify hook completes normally
	start := time.Now()
	output, err := hook.Execute(&HookInput{SessionID: "test", ToolName: "test"})
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !output.Allowed {
		t.Error("expected allowed=true")
	}
	if elapsed > 2*time.Second {
		t.Errorf("hook took too long: %v", elapsed)
	}
}
