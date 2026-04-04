// Package mcp implements plugin management
package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"
)

// Plugin represents an MCP plugin
type Plugin struct {
	Name        string
	Version     string
	Description string
	Command     string
	Args        []string
	Env         map[string]string
	Disabled    bool

	// Runtime state
	process    *exec.Cmd
	client     *Client
	tools      map[string]ToolDefinition
	resources  map[string]ResourceDefinition
	prompts    map[string]PromptDefinition
	mu         sync.RWMutex
}

// ToolDefinition represents a tool from a plugin
type ToolDefinition struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	InputSchema map[string]interface{} `json:"inputSchema"`
}

// ResourceDefinition represents a resource from a plugin
type ResourceDefinition struct {
	URI         string `json:"uri"`
	Name        string `json:"name"`
	Description string `json:"description"`
	MimeType    string `json:"mimeType"`
}

// PromptDefinition represents a prompt from a plugin
type PromptDefinition struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

// PluginConfig represents plugin configuration
type PluginConfig struct {
	Name        string            `json:"name"`
	Version     string            `json:"version,omitempty"`
	Description string            `json:"description,omitempty"`
	Command     string            `json:"command"`
	Args        []string          `json:"args,omitempty"`
	Env         map[string]string `json:"env,omitempty"`
	Disabled    bool              `json:"disabled,omitempty"`
}

// PluginManager manages MCP plugins
type PluginManager struct {
	mu      sync.RWMutex
	plugins map[string]*Plugin
	configs map[string]PluginConfig
	configPath string
}

// NewPluginManager creates a new plugin manager
func NewPluginManager() *PluginManager {
	return &PluginManager{
		plugins: make(map[string]*Plugin),
		configs: make(map[string]PluginConfig),
	}
}

// LoadConfig loads plugin configuration from a file
func (m *PluginManager) LoadConfig(path string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // No config file is OK
		}
		return fmt.Errorf("failed to read config: %w", err)
	}

	var configs []PluginConfig
	if err := json.Unmarshal(data, &configs); err != nil {
		return fmt.Errorf("failed to parse config: %w", err)
	}

	m.configPath = path
	for _, config := range configs {
		m.configs[config.Name] = config
	}

	return nil
}

// SaveConfig saves plugin configuration to a file
func (m *PluginManager) SaveConfig(path string) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	configs := make([]PluginConfig, 0, len(m.configs))
	for _, config := range m.configs {
		configs = append(configs, config)
	}

	data, err := json.MarshalIndent(configs, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	return os.WriteFile(path, data, 0644)
}

// RegisterPlugin registers a new plugin
func (m *PluginManager) RegisterPlugin(config PluginConfig) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.configs[config.Name]; exists {
		return fmt.Errorf("plugin already registered: %s", config.Name)
	}

	m.configs[config.Name] = config
	return nil
}

// UnregisterPlugin unregisters a plugin
func (m *PluginManager) UnregisterPlugin(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if plugin, exists := m.plugins[name]; exists {
		plugin.Stop()
		delete(m.plugins, name)
	}

	delete(m.configs, name)
	return nil
}

// StartPlugin starts a plugin
func (m *PluginManager) StartPlugin(ctx context.Context, name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	config, exists := m.configs[name]
	if !exists {
		return fmt.Errorf("plugin not found: %s", name)
	}

	if config.Disabled {
		return fmt.Errorf("plugin is disabled: %s", name)
	}

	if plugin, exists := m.plugins[name]; exists {
		if plugin.client != nil {
			return nil // Already running
		}
	}

	// Create plugin instance
	plugin := &Plugin{
		Name:        config.Name,
		Version:     config.Version,
		Description: config.Description,
		Command:     config.Command,
		Args:        config.Args,
		Env:         config.Env,
		Disabled:    config.Disabled,
		tools:       make(map[string]ToolDefinition),
		resources:   make(map[string]ResourceDefinition),
		prompts:     make(map[string]PromptDefinition),
	}

	// Start plugin process
	if err := plugin.Start(ctx); err != nil {
		return fmt.Errorf("failed to start plugin: %w", err)
	}

	m.plugins[name] = plugin
	return nil
}

// StopPlugin stops a plugin
func (m *PluginManager) StopPlugin(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	plugin, exists := m.plugins[name]
	if !exists {
		return fmt.Errorf("plugin not found: %s", name)
	}

	return plugin.Stop()
}

