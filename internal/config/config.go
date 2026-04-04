// Package config handles configuration file management
package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// Config represents the application configuration
type Config struct {
	// API configuration
	API APIConfig `json:"api"`

	// Model configuration
	Model string `json:"model,omitempty"`

	// Permission mode
	PermissionMode string `json:"permission_mode,omitempty"`

	// Maximum turns (0 = unlimited)
	MaxTurns int `json:"max_turns,omitempty"`

	// Debug mode
	Debug bool `json:"debug,omitempty"`

	// Tool configurations
	Tools ToolConfigs `json:"tools,omitempty"`

	// MCP server configurations
	MCPServers map[string]MCPServerConfig `json:"mcp_servers,omitempty"`
}

// APIConfig contains API-related configuration
type APIConfig struct {
	// Base URL for API
	BaseURL string `json:"base_url,omitempty"`

	// API key (can also be set via POYO_API_KEY env)
	APIKey string `json:"api_key,omitempty"`

	// API type: "anthropic" or "openai"
	Type string `json:"type,omitempty"`

	// Custom headers
	CustomHeaders map[string]string `json:"custom_headers,omitempty"`

	// Timeout in seconds
	Timeout int `json:"timeout,omitempty"`

	// Default max tokens
	MaxTokens int `json:"max_tokens,omitempty"`
}

// ToolConfigs contains tool-specific configurations
type ToolConfigs struct {
	// Enable/disable specific tools
	Enabled  []string `json:"enabled,omitempty"`
	Disabled []string `json:"disabled,omitempty"`

	// Bash tool configuration
	Bash BashToolConfig `json:"bash,omitempty"`

	// File tool configuration
	File FileToolConfig `json:"file,omitempty"`
}

// BashToolConfig contains bash tool configuration
type BashToolConfig struct {
	// Allowed commands (glob patterns)
	Allowed []string `json:"allowed,omitempty"`

	// Blocked commands (glob patterns)
	Blocked []string `json:"blocked,omitempty"`

	// Default timeout in milliseconds
	Timeout int `json:"timeout,omitempty"`
}

// FileToolConfig contains file tool configuration
type FileToolConfig struct {
	// Allowed paths
	AllowedPaths []string `json:"allowed_paths,omitempty"`

	// Max file size to read (in bytes)
	MaxFileSize int64 `json:"max_file_size,omitempty"`
}

// MCPServerConfig contains MCP server configuration
type MCPServerConfig struct {
	// Command to start the MCP server
	Command string `json:"command"`

	// Arguments for the command
	Args []string `json:"args,omitempty"`

	// Environment variables
	Env map[string]string `json:"env,omitempty"`

	// Working directory
	Cwd string `json:"cwd,omitempty"`
}

// DefaultConfig returns the default configuration
func DefaultConfig() *Config {
	return &Config{
		Model:          "claude-sonnet-4-6",
		PermissionMode: "default",
		API: APIConfig{
			Type:     "openai",
			Timeout:  120,
			MaxTokens: 4096,
		},
		Tools: ToolConfigs{
			Bash: BashToolConfig{
				Timeout: 120000,
			},
			File: FileToolConfig{
				MaxFileSize: 10 * 1024 * 1024, // 10MB
			},
		},
		MCPServers: make(map[string]MCPServerConfig),
	}
}

// ConfigPaths returns the default configuration file paths in order of priority
func ConfigPaths() []string {
	// Priority: local > user > system
	paths := []string{}

	// Local config in current directory
	if cwd, err := os.Getwd(); err == nil {
		paths = append(paths,
			filepath.Join(cwd, ".poyo.json"),
			filepath.Join(cwd, ".poyo", "config.json"),
		)
	}

	// User config
	if home, err := os.UserHomeDir(); err == nil {
		paths = append(paths,
			filepath.Join(home, ".poyo.json"),
			filepath.Join(home, ".config", "poyo", "config.json"),
		)
	}

	// System config
	paths = append(paths, "/etc/poyo/config.json")

	return paths
}

// Load loads configuration from the default paths
func Load() (*Config, error) {
	config := DefaultConfig()

	for _, path := range ConfigPaths() {
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}

		if err := json.Unmarshal(data, config); err != nil {
			return nil, fmt.Errorf("parse config %s: %w", path, err)
		}

		// Use first found config file
		break
	}

	// Apply environment variable overrides
	applyEnvOverrides(config)

	return config, nil
}

// LoadFromPath loads configuration from a specific path
func LoadFromPath(path string) (*Config, error) {
	config := DefaultConfig()

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}

	if err := json.Unmarshal(data, config); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}

	applyEnvOverrides(config)

	return config, nil
}

// applyEnvOverrides applies environment variable overrides
func applyEnvOverrides(config *Config) {
	// API key from environment
	if apiKey := os.Getenv("POYO_API_KEY"); apiKey != "" {
		config.API.APIKey = apiKey
	}

	// Base URL from environment
	if baseURL := os.Getenv("POYO_API_URL"); baseURL != "" {
		config.API.BaseURL = baseURL
	}

	// Model from environment
	if model := os.Getenv("POYO_MODEL"); model != "" {
		config.Model = model
	}

	// Permission mode from environment
	if mode := os.Getenv("POYO_PERMISSION_MODE"); mode != "" {
		config.PermissionMode = mode
	}

	// Debug from environment
	if debug := os.Getenv("POYO_DEBUG"); debug == "true" || debug == "1" {
		config.Debug = true
	}
}

// Save saves the configuration to a file
func (c *Config) Save(path string) error {
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}

	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create config directory: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("write config: %w", err)
	}

	return nil
}

// Validate validates the configuration
func (c *Config) Validate() error {
	// Validate permission mode
	validModes := map[string]bool{
		"default":          true,
		"plan":             true,
		"acceptEdits":      true,
		"auto":             true,
		"bypassPermissions": true,
		"dontAsk":          true,
	}
	if !validModes[c.PermissionMode] {
		return fmt.Errorf("invalid permission mode: %s", c.PermissionMode)
	}

	// Validate API type
	validAPITypes := map[string]bool{
		"anthropic": true,
		"openai":    true,
	}
	if !validAPITypes[c.API.Type] {
		return fmt.Errorf("invalid API type: %s", c.API.Type)
	}

	return nil
}

// MergeFlags merges command line flags into the configuration
func (c *Config) MergeFlags(flags FlagOverrides) {
	if flags.Model != "" {
		c.Model = flags.Model
	}
	if flags.PermissionMode != "" {
		c.PermissionMode = flags.PermissionMode
	}
	if flags.MaxTurns > 0 {
		c.MaxTurns = flags.MaxTurns
	}
	if flags.Debug {
		c.Debug = true
	}
	if flags.APIURL != "" {
		c.API.BaseURL = flags.APIURL
	}
	if flags.APIKey != "" {
		c.API.APIKey = flags.APIKey
	}
	if flags.APIType != "" {
		c.API.Type = flags.APIType
	}
}

// FlagOverrides contains flag values to merge into config
type FlagOverrides struct {
	Model          string
	PermissionMode string
	MaxTurns       int
	Debug          bool
	APIURL         string
	APIKey         string
	APIType        string
}
