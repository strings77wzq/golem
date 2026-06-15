package session

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/strings77wzq/golem/core/providers"
)

func TestSessionSaveAndLoad(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	adapter, err := NewSQLiteAdapter(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer adapter.Close()

	// Create and save a session
	sess := NewSession("test-session")
	sess.AddMessage(providers.Message{Role: providers.RoleUser, Content: "Hello"})
	sess.AddMessage(providers.Message{Role: providers.RoleAssistant, Content: "Hi there"})
	sess.AddMessage(providers.Message{Role: providers.RoleUser, Content: "What's the weather?"})

	if err := adapter.Save(sess); err != nil {
		t.Fatal(err)
	}

	// Load the session
	loaded, ok := adapter.Get("test-session")
	if !ok {
		t.Fatal("expected session to be found")
	}

	if len(loaded.Messages) != 3 {
		t.Errorf("expected 3 messages, got %d", len(loaded.Messages))
	}

	if loaded.Messages[0].Content != "Hello" {
		t.Errorf("expected first message 'Hello', got %q", loaded.Messages[0].Content)
	}

	if loaded.Messages[2].Content != "What's the weather?" {
		t.Errorf("expected third message 'What's the weather?', got %q", loaded.Messages[2].Content)
	}
}

func TestSessionTimestampUpdate(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	adapter, err := NewSQLiteAdapter(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer adapter.Close()

	sess := NewSession("timestamp-test")
	sess.AddMessage(providers.Message{Role: providers.RoleUser, Content: "First"})
	
	firstUpdate := sess.UpdatedAt
	time.Sleep(10 * time.Millisecond)

	sess.AddMessage(providers.Message{Role: providers.RoleAssistant, Content: "Second"})
	
	if !sess.UpdatedAt.After(firstUpdate) {
		t.Error("expected UpdatedAt to be updated after adding message")
	}

	if err := adapter.Save(sess); err != nil {
		t.Fatal(err)
	}

	// Load and verify timestamp preserved (SQLite has second-level precision)
	loaded, ok := adapter.Get("timestamp-test")
	if !ok {
		t.Fatal("expected session to be found")
	}

	// Compare at second-level precision (SQLite truncates nanoseconds)
	if loaded.UpdatedAt.Unix() != sess.UpdatedAt.Unix() {
		t.Errorf("expected UpdatedAt to be preserved after save/load, got %v want %v", loaded.UpdatedAt, sess.UpdatedAt)
	}
}

func TestSessionMessageOrderPreserved(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	adapter, err := NewSQLiteAdapter(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer adapter.Close()

	sess := NewSession("order-test")
	for i := 0; i < 10; i++ {
		role := providers.RoleUser
		if i%2 == 1 {
			role = providers.RoleAssistant
		}
		sess.AddMessage(providers.Message{Role: role, Content: string(rune('A' + i))})
	}

	if err := adapter.Save(sess); err != nil {
		t.Fatal(err)
	}

	loaded, ok := adapter.Get("order-test")
	if !ok {
		t.Fatal("expected session to be found")
	}

	if len(loaded.Messages) != 10 {
		t.Fatalf("expected 10 messages, got %d", len(loaded.Messages))
	}

	// Verify order
	for i, msg := range loaded.Messages {
		expected := string(rune('A' + i))
		if msg.Content != expected {
			t.Errorf("message %d: expected %q, got %q", i, expected, msg.Content)
		}
	}
}

func TestSessionDeleteAndList(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	adapter, err := NewSQLiteAdapter(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer adapter.Close()

	// Create 3 sessions
	for i := 0; i < 3; i++ {
		sess := NewSession(string(rune('a' + i)))
		sess.AddMessage(providers.Message{Role: providers.RoleUser, Content: "test"})
		if err := adapter.Save(sess); err != nil {
			t.Fatal(err)
		}
	}

	// List should show 3 sessions
	sessions := adapter.List()
	if len(sessions) != 3 {
		t.Errorf("expected 3 sessions, got %d", len(sessions))
	}

	// Delete one
	if err := adapter.Delete("b"); err != nil {
		t.Fatal(err)
	}

	// List should show 2 sessions
	sessions = adapter.List()
	if len(sessions) != 2 {
		t.Errorf("expected 2 sessions after delete, got %d", len(sessions))
	}

	// Get deleted should fail
	_, ok := adapter.Get("b")
	if ok {
		t.Error("expected deleted session to not be found")
	}
}
