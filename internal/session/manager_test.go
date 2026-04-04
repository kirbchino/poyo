package session

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestNewSessionManager(t *testing.T) {
	// Create temp directory for testing
	tmpDir := t.TempDir()
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", originalHome)

	mgr, err := NewSessionManager()
	if err != nil {
		t.Fatalf("NewSessionManager() error: %v", err)
	}

	if mgr == nil {
		t.Fatal("NewSessionManager() returned nil")
	}

	// Check that sessions directory was created
	sessionsDir := filepath.Join(tmpDir, ".claude", "sessions")
	if _, err := os.Stat(sessionsDir); os.IsNotExist(err) {
		t.Errorf("Sessions directory not created: %s", sessionsDir)
	}
}

func TestNewSession(t *testing.T) {
	mgr := &SessionManager{
		sessionsDir: t.TempDir(),
		autoSave:    false,
	}

	session := mgr.NewSession("/test/path", "claude-sonnet-4")

	if session == nil {
		t.Fatal("NewSession() returned nil")
	}

	if session.WorkingDir != "/test/path" {
		t.Errorf("WorkingDir = %q, want %q", session.WorkingDir, "/test/path")
	}

	if session.Model != "claude-sonnet-4" {
		t.Errorf("Model = %q, want %q", session.Model, "claude-sonnet-4")
	}

	if len(session.Messages) != 0 {
		t.Errorf("Messages = %d, want 0", len(session.Messages))
	}

	if session.Config.PermissionMode != "default" {
		t.Errorf("PermissionMode = %q, want %q", session.Config.PermissionMode, "default")
	}
}

func TestAddMessage(t *testing.T) {
	mgr := &SessionManager{
		sessionsDir: t.TempDir(),
		autoSave:    false,
	}

	session := mgr.NewSession("/test", "test-model")
	mgr.current = session

	msg := Message{
		ID:        "msg_1",
		Role:      "user",
		Content:   "Hello",
		Timestamp: time.Now(),
	}

	err := mgr.AddMessage(msg)
	if err != nil {
		t.Fatalf("AddMessage() error: %v", err)
	}

	if len(session.Messages) != 1 {
		t.Errorf("Messages count = %d, want 1", len(session.Messages))
	}

	if session.Messages[0].Content != "Hello" {
		t.Errorf("Message content = %q, want %q", session.Messages[0].Content, "Hello")
	}
}

func TestSaveAndLoadSession(t *testing.T) {
	mgr := &SessionManager{
		sessionsDir: t.TempDir(),
		autoSave:    false,
	}

	// Create and save a session
	session := mgr.NewSession("/test", "test-model")
	session.Messages = append(session.Messages, Message{
		ID:        "msg_1",
		Role:      "user",
		Content:   "Test message",
		Timestamp: time.Now(),
	})
	mgr.current = session

	err := mgr.SaveSession()
	if err != nil {
		t.Fatalf("SaveSession() error: %v", err)
	}

	// Load the session
	loaded, err := mgr.LoadSession(session.ID)
	if err != nil {
		t.Fatalf("LoadSession() error: %v", err)
	}

	if loaded.ID != session.ID {
		t.Errorf("ID = %q, want %q", loaded.ID, session.ID)
	}

	if len(loaded.Messages) != 1 {
		t.Errorf("Messages count = %d, want 1", len(loaded.Messages))
	}
}

func TestListSessions(t *testing.T) {
	mgr := &SessionManager{
		sessionsDir: t.TempDir(),
		autoSave:    false,
	}

	// Create multiple sessions
	for i := 0; i < 3; i++ {
		session := mgr.NewSession("/test", "test-model")
		session.Messages = append(session.Messages, Message{
			ID:        "msg_" + string(rune('1'+i)),
			Role:      "user",
			Content:   "Test",
			Timestamp: time.Now(),
		})
		mgr.current = session
		mgr.SaveSession()
		mgr.current = nil
	}

	sessions, err := mgr.ListSessions()
	if err != nil {
		t.Fatalf("ListSessions() error: %v", err)
	}

	if len(sessions) != 3 {
		t.Errorf("Sessions count = %d, want 3", len(sessions))
	}
}

