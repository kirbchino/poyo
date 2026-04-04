package filehistory

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestNewManager(t *testing.T) {
	m := NewManager()
	if m == nil {
		t.Fatal("NewManager() returned nil")
	}

	if m.histories == nil {
		t.Error("histories should be initialized")
	}

	if m.maxSnippets != 100 {
		t.Errorf("maxSnippets = %d, want 100", m.maxSnippets)
	}
}

func TestManagerWithOptions(t *testing.T) {
	m := NewManager(
		WithMaxSnapshots(50),
		WithStorageDir("/tmp/history"),
	)

	if m.maxSnippets != 50 {
		t.Errorf("maxSnippets = %d, want 50", m.maxSnippets)
	}

	if m.storageDir != "/tmp/history" {
		t.Errorf("storageDir = %q, want '/tmp/history'", m.storageDir)
	}
}

func TestManagerRecord(t *testing.T) {
	m := NewManager()

	snapshot, err := m.Record("/test/file.txt", "content", "edit", "test edit")
	if err != nil {
		t.Fatalf("Record() error: %v", err)
	}

	if snapshot == nil {
		t.Fatal("Record() returned nil snapshot")
	}

	if snapshot.ID == "" {
		t.Error("Snapshot should have ID")
	}

	if snapshot.FilePath != "/test/file.txt" {
		t.Errorf("FilePath = %q, want '/test/file.txt'", snapshot.FilePath)
	}

	if snapshot.Content != "content" {
		t.Errorf("Content = %q, want 'content'", snapshot.Content)
	}

	if snapshot.Checksum == "" {
		t.Error("Snapshot should have checksum")
	}

	if snapshot.Operation != "edit" {
		t.Errorf("Operation = %q, want 'edit'", snapshot.Operation)
	}
}

func TestManagerRecordSameContent(t *testing.T) {
	m := NewManager()

	// Record first snapshot
	m.Record("/test/file.txt", "content", "edit", "first")

	// Record same content again
	snapshot, err := m.Record("/test/file.txt", "content", "edit", "second")
	if err != nil {
		t.Fatalf("Record() error: %v", err)
	}

	// Should return the same snapshot (no new snapshot created)
	history, _ := m.GetHistory("/test/file.txt")
	if len(history.Snapshots) != 1 {
		t.Errorf("Should have 1 snapshot, got %d", len(history.Snapshots))
	}

	// Check it returns the first snapshot
	if snapshot.Description != "first" {
		t.Errorf("Should return first snapshot, got description: %q", snapshot.Description)
	}
}

func TestManagerRecordMaxSnapshots(t *testing.T) {
	m := NewManager(WithMaxSnapshots(5))

	// Record 10 snapshots
	for i := 0; i < 10; i++ {
		m.Record("/test/file.txt", "content"+string(rune('0'+i)), "edit", "edit")
	}

	history, _ := m.GetHistory("/test/file.txt")

	// Should only keep last 5
	if len(history.Snapshots) != 5 {
		t.Errorf("Snapshots count = %d, want 5", len(history.Snapshots))
	}
}

func TestManagerGetHistory(t *testing.T) {
	m := NewManager()

	// Record some snapshots
	m.Record("/test/file.txt", "content1", "edit", "edit 1")
	m.Record("/test/file.txt", "content2", "edit", "edit 2")

	history, err := m.GetHistory("/test/file.txt")
	if err != nil {
		t.Fatalf("GetHistory() error: %v", err)
	}

	if history.FilePath != "/test/file.txt" {
		t.Errorf("FilePath = %q, want '/test/file.txt'", history.FilePath)
	}

	if len(history.Snapshots) != 2 {
		t.Errorf("Snapshots count = %d, want 2", len(history.Snapshots))
	}

	// Get non-existent history
	_, err = m.GetHistory("/nonexistent/file.txt")
	if err == nil {
		t.Error("GetHistory() should return error for non-existent file")
	}
}

