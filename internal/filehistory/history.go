// Package filehistory provides file history tracking and undo capabilities.
package filehistory

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

// Snapshot represents a file snapshot at a point in time
type Snapshot struct {
	ID          string    `json:"id"`
	FilePath    string    `json:"file_path"`
	Content     string    `json:"content"`
	Checksum    string    `json:"checksum"`
	Timestamp   time.Time `json:"timestamp"`
	Operation   string    `json:"operation"`
	Description string    `json:"description,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// History represents the complete history of a file
type History struct {
	FilePath  string     `json:"file_path"`
	Snapshots []Snapshot `json:"snapshots"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
}

// Manager manages file histories
type Manager struct {
	histories map[string]*History
	maxSnippets int
	mu         sync.RWMutex
	storageDir string
}

// NewManager creates a new history manager
func NewManager(options ...Option) *Manager {
	m := &Manager{
		histories:   make(map[string]*History),
		maxSnippets: 100, // Default max snapshots per file
	}

	for _, opt := range options {
		opt(m)
	}

	return m
}

// Option is a functional option for Manager
type Option func(*Manager)

// WithMaxSnapshots sets the maximum number of snapshots per file
func WithMaxSnapshots(max int) Option {
	return func(m *Manager) {
		m.maxSnippets = max
	}
}

// WithStorageDir sets the storage directory for persisting histories
func WithStorageDir(dir string) Option {
	return func(m *Manager) {
		m.storageDir = dir
	}
}

// Record records a new snapshot for a file
func (m *Manager) Record(filePath, content, operation, description string) (*Snapshot, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Normalize file path
	filePath = filepath.Clean(filePath)

	// Calculate checksum
	checksum := calculateChecksum(content)

	// Create snapshot
	snapshot := Snapshot{
		ID:          generateSnapshotID(),
		FilePath:    filePath,
		Content:     content,
		Checksum:    checksum,
		Timestamp:   time.Now(),
		Operation:   operation,
		Description: description,
	}

	// Get or create history
	history, exists := m.histories[filePath]
	if !exists {
		history = &History{
			FilePath:  filePath,
			Snapshots: []Snapshot{},
			CreatedAt: time.Now(),
		}
		m.histories[filePath] = history
	}

	// Check if content changed (skip if same as last)
	if len(history.Snapshots) > 0 {
		last := history.Snapshots[len(history.Snapshots)-1]
		if last.Checksum == checksum {
			return &last, nil // No change
		}
	}

	// Add snapshot
	history.Snapshots = append(history.Snapshots, snapshot)
	history.UpdatedAt = time.Now()

	// Trim old snapshots if needed
	if len(history.Snapshots) > m.maxSnippets {
		history.Snapshots = history.Snapshots[len(history.Snapshots)-m.maxSnippets:]
	}

	return &snapshot, nil
}

// GetHistory returns the history for a file
func (m *Manager) GetHistory(filePath string) (*History, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	filePath = filepath.Clean(filePath)
	history, exists := m.histories[filePath]
	if !exists {
		return nil, fmt.Errorf("no history found for file: %s", filePath)
	}

	return history, nil
}

// GetSnapshot returns a specific snapshot
func (m *Manager) GetSnapshot(filePath, snapshotID string) (*Snapshot, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	history, exists := m.histories[filepath.Clean(filePath)]
	if !exists {
		return nil, fmt.Errorf("no history found for file: %s", filePath)
	}

	for i := range history.Snapshots {
		if history.Snapshots[i].ID == snapshotID {
			return &history.Snapshots[i], nil
		}
	}

	return nil, fmt.Errorf("snapshot not found: %s", snapshotID)
}

// GetLatest returns the latest snapshot for a file
func (m *Manager) GetLatest(filePath string) (*Snapshot, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	history, exists := m.histories[filepath.Clean(filePath)]
	if !exists || len(history.Snapshots) == 0 {
		return nil, fmt.Errorf("no snapshots found for file: %s", filePath)
	}

	return &history.Snapshots[len(history.Snapshots)-1], nil
}

