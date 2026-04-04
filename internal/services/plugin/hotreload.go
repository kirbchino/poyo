// Package plugin provides hot-reload support for plugins
package plugin

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

// HotReloader manages hot-reloading of plugins
type HotReloader struct {
	mu          sync.RWMutex
	manager     *PluginManager
	watcher     *fsnotify.Watcher
	watchedDirs map[string]bool
	pluginMTimes map[string]time.Time
	running     bool
	stopChan    chan struct{}
}

// NewHotReloader creates a new hot reloader
func NewHotReloader(manager *PluginManager) *HotReloader {
	return &HotReloader{
		manager:      manager,
		watchedDirs:  make(map[string]bool),
		pluginMTimes: make(map[string]time.Time),
		stopChan:     make(chan struct{}),
	}
}

// Start starts watching for plugin changes
func (r *HotReloader) Start() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.running {
		return nil
	}

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("create watcher: %w", err)
	}

	r.watcher = watcher
	r.running = true

	// Watch all plugin directories
	for _, dir := range r.manager.pluginDirs {
		if err := r.watchDir(dir); err == nil {
			r.watchedDirs[dir] = true
		}
	}

	// Start watching goroutine
	go r.watchLoop()

	return nil
}

// Stop stops watching for plugin changes
func (r *HotReloader) Stop() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if !r.running {
		return nil
	}

	close(r.stopChan)
	r.running = false

	if r.watcher != nil {
		return r.watcher.Close()
	}

	return nil
}

// watchDir adds a directory to the watch list
func (r *HotReloader) watchDir(dir string) error {
	// Watch the directory itself
	if err := r.watcher.Add(dir); err != nil {
		return err
	}

	// Watch all subdirectories (plugin directories)
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		if entry.IsDir() {
			pluginPath := filepath.Join(dir, entry.Name())
			r.watcher.Add(pluginPath)

			// Record initial modification time
			if info, err := entry.Info(); err == nil {
				r.pluginMTimes[pluginPath] = info.ModTime()
			}
		}
	}

	return nil
}

// watchLoop is the main watching loop
func (r *HotReloader) watchLoop() {
	for {
		select {
		case <-r.stopChan:
			return
		case event, ok := <-r.watcher.Events:
			if !ok {
				return
			}
			r.handleEvent(event)
		case err, ok := <-r.watcher.Errors:
			if !ok {
				return
			}
			// Log error (in production, would use proper logging)
			fmt.Printf("hot-reload watcher error: %v\n", err)
		}
	}
}

// handleEvent handles a filesystem event
func (r *HotReloader) handleEvent(event fsnotify.Event) {
	// Only handle write and create events
	if event.Op&fsnotify.Write == 0 && event.Op&fsnotify.Create == 0 {
		return
	}

	// Check if it's a plugin.json or main file
	filename := filepath.Base(event.Name)
	if filename != "plugin.json" && filename != "main.lua" && filename != "main.js" && filename != "main.py" {
		return
	}

	// Get plugin directory
	pluginDir := filepath.Dir(event.Name)

	// Check if this is a known plugin
	pluginID := filepath.Base(pluginDir)
	plugin := r.manager.Get(pluginID)

	if plugin == nil {
		// New plugin discovered
		r.loadPlugin(pluginDir)
		return
	}

	// Check modification time to avoid duplicate reloads
	info, err := os.Stat(event.Name)
	if err != nil {
		return
	}

	r.mu.RLock()
	lastMTime, exists := r.pluginMTimes[pluginDir]
	r.mu.RUnlock()

	if exists && info.ModTime().Sub(lastMTime) < time.Second {
		// Skip duplicate event
		return
	}

	r.mu.Lock()
	r.pluginMTimes[pluginDir] = info.ModTime()
	r.mu.Unlock()

	// Reload the plugin
	r.reloadPlugin(pluginDir, pluginID)
}

// loadPlugin loads a new plugin
func (r *HotReloader) loadPlugin(pluginDir string) {
	plugin, err := r.manager.loadPluginManifest(pluginDir)
	if err != nil {
		fmt.Printf("failed to load plugin manifest: %v\n", err)
		return
	}

	plugin.Path = pluginDir
	plugin.Enabled = true

	r.manager.Register(plugin)

	if err := r.manager.Load(context.Background(), plugin.ID); err != nil {
		fmt.Printf("failed to load plugin %s: %v\n", plugin.ID, err)
		return
	}

	r.manager.emitEvent(PluginEvent{
		Type:      PluginEventLoaded,
		PluginID:  plugin.ID,
		Timestamp: time.Now(),
		Data:      "hot-loaded",
	})

	fmt.Printf("Plugin %s hot-loaded\n", plugin.ID)
}

// reloadPlugin reloads an existing plugin
func (r *HotReloader) reloadPlugin(pluginDir string, pluginID string) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Unload old plugin
	if err := r.manager.Unload(ctx, pluginID); err != nil {
		fmt.Printf("failed to unload plugin %s: %v\n", pluginID, err)
	}

	// Load new manifest
	plugin, err := r.manager.loadPluginManifest(pluginDir)
	if err != nil {
		fmt.Printf("failed to reload plugin manifest: %v\n", err)
		return
	}

	// Update plugin
	plugin.Path = pluginDir
	plugin.Enabled = true
	r.manager.Register(plugin)

	// Load plugin
	if err := r.manager.Load(ctx, plugin.ID); err != nil {
		fmt.Printf("failed to reload plugin %s: %v\n", plugin.ID, err)
		return
	}

	r.manager.emitEvent(PluginEvent{
		Type:      PluginEventLoaded,
		PluginID:  pluginID,
		Timestamp: time.Now(),
		Data:      "hot-reloaded",
	})

	fmt.Printf("Plugin %s hot-reloaded\n", pluginID)
}

// WatchPlugin adds a specific plugin directory to watch
func (r *HotReloader) WatchPlugin(pluginPath string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.watcher == nil {
		return fmt.Errorf("watcher not started")
	}

	return r.watcher.Add(pluginPath)
}

// UnwatchPlugin removes a specific plugin directory from watch
func (r *HotReloader) UnwatchPlugin(pluginPath string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.watcher == nil {
		return fmt.Errorf("watcher not started")
	}

	return r.watcher.Remove(pluginPath)
}
