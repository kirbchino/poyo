// Package plugin provides compatibility layer for Poyo plugins
// Supports native Poyo format plus compatibility with Claude Code and OpenClaw formats
package plugin

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// PluginSpec represents the plugin specification format
type PluginSpec string

const (
	SpecKirby      PluginSpec = "kirby"       // Kirby (星之卡比) native spec
	SpecClaudeCode PluginSpec = "claude-code" // Original CC spec (兼容)
	SpecOpenClaw   PluginSpec = "openclaw"    // OpenClaw spec (兼容)
	SpecUnified    PluginSpec = "unified"     // Unified Go spec
)

// ClaudeCodeManifest represents the original Claude Code plugin.json format
type ClaudeCodeManifest struct {
	Name        string                 `json:"name"`
	Version     string                 `json:"version"`
	Description string                 `json:"description"`
	Type        string                 `json:"type"` // lua, mcp, script
	Main        string                 `json:"main,omitempty"`
	Entrypoint  string                 `json:"entrypoint,omitempty"`
	Tools       []CCToolDef            `json:"tools,omitempty"`
	Commands    []CCCommandDef         `json:"commands,omitempty"`
	Hooks       []CCHookDef            `json:"hooks,omitempty"`
	Config      map[string]interface{} `json:"config,omitempty"`
	Permissions []string               `json:"permissions,omitempty"`
	Dependencies []string              `json:"dependencies,omitempty"`
}

// CCToolDef represents a tool definition in CC format
type CCToolDef struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	InputSchema map[string]interface{} `json:"input_schema"`
	Handler     string                 `json:"handler,omitempty"`
	Concurrent  bool                   `json:"concurrent,omitempty"`
}

// CCCommandDef represents a command definition in CC format
type CCCommandDef struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Handler     string `json:"handler,omitempty"`
}

// CCHookDef represents a hook definition in CC format
type CCHookDef struct {
	Type     string `json:"type"`     // PreToolUse, PostToolUse, etc.
	Priority int    `json:"priority"` // Higher = earlier execution
	Handler  string `json:"handler"`
}

// OpenClawManifest represents the OpenClaw plugin format
type OpenClawManifest struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Version     string                 `json:"version"`
	Description string                 `json:"description"`
	Author      string                 `json:"author,omitempty"`
	License     string                 `json:"license,omitempty"`
	Repository  string                 `json:"repository,omitempty"`
	Runtime     string                 `json:"runtime"` // lua, python, node, wasm, native
	Entry       string                 `json:"entry"`
	Provides    OpenClawProvides       `json:"provides"`
	Config      map[string]interface{} `json:"config,omitempty"`
	Requires    []OpenClawRequire      `json:"requires,omitempty"`
}

// OpenClawProvides defines what the plugin provides
type OpenClawProvides struct {
	Tools    []OpenClawTool `json:"tools,omitempty"`
	Commands []OpenClawCmd  `json:"commands,omitempty"`
	Hooks    []OpenClawHook `json:"hooks,omitempty"`
	Routes   []OpenClawRoute `json:"routes,omitempty"`
}

// OpenClawTool represents a tool in OpenClaw format
type OpenClawTool struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Parameters  map[string]interface{} `json:"parameters"`
	Returns     map[string]interface{} `json:"returns,omitempty"`
	Examples    []map[string]interface{} `json:"examples,omitempty"`
}

// OpenClawCmd represents a command in OpenClaw format
type OpenClawCmd struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Usage       string `json:"usage,omitempty"`
}

// OpenClawHook represents a hook in OpenClaw format
type OpenClawHook struct {
	Event    string `json:"event"`    // tool.use.before, tool.use.after, etc.
	Priority int    `json:"priority"` // 0-100, higher = earlier
	Handler  string `json:"handler"`
}

// OpenClawRoute represents an HTTP route in OpenClaw format
type OpenClawRoute struct {
	Method  string `json:"method"`
	Path    string `json:"path"`
	Handler string `json:"handler"`
}