func TestManagerGetSnapshot(t *testing.T) {
	m := NewManager()

	s1, _ := m.Record("/test/file.txt", "content1", "edit", "edit 1")
	s2, _ := m.Record("/test/file.txt", "content2", "edit", "edit 2")

	// Get first snapshot
	snapshot, err := m.GetSnapshot("/test/file.txt", s1.ID)
	if err != nil {
		t.Fatalf("GetSnapshot() error: %v", err)
	}

	if snapshot.Content != "content1" {
		t.Errorf("Content = %q, want 'content1'", snapshot.Content)
	}

	// Get second snapshot
	snapshot, err = m.GetSnapshot("/test/file.txt", s2.ID)
	if err != nil {
		t.Fatalf("GetSnapshot() error: %v", err)
	}

	if snapshot.Content != "content2" {
		t.Errorf("Content = %q, want 'content2'", snapshot.Content)
	}

	// Get non-existent snapshot
	_, err = m.GetSnapshot("/test/file.txt", "nonexistent")
	if err == nil {
		t.Error("GetSnapshot() should return error for non-existent ID")
	}
}

func TestManagerGetLatest(t *testing.T) {
	m := NewManager()

	m.Record("/test/file.txt", "content1", "edit", "edit 1")
	m.Record("/test/file.txt", "content2", "edit", "edit 2")
	s3, _ := m.Record("/test/file.txt", "content3", "edit", "edit 3")

	latest, err := m.GetLatest("/test/file.txt")
	if err != nil {
		t.Fatalf("GetLatest() error: %v", err)
	}

	if latest.ID != s3.ID {
		t.Error("GetLatest() should return the most recent snapshot")
	}

	// Get latest for non-existent file
	_, err = m.GetLatest("/nonexistent/file.txt")
	if err == nil {
		t.Error("GetLatest() should return error for non-existent file")
	}
}

func TestManagerUndo(t *testing.T) {
	m := NewManager()

	m.Record("/test/file.txt", "content1", "edit", "edit 1")
	m.Record("/test/file.txt", "content2", "edit", "edit 2")

	// Undo should revert to content1
	snapshot, err := m.Undo("/test/file.txt")
	if err != nil {
		t.Fatalf("Undo() error: %v", err)
	}

	if snapshot.Content != "content1" {
		t.Errorf("After undo, content = %q, want 'content1'", snapshot.Content)
	}

	// History should now have only 1 snapshot
	history, _ := m.GetHistory("/test/file.txt")
	if len(history.Snapshots) != 1 {
		t.Errorf("After undo, snapshots count = %d, want 1", len(history.Snapshots))
	}

	// Cannot undo with only 1 snapshot
	_, err = m.Undo("/test/file.txt")
	if err == nil {
		t.Error("Undo() should fail with only 1 snapshot")
	}
}

func TestManagerUndoTo(t *testing.T) {
	m := NewManager()

	s1, _ := m.Record("/test/file.txt", "content1", "edit", "edit 1")
	m.Record("/test/file.txt", "content2", "edit", "edit 2")
	m.Record("/test/file.txt", "content3", "edit", "edit 3")

	// Undo to first snapshot
	snapshot, err := m.UndoTo("/test/file.txt", s1.ID)
	if err != nil {
		t.Fatalf("UndoTo() error: %v", err)
	}

	if snapshot.Content != "content1" {
		t.Errorf("After undoTo, content = %q, want 'content1'", snapshot.Content)
	}

	// History should now have only 1 snapshot
	history, _ := m.GetHistory("/test/file.txt")
	if len(history.Snapshots) != 1 {
		t.Errorf("After undoTo, snapshots count = %d, want 1", len(history.Snapshots))
	}
}

func TestManagerListFiles(t *testing.T) {
	m := NewManager()

	m.Record("/test/file1.txt", "content1", "edit", "edit")
	m.Record("/test/file2.txt", "content2", "edit", "edit")
	m.Record("/test/file3.txt", "content3", "edit", "edit")

	files := m.ListFiles()

	if len(files) != 3 {
		t.Errorf("ListFiles() count = %d, want 3", len(files))
	}

	// Files should be sorted
	for i := 1; i < len(files); i++ {
		if files[i] < files[i-1] {
			t.Error("ListFiles() should return sorted files")
		}
	}
}

func TestManagerClearHistory(t *testing.T) {
	m := NewManager()

	m.Record("/test/file.txt", "content", "edit", "edit")

	err := m.ClearHistory("/test/file.txt")
	if err != nil {
		t.Fatalf("ClearHistory() error: %v", err)
	}

	_, err = m.GetHistory("/test/file.txt")
	if err == nil {
		t.Error("GetHistory() should return error after clearing")
	}
}