// Undo reverts to the previous snapshot
func (m *Manager) Undo(filePath string) (*Snapshot, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	history, exists := m.histories[filepath.Clean(filePath)]
	if !exists || len(history.Snapshots) < 2 {
		return nil, fmt.Errorf("cannot undo: not enough history")
	}

	// Remove the last snapshot
	lastIdx := len(history.Snapshots) - 1
	history.Snapshots = history.Snapshots[:lastIdx]
	history.UpdatedAt = time.Now()

	// Return the new latest
	return &history.Snapshots[len(history.Snapshots)-1], nil
}

// UndoTo reverts to a specific snapshot
func (m *Manager) UndoTo(filePath, snapshotID string) (*Snapshot, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	history, exists := m.histories[filepath.Clean(filePath)]
	if !exists {
		return nil, fmt.Errorf("no history found for file: %s", filePath)
	}

	// Find the snapshot
	targetIdx := -1
	for i, s := range history.Snapshots {
		if s.ID == snapshotID {
			targetIdx = i
			break
		}
	}

	if targetIdx == -1 {
		return nil, fmt.Errorf("snapshot not found: %s", snapshotID)
	}

	// Truncate to target (inclusive)
	history.Snapshots = history.Snapshots[:targetIdx+1]
	history.UpdatedAt = time.Now()

	return &history.Snapshots[len(history.Snapshots)-1], nil
}

// Redo re-applies a previously undone change (if available)
func (m *Manager) Redo(filePath string) (*Snapshot, error) {
	// In this implementation, redo is not directly supported
	// because we truncate history on undo. A full implementation
	// would maintain a separate redo stack.
	return nil, fmt.Errorf("redo not supported in this implementation")
}

// ListFiles returns all files with history
func (m *Manager) ListFiles() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	files := make([]string, 0, len(m.histories))
	for path := range m.histories {
		files = append(files, path)
	}
	sort.Strings(files)
	return files
}

// ClearHistory clears the history for a file
func (m *Manager) ClearHistory(filePath string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	filePath = filepath.Clean(filePath)
	delete(m.histories, filePath)
	return nil
}

// ClearAll clears all histories
func (m *Manager) ClearAll() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.histories = make(map[string]*History)
}

// Diff returns the diff between two snapshots
func (m *Manager) Diff(filePath, fromID, toID string) (*Diff, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	history, exists := m.histories[filepath.Clean(filePath)]
	if !exists {
		return nil, fmt.Errorf("no history found for file: %s", filePath)
	}

	var from, to *Snapshot
	for i := range history.Snapshots {
		if history.Snapshots[i].ID == fromID {
			from = &history.Snapshots[i]
		}
		if history.Snapshots[i].ID == toID {
			to = &history.Snapshots[i]
		}
	}

	if from == nil || to == nil {
		return nil, fmt.Errorf("one or both snapshots not found")
	}

	return computeDiff(from, to), nil
}

// Diff represents the difference between two snapshots
type Diff struct {
	FromID    string     `json:"from_id"`
	ToID      string     `json:"to_id"`
	FromTime  time.Time  `json:"from_time"`
	ToTime    time.Time  `json:"to_time"`
	Changes   []DiffLine `json:"changes"`
	Additions int        `json:"additions"`
	Deletions int        `json:"deletions"`
}

// DiffLine represents a line in a diff
type DiffLine struct {
	Type     string `json:"type"` // "add", "delete", "context"
	Content  string `json:"content"`
	OldLine  int    `json:"old_line,omitempty"`
	NewLine  int    `json:"new_line,omitempty"`
}

