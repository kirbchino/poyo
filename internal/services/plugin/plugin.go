// Package plugin provides a plugin system for extending Poyo functionality
package plugin

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Plugin represents a loaded plugin
type Plugin struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Version     string                 `json:"version"`
	Description string                 `json:"description"`
	Author      string                 `json:"author,omitempty"`
	License     string                 `json:"license,omitempty"`
	Repository  string                 `json:"repository,omitempty"`
	Main        string                 `json:"main"`      // Entry point (script or binary)
	Type        PluginType             `json:"type"`      // native, script, wasm, lua, mcp
	Enabled     bool                   `json:"enabled"`
	Config      map[string]interface{} `json:"config,omitempty"`
	Hooks       []HookDefinition       `json:"hooks,omitempty"`
	Tools       []ToolDefinition       `json:"tools,omitempty"`
	Commands    []CommandDefinition    `json:"commands,omitempty"`
	Permissions []Permission           `json:"permissions,omitempty"`
	Path        string                 `json:"-"` // Plugin directory path
}

// PluginType represents the type of plugin
type PluginType string

const (
	PluginTypeNative PluginType = "native"  // Go plugin
	PluginTypeScript PluginType = "script"  // Shell/Python script
	PluginTypeWasm   PluginType = "wasm"    // WebAssembly
	PluginTypeLua    PluginType = "lua"     // Lua script
	PluginTypeMCP    PluginType = "mcp"     // MCP server
)

// Permission represents a plugin permission
type Permission struct {
	Type PermissionType `json:"type"`
}

// PermissionType represents a permission type
type PermissionType string

const (
	PermissionFileRead   PermissionType = "file:read"
	PermissionFileWrite  PermissionType = "file:write"
	PermissionNetwork    PermissionType = "network"
	PermissionExecute    PermissionType = "execute"
	PermissionEnv        PermissionType = "env"
	PermissionWorkspace  PermissionType = "workspace"
)

// HookDefinition defines a hook provided by a plugin
type HookDefinition struct {
	Type     string `json:"type"`     // PreToolUse, PostToolUse, etc.
	Priority int    `json:"priority"` // Execution priority
	Handler  string `json:"handler,omitempty"`  // Handler function name
}

// ToolDefinition defines a tool provided by a plugin
type ToolDefinition struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	InputSchema interface{}            `json:"inputSchema"`
	Handler     string                 `json:"handler,omitempty"`     // Handler function name
	Concurrent  bool                   `json:"concurrent,omitempty"`  // Can run concurrently
}

// CommandDefinition defines a slash command provided by a plugin
type CommandDefinition struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Usage       string `json:"usage"`
	Handler     string `json:"handler,omitempty"` // Handler function name
}

// PluginManager manages plugins
type PluginManager struct {
	mu            sync.RWMutex
	plugins       map[string]*Plugin
	pluginDirs    []string
	configDir     string
	handlers      map[string]PluginHandler
	eventHandlers []EventHandler
}

// PluginHandler handles plugin operations
type PluginHandler interface {
	Load(ctx context.Context, plugin *Plugin) error
	Unload(ctx context.Context, plugin *Plugin) error
	Execute(ctx context.Context, plugin *Plugin, method string, input interface{}) (interface{}, error)
}

// EventHandler handles plugin events
type EventHandler func(event PluginEvent)

// PluginEvent represents a plugin event
type PluginEvent struct {
	Type      PluginEventType
	PluginID  string
	Timestamp time.Time
	Data      interface{}
}

// PluginEventType represents plugin event types
type PluginEventType string

const (
	PluginEventLoaded   PluginEventType = "loaded"
	PluginEventUnloaded PluginEventType = "unloaded"
	PluginEventEnabled  PluginEventType = "enabled"
	PluginEventDisabled PluginEventType = "disabled"
	PluginEventError    PluginEventType = "error"
)

// NewPluginManager creates a new plugin manager
func NewPluginManager(configDir string) *PluginManager {
	return &PluginManager{
		plugins:       make(map[string]*Plugin),
		pluginDirs:    []string{filepath.Join(configDir, "plugins")},
		configDir:     configDir,
		handlers:      make(map[string]PluginHandler),
		eventHandlers: make([]EventHandler, 0),
	}
}

// AddPluginDir adds a plugin directory
func (pm *PluginManager) AddPluginDir(dir string) {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	pm.pluginDirs = append(pm.pluginDirs, dir)
}

// RegisterHandler registers a plugin handler for a plugin type
func (pm *PluginManager) RegisterHandler(pluginType PluginType, handler PluginHandler) {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	pm.handlers[string(pluginType)] = handler
}

// OnEvent registers an event handler
func (pm *PluginManager) OnEvent(handler EventHandler) {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	pm.eventHandlers = append(pm.eventHandlers, handler)
}

// Discover discovers plugins in plugin directories
func (pm *PluginManager) Discover() ([]*Plugin, error) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	var discovered []*Plugin

	for _, dir := range pm.pluginDirs {
		entries, err := os.ReadDir(dir)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, fmt.Errorf("read plugin dir %s: %w", dir, err)
		}

		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}

			pluginPath := filepath.Join(dir, entry.Name())
			plugin, err := pm.loadPluginManifest(pluginPath)
			if err != nil {
				continue
			}

			plugin.Path = pluginPath
			discovered = append(discovered, plugin)
		}
	}

	return discovered, nil
}

// loadPluginManifest loads a plugin manifest
func (pm *PluginManager) loadPluginManifest(pluginPath string) (*Plugin, error) {
	manifestPath := filepath.Join(pluginPath, "plugin.json")
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		return nil, fmt.Errorf("read manifest: %w", err)
	}

	var plugin Plugin
	if err := json.Unmarshal(data, &plugin); err != nil {
		return nil, fmt.Errorf("parse manifest: %w", err)
	}

	if plugin.ID == "" {
		plugin.ID = filepath.Base(pluginPath)
	}

	return &plugin, nil
}