// OpenClawRequire represents a requirement in OpenClaw format
type OpenClawRequire struct {
	Type    string `json:"type"`    // permission, plugin, api
	Name    string `json:"name"`
	Version string `json:"version,omitempty"`
}

// CompatibilityLayer handles conversion between plugin specs
type CompatibilityLayer struct {
	pluginDirs []string
}

// NewCompatibilityLayer creates a new compatibility layer
func NewCompatibilityLayer(pluginDirs []string) *CompatibilityLayer {
	return &CompatibilityLayer{pluginDirs: pluginDirs}
}

// DetectSpec detects the plugin specification format
func (c *CompatibilityLayer) DetectSpec(manifestPath string) (PluginSpec, error) {
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		return "", err
	}

	// Try to detect format
	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		return "", err
	}

	// Check for OpenClaw-specific fields
	if _, ok := raw["runtime"]; ok {
		if _, ok := raw["provides"]; ok {
			return SpecOpenClaw, nil
		}
	}

	// Check for Claude Code format (legacy compatibility)
	if _, ok := raw["type"]; ok {
		return SpecClaudeCode, nil
	}

	// Default to unified
	return SpecUnified, nil
}

// ParseManifest parses a manifest file and returns a unified Plugin
func (c *CompatibilityLayer) ParseManifest(manifestPath string) (*Plugin, error) {
	spec, err := c.DetectSpec(manifestPath)
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(manifestPath)
	if err != nil {
		return nil, err
	}

	switch spec {
	case SpecClaudeCode:
		return c.parseClaudeCode(data, manifestPath)
	case SpecOpenClaw:
		return c.parseOpenClaw(data, manifestPath)
	default:
		return c.parseUnified(data, manifestPath)
	}
}

// parseClaudeCode parses a Claude Code format manifest (legacy compatibility)
func (c *CompatibilityLayer) parseClaudeCode(data []byte, manifestPath string) (*Plugin, error) {
	var cc ClaudeCodeManifest
	if err := json.Unmarshal(data, &cc); err != nil {
		return nil, err
	}

	plugin := &Plugin{
		ID:          filepath.Base(filepath.Dir(manifestPath)),
		Name:        cc.Name,
		Version:     cc.Version,
		Description: cc.Description,
		Type:        PluginType(cc.Type),
		Path:        filepath.Dir(manifestPath),
		Enabled:     true,
		Config:      cc.Config,
	}

	// Set main/entrypoint
	if cc.Main != "" {
		plugin.Main = cc.Main
	} else if cc.Entrypoint != "" {
		plugin.Main = cc.Entrypoint
	}

	// Convert tools
	for _, t := range cc.Tools {
		plugin.Tools = append(plugin.Tools, ToolDefinition{
			Name:        t.Name,
			Description: t.Description,
			InputSchema: t.InputSchema,
			Handler:     t.Handler,
			Concurrent:  t.Concurrent,
		})
	}

	// Convert commands
	for _, cmd := range cc.Commands {
		plugin.Commands = append(plugin.Commands, CommandDefinition{
			Name:        cmd.Name,
			Description: cmd.Description,
			Handler:     cmd.Handler,
		})
	}

	// Convert hooks
	for _, h := range cc.Hooks {
		plugin.Hooks = append(plugin.Hooks, HookDefinition{
			Type:     h.Type,
			Priority: h.Priority,
			Handler:  h.Handler,
		})
	}

	// Convert permissions
	for _, p := range cc.Permissions {
		plugin.Permissions = append(plugin.Permissions, Permission{
			Type: PermissionType(p),
		})
	}

	return plugin, nil
}