func TestManagerClearAll(t *testing.T) {
	m := NewManager()

	m.Record("/test/file1.txt", "content", "edit", "edit")
	m.Record("/test/file2.txt", "content", "edit", "edit")

	m.ClearAll()

	if len(m.ListFiles()) != 0 {
		t.Error("ClearAll() should remove all histories")
	}
}

func TestManagerDiff(t *testing.T) {
	m := NewManager()

	s1, _ := m.Record("/test/file.txt", "line1\nline2\nline3", "edit", "edit 1")
	s2, _ := m.Record("/test/file.txt", "line1\nline2 modified\nline3\nline4", "edit", "edit 2")

	diff, err := m.Diff("/test/file.txt", s1.ID, s2.ID)
	if err != nil {
		t.Fatalf("Diff() error: %v", err)
	}

	if diff.FromID != s1.ID {
		t.Error("Diff FromID mismatch")
	}

	if diff.ToID != s2.ID {
		t.Error("Diff ToID mismatch")
	}

	if len(diff.Changes) == 0 {
		t.Error("Diff should have changes")
	}
}

func TestManagerSaveLoad(t *testing.T) {
	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "history-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create and save
	m1 := NewManager(WithStorageDir(tmpDir))
	m1.Record("/test/file.txt", "content", "edit", "test edit")

	if err := m1.Save(); err != nil {
		t.Fatalf("Save() error: %v", err)
	}

	// Create new manager and load
	m2 := NewManager(WithStorageDir(tmpDir))
	if err := m2.Load(); err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	// Verify history was loaded
	history, err := m2.GetHistory("/test/file.txt")
	if err != nil {
		t.Fatalf("GetHistory() error after load: %v", err)
	}

	if len(history.Snapshots) != 1 {
		t.Errorf("After load, snapshots count = %d, want 1", len(history.Snapshots))
	}
}

func TestFileEditWithHistory(t *testing.T) {
	m := NewManager()
	editor := NewFileEditWithHistory(m)

	// Perform edit
	snapshot, err := editor.Edit("/test/file.txt", "old content", "new content", "test edit")
	if err != nil {
		t.Fatalf("Edit() error: %v", err)
	}

	if snapshot.Content != "new content" {
		t.Errorf("Content = %q, want 'new content'", snapshot.Content)
	}

	// Check history
	history, _ := m.GetHistory("/test/file.txt")
	if len(history.Snapshots) != 2 {
		t.Errorf("Should have 2 snapshots (before + after), got %d", len(history.Snapshots))
	}
}

func TestFileEditWithHistoryUndo(t *testing.T) {
	m := NewManager()
	editor := NewFileEditWithHistory(m)

	editor.Edit("/test/file.txt", "", "content1", "edit 1")
	editor.Edit("/test/file.txt", "content1", "content2", "edit 2")

	// Undo should return previous content
	content, err := editor.UndoEdit("/test/file.txt")
	if err != nil {
		t.Fatalf("UndoEdit() error: %v", err)
	}

	if content != "content1" {
		t.Errorf("After undo, content = %q, want 'content1'", content)
	}
}

func TestFileEditWithHistoryGetPreviousVersion(t *testing.T) {
	m := NewManager()
	editor := NewFileEditWithHistory(m)

	editor.Edit("/test/file.txt", "", "content1", "edit 1")
	editor.Edit("/test/file.txt", "content1", "content2", "edit 2")

	// Get previous version
	content, err := editor.GetPreviousVersion("/test/file.txt")
	if err != nil {
		t.Fatalf("GetPreviousVersion() error: %v", err)
	}

	if content != "content1" {
		t.Errorf("Previous content = %q, want 'content1'", content)
	}

	// Get previous version with only one snapshot
	editor.Edit("/test/single.txt", "", "only content", "edit")
	_, err = editor.GetPreviousVersion("/test/single.txt")
	if err == nil {
		t.Error("GetPreviousVersion() should fail with only one snapshot")
	}
}

func TestSnapshot(t *testing.T) {
	snapshot := Snapshot{
		ID:          "snap-123",
		FilePath:    "/test/file.txt",
		Content:     "test content",
		Checksum:    "abc123",
		Timestamp:   time.Now(),
		Operation:   "edit",
		Description: "test description",
		Metadata: map[string]interface{}{
			"key": "value",
		},
	}

	if snapshot.ID != "snap-123" {
		t.Errorf("ID = %q, want 'snap-123'", snapshot.ID)
	}

	if snapshot.Operation != "edit" {
		t.Errorf("Operation = %q, want 'edit'", snapshot.Operation)
	}
}