// computeDiff computes the diff between two snapshots
func computeDiff(from, to *Snapshot) *Diff {
	fromLines := strings.Split(from.Content, "\n")
	toLines := strings.Split(to.Content, "\n")

	diff := &Diff{
		FromID:   from.ID,
		ToID:     to.ID,
		FromTime: from.Timestamp,
		ToTime:   to.Timestamp,
		Changes:  []DiffLine{},
	}

	// Simple line-by-line diff (a more sophisticated implementation would use LCS)
	oldIdx, newIdx := 0, 0

	for oldIdx < len(fromLines) || newIdx < len(toLines) {
		if oldIdx >= len(fromLines) {
			// Remaining lines are additions
			diff.Changes = append(diff.Changes, DiffLine{
				Type:    "add",
				Content: toLines[newIdx],
				NewLine: newIdx + 1,
			})
			diff.Additions++
			newIdx++
		} else if newIdx >= len(toLines) {
			// Remaining lines are deletions
			diff.Changes = append(diff.Changes, DiffLine{
				Type:    "delete",
				Content: fromLines[oldIdx],
				OldLine: oldIdx + 1,
			})
			diff.Deletions++
			oldIdx++
		} else if fromLines[oldIdx] == toLines[newIdx] {
			// Lines match
			diff.Changes = append(diff.Changes, DiffLine{
				Type:     "context",
				Content:  fromLines[oldIdx],
				OldLine:  oldIdx + 1,
				NewLine:  newIdx + 1,
			})
			oldIdx++
			newIdx++
		} else {
			// Lines differ - check if it's an addition or deletion
			// Simple heuristic: prefer addition
			diff.Changes = append(diff.Changes, DiffLine{
				Type:    "add",
				Content: toLines[newIdx],
				NewLine: newIdx + 1,
			})
			diff.Additions++
			newIdx++
		}
	}

	return diff
}

// Save saves histories to disk
func (m *Manager) Save() error {
	if m.storageDir == "" {
		return nil
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	// Ensure directory exists
	if err := os.MkdirAll(m.storageDir, 0755); err != nil {
		return fmt.Errorf("failed to create storage directory: %w", err)
	}

	// Save each history
	for path, history := range m.histories {
		data, err := json.MarshalIndent(history, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal history: %w", err)
		}

		filename := filepath.Join(m.storageDir, strings.ReplaceAll(path, "/", "_")+".json")
		if err := ioutil.WriteFile(filename, data, 0644); err != nil {
			return fmt.Errorf("failed to write history file: %w", err)
		}
	}

	return nil
}

// Load loads histories from disk
func (m *Manager) Load() error {
	if m.storageDir == "" {
		return nil
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	files, err := filepath.Glob(filepath.Join(m.storageDir, "*.json"))
	if err != nil {
		return fmt.Errorf("failed to list history files: %w", err)
	}

	for _, file := range files {
		data, err := ioutil.ReadFile(file)
		if err != nil {
			continue // Skip files that can't be read
		}

		var history History
		if err := json.Unmarshal(data, &history); err != nil {
			continue // Skip invalid files
		}

		m.histories[history.FilePath] = &history
	}

	return nil
}

// FileEditWithHistory wraps file editing with history tracking
type FileEditWithHistory struct {
	manager *Manager
}

// NewFileEditWithHistory creates a new FileEditWithHistory
func NewFileEditWithHistory(manager *Manager) *FileEditWithHistory {
	return &FileEditWithHistory{manager: manager}
}

// Edit performs an edit and records it in history
func (e *FileEditWithHistory) Edit(filePath, oldContent, newContent, description string) (*Snapshot, error) {
	// Record the current state first (if file exists)
	if oldContent != "" {
		e.manager.Record(filePath, oldContent, "before_edit", "State before: "+description)
	}

	// Record the new state
	return e.manager.Record(filePath, newContent, "edit", description)
}

// UndoEdit undoes the last edit and returns the previous content
func (e *FileEditWithHistory) UndoEdit(filePath string) (string, error) {
	snapshot, err := e.manager.Undo(filePath)
	if err != nil {
		return "", err
	}
	return snapshot.Content, nil
}

// GetPreviousVersion returns the previous version of a file
func (e *FileEditWithHistory) GetPreviousVersion(filePath string) (string, error) {
	m := e.manager

	m.mu.RLock()
	defer m.mu.RUnlock()

	history, exists := m.histories[filepath.Clean(filePath)]
	if !exists || len(history.Snapshots) < 2 {
		return "", fmt.Errorf("no previous version available")
	}

	return history.Snapshots[len(history.Snapshots)-2].Content, nil
}

// Helper functions

func generateSnapshotID() string {
	return fmt.Sprintf("snap-%d", time.Now().UnixNano())
}

func calculateChecksum(content string) string {
	hash := sha256.Sum256([]byte(content))
	return hex.EncodeToString(hash[:])
}