// StartAll starts all registered plugins
func (m *PluginManager) StartAll(ctx context.Context) error {
	m.mu.RLock()
	configs := make([]PluginConfig, 0, len(m.configs))
	for _, config := range m.configs {
		configs = append(configs, config)
	}
	m.mu.RUnlock()

	var errors []error
	for _, config := range configs {
		if config.Disabled {
			continue
		}
		if err := m.StartPlugin(ctx, config.Name); err != nil {
			errors = append(errors, fmt.Errorf("%s: %w", config.Name, err))
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("errors starting plugins: %v", errors)
	}
	return nil
}

// StopAll stops all plugins
func (m *PluginManager) StopAll() error {
	m.mu.RLock()
	plugins := make([]*Plugin, 0, len(m.plugins))
	for _, plugin := range m.plugins {
		plugins = append(plugins, plugin)
	}
	m.mu.RUnlock()

	var errors []error
	for _, plugin := range plugins {
		if err := plugin.Stop(); err != nil {
			errors = append(errors, fmt.Errorf("%s: %w", plugin.Name, err))
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("errors stopping plugins: %v", errors)
	}
	return nil
}

// GetPlugin gets a plugin by name
func (m *PluginManager) GetPlugin(name string) (*Plugin, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	plugin, exists := m.plugins[name]
	if !exists {
		return nil, fmt.Errorf("plugin not found: %s", name)
	}
	return plugin, nil
}

// ListPlugins lists all plugins
func (m *PluginManager) ListPlugins() []PluginInfo {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var infos []PluginInfo
	for name, config := range m.configs {
		info := PluginInfo{
			Name:        name,
			Version:     config.Version,
			Description: config.Description,
			Disabled:    config.Disabled,
		}

		if plugin, exists := m.plugins[name]; exists {
			info.Running = plugin.client != nil
			info.Tools = len(plugin.tools)
			info.Resources = len(plugin.resources)
			info.Prompts = len(plugin.prompts)
		}

		infos = append(infos, info)
	}

	return infos
}

// PluginInfo contains plugin information
type PluginInfo struct {
	Name        string `json:"name"`
	Version     string `json:"version"`
	Description string `json:"description"`
	Disabled    bool   `json:"disabled"`
	Running     bool   `json:"running"`
	Tools       int    `json:"tools"`
	Resources   int    `json:"resources"`
	Prompts     int    `json:"prompts"`
}

// GetAllTools returns all tools from all plugins
func (m *PluginManager) GetAllTools() map[string]ToolDefinition {
	m.mu.RLock()
	defer m.mu.RUnlock()

	tools := make(map[string]ToolDefinition)
	for _, plugin := range m.plugins {
		for name, tool := range plugin.tools {
			tools[plugin.Name+"."+name] = tool
		}
	}
	return tools
}

// CallTool calls a tool on a plugin
func (m *PluginManager) CallTool(ctx context.Context, pluginName, toolName string, args map[string]interface{}) (*ToolResult, error) {
	m.mu.RLock()
	plugin, exists := m.plugins[pluginName]
	m.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("plugin not found: %s", pluginName)
	}

	return plugin.CallTool(ctx, toolName, args)
}

// Start starts the plugin process
func (p *Plugin) Start(ctx context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	// Create command
	cmd := exec.CommandContext(ctx, p.Command, p.Args...)

	// Set environment
	cmd.Env = os.Environ()
	for k, v := range p.Env {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, v))
	}

	// Set up pipes
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdin pipe: %w", err)
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	// Start process
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start process: %w", err)
	}

	p.process = cmd

	// Create client
	p.client = NewClient("poyo", "1.0.0", stdout, stdin)

	// Initialize connection
	initResult, err := p.client.Initialize(ctx)
	if err != nil {
		p.Stop()
		return fmt.Errorf("failed to initialize: %w", err)
	}

	p.Version = initResult.ServerInfo.Version

	// Load tools
	if initResult.Capabilities.Tools != nil {
		tools, err := p.client.ListTools(ctx)
		if err != nil {
			p.Stop()
			return fmt.Errorf("failed to list tools: %w", err)
		}

		for _, tool := range tools {
			p.tools[tool.Name] = ToolDefinition{
				Name:        tool.Name,
				Description: tool.Description,
				InputSchema: tool.InputSchema.(map[string]interface{}),
			}
		}
	}

	return nil
}

// Stop stops the plugin process
func (p *Plugin) Stop() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.process == nil {
		return nil
	}

	// Send shutdown signal
	if p.process.Process != nil {
		if err := p.process.Process.Signal(os.Interrupt); err != nil {
			p.process.Process.Kill()
		}
	}

	// Wait for process to exit
	done := make(chan error, 1)
	go func() {
		done <- p.process.Wait()
	}()

	select {
	case <-time.After(5 * time.Second):
		if p.process.Process != nil {
			p.process.Process.Kill()
		}
	case <-done:
	}

	p.process = nil
	p.client = nil
	return nil
}

// CallTool calls a tool on this plugin
func (p *Plugin) CallTool(ctx context.Context, name string, args map[string]interface{}) (*ToolResult, error) {
	p.mu.RLock()
	client := p.client
	p.mu.RUnlock()

	if client == nil {
		return nil, fmt.Errorf("plugin not running: %s", p.Name)
	}

	return client.CallTool(ctx, name, args)
}

// GetTools returns the plugin's tools
func (p *Plugin) GetTools() map[string]ToolDefinition {
	p.mu.RLock()
	defer p.mu.RUnlock()

	tools := make(map[string]ToolDefinition)
	for k, v := range p.tools {
		tools[k] = v
	}
	return tools
}

// GetResources returns the plugin's resources
func (p *Plugin) GetResources() map[string]ResourceDefinition {
	p.mu.RLock()
	defer p.mu.RUnlock()

	resources := make(map[string]ResourceDefinition)
	for k, v := range p.resources {
		resources[k] = v
	}
	return resources
}

// IsRunning returns whether the plugin is running
func (p *Plugin) IsRunning() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.client != nil
}

// DefaultPluginConfigPath returns the default plugin config path
func DefaultPluginConfigPath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	configDir := filepath.Join(homeDir, ".claude")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return "", err
	}

	return filepath.Join(configDir, "plugins.json"), nil
}

// Helper to get os environment
func osEnviron() []string {
	return os.Environ()
}