func TestDeleteSession(t *testing.T) {
	mgr := &SessionManager{
		sessionsDir: t.TempDir(),
		autoSave:    false,
	}

	session := mgr.NewSession("/test", "test-model")
	mgr.current = session
	mgr.SaveSession()

	// Delete the session
	err := mgr.DeleteSession(session.ID)
	if err != nil {
		t.Fatalf("DeleteSession() error: %v", err)
	}

	// Try to load deleted session
	_, err = mgr.LoadSession(session.ID)
	if err == nil {
		t.Error("LoadSession() should fail for deleted session")
	}
}

func TestGenerateSummary(t *testing.T) {
	mgr := &SessionManager{}

	messages := []Message{
		{
			Role:    "user",
			Content: "What is the weather like?",
		},
		{
			Role:    "assistant",
			Content: "I can help you check the weather.",
			ToolCalls: []ToolCall{
				{Name: "WebSearch", Input: map[string]interface{}{"query": "weather"}},
			},
		},
		{
			Role:    "user",
			Content: "Write a function to calculate Fibonacci",
		},
		{
			Role:    "assistant",
			Content: "我建议使用递归方式实现。Here is the code...",
		},
	}

	summary := mgr.generateSummary(messages)

	if summary == "" {
		t.Error("generateSummary() returned empty string")
	}

	// Should contain topic information
	if len(summary) < 50 {
		t.Errorf("Summary too short: %q", summary)
	}
}

func TestCompact(t *testing.T) {
	mgr := &SessionManager{
		sessionsDir: t.TempDir(),
		autoSave:    false,
	}

	session := &Session{
		ID:        "test_session",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Model:     "test-model",
	}

	// Add 20 messages
	for i := 0; i < 20; i++ {
		session.Messages = append(session.Messages, Message{
			ID:        generateMessageID(),
			Role:      "user",
			Content:   "Message " + string(rune('0'+i)),
			Timestamp: time.Now(),
		})
	}

	mgr.current = session

	// Compact to keep 10 messages
	err := mgr.Compact(10)
	if err != nil {
		t.Fatalf("Compact() error: %v", err)
	}

	// Should have summary + 10 recent messages
	if len(session.Messages) > 11 {
		t.Errorf("Messages count = %d, want <= 11", len(session.Messages))
	}
}

func TestExportMarkdown(t *testing.T) {
	mgr := &SessionManager{
		sessionsDir: t.TempDir(),
		autoSave:    false,
	}

	session := &Session{
		ID:        "test_session",
		CreatedAt: time.Now().Truncate(time.Second),
		Model:     "test-model",
		Messages: []Message{
			{ID: "1", Role: "user", Content: "Hello", Timestamp: time.Now()},
			{ID: "2", Role: "assistant", Content: "Hi there!", Timestamp: time.Now()},
		},
	}

	outputPath := filepath.Join(t.TempDir(), "export.md")
	err := mgr.exportMarkdown(session, outputPath)
	if err != nil {
		t.Fatalf("exportMarkdown() error: %v", err)
	}

	// Check file was created
	if _, err := os.Stat(outputPath); os.IsNotExist(err) {
		t.Fatal("Export file not created")
	}

	// Read and verify content
	content, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("ReadFile() error: %v", err)
	}

	if len(content) == 0 {
		t.Error("Export file is empty")
	}
}

func TestGetCurrentSession(t *testing.T) {
	mgr := &SessionManager{
		sessionsDir: t.TempDir(),
		autoSave:    false,
	}

	// No session initially
	if mgr.GetCurrentSession() != nil {
		t.Error("GetCurrentSession() should return nil when no session")
	}

	// Create a session
	session := mgr.NewSession("/test", "test-model")

	if mgr.GetCurrentSession() != session {
		t.Error("GetCurrentSession() should return current session")
	}
}

func TestSetAutoSave(t *testing.T) {
	mgr := &SessionManager{
		autoSave: true,
	}

	mgr.SetAutoSave(false)

	if mgr.autoSave {
		t.Error("SetAutoSave(false) did not disable auto-save")
	}
}

func TestTruncate(t *testing.T) {
	tests := []struct {
		input    string
		maxLen   int
		expected string
	}{
		{"short", 10, "short"},
		{"this is a long string", 10, "this is..."},
		{"exact", 5, "exact"},
		{"", 5, ""},
	}

	for _, tt := range tests {
		result := truncate(tt.input, tt.maxLen)
		if result != tt.expected {
			t.Errorf("truncate(%q, %d) = %q, want %q", tt.input, tt.maxLen, result, tt.expected)
		}
	}
}
