package hooks

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

// FileWatcher monitors files for changes and triggers hooks
type FileWatcher struct {
	mu       sync.RWMutex
	watcher  *fsnotify.Watcher
	paths    map[string]bool
	handlers []FileChangeHandler
	ctx      context.Context
	cancel   context.CancelFunc
	running  bool
}

// FileChangeHandler handles file change events
type FileChangeHandler func(event FileChangeEvent)

// FileChangeEvent represents a file change event
type FileChangeEvent struct {
	Path      string
	Operation string // "create", "write", "remove", "rename", "chmod"
	Timestamp time.Time
}

// NewFileWatcher creates a new file watcher
func NewFileWatcher() (*FileWatcher, error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("failed to create file watcher: %w", err)
	}

	return &FileWatcher{
		watcher: watcher,
		paths:   make(map[string]bool),
	}, nil
}

// AddPath adds a path to watch
func (w *FileWatcher) AddPath(path string) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	// Resolve to absolute path
	absPath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("failed to resolve path: %w", err)
	}

	// Check if path exists
	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		return fmt.Errorf("path does not exist: %s", absPath)
	}

	// Add to watcher
	if err := w.watcher.Add(absPath); err != nil {
		return fmt.Errorf("failed to add path to watcher: %w", err)
	}

	w.paths[absPath] = true
	return nil
}

// RemovePath removes a path from watching
func (w *FileWatcher) RemovePath(path string) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	absPath, err := filepath.Abs(path)
	if err != nil {
		return err
	}

	if err := w.watcher.Remove(absPath); err != nil {
		return err
	}

	delete(w.paths, absPath)
	return nil
}

// RegisterHandler registers a file change handler
func (w *FileWatcher) RegisterHandler(handler FileChangeHandler) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.handlers = append(w.handlers, handler)
}

// Start starts watching for file changes
func (w *FileWatcher) Start(parentCtx context.Context) error {
	w.mu.Lock()
	if w.running {
		w.mu.Unlock()
		return nil
	}

	w.ctx, w.cancel = context.WithCancel(parentCtx)
	w.running = true
	w.mu.Unlock()

	go w.watchLoop()

	return nil
}

// Stop stops watching for file changes
func (w *FileWatcher) Stop() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if !w.running {
		return nil
	}

	w.cancel()
	w.running = false

	return w.watcher.Close()
}

// watchLoop is the main watching loop
func (w *FileWatcher) watchLoop() {
	for {
		select {
		case <-w.ctx.Done():
			return

		case event, ok := <-w.watcher.Events:
			if !ok {
				return
			}
			w.handleEvent(event)

		case err, ok := <-w.watcher.Errors:
			if !ok {
				return
			}
			// Log error but continue watching
			fmt.Fprintf(os.Stderr, "[FileWatcher] Error: %v\n", err)
		}
	}
}

// handleEvent handles a file system event
func (w *FileWatcher) handleEvent(event fsnotify.Event) {
	// Determine operation
	var operation string
	switch {
	case event.Op&fsnotify.Create == fsnotify.Create:
		operation = "create"
	case event.Op&fsnotify.Write == fsnotify.Write:
		operation = "write"
	case event.Op&fsnotify.Remove == fsnotify.Remove:
		operation = "remove"
	case event.Op&fsnotify.Rename == fsnotify.Rename:
		operation = "rename"
	case event.Op&fsnotify.Chmod == fsnotify.Chmod:
		operation = "chmod"
	default:
		return
	}

	changeEvent := FileChangeEvent{
		Path:      event.Name,
		Operation: operation,
		Timestamp: time.Now(),
	}

	// Call handlers
	w.mu.RLock()
	handlers := make([]FileChangeHandler, len(w.handlers))
	copy(handlers, w.handlers)
	w.mu.RUnlock()

	for _, handler := range handlers {
		go handler(changeEvent)
	}
}

// GetWatchedPaths returns the list of watched paths
func (w *FileWatcher) GetWatchedPaths() []string {
	w.mu.RLock()
	defer w.mu.RUnlock()

	paths := make([]string, 0, len(w.paths))
	for path := range w.paths {
		paths = append(paths, path)
	}
	return paths
}

// UpdateWatchPaths updates the set of watched paths
func (w *FileWatcher) UpdateWatchPaths(newPaths []string) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	// Build set of new paths
	newPathSet := make(map[string]bool)
	for _, path := range newPaths {
		absPath, err := filepath.Abs(path)
		if err != nil {
			continue
		}
		newPathSet[absPath] = true
	}

	// Remove paths no longer needed
	for path := range w.paths {
		if !newPathSet[path] {
			w.watcher.Remove(path)
			delete(w.paths, path)
		}
	}

	// Add new paths
	for path := range newPathSet {
		if !w.paths[path] {
			if err := w.watcher.Add(path); err == nil {
				w.paths[path] = true
			}
		}
	}

	return nil
}

// MatchesMatcher checks if a string matches a glob matcher pattern
func MatchesMatcher(pattern, value string) bool {
	if pattern == "" || pattern == "*" {
		return true
	}

	// Try glob matching
	matched, err := filepath.Match(pattern, value)
	if err != nil {
		// Invalid pattern, try simple contains
		return strings.Contains(value, pattern)
	}

	return matched
}
