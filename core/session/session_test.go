package session

import (
	"bytes"
	"sync"
	"testing"
	"time"

	"github.com/strings77wzq/golem/core/providers"
)

func TestNewSession(t *testing.T) {
	id := "test-session"
	session := NewSession(id)

	if session.ID != id {
		t.Errorf("expected ID %q, got %q", id, session.ID)
	}

	if len(session.Messages) != 0 {
		t.Errorf("expected 0 messages, got %d", len(session.Messages))
	}

	if session.CreatedAt.IsZero() {
		t.Error("expected CreatedAt to be set")
	}

	if session.UpdatedAt.IsZero() {
		t.Error("expected UpdatedAt to be set")
	}

	if !session.CreatedAt.Equal(session.UpdatedAt) {
		t.Error("expected CreatedAt and UpdatedAt to be equal for new session")
	}
}

func TestAddMessage(t *testing.T) {
	session := NewSession("test")

	msg1 := providers.Message{Role: providers.RoleUser, Content: "Hello"}
	msg2 := providers.Message{Role: providers.RoleAssistant, Content: "Hi"}

	session.AddMessage(msg1)
	if session.MessageCount() != 1 {
		t.Errorf("expected 1 message, got %d", session.MessageCount())
	}

	session.AddMessage(msg2)
	if session.MessageCount() != 2 {
		t.Errorf("expected 2 messages, got %d", session.MessageCount())
	}

	messages := session.GetMessages()
	if messages[0].Content != "Hello" {
		t.Errorf("expected first message 'Hello', got %q", messages[0].Content)
	}
	if messages[1].Content != "Hi" {
		t.Errorf("expected second message 'Hi', got %q", messages[1].Content)
	}
}

func TestGetMessages(t *testing.T) {
	session := NewSession("test")

	msg := providers.Message{Role: providers.RoleUser, Content: "Test"}
	session.AddMessage(msg)

	messages := session.GetMessages()
	messages[0].Content = "Modified"

	origMessages := session.GetMessages()
	if origMessages[0].Content != "Test" {
		t.Error("modifying returned slice should not affect session")
	}
}

func TestClear(t *testing.T) {
	session := NewSession("test")

	session.AddMessage(providers.Message{Role: providers.RoleUser, Content: "Hello"})
	session.AddMessage(providers.Message{Role: providers.RoleAssistant, Content: "Hi"})

	if session.MessageCount() != 2 {
		t.Errorf("expected 2 messages before clear, got %d", session.MessageCount())
	}

	oldUpdatedAt := session.UpdatedAt
	time.Sleep(10 * time.Millisecond)

	session.Clear()

	if session.MessageCount() != 0 {
		t.Errorf("expected 0 messages after clear, got %d", session.MessageCount())
	}

	if !session.UpdatedAt.After(oldUpdatedAt) {
		t.Error("expected UpdatedAt to be updated after clear")
	}
}

func TestConcurrentAddMessage(t *testing.T) {
	session := NewSession("test")

	var wg sync.WaitGroup
	numGoroutines := 50

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			msg := providers.Message{
				Role:    providers.RoleUser,
				Content: "message",
			}
			session.AddMessage(msg)
		}(i)
	}

	wg.Wait()

	if session.MessageCount() != numGoroutines {
		t.Errorf("expected %d messages, got %d", numGoroutines, session.MessageCount())
	}
}

func TestExport(t *testing.T) {
	s := NewSession("test-session")
	s.AddMessage(providers.Message{Role: providers.RoleUser, Content: "Hello"})
	s.AddMessage(providers.Message{Role: providers.RoleAssistant, Content: "Hi there!"})

	exportData := s.Export()

	if exportData.Version != "1.0" {
		t.Errorf("expected version '1.0', got %q", exportData.Version)
	}
	if exportData.Session.ID != "test-session" {
		t.Errorf("expected session ID 'test-session', got %q", exportData.Session.ID)
	}
	if len(exportData.Session.Messages) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(exportData.Session.Messages))
	}
	if exportData.Session.Messages[0].Content != "Hello" {
		t.Errorf("expected first message 'Hello', got %q", exportData.Session.Messages[0].Content)
	}
}

func TestExportToJSON(t *testing.T) {
	s := NewSession("json-test")
	s.AddMessage(providers.Message{Role: providers.RoleUser, Content: "Test"})

	jsonData, err := s.ExportToJSON()
	if err != nil {
		t.Fatalf("ExportToJSON failed: %v", err)
	}

	if len(jsonData) == 0 {
		t.Fatal("expected non-empty JSON")
	}

	if !bytes.Contains(jsonData, []byte("json-test")) {
		t.Error("expected JSON to contain session ID")
	}
}

func TestImportFromJSON(t *testing.T) {
	original := NewSession("import-test")
	original.AddMessage(providers.Message{Role: providers.RoleUser, Content: "Original message"})

	jsonData, err := original.ExportToJSON()
	if err != nil {
		t.Fatalf("ExportToJSON failed: %v", err)
	}

	imported, err := ImportFromJSON(jsonData)
	if err != nil {
		t.Fatalf("ImportFromJSON failed: %v", err)
	}

	if imported.ID != "import-test" {
		t.Errorf("expected ID 'import-test', got %q", imported.ID)
	}
	if imported.MessageCount() != 1 {
		t.Fatalf("expected 1 message, got %d", imported.MessageCount())
	}
	if imported.GetMessages()[0].Content != "Original message" {
		t.Errorf("expected message 'Original message', got %q", imported.GetMessages()[0].Content)
	}
}

func TestImportFromJSONInvalid(t *testing.T) {
	_, err := ImportFromJSON([]byte("invalid json"))
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestImportFromJSONMissingVersion(t *testing.T) {
	jsonData := `{"session": {"id": "test"}}`
	_, err := ImportFromJSON([]byte(jsonData))
	if err == nil {
		t.Fatal("expected error for missing version")
	}
}

func TestImportFromJSONMissingID(t *testing.T) {
	jsonData := `{"version": "1.0", "session": {}}`
	_, err := ImportFromJSON([]byte(jsonData))
	if err == nil {
		t.Fatal("expected error for missing session ID")
	}
}
