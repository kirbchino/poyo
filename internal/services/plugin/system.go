// Package plugin provides a unified plugin system for Poyo
package plugin

import (
	"context"
	"fmt"
	"sync"
)

// System represents the complete plugin system
type System struct {
	mu           sync.RWMutex
	manager      *PluginManager
	hookExecutor *HookExecutor
	hotReloader  *HotReloader
	luaHandler   *LuaPluginHandler
	mcpHandler   *MCPPluginHandler
	config       *SystemConfig
}

// SystemConfig configures the plugin system
type SystemConfig struct {
	ConfigDir    string
	WorkingDir   string
	EnableHotReload bool
	EnableMCP    bool
	EnableLua    bool
	ToolExecutor ToolExecutor
	LuaAPI       LuaAPI
}

// NewSystem creates a new plugin system
func NewSystem(config *SystemConfig) *System {
	manager := NewPluginManager(config.ConfigDir)

	system := &System{
		manager: manager,
		config:  config,
	}

	// Register handlers
	if config.EnableLua {
		system.luaHandler = NewLuaPluginHandler(config.WorkingDir, config.LuaAPI, config.ToolExecutor)
		manager.RegisterHandler(PluginType("lua"), system.luaHandler)
		// Also register as "native" Lua type
		manager.RegisterHandler(PluginTypeLua, system.luaHandler)
	}

	if config.EnableMCP {
		system.mcpHandler = NewMCPPluginHandler(config.WorkingDir)
		manager.RegisterHandler(PluginType("mcp"), system.mcpHandler)
		manager.RegisterHandler(PluginTypeMCP, system.mcpHandler)
	}

	// Script handler (shell, python, node, etc.)
	scriptHandler := NewScriptPluginHandler(config.WorkingDir)
	if config.ToolExecutor != nil {
		scriptHandler.SetToolExecutor(config.ToolExecutor)
	}
	manager.RegisterHandler(PluginTypeScript, scriptHandler)
	manager.RegisterHandler(PluginType("python"), scriptHandler)
	manager.RegisterHandler(PluginType("node"), scriptHandler)
	manager.RegisterHandler(PluginType("shell"), scriptHandler)

	// Create hook executor
	system.hookExecutor = NewHookExecutor(manager)

	// Create hot reloader
	if config.EnableHotReload {
		system.hotReloader = NewHotReloader(manager)
	}

	return system
}

// Initialize initializes the plugin system
func (s *System) Initialize(ctx context.Context) error {
	// Discover plugins
	plugins, err := s.manager.Discover()
	if err != nil {
		return fmt.Errorf("discover plugins: %w", err)
	}

	// Register all discovered plugins
	for _, plugin := range plugins {
		s.manager.Register(plugin)
	}

	// Load configuration
	if err := s.manager.LoadConfig(); err != nil {
		return fmt.Errorf("load plugin config: %w", err)
	}

	// Load all enabled plugins
	for _, plugin := range s.manager.GetEnabled() {
		if err := s.manager.Load(ctx, plugin.ID); err != nil {
			// Log error but continue
			fmt.Printf("Failed to load plugin %s: %v\n", plugin.ID, err)
		}
	}

	// Start hot reloader
	if s.hotReloader != nil {
		if err := s.hotReloader.Start(); err != nil {
			fmt.Printf("Failed to start hot reloader: %v\n", err)
		}
	}

	return nil
}

// Shutdown shuts down the plugin system
func (s *System) Shutdown(ctx context.Context) error {
	// Stop hot reloader
	if s.hotReloader != nil {
		s.hotReloader.Stop()
	}

	// Unload all plugins
	for _, plugin := range s.manager.GetAll() {
		s.manager.Unload(ctx, plugin.ID)
	}

	// Close Lua handler
	if s.luaHandler != nil {
		s.luaHandler.Close()
	}

	// Save configuration
	return s.manager.SaveConfig()
}

// Manager returns the plugin manager
func (s *System) Manager() *PluginManager {
	return s.manager
}

// Hooks returns the hook executor
func (s *System) Hooks() *HookExecutor {
	return s.hookExecutor
}

// PreToolUse executes PreToolUse hooks
func (s *System) PreToolUse(ctx context.Context, tool string, input map[string]interface{}) (*HookResult, error) {
	return s.hookExecutor.PreToolUse(ctx, tool, input)
}

// PostToolUse executes PostToolUse hooks
func (s *System) PostToolUse(ctx context.Context, tool string, input map[string]interface{}, success bool, err string, result interface{}) (*HookResult, error) {
	return s.hookExecutor.PostToolUse(ctx, tool, input, success, err, result)
}

// ExecuteTool executes a tool provided by a plugin
func (s *System) ExecuteTool(ctx context.Context, pluginID string, toolName string, input interface{}) (interface{}, error) {
	return s.manager.Execute(ctx, pluginID, toolName, input)
}

// RegisterPlugin registers a plugin programmatically
func (s *System) RegisterPlugin(plugin *Plugin) error {
	s.manager.Register(plugin)
	return s.manager.Load(context.Background(), plugin.ID)
}

// UnregisterPlugin unregisters a plugin
func (s *System) UnregisterPlugin(ctx context.Context, pluginID string) error {
	if err := s.manager.Unload(ctx, pluginID); err != nil {
		return err
	}
	s.manager.Unregister(pluginID)
	return nil
}

// GetTools returns all tools from all enabled plugins
func (s *System) GetTools() []ToolDefinition {
	var tools []ToolDefinition
	for _, plugin := range s.manager.GetEnabled() {
		tools = append(tools, plugin.Tools...)
	}
	return tools
}

// GetCommands returns all commands from all enabled plugins
func (s *System) GetCommands() []CommandDefinition {
	var commands []CommandDefinition
	for _, plugin := range s.manager.GetEnabled() {
		commands = append(commands, plugin.Commands...)
	}
	return commands
}

// SetToolExecutor sets the tool executor for Lua plugins
func (s *System) SetToolExecutor(executor ToolExecutor) {
	if s.luaHandler != nil {
		s.luaHandler.toolExec = executor
	}
}

// SetLuaContext sets the context for Lua plugin execution
func (s *System) SetLuaContext(ctx context.Context) {
	if s.luaHandler != nil {
		s.luaHandler.SetContext(ctx)
	}
}

// OnPluginEvent registers a plugin event handler
func (s *System) OnPluginEvent(handler EventHandler) {
	s.manager.OnEvent(handler)
}