func TestHistory(t *testing.T) {
	history := History{
		FilePath: "/test/file.txt",
		Snapshots: []Snapshot{
			{ID: "snap-1", Content: "content1"},
			{ID: "snap-2", Content: "content2"},
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if history.FilePath != "/test/file.txt" {
		t.Errorf("FilePath = %q", history.FilePath)
	}

	if len(history.Snapshots) != 2 {
		t.Errorf("Snapshots count = %d", len(history.Snapshots))
	}
}

func TestDiffLine(t *testing.T) {
	lines := []DiffLine{
		{Type: "add", Content: "new line", NewLine: 1},
		{Type: "delete", Content: "old line", OldLine: 1},
		{Type: "context", Content: "same line", OldLine: 2, NewLine: 2},
	}

	for _, line := range lines {
		if line.Content == "" {
			t.Error("DiffLine content should not be empty")
		}
	}
}

func TestCalculateChecksum(t *testing.T) {
	content1 := "test content"
	content2 := "test content"
	content3 := "different content"

	checksum1 := calculateChecksum(content1)
	checksum2 := calculateChecksum(content2)
	checksum3 := calculateChecksum(content3)

	if checksum1 != checksum2 {
		t.Error("Same content should produce same checksum")
	}

	if checksum1 == checksum3 {
		t.Error("Different content should produce different checksum")
	}

	if len(checksum1) != 64 { // SHA-256 produces 64 hex chars
		t.Errorf("Checksum length = %d, want 64", len(checksum1))
	}
}

func TestComputeDiff(t *testing.T) {
	from := &Snapshot{
		ID:        "snap-1",
		Content:   "line1\nline2\nline3",
		Timestamp: time.Now(),
	}

	to := &Snapshot{
		ID:        "snap-2",
		Content:   "line1\nline2 modified\nline3\nline4",
		Timestamp: time.Now(),
	}

	diff := computeDiff(from, to)

	if diff.FromID != "snap-1" {
		t.Errorf("FromID = %q, want 'snap-1'", diff.FromID)
	}

	if diff.ToID != "snap-2" {
		t.Errorf("ToID = %q, want 'snap-2'", diff.ToID)
	}

	if diff.Additions == 0 {
		t.Error("Diff should have additions")
	}

	if len(diff.Changes) == 0 {
		t.Error("Diff should have changes")
	}
}

func TestHistoryJSON(t *testing.T) {
	history := History{
		FilePath: "/test/file.txt",
		Snapshots: []Snapshot{
			{
				ID:          "snap-1",
				FilePath:    "/test/file.txt",
				Content:     "content",
				Checksum:    "abc123",
				Timestamp:   time.Date(2026, 4, 3, 12, 0, 0, 0, time.UTC),
				Operation:   "edit",
				Description: "test",
			},
		},
		CreatedAt: time.Date(2026, 4, 3, 12, 0, 0, 0, time.UTC),
		UpdatedAt: time.Date(2026, 4, 3, 12, 0, 0, 0, time.UTC),
	}

	data, err := json.MarshalIndent(history, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal history: %v", err)
	}

	var parsed History
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("Failed to unmarshal history: %v", err)
	}

	if parsed.FilePath != history.FilePath {
		t.Errorf("Parsed FilePath = %q, want %q", parsed.FilePath, history.FilePath)
	}

	if len(parsed.Snapshots) != len(history.Snapshots) {
		t.Errorf("Parsed Snapshots count = %d, want %d", len(parsed.Snapshots), len(history.Snapshots))
	}
}

func TestManagerConcurrency(t *testing.T) {
	m := NewManager()
	done := make(chan bool, 100)

	// Concurrent records
	for i := 0; i < 50; i++ {
		go func(idx int) {
			m.Record("/test/file.txt", "content"+string(rune('0'+idx)), "edit", "edit")
			done <- true
		}(i)
	}

	// Concurrent reads
	for i := 0; i < 50; i++ {
		go func(idx int) {
			m.GetHistory("/test/file.txt")
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 100; i++ {
		<-done
	}

	// Should not have race conditions
	history, _ := m.GetHistory("/test/file.txt")
	if history == nil {
		t.Error("History should exist after concurrent operations")
	}
}