// parseOpenClaw parses an OpenClaw manifest
func (c *CompatibilityLayer) parseOpenClaw(data []byte, manifestPath string) (*Plugin, error) {
	var oc OpenClawManifest
	if err := json.Unmarshal(data, &oc); err != nil {
		return nil, err
	}

	plugin := &Plugin{
		ID:          oc.ID,
		Name:        oc.Name,
		Version:     oc.Version,
		Description: oc.Description,
		Author:      oc.Author,
		License:     oc.License,
		Repository:  oc.Repository,
		Type:        c.openClawRuntimeToType(oc.Runtime),
		Path:        filepath.Dir(manifestPath),
		Enabled:     true,
		Main:        oc.Entry,
		Config:      oc.Config,
	}

	// Convert tools
	for _, t := range oc.Provides.Tools {
		plugin.Tools = append(plugin.Tools, ToolDefinition{
			Name:        t.Name,
			Description: t.Description,
			InputSchema: t.Parameters,
		})
	}

	// Convert commands
	for _, cmd := range oc.Provides.Commands {
		plugin.Commands = append(plugin.Commands, CommandDefinition{
			Name:        cmd.Name,
			Description: cmd.Description,
		})
	}

	// Convert hooks (OpenClaw uses different event names)
	for _, h := range oc.Provides.Hooks {
		plugin.Hooks = append(plugin.Hooks, HookDefinition{
			Type:     c.openClawEventToCC(h.Event),
			Priority: h.Priority,
			Handler:  h.Handler,
		})
	}

	// Convert requirements to permissions
	for _, r := range oc.Requires {
		if r.Type == "permission" {
			plugin.Permissions = append(plugin.Permissions, Permission{
				Type: PermissionType(r.Name),
			})
		}
	}

	return plugin, nil
}

// parseUnified parses a unified manifest
func (c *CompatibilityLayer) parseUnified(data []byte, manifestPath string) (*Plugin, error) {
	var plugin Plugin
	if err := json.Unmarshal(data, &plugin); err != nil {
		return nil, err
	}

	if plugin.ID == "" {
		plugin.ID = filepath.Base(filepath.Dir(manifestPath))
	}
	plugin.Path = filepath.Dir(manifestPath)

	return &plugin, nil
}

// openClawRuntimeToType converts OpenClaw runtime to plugin type
func (c *CompatibilityLayer) openClawRuntimeToType(runtime string) PluginType {
	switch runtime {
	case "lua":
		return PluginTypeLua
	case "python":
		return PluginTypeScript
	case "node", "javascript":
		return PluginTypeScript
	case "wasm":
		return PluginTypeWasm
	case "native", "go":
		return PluginTypeNative
	case "mcp":
		return PluginTypeMCP
	default:
		return PluginTypeScript
	}
}

// openClawEventToCC converts OpenClaw event names to CC hook types
func (c *CompatibilityLayer) openClawEventToCC(event string) string {
	mapping := map[string]string{
		"tool.use.before":    "PreToolUse",
		"tool.use.after":     "PostToolUse",
		"prompt.send.before": "PrePrompt",
		"prompt.send.after":  "PostPrompt",
		"error.occurred":     "OnError",
		"plugin.load":        "OnLoad",
		"plugin.unload":      "OnUnload",
	}

	if mapped, ok := mapping[event]; ok {
		return mapped
	}
	return event
}

// ExportToClaudeCode exports a plugin to Claude Code format
func (c *CompatibilityLayer) ExportToClaudeCode(plugin *Plugin) ([]byte, error) {
	cc := ClaudeCodeManifest{
		Name:        plugin.Name,
		Version:     plugin.Version,
		Description: plugin.Description,
		Type:        string(plugin.Type),
		Main:        plugin.Main,
		Config:      plugin.Config,
	}

	for _, t := range plugin.Tools {
		var inputSchema map[string]interface{}
		if t.InputSchema != nil {
			if schema, ok := t.InputSchema.(map[string]interface{}); ok {
				inputSchema = schema
			}
		}
		cc.Tools = append(cc.Tools, CCToolDef{
			Name:        t.Name,
			Description: t.Description,
			InputSchema: inputSchema,
			Handler:     t.Handler,
			Concurrent:  t.Concurrent,
		})
	}

	for _, cmd := range plugin.Commands {
		cc.Commands = append(cc.Commands, CCCommandDef{
			Name:        cmd.Name,
			Description: cmd.Description,
			Handler:     cmd.Handler,
		})
	}

	for _, h := range plugin.Hooks {
		cc.Hooks = append(cc.Hooks, CCHookDef{
			Type:     h.Type,
			Priority: h.Priority,
			Handler:  h.Handler,
		})
	}

	for _, p := range plugin.Permissions {
		cc.Permissions = append(cc.Permissions, string(p.Type))
	}

	return json.MarshalIndent(cc, "", "  ")
}

