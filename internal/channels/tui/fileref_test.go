package tui

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestResolveFileRefs_SimpleFile(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "test.txt", "hello world")

	m := Model{ctx: contextForTest(t)}
	text := m.resolveFileRefsWithDir("@test.txt", dir)

	if !strings.Contains(text, "hello world") {
		t.Errorf("expected file content in output, got: %s", text)
	}
	if !strings.Contains(text, "<file path=") {
		t.Errorf("expected <file> tag, got: %s", text)
	}
}

func TestResolveFileRefs_InlineFile(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "schema.sql", "CREATE TABLE t (id INT);")

	m := Model{ctx: contextForTest(t)}
	text := m.resolveFileRefsWithDir("分析 @schema.sql 表结构", dir)

	if !strings.Contains(text, "CREATE TABLE") {
		t.Errorf("expected file content, got: %s", text)
	}
	if !strings.Contains(text, "分析") {
		t.Errorf("expected prefix preserved, got: %s", text)
	}
	if !strings.Contains(text, "表结构") {
		t.Errorf("expected suffix preserved, got: %s", text)
	}
}

func TestResolveFileRefs_NonexistentFile(t *testing.T) {
	m := Model{ctx: contextForTest(t)}
	text := m.resolveFileRefs("@nope.txt")

	if text != "@nope.txt" {
		t.Errorf("expected original text preserved, got: %s", text)
	}
}

func TestResolveFileRefs_LargeFile(t *testing.T) {
	dir := t.TempDir()
	bigContent := strings.Repeat("x", 60000) // 60KB
	writeFile(t, dir, "big.txt", bigContent)

	m := Model{ctx: contextForTest(t)}
	text := m.resolveFileRefsWithDir("@big.txt", dir)

	t.Logf("output length: %d", len(text))
	t.Logf("output first 200: %q", text[:min(200, len(text))])
	t.Logf("output last 100: %q", text[max(0, len(text)-100):])

	if !strings.Contains(text, "truncated") {
		t.Errorf("expected truncation notice in output of length %d", len(text))
	}
	// Total length includes <file> tags + truncation notice, so > maxFileSize is OK
	// Key check: the xxx content should be truncated to 50KB
	trimmed := strings.TrimPrefix(text, "\n<file path=\"big.txt\" size=\"60000\">\n")
	trimmed = strings.TrimSuffix(trimmed, "\n[truncated — file too large]\n</file>\n")
	if len(trimmed) > maxFileSize {
		t.Errorf("expected file content truncated to %d chars, got %d", maxFileSize, len(trimmed))
	}
}

func TestResolveFileRefs_BinaryFile(t *testing.T) {
	dir := t.TempDir()
	// Write binary content with null bytes
	binaryContent := []byte{0x00, 0x01, 0x02, 0x00, 0x04}
	os.WriteFile(filepath.Join(dir, "image.png"), binaryContent, 0644)

	m := Model{ctx: contextForTest(t)}
	text := m.resolveFileRefsWithDir("@image.png", dir)

	if text != "@image.png" {
		t.Errorf("expected binary file skipped, got: %s", text)
	}
}

func TestResolveFileRefs_MultipleRefs(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "a.txt", "content A")
	writeFile(t, dir, "b.txt", "content B")

	m := Model{ctx: contextForTest(t)}
	text := m.resolveFileRefsWithDir("@a.txt and @b.txt", dir)

	if !strings.Contains(text, "content A") {
		t.Errorf("expected content A, got: %s", text)
	}
	if !strings.Contains(text, "content B") {
		t.Errorf("expected content B, got: %s", text)
	}
}

func TestResolveFileRefs_TildePath(t *testing.T) {
	home, _ := os.UserHomeDir()
	// Don't create actual file in home dir, just test path expansion
	m := Model{ctx: contextForTest(t)}
	text := m.resolveFileRefsWithDir("~/nonexistent_for_test.txt", home)

	// Should try to read from home dir, fail, preserve original
	if text != "~/nonexistent_for_test.txt" {
		t.Errorf("expected original text for missing tilde path, got: %s", text)
	}
}

func TestResolveFileRefs_NoAtSign(t *testing.T) {
	m := Model{ctx: contextForTest(t)}
	text := m.resolveFileRefs("just plain text")

	if text != "just plain text" {
		t.Errorf("expected unchanged text, got: %s", text)
	}
}

func TestResolveFileRefs_EmptyInput(t *testing.T) {
	m := Model{ctx: contextForTest(t)}
	text := m.resolveFileRefs("")

	if text != "" {
		t.Errorf("expected empty string, got: %s", text)
	}
}

func TestResolveFileRefs_RelativePath(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "sub/nested.txt", "nested content")

	m := Model{ctx: contextForTest(t)}
	text := m.resolveFileRefsWithDir("@sub/nested.txt", dir)

	if !strings.Contains(text, "nested content") {
		t.Errorf("expected nested content, got: %s", text)
	}
}

func TestIsBinaryFile(t *testing.T) {
	tests := []struct {
		name     string
		content  []byte
		expected bool
	}{
		{"text", []byte("hello world"), false},
		{"null bytes", []byte{0x00, 0x01, 0x02}, true},
		{"empty", []byte{}, false},
		{"json", []byte(`{"key":"value"}`), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isBinaryFile(tt.content); got != tt.expected {
				t.Errorf("isBinaryFile() = %v, want %v", got, tt.expected)
			}
		})
	}
}

// helpers

func writeFile(t *testing.T, dir, name, content string) {
	t.Helper()
	path := filepath.Join(dir, name)
	os.MkdirAll(filepath.Dir(path), 0755)
	os.WriteFile(path, []byte(content), 0644)
}

func contextForTest(t *testing.T) context.Context {
	t.Helper()
	return context.Background()
}