// Load loads a plugin
func (pm *PluginManager) Load(ctx context.Context, pluginID string) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	plugin, ok := pm.plugins[pluginID]
	if !ok {
		return fmt.Errorf("plugin %s not found", pluginID)
	}

	handler, ok := pm.handlers[string(plugin.Type)]
	if !ok {
		return fmt.Errorf("no handler for plugin type %s", plugin.Type)
	}

	if err := handler.Load(ctx, plugin); err != nil {
		pm.emitEvent(PluginEvent{
			Type:      PluginEventError,
			PluginID:  pluginID,
			Timestamp: time.Now(),
			Data:      err.Error(),
		})
		return err
	}

	pm.emitEvent(PluginEvent{
		Type:      PluginEventLoaded,
		PluginID:  pluginID,
		Timestamp: time.Now(),
	})

	return nil
}

// Unload unloads a plugin
func (pm *PluginManager) Unload(ctx context.Context, pluginID string) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	plugin, ok := pm.plugins[pluginID]
	if !ok {
		return fmt.Errorf("plugin %s not found", pluginID)
	}

	handler, ok := pm.handlers[string(plugin.Type)]
	if !ok {
		return nil
	}

	if err := handler.Unload(ctx, plugin); err != nil {
		return err
	}

	pm.emitEvent(PluginEvent{
		Type:      PluginEventUnloaded,
		PluginID:  pluginID,
		Timestamp: time.Now(),
	})

	return nil
}

// Enable enables a plugin
func (pm *PluginManager) Enable(pluginID string) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	plugin, ok := pm.plugins[pluginID]
	if !ok {
		return fmt.Errorf("plugin %s not found", pluginID)
	}

	plugin.Enabled = true

	pm.emitEvent(PluginEvent{
		Type:      PluginEventEnabled,
		PluginID:  pluginID,
		Timestamp: time.Now(),
	})

	return nil
}

// Disable disables a plugin
func (pm *PluginManager) Disable(pluginID string) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	plugin, ok := pm.plugins[pluginID]
	if !ok {
		return fmt.Errorf("plugin %s not found", pluginID)
	}

	plugin.Enabled = false

	pm.emitEvent(PluginEvent{
		Type:      PluginEventDisabled,
		PluginID:  pluginID,
		Timestamp: time.Now(),
	})

	return nil
}

// Register registers a plugin
func (pm *PluginManager) Register(plugin *Plugin) {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	pm.plugins[plugin.ID] = plugin
}

// Unregister unregisters a plugin
func (pm *PluginManager) Unregister(pluginID string) {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	delete(pm.plugins, pluginID)
}

// Get gets a plugin by ID
func (pm *PluginManager) Get(pluginID string) *Plugin {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	return pm.plugins[pluginID]
}

// GetAll gets all plugins
func (pm *PluginManager) GetAll() []*Plugin {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	plugins := make([]*Plugin, 0, len(pm.plugins))
	for _, p := range pm.plugins {
		plugins = append(plugins, p)
	}
	return plugins
}

// GetEnabled gets all enabled plugins
func (pm *PluginManager) GetEnabled() []*Plugin {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	var plugins []*Plugin
	for _, p := range pm.plugins {
		if p.Enabled {
			plugins = append(plugins, p)
		}
	}
	return plugins
}

// Execute executes a plugin method
func (pm *PluginManager) Execute(ctx context.Context, pluginID string, method string, input interface{}) (interface{}, error) {
	pm.mu.RLock()
	plugin, ok := pm.plugins[pluginID]
	pm.mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("plugin %s not found", pluginID)
	}

	if !plugin.Enabled {
		return nil, fmt.Errorf("plugin %s is disabled", pluginID)
	}

	handler, ok := pm.handlers[string(plugin.Type)]
	if !ok {
		return nil, fmt.Errorf("no handler for plugin type %s", plugin.Type)
	}

	return handler.Execute(ctx, plugin, method, input)
}

// emitEvent emits a plugin event
func (pm *PluginManager) emitEvent(event PluginEvent) {
	for _, handler := range pm.eventHandlers {
		handler(event)
	}
}

// PluginConfig represents plugin configuration
type PluginConfig struct {
	Plugins []PluginConfigEntry `json:"plugins"`
}

// PluginConfigEntry represents a plugin configuration entry
type PluginConfigEntry struct {
	ID      string                 `json:"id"`
	Enabled bool                   `json:"enabled"`
	Config  map[string]interface{} `json:"config,omitempty"`
}

// LoadConfig loads plugin configuration
func (pm *PluginManager) LoadConfig() error {
	configPath := filepath.Join(pm.configDir, "plugins.json")
	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("read plugin config: %w", err)
	}

	var config PluginConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return fmt.Errorf("parse plugin config: %w", err)
	}

	for _, entry := range config.Plugins {
		if plugin, ok := pm.plugins[entry.ID]; ok {
			plugin.Enabled = entry.Enabled
			plugin.Config = entry.Config
		}
	}

	return nil
}

// SaveConfig saves plugin configuration
func (pm *PluginManager) SaveConfig() error {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	var config PluginConfig
	for _, plugin := range pm.plugins {
		config.Plugins = append(config.Plugins, PluginConfigEntry{
			ID:      plugin.ID,
			Enabled: plugin.Enabled,
			Config:  plugin.Config,
		})
	}

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal plugin config: %w", err)
	}

	configPath := filepath.Join(pm.configDir, "plugins.json")
	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("write plugin config: %w", err)
	}

	return nil
}