// ExportToOpenClaw exports a plugin to OpenClaw format
func (c *CompatibilityLayer) ExportToOpenClaw(plugin *Plugin) ([]byte, error) {
	oc := OpenClawManifest{
		ID:          plugin.ID,
		Name:        plugin.Name,
		Version:     plugin.Version,
		Description: plugin.Description,
		Author:      plugin.Author,
		License:     plugin.License,
		Repository:  plugin.Repository,
		Runtime:     c.pluginTypeToOpenClawRuntime(plugin.Type),
		Entry:       plugin.Main,
		Config:      plugin.Config,
	}

	for _, t := range plugin.Tools {
		var parameters map[string]interface{}
		if t.InputSchema != nil {
			if schema, ok := t.InputSchema.(map[string]interface{}); ok {
				parameters = schema
			}
		}
		oc.Provides.Tools = append(oc.Provides.Tools, OpenClawTool{
			Name:        t.Name,
			Description: t.Description,
			Parameters:  parameters,
		})
	}

	for _, cmd := range plugin.Commands {
		oc.Provides.Commands = append(oc.Provides.Commands, OpenClawCmd{
			Name:        cmd.Name,
			Description: cmd.Description,
		})
	}

	for _, h := range plugin.Hooks {
		oc.Provides.Hooks = append(oc.Provides.Hooks, OpenClawHook{
			Event:    c.ccHookToOpenClawEvent(h.Type),
			Priority: h.Priority,
			Handler:  h.Handler,
		})
	}

	for _, p := range plugin.Permissions {
		oc.Requires = append(oc.Requires, OpenClawRequire{
			Type: "permission",
			Name: string(p.Type),
		})
	}

	return json.MarshalIndent(oc, "", "  ")
}

// pluginTypeToOpenClawRuntime converts plugin type to OpenClaw runtime
func (c *CompatibilityLayer) pluginTypeToOpenClawRuntime(t PluginType) string {
	switch t {
	case PluginTypeLua:
		return "lua"
	case PluginTypeWasm:
		return "wasm"
	case PluginTypeNative:
		return "native"
	case PluginTypeMCP:
		return "mcp"
	default:
		return "script"
	}
}

// ccHookToOpenClawEvent converts CC hook types to OpenClaw event names
func (c *CompatibilityLayer) ccHookToOpenClawEvent(hookType string) string {
	mapping := map[string]string{
		"PreToolUse":  "tool.use.before",
		"PostToolUse": "tool.use.after",
		"PrePrompt":   "prompt.send.before",
		"PostPrompt":  "prompt.send.after",
		"OnError":     "error.occurred",
		"OnLoad":      "plugin.load",
		"OnUnload":    "plugin.unload",
	}

	if mapped, ok := mapping[hookType]; ok {
		return mapped
	}
	return hookType
}

// DiscoverPlugins discovers plugins from all directories
func (c *CompatibilityLayer) DiscoverPlugins() ([]*Plugin, error) {
	var plugins []*Plugin

	for _, dir := range c.pluginDirs {
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}

		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}

			pluginPath := filepath.Join(dir, entry.Name())

			// Try different manifest files
			manifestFiles := []string{
				"plugin.json",      // CC format
				"openclaw.json",    // OpenClaw format
				"manifest.json",    // Generic
			}

			for _, manifestFile := range manifestFiles {
				manifestPath := filepath.Join(pluginPath, manifestFile)
				if _, err := os.Stat(manifestPath); err == nil {
					plugin, err := c.ParseManifest(manifestPath)
					if err != nil {
						fmt.Printf("Warning: failed to parse %s: %v\n", manifestPath, err)
						continue
					}
					plugins = append(plugins, plugin)
					break
				}
			}
		}
	}

	return plugins, nil
}
